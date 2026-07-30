package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/crypto-market-platform/internal/market"
)

const (
	latestKeyPrefix = "market:latest:"
	streamKey       = "market:events"
	streamMaxLen    = 10000
)

// MarketCache provides Redis-backed caching for latest market data.
type MarketCache struct {
	client *redis.Client
}

// NewMarketCache creates a new MarketCache.
func NewMarketCache(client *redis.Client) *MarketCache {
	return &MarketCache{client: client}
}

// SetLatest stores the latest market data for a symbol in Redis.
func (c *MarketCache) SetLatest(ctx context.Context, data market.LatestMarketData) error {
	key := latestKeyPrefix + data.Symbol
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal latest market data: %w", err)
	}
	if err := c.client.Set(ctx, key, payload, 5*time.Minute).Err(); err != nil {
		return fmt.Errorf("set latest for %s: %w", data.Symbol, err)
	}
	return nil
}

// SetLatestBatch stores multiple market data entries using Redis pipelining
// for reduced round-trip overhead under high ingestion throughput.
func (c *MarketCache) SetLatestBatch(ctx context.Context, entries []market.LatestMarketData) error {
	if len(entries) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()
	for _, data := range entries {
		key := latestKeyPrefix + data.Symbol
		payload, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("marshal latest market data for %s: %w", data.Symbol, err)
		}
		pipe.Set(ctx, key, payload, 5*time.Minute)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("pipeline set batch (%d entries): %w", len(entries), err)
	}
	return nil
}

// GetLatest retrieves the latest market data for a symbol from Redis.
func (c *MarketCache) GetLatest(ctx context.Context, symbol string) (*market.LatestMarketData, error) {
	key := latestKeyPrefix + symbol
	payload, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest for %s: %w", symbol, err)
	}

	var data market.LatestMarketData
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, fmt.Errorf("unmarshal latest for %s: %w", symbol, err)
	}
	return &data, nil
}

// GetAllLatest retrieves latest market data for all given symbols.
func (c *MarketCache) GetAllLatest(ctx context.Context, symbols []string) ([]market.LatestMarketData, error) {
	if len(symbols) == 0 {
		return nil, nil
	}

	keys := make([]string, len(symbols))
	for i, s := range symbols {
		keys[i] = latestKeyPrefix + s
	}

	payloads, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("mget latest: %w", err)
	}

	results := make([]market.LatestMarketData, 0, len(payloads))
	for _, p := range payloads {
		if p == nil {
			continue
		}
		str, ok := p.(string)
		if !ok {
			continue
		}
		var data market.LatestMarketData
		if err := json.Unmarshal([]byte(str), &data); err != nil {
			continue
		}
		results = append(results, data)
	}
	return results, nil
}

// PriceEvent is the canonical event schema published to the stream.
type PriceEvent struct {
	EventID     string `json:"event_id"`
	EventType   string `json:"event_type"`
	Symbol      string `json:"symbol"`
	PriceUSD    string `json:"price_usd"`
	MarketCap   string `json:"market_cap"`
	Volume24h   string `json:"volume_24h"`
	Change24h   string `json:"change_24h"`
	Provider    string `json:"provider"`
	ObservedAt  string `json:"observed_at"`
	PublishedAt string `json:"published_at"`
}

// PublishEvent publishes a price update event to a Redis Stream.
// It emits the full event schema while retaining the legacy "data" field wrapper
// for backward compatibility with existing consumers.
func (c *MarketCache) PublishEvent(ctx context.Context, data market.LatestMarketData) error {
	now := time.Now().UTC()
	event := PriceEvent{
		EventID:     uuid.New().String(),
		EventType:   "market.price.updated",
		Symbol:      data.Symbol,
		PriceUSD:    data.PriceUSD,
		MarketCap:   data.MarketCap,
		Volume24h:   data.Volume24h,
		Change24h:   data.Change24h,
		Provider:    data.Provider,
		ObservedAt:  data.CapturedAt.UTC().Format(time.RFC3339),
		PublishedAt: now.Format(time.RFC3339),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	err = c.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		MaxLen: streamMaxLen,
		Approx: true,
		Values: map[string]interface{}{
			"symbol": data.Symbol,
			"data":   string(payload),
		},
	}).Err()
	if err != nil {
		return fmt.Errorf("publish event for %s: %w", data.Symbol, err)
	}
	return nil
}

// Ping checks Redis connectivity.
func (c *MarketCache) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return nil
}
