package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

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
	Provider    string
	StatusCode  int
	Message     string
	Err         error
	RateLimited bool
	Transient   bool
	Permanent   bool
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

// IsRateLimited returns true if the error indicates an HTTP 429 rate-limit response.
func IsRateLimited(err error) bool {
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe.RateLimited || pe.StatusCode == http.StatusTooManyRequests
	}
	return false
}

// IsTransient returns true if the error is transient and may succeed on retry.
// Transient errors include: connection resets, 5xx responses, timeouts, and rate limits.
func IsTransient(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var pe *ProviderError
	if errors.As(err, &pe) {
		if pe.Transient {
			return true
		}
		if pe.StatusCode >= 500 && pe.StatusCode < 600 {
			return true
		}
		if pe.StatusCode == http.StatusTooManyRequests {
			return true
		}
	}
	return false
}

// IsPermanent returns true if the error is permanent and should not be retried.
// Permanent errors include: 4xx (except 429), malformed responses, validation failures.
func IsPermanent(err error) bool {
	if err == nil {
		return false
	}
	var pe *ProviderError
	if errors.As(err, &pe) {
		if pe.Permanent {
			return true
		}
		if pe.StatusCode >= 400 && pe.StatusCode < 500 && pe.StatusCode != http.StatusTooManyRequests {
			return true
		}
	}
	return false
}
