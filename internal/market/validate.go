package market

import (
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var validationFailuresTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "data_validation_failures_total",
	Help: "Total number of data validation failures by provider and field.",
}, []string{"provider", "field"})

// ValidationError represents a typed validation failure.
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field %s: %s (value: %q)", e.Field, e.Message, e.Value)
}

// maxFutureTimestamp is the maximum allowed time in the future for timestamps.
const maxFutureTimestamp = 5 * time.Minute

// Validate checks that MarketData contains all required values.
func (m *MarketData) Validate() error {
	if m.Symbol == "" {
		validationFailuresTotal.WithLabelValues(m.Provider, "symbol").Inc()
		return &ValidationError{Field: "symbol", Value: m.Symbol, Message: "symbol is required"}
	}
	if m.PriceUSD == "" {
		validationFailuresTotal.WithLabelValues(m.Provider, "price_usd").Inc()
		return &ValidationError{Field: "price_usd", Value: m.PriceUSD, Message: fmt.Sprintf("price_usd is required for %s", m.Symbol)}
	}

	price, err := strconv.ParseFloat(m.PriceUSD, 64)
	if err != nil {
		validationFailuresTotal.WithLabelValues(m.Provider, "price_usd").Inc()
		return &ValidationError{Field: "price_usd", Value: m.PriceUSD, Message: fmt.Sprintf("invalid price_usd for %s: %v", m.Symbol, err)}
	}
	if price <= 0 {
		validationFailuresTotal.WithLabelValues(m.Provider, "price_usd").Inc()
		return &ValidationError{Field: "price_usd", Value: m.PriceUSD, Message: fmt.Sprintf("price must be greater than zero for %s", m.Symbol)}
	}

	if m.MarketCap != "" {
		mc, err := strconv.ParseFloat(m.MarketCap, 64)
		if err != nil {
			validationFailuresTotal.WithLabelValues(m.Provider, "market_cap").Inc()
			return &ValidationError{Field: "market_cap", Value: m.MarketCap, Message: fmt.Sprintf("invalid market_cap for %s: %v", m.Symbol, err)}
		}
		if mc < 0 {
			validationFailuresTotal.WithLabelValues(m.Provider, "market_cap").Inc()
			return &ValidationError{Field: "market_cap", Value: m.MarketCap, Message: fmt.Sprintf("market_cap cannot be negative for %s", m.Symbol)}
		}
	}

	if m.Volume24h != "" {
		vol, err := strconv.ParseFloat(m.Volume24h, 64)
		if err != nil {
			validationFailuresTotal.WithLabelValues(m.Provider, "volume_24h").Inc()
			return &ValidationError{Field: "volume_24h", Value: m.Volume24h, Message: fmt.Sprintf("invalid volume_24h for %s: %v", m.Symbol, err)}
		}
		if vol < 0 {
			validationFailuresTotal.WithLabelValues(m.Provider, "volume_24h").Inc()
			return &ValidationError{Field: "volume_24h", Value: m.Volume24h, Message: fmt.Sprintf("volume_24h cannot be negative for %s", m.Symbol)}
		}
	}

	if m.Change24h != "" {
		if _, err := strconv.ParseFloat(m.Change24h, 64); err != nil {
			validationFailuresTotal.WithLabelValues(m.Provider, "change_24h").Inc()
			return &ValidationError{Field: "change_24h", Value: m.Change24h, Message: fmt.Sprintf("invalid change_24h for %s: %v", m.Symbol, err)}
		}
	}

	if m.Provider == "" {
		validationFailuresTotal.WithLabelValues("unknown", "provider").Inc()
		return &ValidationError{Field: "provider", Value: m.Provider, Message: fmt.Sprintf("provider is required for %s", m.Symbol)}
	}

	// Validate timestamp is not excessively in the future.
	if !m.FetchedAt.IsZero() {
		if m.FetchedAt.After(time.Now().Add(maxFutureTimestamp)) {
			validationFailuresTotal.WithLabelValues(m.Provider, "fetched_at").Inc()
			return &ValidationError{Field: "fetched_at", Value: m.FetchedAt.String(), Message: fmt.Sprintf("timestamp is too far in the future for %s", m.Symbol)}
		}
	}

	return nil
}
