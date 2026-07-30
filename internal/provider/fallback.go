package provider

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/crypto-market-platform/internal/market"
	"github.com/crypto-market-platform/internal/resilience"
)

var (
	fallbackTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "provider_fallback_total",
		Help: "Total number of provider fallback events.",
	}, []string{"from", "to"})

	providerSwitchTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "provider_switch_total",
		Help: "Total number of provider switches.",
	}, []string{"from", "to"})

	activeProviderGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "provider_active",
		Help: "Currently active provider (1=active, 0=inactive).",
	}, []string{"provider"})
)

// FallbackConfig holds configuration for the fallback orchestrator.
type FallbackConfig struct {
	RetryConfig       resilience.RetryConfig
	CircuitBreakerCfg resilience.CircuitBreakerConfig
}

// FallbackOrchestrator manages provider fallback with circuit breakers and retries.
type FallbackOrchestrator struct {
	selector    *Selector
	cbManager   *resilience.Manager
	rateTracker *resilience.RateLimitTracker
	retryConfig resilience.RetryConfig
	logger      *slog.Logger

	mu             sync.RWMutex
	activeProvider string
	lastSwitch     time.Time
	degraded       bool
}

// NewFallbackOrchestrator creates a new fallback orchestrator.
func NewFallbackOrchestrator(
	selector *Selector,
	cbManager *resilience.Manager,
	rateTracker *resilience.RateLimitTracker,
	retryConfig resilience.RetryConfig,
	logger *slog.Logger,
) *FallbackOrchestrator {
	fo := &FallbackOrchestrator{
		selector:    selector,
		cbManager:   cbManager,
		rateTracker: rateTracker,
		retryConfig: retryConfig,
		logger:      logger,
	}

	// Set initial active provider.
	if primary, err := selector.Primary(); err == nil {
		fo.activeProvider = primary.Name()
		activeProviderGauge.WithLabelValues(primary.Name()).Set(1)
	}

	return fo
}

// FetchMarketData attempts to fetch market data using the fallback flow:
// 1. Attempt primary provider
// 2. Retry transient failures within limits
// 3. Circuit breaker or retry policy rejects further attempts
// 4. Select next eligible provider
// 5. Fetch and normalize data
// 6. Record actual provider used
func (fo *FallbackOrchestrator) FetchMarketData(ctx context.Context, providerSymbols []string) ([]market.MarketData, error) {
	eligible := fo.selector.Eligible()
	if len(eligible) == 0 {
		return nil, &ProviderError{
			Provider:  "none",
			Message:   "no eligible providers available",
			Permanent: true,
		}
	}

	var lastErr error
	for i, prov := range eligible {
		name := prov.Name()

		// Check circuit breaker.
		cb := fo.cbManager.Get(name)
		if cb.State() == resilience.StateOpen {
			fo.logger.Debug("circuit open, skipping provider",
				slog.String("provider", name))
			continue
		}

		// Check rate limit.
		if wait, remaining := fo.rateTracker.ShouldWait(name); wait {
			fo.logger.Debug("rate limited, skipping provider",
				slog.String("provider", name),
				slog.Duration("remaining", remaining))
			continue
		}

		// Attempt fetch with retry.
		data, err := fo.attemptFetch(ctx, prov, providerSymbols)
		if err != nil {
			lastErr = err

			// Record failure in circuit breaker.
			cb.RecordFailure()

			// Track rate limits.
			if IsRateLimited(err) {
				fo.rateTracker.RecordRateLimit(name, resilience.DefaultRetryAfter())
			}

			fo.logger.Warn("provider fetch failed",
				slog.String("provider", name),
				slog.String("error", err.Error()),
				slog.Int("attempt", i+1))

			// If this was the primary and we're falling back, record it.
			if i == 0 && len(eligible) > 1 {
				fallbackTotal.WithLabelValues(name, eligible[1].Name()).Inc()
			}
			continue
		}

		// Success - record in circuit breaker.
		cb.RecordSuccess()

		// Track provider switch if different from active.
		fo.setActiveProvider(name)

		return data, nil
	}

	// All providers failed.
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, &ProviderError{
		Provider:  "all",
		Message:   "all providers unavailable",
		Permanent: false,
		Transient: true,
	}
}

// attemptFetch tries to fetch data from a single provider with retries.
func (fo *FallbackOrchestrator) attemptFetch(ctx context.Context, prov Provider, symbols []string) ([]market.MarketData, error) {
	cfg := fo.retryConfig
	cfg.RetryableFunc = func(err error) bool {
		return IsTransient(err) && !IsPermanent(err)
	}

	data, err := resilience.RetryWithResult(ctx, cfg, func(ctx context.Context) ([]market.MarketData, error) {
		fetched, fetchErr := prov.FetchMarketData(ctx, symbols)
		if fetchErr != nil {
			return nil, fmt.Errorf("fetch from %s: %w", prov.Name(), fetchErr)
		}
		return fetched, nil
	})
	if err != nil {
		return nil, fmt.Errorf("attempt fetch: %w", err)
	}
	return data, nil
}

// setActiveProvider updates the active provider and emits metrics if changed.
func (fo *FallbackOrchestrator) setActiveProvider(name string) {
	fo.mu.Lock()
	defer fo.mu.Unlock()

	if fo.activeProvider != name {
		from := fo.activeProvider
		fo.activeProvider = name
		fo.lastSwitch = time.Now()

		// Update metrics.
		if from != "" {
			activeProviderGauge.WithLabelValues(from).Set(0)
			providerSwitchTotal.WithLabelValues(from, name).Inc()
		}
		activeProviderGauge.WithLabelValues(name).Set(1)

		// Determine if degraded (using non-primary provider).
		primary, _ := fo.selector.Primary()
		fo.degraded = primary == nil || primary.Name() != name

		fo.logger.Info("provider switched",
			slog.String("from", from),
			slog.String("to", name),
			slog.Bool("degraded", fo.degraded))
	}
}

// ActiveProvider returns the name of the currently active provider.
func (fo *FallbackOrchestrator) ActiveProvider() string {
	fo.mu.RLock()
	defer fo.mu.RUnlock()
	return fo.activeProvider
}

// IsDegraded returns true if operating on a fallback provider.
func (fo *FallbackOrchestrator) IsDegraded() bool {
	fo.mu.RLock()
	defer fo.mu.RUnlock()
	return fo.degraded
}

// LastSwitch returns the time of the last provider switch.
func (fo *FallbackOrchestrator) LastSwitch() time.Time {
	fo.mu.RLock()
	defer fo.mu.RUnlock()
	return fo.lastSwitch
}

// CircuitStates returns the circuit breaker states for all providers.
func (fo *FallbackOrchestrator) CircuitStates() map[string]string {
	return fo.cbManager.States()
}

// Status returns a summary of the orchestrator status.
type OrchestratorStatus struct {
	ActiveProvider string            `json:"active_provider"`
	Degraded       bool              `json:"degraded"`
	LastSwitch     *time.Time        `json:"last_switch,omitempty"`
	CircuitStates  map[string]string `json:"circuit_states"`
	Providers      []string          `json:"providers"`
}

// Status returns the current orchestrator status.
func (fo *FallbackOrchestrator) Status() OrchestratorStatus {
	fo.mu.RLock()
	defer fo.mu.RUnlock()

	status := OrchestratorStatus{
		ActiveProvider: fo.activeProvider,
		Degraded:       fo.degraded,
		CircuitStates:  fo.cbManager.States(),
		Providers:      fo.selector.Names(),
	}
	if !fo.lastSwitch.IsZero() {
		status.LastSwitch = &fo.lastSwitch
	}
	return status
}
