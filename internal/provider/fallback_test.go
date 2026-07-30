package provider

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/crypto-market-platform/internal/market"
	"github.com/crypto-market-platform/internal/resilience"
)

// testProvider is a configurable test provider.
type testProvider struct {
	name string
	data []market.MarketData
	err  error
}

func (p *testProvider) Name() string { return p.name }
func (p *testProvider) FetchMarketData(ctx context.Context, symbols []string) ([]market.MarketData, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.data, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func testOrchestrator(providers []Provider) *FallbackOrchestrator {
	selector := NewSelector(providers, nil)
	cbManager := resilience.NewManager(resilience.CircuitBreakerConfig{
		FailureThreshold: 2,
		OpenDuration:     100 * time.Millisecond,
		SuccessThreshold: 1,
	})
	rateTracker := resilience.NewRateLimitTracker()
	retryConfig := resilience.RetryConfig{
		MaxAttempts: 1, // No retries in tests for simplicity
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}
	return NewFallbackOrchestrator(selector, cbManager, rateTracker, retryConfig, testLogger())
}

func TestFallbackOrchestrator_PrimarySuccess(t *testing.T) {
	primary := &testProvider{
		name: "primary",
		data: []market.MarketData{{Symbol: "BTC", PriceUSD: "50000", Provider: "primary"}},
	}
	fallback := &testProvider{
		name: "fallback",
		data: []market.MarketData{{Symbol: "BTC", PriceUSD: "50001", Provider: "fallback"}},
	}

	fo := testOrchestrator([]Provider{primary, fallback})
	data, err := fo.FetchMarketData(context.Background(), []string{"bitcoin"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 1 {
		t.Fatalf("expected 1 result, got %d", len(data))
	}
	if data[0].Provider != "primary" {
		t.Errorf("expected primary provider data, got %s", data[0].Provider)
	}
	if fo.ActiveProvider() != "primary" {
		t.Errorf("expected active provider 'primary', got %s", fo.ActiveProvider())
	}
	if fo.IsDegraded() {
		t.Error("should not be degraded when using primary")
	}
}

func TestFallbackOrchestrator_FallbackOnPrimaryFailure(t *testing.T) {
	primary := &testProvider{
		name: "primary",
		err:  &ProviderError{Provider: "primary", Message: "failed", Transient: true},
	}
	fallback := &testProvider{
		name: "fallback",
		data: []market.MarketData{{Symbol: "BTC", PriceUSD: "50001", Provider: "fallback"}},
	}

	fo := testOrchestrator([]Provider{primary, fallback})
	data, err := fo.FetchMarketData(context.Background(), []string{"bitcoin"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 1 {
		t.Fatalf("expected 1 result, got %d", len(data))
	}
	if data[0].Provider != "fallback" {
		t.Errorf("expected fallback provider data, got %s", data[0].Provider)
	}
	if fo.ActiveProvider() != "fallback" {
		t.Errorf("expected active provider 'fallback', got %s", fo.ActiveProvider())
	}
	if !fo.IsDegraded() {
		t.Error("should be degraded when using fallback")
	}
}

func TestFallbackOrchestrator_AllProvidersFail(t *testing.T) {
	primary := &testProvider{
		name: "primary",
		err:  &ProviderError{Provider: "primary", Message: "failed", Transient: true},
	}
	fallback := &testProvider{
		name: "fallback",
		err:  &ProviderError{Provider: "fallback", Message: "failed", Transient: true},
	}

	fo := testOrchestrator([]Provider{primary, fallback})
	_, err := fo.FetchMarketData(context.Background(), []string{"bitcoin"})

	if err == nil {
		t.Error("expected error when all providers fail")
	}
}

func TestFallbackOrchestrator_CircuitBreakerSkipsOpen(t *testing.T) {
	primary := &testProvider{
		name: "primary",
		err:  &ProviderError{Provider: "primary", Message: "failed", Transient: true},
	}
	fallback := &testProvider{
		name: "fallback",
		data: []market.MarketData{{Symbol: "BTC", PriceUSD: "50001", Provider: "fallback"}},
	}

	fo := testOrchestrator([]Provider{primary, fallback})

	// Fail primary twice to open circuit (threshold is 2).
	fo.FetchMarketData(context.Background(), []string{"bitcoin"})
	fo.FetchMarketData(context.Background(), []string{"bitcoin"})

	// Circuit should be open now, third call should skip primary entirely.
	data, err := fo.FetchMarketData(context.Background(), []string{"bitcoin"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data[0].Provider != "fallback" {
		t.Errorf("expected fallback data, got %s", data[0].Provider)
	}

	states := fo.CircuitStates()
	if states["primary"] != "open" {
		t.Errorf("expected primary circuit open, got %s", states["primary"])
	}
}

func TestFallbackOrchestrator_ProviderSourcePreserved(t *testing.T) {
	fallback := &testProvider{
		name: "coincap",
		data: []market.MarketData{
			{Symbol: "BTC", PriceUSD: "50000", Provider: "coincap"},
			{Symbol: "ETH", PriceUSD: "3000", Provider: "coincap"},
		},
	}
	primary := &testProvider{
		name: "coingecko",
		err:  errors.New("down"),
	}

	fo := testOrchestrator([]Provider{primary, fallback})
	data, err := fo.FetchMarketData(context.Background(), []string{"bitcoin", "ethereum"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify provider source is preserved in data.
	for _, d := range data {
		if d.Provider != "coincap" {
			t.Errorf("expected provider 'coincap' in data, got %s", d.Provider)
		}
	}
}

func TestFallbackOrchestrator_Status(t *testing.T) {
	primary := &testProvider{name: "primary", data: []market.MarketData{}}
	fallback := &testProvider{name: "fallback", data: []market.MarketData{}}

	fo := testOrchestrator([]Provider{primary, fallback})
	fo.FetchMarketData(context.Background(), []string{"bitcoin"})

	status := fo.Status()
	if status.ActiveProvider != "primary" {
		t.Errorf("expected active provider 'primary', got %s", status.ActiveProvider)
	}
	if status.Degraded {
		t.Error("should not be degraded")
	}
	if len(status.Providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(status.Providers))
	}
}

func TestFallbackOrchestrator_NoEligibleProviders(t *testing.T) {
	selector := NewSelector(nil, nil)
	cbManager := resilience.NewManager(resilience.DefaultCircuitBreakerConfig("test"))
	rateTracker := resilience.NewRateLimitTracker()
	retryConfig := resilience.DefaultRetryConfig()

	fo := NewFallbackOrchestrator(selector, cbManager, rateTracker, retryConfig, testLogger())
	_, err := fo.FetchMarketData(context.Background(), []string{"bitcoin"})

	if err == nil {
		t.Error("expected error with no providers")
	}
}
