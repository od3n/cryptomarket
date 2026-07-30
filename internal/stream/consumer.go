package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// StreamKey is the Redis Stream key for market events.
	StreamKey = "market:events"
	// ConsumerGroup is the consumer group name for the realtime gateway.
	ConsumerGroup = "realtime-gateway"
	// ConsumerName identifies this consumer instance within the group.
	ConsumerName = "realtime-1"

	defaultBatchSize     = 100
	defaultBlockTime     = 5 * time.Second
	defaultClaimInterval = 30 * time.Second
	defaultClaimMinIdle  = 60 * time.Second
)

// ConsumerConfig holds configuration for the stream consumer.
type ConsumerConfig struct {
	StreamKey     string
	ConsumerGroup string
	ConsumerName  string
	BatchSize     int64
	BlockTime     time.Duration
	ClaimInterval time.Duration
	ClaimMinIdle  time.Duration
}

// DefaultConsumerConfig returns sensible defaults.
func DefaultConsumerConfig() ConsumerConfig {
	return ConsumerConfig{
		StreamKey:     StreamKey,
		ConsumerGroup: ConsumerGroup,
		ConsumerName:  ConsumerName,
		BatchSize:     defaultBatchSize,
		BlockTime:     defaultBlockTime,
		ClaimInterval: defaultClaimInterval,
		ClaimMinIdle:  defaultClaimMinIdle,
	}
}

// EventHandler is called for each valid event consumed from the stream.
type EventHandler func(ctx context.Context, msg StreamMessage) error

// MetricsReporter allows the consumer to report operational metrics.
type MetricsReporter interface {
	IncEventsConsumed()
	IncValidationFailures()
	IncRedisReconnects()
	SetConsumerLag(lag float64)
}

// Consumer reads events from a Redis Stream using consumer groups.
type Consumer struct {
	client  *redis.Client
	cfg     ConsumerConfig
	handler EventHandler
	metrics MetricsReporter
	logger  *slog.Logger

	mu      sync.Mutex
	seen    map[string]struct{} // event IDs for deduplication
	running bool
}

// NewConsumer creates a new stream consumer.
func NewConsumer(client *redis.Client, cfg ConsumerConfig, handler EventHandler, metrics MetricsReporter, logger *slog.Logger) *Consumer {
	return &Consumer{
		client:  client,
		cfg:     cfg,
		handler: handler,
		metrics: metrics,
		logger:  logger,
		seen:    make(map[string]struct{}, 1024),
	}
}

// EnsureGroup creates the consumer group if it does not exist.
func (c *Consumer) EnsureGroup(ctx context.Context) error {
	err := c.client.XGroupCreateMkStream(ctx, c.cfg.StreamKey, c.cfg.ConsumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("create consumer group: %w", err)
	}
	c.logger.Info("consumer group ready",
		slog.String("stream", c.cfg.StreamKey),
		slog.String("group", c.cfg.ConsumerGroup))
	return nil
}

// Run starts consuming events. It blocks until the context is canceled.
func (c *Consumer) Run(ctx context.Context) error {
	c.mu.Lock()
	c.running = true
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
	}()

	// Start pending message recovery in background.
	go c.claimPendingLoop(ctx)

	c.logger.Info("stream consumer started",
		slog.String("stream", c.cfg.StreamKey),
		slog.String("consumer", c.cfg.ConsumerName))

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("stream consumer stopping")
			return fmt.Errorf("consumer stopped: %w", ctx.Err())
		default:
		}

		entries, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.cfg.ConsumerGroup,
			Consumer: c.cfg.ConsumerName,
			Streams:  []string{c.cfg.StreamKey, ">"},
			Count:    c.cfg.BatchSize,
			Block:    c.cfg.BlockTime,
		}).Result()

		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			if ctx.Err() != nil {
				return fmt.Errorf("consumer stopped: %w", ctx.Err())
			}
			c.logger.Error("xreadgroup failed", slog.String("error", err.Error()))
			if c.metrics != nil {
				c.metrics.IncRedisReconnects()
			}
			// Back off on transient errors.
			select {
			case <-ctx.Done():
				return fmt.Errorf("consumer stopped: %w", ctx.Err())
			case <-time.After(2 * time.Second):
			}
			continue
		}

		for _, stream := range entries {
			for _, entry := range stream.Messages {
				c.processEntry(ctx, entry)
			}
		}
	}
}

