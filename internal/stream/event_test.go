package stream

import (
	"errors"
	"testing"
	"time"
)

func TestPriceEventValidate_Valid(t *testing.T) {
	event := &PriceEvent{
		EventID:     "test-id-123",
		EventType:   EventTypePriceUpdated,
		Symbol:      "BTC",
		PriceUSD:    "65000.50",
		MarketCap:   "1300000000000",
		Volume24h:   "50000000000",
		Change24h:   "2.5",
		Provider:    "coingecko",
		ObservedAt:  "2026-07-20T08:15:00Z",
		PublishedAt: "2026-07-20T08:15:01Z",
	}

	if err := event.Validate(); err != nil {
		t.Errorf("expected valid event, got error: %v", err)
	}
}

func TestPriceEventValidate_MissingEventID(t *testing.T) {
	event := &PriceEvent{
		EventType: EventTypePriceUpdated,
		Symbol:    "BTC",
		PriceUSD:  "65000.50",
		Provider:  "coingecko",
	}

	if err := event.Validate(); !errors.Is(err, ErrMissingEventID) {
		t.Errorf("expected ErrMissingEventID, got: %v", err)
	}
}

func TestPriceEventValidate_MissingEventType(t *testing.T) {
	event := &PriceEvent{
		EventID:  "test-id",
		Symbol:   "BTC",
		PriceUSD: "65000.50",
		Provider: "coingecko",
	}

	if err := event.Validate(); !errors.Is(err, ErrMissingEventType) {
		t.Errorf("expected ErrMissingEventType, got: %v", err)
	}
}

func TestPriceEventValidate_UnknownEventType(t *testing.T) {
	event := &PriceEvent{
		EventID:   "test-id",
		EventType: "unknown.event",
		Symbol:    "BTC",
		PriceUSD:  "65000.50",
		Provider:  "coingecko",
	}

	if err := event.Validate(); err == nil {
		t.Error("expected error for unknown event type, got nil")
	}
}

func TestPriceEventValidate_MissingSymbol(t *testing.T) {
	event := &PriceEvent{
		EventID:   "test-id",
		EventType: EventTypePriceUpdated,
		PriceUSD:  "65000.50",
		Provider:  "coingecko",
	}

	if err := event.Validate(); !errors.Is(err, ErrMissingSymbol) {
		t.Errorf("expected ErrMissingSymbol, got: %v", err)
	}
}

func TestPriceEventValidate_MissingPrice(t *testing.T) {
	event := &PriceEvent{
		EventID:   "test-id",
		EventType: EventTypePriceUpdated,
		Symbol:    "BTC",
		Provider:  "coingecko",
	}

	if err := event.Validate(); !errors.Is(err, ErrMissingPrice) {
		t.Errorf("expected ErrMissingPrice, got: %v", err)
	}
}

func TestPriceEventValidate_MissingProvider(t *testing.T) {
	event := &PriceEvent{
		EventID:   "test-id",
		EventType: EventTypePriceUpdated,
		Symbol:    "BTC",
		PriceUSD:  "65000.50",
	}

	if err := event.Validate(); !errors.Is(err, ErrMissingProvider) {
		t.Errorf("expected ErrMissingProvider, got: %v", err)
	}
}

func TestPriceEventValidate_InvalidTimestamp(t *testing.T) {
	event := &PriceEvent{
		EventID:     "test-id",
		EventType:   EventTypePriceUpdated,
		Symbol:      "BTC",
		PriceUSD:    "65000.50",
		Provider:    "coingecko",
		PublishedAt: "not-a-timestamp",
	}

	if err := event.Validate(); err == nil {
		t.Error("expected error for invalid timestamp, got nil")
	}
}

func TestPriceEventValidate_EmptyTimestampsAllowed(t *testing.T) {
	event := &PriceEvent{
		EventID:   "test-id",
		EventType: EventTypePriceUpdated,
		Symbol:    "BTC",
		PriceUSD:  "65000.50",
		Provider:  "coingecko",
	}

	if err := event.Validate(); err != nil {
		t.Errorf("expected empty timestamps to be valid, got: %v", err)
	}
}

func TestPriceEventPublishedTime(t *testing.T) {
	event := &PriceEvent{
		PublishedAt: "2026-07-20T08:15:01Z",
	}

	got := event.PublishedTime()
	expected := time.Date(2026, 7, 20, 8, 15, 1, 0, time.UTC)
	if !got.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func TestPriceEventPublishedTime_Empty(t *testing.T) {
	event := &PriceEvent{}
	got := event.PublishedTime()
	if !got.IsZero() {
		t.Errorf("expected zero time for empty published_at, got %v", got)
	}
}

func TestParseEvent_Valid(t *testing.T) {
	payload := []byte(`{
		"event_id": "abc-123",
		"event_type": "market.price.updated",
		"symbol": "ETH",
		"price_usd": "3500.00",
		"market_cap": "420000000000",
		"volume_24h": "20000000000",
		"change_24h": "-1.2",
		"provider": "coingecko",
		"observed_at": "2026-07-20T08:00:00Z",
		"published_at": "2026-07-20T08:00:01Z"
	}`)

	event, err := ParseEvent(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Symbol != "ETH" {
		t.Errorf("expected symbol ETH, got %s", event.Symbol)
	}
	if event.PriceUSD != "3500.00" {
		t.Errorf("expected price 3500.00, got %s", event.PriceUSD)
	}
}

func TestParseEvent_InvalidJSON(t *testing.T) {
	payload := []byte(`{invalid json}`)
	_, err := ParseEvent(payload)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestParseEvent_MalformedPayload(t *testing.T) {
	// Missing required fields should still parse (validation is separate)
	payload := []byte(`{"symbol": "BTC"}`)
	event, err := ParseEvent(payload)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if event.Symbol != "BTC" {
		t.Errorf("expected symbol BTC, got %s", event.Symbol)
	}
	// But validation should fail
	if err := event.Validate(); err == nil {
		t.Error("expected validation error for incomplete event")
	}
}
