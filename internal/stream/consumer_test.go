package stream

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestParseEntry_NewFormat(t *testing.T) {
	c := &Consumer{
		cfg:    DefaultConsumerConfig(),
		logger: testLogger(),
		seen:   make(map[string]struct{}),
	}

	entry := redis.XMessage{
		ID: "1234567890-0",
		Values: map[string]interface{}{
			"symbol": "BTC",
			"data": `{
				"event_id": "evt-123",
				"event_type": "market.price.updated",
				"symbol": "BTC",
				"price_usd": "65000.50",
				"market_cap": "1300000000000",
				"volume_24h": "50000000000",
				"change_24h": "2.5",
				"provider": "coingecko",
				"observed_at": "2026-07-20T08:15:00Z",
				"published_at": "2026-07-20T08:15:01Z"
			}`,
		},
	}

	event, err := c.parseEntry(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.EventID != "evt-123" {
		t.Errorf("expected event_id evt-123, got %s", event.EventID)
	}
	if event.Symbol != "BTC" {
		t.Errorf("expected symbol BTC, got %s", event.Symbol)
	}
	if event.PriceUSD != "65000.50" {
		t.Errorf("expected price 65000.50, got %s", event.PriceUSD)
	}
}

func TestParseEntry_LegacyFormat(t *testing.T) {
	c := &Consumer{
		cfg:    DefaultConsumerConfig(),
		logger: testLogger(),
		seen:   make(map[string]struct{}),
	}

	entry := redis.XMessage{
		ID: "1234567890-1",
		Values: map[string]interface{}{
			"symbol": "ETH",
			"data": `{
				"symbol": "ETH",
				"name": "Ethereum",
				"price_usd": "3500.00",
				"market_cap": "420000000000",
				"volume_24h": "20000000000",
				"change_24h": "-1.2",
				"provider": "coingecko",
				"captured_at": "2026-07-20T08:00:00Z"
			}`,
		},
	}

	event, err := c.parseEntry(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Symbol != "ETH" {
		t.Errorf("expected symbol ETH, got %s", event.Symbol)
	}
	if event.PriceUSD != "3500.00" {
		t.Errorf("expected price 3500.00, got %s", event.PriceUSD)
	}
	if event.EventID != "1234567890-1" {
		t.Errorf("expected stream ID as event_id, got %s", event.EventID)
	}
	if event.EventType != EventTypePriceUpdated {
		t.Errorf("expected event type %s, got %s", EventTypePriceUpdated, event.EventType)
	}
}

func TestParseEntry_MalformedJSON(t *testing.T) {
	c := &Consumer{
		cfg:    DefaultConsumerConfig(),
		logger: testLogger(),
		seen:   make(map[string]struct{}),
	}

	entry := redis.XMessage{
		ID: "1234567890-2",
		Values: map[string]interface{}{
			"symbol": "BTC",
			"data":   `{invalid json}`,
		},
	}

	_, err := c.parseEntry(entry)
	if err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}
}

func TestParseEntry_InvalidEvent(t *testing.T) {
	c := &Consumer{
		cfg:    DefaultConsumerConfig(),
		logger: testLogger(),
		seen:   make(map[string]struct{}),
	}

	// Valid JSON but missing required fields for new format
	entry := redis.XMessage{
		ID: "1234567890-3",
		Values: map[string]interface{}{
			"data": `{
				"event_id": "evt-456",
				"event_type": "market.price.updated",
				"symbol": "",
				"price_usd": "65000.50",
				"provider": "coingecko"
			}`,
		},
	}

	_, err := c.parseEntry(entry)
	if err == nil {
		t.Error("expected validation error for missing symbol, got nil")
	}
}

func TestParseEntry_DirectFieldFormat(t *testing.T) {
	c := &Consumer{
		cfg:    DefaultConsumerConfig(),
		logger: testLogger(),
		seen:   make(map[string]struct{}),
	}

	// All fields at top level (no "data" wrapper)
	entry := redis.XMessage{
		ID: "1234567890-4",
		Values: map[string]interface{}{
			"event_id":     "evt-direct",
			"event_type":   "market.price.updated",
			"symbol":       "SOL",
			"price_usd":    "150.00",
			"market_cap":   "65000000000",
			"volume_24h":   "3000000000",
			"change_24h":   "5.0",
			"provider":     "coingecko",
			"observed_at":  "2026-07-20T08:00:00Z",
			"published_at": "2026-07-20T08:00:01Z",
		},
	}

	event, err := c.parseEntry(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Symbol != "SOL" {
		t.Errorf("expected symbol SOL, got %s", event.Symbol)
	}
	if event.EventID != "evt-direct" {
		t.Errorf("expected event_id evt-direct, got %s", event.EventID)
	}
}

func TestConsumerDeduplication(t *testing.T) {
	c := &Consumer{
		cfg:    DefaultConsumerConfig(),
		logger: testLogger(),
		seen:   make(map[string]struct{}),
	}

	if c.isDuplicate("evt-1") {
		t.Error("expected evt-1 to not be duplicate initially")
	}

	c.markSeen("evt-1")

	if !c.isDuplicate("evt-1") {
		t.Error("expected evt-1 to be duplicate after markSeen")
	}

	if c.isDuplicate("evt-2") {
		t.Error("expected evt-2 to not be duplicate")
	}
}

func TestConsumerSeenEviction(t *testing.T) {
	c := &Consumer{
		cfg:    DefaultConsumerConfig(),
		logger: testLogger(),
		seen:   make(map[string]struct{}),
	}

	// Fill beyond capacity to trigger eviction
	for i := 0; i < 10001; i++ {
		c.markSeen(fmt.Sprintf("evt-%d", i))
	}

	// After eviction, the map should be reset (small)
	c.mu.Lock()
	size := len(c.seen)
	c.mu.Unlock()

	// markSeen evicts when > 10000, resetting to empty map then adding the current one
	if size > 10000 {
		t.Errorf("expected seen map to be evicted, got size %d", size)
	}
}
