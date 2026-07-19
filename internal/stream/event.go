// Package stream defines the canonical price event schema and Redis Streams consumer logic.
package stream

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// EventType constants for market events.
const (
	EventTypePriceUpdated = "market.price.updated"
)

// PriceEvent is the canonical event schema for market price updates.
// Decimal values are strings to avoid floating-point precision loss.
// All timestamps are UTC.
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

// Validation errors.
var (
	ErrMissingEventID   = errors.New("event_id is required")
	ErrMissingEventType = errors.New("event_type is required")
	ErrMissingSymbol    = errors.New("symbol is required")
	ErrMissingPrice     = errors.New("price_usd is required")
	ErrMissingProvider  = errors.New("provider is required")
	ErrInvalidTimestamp = errors.New("invalid timestamp format, expected RFC3339 UTC")
	ErrUnknownEventType = errors.New("unknown event_type")
)

// Validate checks that all required fields are present and well-formed.
func (e *PriceEvent) Validate() error {
	if e.EventID == "" {
		return ErrMissingEventID
	}
	if e.EventType == "" {
		return ErrMissingEventType
	}
	if e.EventType != EventTypePriceUpdated {
		return fmt.Errorf("%w: %s", ErrUnknownEventType, e.EventType)
	}
	if e.Symbol == "" {
		return ErrMissingSymbol
	}
	if e.PriceUSD == "" {
		return ErrMissingPrice
	}
	if e.Provider == "" {
		return ErrMissingProvider
	}
	if e.ObservedAt != "" {
		if _, err := time.Parse(time.RFC3339, e.ObservedAt); err != nil {
			return fmt.Errorf("observed_at: %w", ErrInvalidTimestamp)
		}
	}
	if e.PublishedAt != "" {
		if _, err := time.Parse(time.RFC3339, e.PublishedAt); err != nil {
			return fmt.Errorf("published_at: %w", ErrInvalidTimestamp)
		}
	}
	return nil
}

// PublishedTime returns the parsed published_at timestamp, or zero time if unavailable.
func (e *PriceEvent) PublishedTime() time.Time {
	if e.PublishedAt == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, e.PublishedAt)
	if err != nil {
		return time.Time{}
	}
	return t
}

// ParseEvent parses a JSON payload into a PriceEvent.
func ParseEvent(payload []byte) (*PriceEvent, error) {
	var event PriceEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("unmarshal event: %w", err)
	}
	return &event, nil
}

// StreamMessage represents a raw message from Redis Streams.
type StreamMessage struct {
	StreamID string
	Event    *PriceEvent
}
