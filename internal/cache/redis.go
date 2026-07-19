package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

// GetLatest retrieves the latest market data for a symbol from Redis.
func (c *MarketCache) GetLatest(ctx context.Context, symbol string) (*market.LatestMarketData, error) {
	key := latestKeyPrefix + symbol
	payload, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
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

	var results []market.LatestMarketData
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

// PublishEvent publishes a price update event to a Redis Stream.
func (c *MarketCache) PublishEvent(ctx context.Context, data market.LatestMarketData) error {
	payload, err := json.Marshal(data)
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
	return c.client.Ping(ctx).Err()
}
