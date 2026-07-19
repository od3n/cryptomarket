package market

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// FreshnessState represents the freshness status of market data.
type FreshnessState string

const (
	// FreshnessFresh indicates data is within the fresh threshold.
	FreshnessFresh FreshnessState = "fresh"
	// FreshnessDelayed indicates data is delayed but not stale.
	FreshnessDelayed FreshnessState = "delayed"
	// FreshnessStale indicates data is stale.
	FreshnessStale FreshnessState = "stale"
	// FreshnessUnknown indicates no valid observation is available.
	FreshnessUnknown FreshnessState = "unknown"
)

var (
	freshnessGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "data_freshness_status",
		Help: "Data freshness status (0=fresh, 1=delayed, 2=stale, 3=unknown).",
	}, []string{"symbol"})

	staleSymbolsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "data_stale_symbols_count",
		Help: "Number of symbols with stale data.",
	})
)

// FreshnessConfig holds configuration for freshness calculations.
type FreshnessConfig struct {
	// FreshThreshold is the maximum age for data to be considered fresh.
	FreshThreshold time.Duration
	// StaleThreshold is the maximum age for data to be considered delayed (beyond this is stale).
	StaleThreshold time.Duration
}

// DefaultFreshnessConfig returns sensible defaults.
func DefaultFreshnessConfig() FreshnessConfig {
	return FreshnessConfig{
		FreshThreshold: 120 * time.Second,
		StaleThreshold: 300 * time.Second,
	}
}

// FreshnessTracker tracks freshness state per symbol.
type FreshnessTracker struct {
	mu         sync.RWMutex
	config     FreshnessConfig
	lastUpdate map[string]time.Time
}

// NewFreshnessTracker creates a new freshness tracker.
func NewFreshnessTracker(config FreshnessConfig) *FreshnessTracker {
	return &FreshnessTracker{
		config:     config,
		lastUpdate: make(map[string]time.Time),
	}
}

// RecordUpdate records a data update for a symbol.
func (ft *FreshnessTracker) RecordUpdate(symbol string, timestamp time.Time) {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	ft.lastUpdate[symbol] = timestamp
}

// State returns the freshness state for a symbol.
func (ft *FreshnessTracker) State(symbol string) FreshnessState {
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	return ft.stateFor(symbol, time.Now())
}

// stateFor calculates freshness state at a given time (for testing).
func (ft *FreshnessTracker) stateFor(symbol string, now time.Time) FreshnessState {
	lastUpdate, ok := ft.lastUpdate[symbol]
	if !ok {
		return FreshnessUnknown
	}

	age := now.Sub(lastUpdate)
	switch {
	case age <= ft.config.FreshThreshold:
		return FreshnessFresh
	case age <= ft.config.StaleThreshold:
		return FreshnessDelayed
	default:
		return FreshnessStale
	}
}

// Age returns the age of data for a symbol.
func (ft *FreshnessTracker) Age(symbol string) time.Duration {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	lastUpdate, ok := ft.lastUpdate[symbol]
	if !ok {
		return -1
	}
	return time.Since(lastUpdate)
}

// Summary provides an aggregate view of freshness across all symbols.
type FreshnessSummary struct {
	Total       int            `json:"total"`
	Fresh       int            `json:"fresh"`
	Delayed     int            `json:"delayed"`
	Stale       int            `json:"stale"`
	Unknown     int            `json:"unknown"`
	OldestAge   *time.Duration `json:"oldest_age,omitempty"`
	OverallState FreshnessState `json:"overall_state"`
}

// Summary returns an aggregate freshness summary.
func (ft *FreshnessTracker) Summary() FreshnessSummary {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	now := time.Now()
	summary := FreshnessSummary{
		Total: len(ft.lastUpdate),
	}

	var oldestAge time.Duration
	hasOldest := false

	for symbol := range ft.lastUpdate {
		state := ft.stateFor(symbol, now)
		switch state {
		case FreshnessFresh:
			summary.Fresh++
		case FreshnessDelayed:
			summary.Delayed++
		case FreshnessStale:
			summary.Stale++
		case FreshnessUnknown:
			summary.Unknown++
		}

		age := now.Sub(ft.lastUpdate[symbol])
		if !hasOldest || age > oldestAge {
			oldestAge = age
			hasOldest = true
		}
	}

	if hasOldest {
		summary.OldestAge = &oldestAge
	}

	// Determine overall state (worst case).
	switch {
	case summary.Stale > 0:
		summary.OverallState = FreshnessStale
	case summary.Delayed > 0:
		summary.OverallState = FreshnessDelayed
	case summary.Fresh > 0:
		summary.OverallState = FreshnessFresh
	default:
		summary.OverallState = FreshnessUnknown
	}

	// Update metrics.
	staleSymbolsGauge.Set(float64(summary.Stale))

	return summary
}

// UpdateMetrics updates Prometheus metrics for all symbols.
func (ft *FreshnessTracker) UpdateMetrics() {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	now := time.Now()
	for symbol := range ft.lastUpdate {
		state := ft.stateFor(symbol, now)
		var value float64
		switch state {
		case FreshnessFresh:
			value = 0
		case FreshnessDelayed:
			value = 1
		case FreshnessStale:
			value = 2
		case FreshnessUnknown:
			value = 3
		}
		freshnessGauge.WithLabelValues(symbol).Set(value)
	}
}

// FreshnessComplianceRatio returns the ratio of fresh symbols to total symbols.
func (ft *FreshnessTracker) FreshnessComplianceRatio() float64 {
	summary := ft.Summary()
	if summary.Total == 0 {
		return 1.0 // No data means no violations
	}
	return float64(summary.Fresh) / float64(summary.Total)
}