// processEntry parses, validates, deduplicates, and dispatches a single stream entry.
func (c *Consumer) processEntry(ctx context.Context, entry redis.XMessage) {
	event, err := c.parseEntry(entry)
	if err != nil {
		c.logger.Warn("malformed event, acknowledging to skip",
			slog.String("stream_id", entry.ID),
			slog.String("error", err.Error()))
		if c.metrics != nil {
			c.metrics.IncValidationFailures()
		}
		// ACK malformed messages to avoid infinite reprocessing.
		c.ack(ctx, entry.ID)
		return
	}

	// Deduplicate by event_id.
	if c.isDuplicate(event.EventID) {
		c.logger.Debug("duplicate event skipped", slog.String("event_id", event.EventID))
		c.ack(ctx, entry.ID)
		return
	}

	msg := StreamMessage{
		StreamID: entry.ID,
		Event:    event,
	}

	if err := c.handler(ctx, msg); err != nil {
		c.logger.Error("event handler failed, not acknowledging",
			slog.String("stream_id", entry.ID),
			slog.String("event_id", event.EventID),
			slog.String("error", err.Error()))
		// Do NOT ack — message will be retried via pending/claim.
		return
	}

	c.markSeen(event.EventID)
	if c.metrics != nil {
		c.metrics.IncEventsConsumed()
	}
	c.ack(ctx, entry.ID)
}

// parseEntry extracts a PriceEvent from a Redis stream entry.
// Supports both the new full-schema format and the legacy {symbol, data} wrapper.
func (c *Consumer) parseEntry(entry redis.XMessage) (*PriceEvent, error) {
	// Try new format: full event JSON in "data" field.
	if dataStr, ok := entry.Values["data"].(string); ok {
		event, err := ParseEvent([]byte(dataStr))
		if err != nil {
			return nil, err
		}
		// If event_id is empty, this might be legacy format.
		if event.EventID != "" {
			if err := event.Validate(); err != nil {
				return nil, err
			}
			return event, nil
		}
		// Legacy format: data contains LatestMarketData JSON.
		return c.parseLegacyEntry(entry, dataStr)
	}

	// Try direct field format (all fields at top level).
	payload, err := json.Marshal(entry.Values)
	if err != nil {
		return nil, fmt.Errorf("marshal entry values: %w", err)
	}
	event, err := ParseEvent(payload)
	if err != nil {
		return nil, err
	}
	if err := event.Validate(); err != nil {
		return nil, err
	}
	return event, nil
}

// parseLegacyEntry handles the legacy {symbol, data: LatestMarketData} format.
func (c *Consumer) parseLegacyEntry(entry redis.XMessage, dataStr string) (*PriceEvent, error) {
	var legacy struct {
		Symbol     string `json:"symbol"`
		Name       string `json:"name"`
		PriceUSD   string `json:"price_usd"`
		MarketCap  string `json:"market_cap"`
		Volume24h  string `json:"volume_24h"`
		Change24h  string `json:"change_24h"`
		Provider   string `json:"provider"`
		CapturedAt string `json:"captured_at"`
	}
	if err := json.Unmarshal([]byte(dataStr), &legacy); err != nil {
		return nil, fmt.Errorf("parse legacy data: %w", err)
	}

	symbol := legacy.Symbol
	if symbol == "" {
		if s, ok := entry.Values["symbol"].(string); ok {
			symbol = s
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	event := &PriceEvent{
		EventID:     entry.ID, // Use stream ID as event ID for legacy events.
		EventType:   EventTypePriceUpdated,
		Symbol:      symbol,
		PriceUSD:    legacy.PriceUSD,
		MarketCap:   legacy.MarketCap,
		Volume24h:   legacy.Volume24h,
		Change24h:   legacy.Change24h,
		Provider:    legacy.Provider,
		ObservedAt:  legacy.CapturedAt,
		PublishedAt: now,
	}

	if err := event.Validate(); err != nil {
		return nil, err
	}
	return event, nil
}

// claimPendingLoop periodically claims and reprocesses pending messages.
func (c *Consumer) claimPendingLoop(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.ClaimInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.claimPending(ctx)
		}
	}
}

// claimPending uses XAUTOCLAIM to recover messages idle beyond ClaimMinIdle.
func (c *Consumer) claimPending(ctx context.Context) {
	entries, _, err := c.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   c.cfg.StreamKey,
		Group:    c.cfg.ConsumerGroup,
		Consumer: c.cfg.ConsumerName,
		MinIdle:  c.cfg.ClaimMinIdle,
		Start:    "0-0",
		Count:    c.cfg.BatchSize,
	}).Result()

	if err != nil {
		if ctx.Err() != nil {
			return
		}
		c.logger.Error("xautoclaim failed", slog.String("error", err.Error()))
		return
	}

	if len(entries) > 0 {
		c.logger.Info("claimed pending messages", slog.Int("count", len(entries)))
	}

	for _, entry := range entries {
		c.processEntry(ctx, entry)
	}
}

func (c *Consumer) ack(ctx context.Context, id string) {
	if err := c.client.XAck(ctx, c.cfg.StreamKey, c.cfg.ConsumerGroup, id).Err(); err != nil {
		c.logger.Error("xack failed", slog.String("id", id), slog.String("error", err.Error()))
	}
}

func (c *Consumer) isDuplicate(eventID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, exists := c.seen[eventID]
	return exists
}

func (c *Consumer) markSeen(eventID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seen[eventID] = struct{}{}
	// Prevent unbounded growth: evict oldest half when exceeding capacity.
	if len(c.seen) > 10000 {
		c.seen = make(map[string]struct{}, 1024)
	}
}
