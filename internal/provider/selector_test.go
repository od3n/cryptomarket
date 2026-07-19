package provider

import (
	"context"
	"testing"

	"github.com/crypto-market-platform/internal/market"
)

// mockProvider is a test provider.
type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) FetchMarketData(ctx context.Context, symbols []string) ([]market.MarketData, error) {
	return nil, nil
}

func TestSelector_Primary(t *testing.T) {
	p1 := &mockProvider{name: "coingecko"}
	p2 := &mockProvider{name: "coincap"}
	s := NewSelector([]Provider{p1, p2}, nil)

	primary, err := s.Primary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if primary.Name() != "coingecko" {
		t.Errorf("expected primary coingecko, got %s", primary.Name())
	}
}

func TestSelector_Fallbacks(t *testing.T) {
	p1 := &mockProvider{name: "coingecko"}
	p2 := &mockProvider{name: "coincap"}
	p3 := &mockProvider{name: "binance"}
	s := NewSelector([]Provider{p1, p2, p3}, nil)

	fallbacks := s.Fallbacks()
	if len(fallbacks) != 2 {
		t.Fatalf("expected 2 fallbacks, got %d", len(fallbacks))
	}
	if fallbacks[0].Name() != "coincap" {
		t.Errorf("expected first fallback coincap, got %s", fallbacks[0].Name())
	}
	if fallbacks[1].Name() != "binance" {
		t.Errorf("expected second fallback binance, got %s", fallbacks[1].Name())
	}
}

func TestSelector_Disabled(t *testing.T) {
	p1 := &mockProvider{name: "coingecko"}
	p2 := &mockProvider{name: "coincap"}
	s := NewSelector([]Provider{p1, p2}, []string{"coingecko"})

	// Primary should now be coincap since coingecko is disabled.
	primary, err := s.Primary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if primary.Name() != "coincap" {
		t.Errorf("expected primary coincap (coingecko disabled), got %s", primary.Name())
	}

	eligible := s.Eligible()
	if len(eligible) != 1 {
		t.Errorf("expected 1 eligible provider, got %d", len(eligible))
	}
}

func TestSelector_DisableEnable(t *testing.T) {
	p1 := &mockProvider{name: "coingecko"}
	p2 := &mockProvider{name: "coincap"}
	s := NewSelector([]Provider{p1, p2}, nil)

	// Disable primary.
	s.Disable("coingecko")
	if !s.IsDisabled("coingecko") {
		t.Error("expected coingecko to be disabled")
	}
	primary, _ := s.Primary()
	if primary.Name() != "coincap" {
		t.Errorf("expected coincap after disabling coingecko, got %s", primary.Name())
	}

	// Re-enable.
	s.Enable("coingecko")
	if s.IsDisabled("coingecko") {
		t.Error("expected coingecko to be enabled")
	}
	primary, _ = s.Primary()
	if primary.Name() != "coingecko" {
		t.Errorf("expected coingecko after re-enabling, got %s", primary.Name())
	}
}

func TestSelector_AllDisabled(t *testing.T) {
	p1 := &mockProvider{name: "coingecko"}
	p2 := &mockProvider{name: "coincap"}
	s := NewSelector([]Provider{p1, p2}, []string{"coingecko", "coincap"})

	_, err := s.Primary()
	if err == nil {
		t.Error("expected error when all providers disabled")
	}

	eligible := s.Eligible()
	if len(eligible) != 0 {
		t.Errorf("expected 0 eligible providers, got %d", len(eligible))
	}
}

func TestSelector_Names(t *testing.T) {
	p1 := &mockProvider{name: "coingecko"}
	p2 := &mockProvider{name: "coincap"}
	s := NewSelector([]Provider{p1, p2}, nil)

	names := s.Names()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "coingecko" || names[1] != "coincap" {
		t.Errorf("unexpected names: %v", names)
	}
}
