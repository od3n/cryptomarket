package provider

import (
	"context"
	"fmt"

	"github.com/crypto-market-platform/internal/market"
)

// Provider defines the interface for fetching market data from external sources.
// Implementations must be replaceable without affecting business logic.
type Provider interface {
	// Name returns the provider identifier.
	Name() string

	// FetchMarketData retrieves current market data for the given provider symbols.
	// It returns normalized MarketData for each successfully fetched symbol.
	FetchMarketData(ctx context.Context, providerSymbols []string) ([]market.MarketData, error)
}

// ProviderError represents a typed error from a provider operation.
type ProviderError struct {
	Provider   string
	StatusCode int
	Message    string
	Err        error
}

func (e *ProviderError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("provider %s: %s (status %d): %v", e.Provider, e.Message, e.StatusCode, e.Err)
	}
	return fmt.Sprintf("provider %s: %s (status %d)", e.Provider, e.Message, e.StatusCode)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}
