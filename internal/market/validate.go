package market

import (
	"fmt"
	"strconv"
)

// Validate checks that MarketData contains all required values.
func (m *MarketData) Validate() error {
	if m.Symbol == "" {
		return fmt.Errorf("market data: symbol is required")
	}
	if m.PriceUSD == "" {
		return fmt.Errorf("market data: price_usd is required for %s", m.Symbol)
	}
	if _, err := strconv.ParseFloat(m.PriceUSD, 64); err != nil {
		return fmt.Errorf("market data: invalid price_usd %q for %s: %w", m.PriceUSD, m.Symbol, err)
	}
	if m.MarketCap != "" {
		if _, err := strconv.ParseFloat(m.MarketCap, 64); err != nil {
			return fmt.Errorf("market data: invalid market_cap %q for %s: %w", m.MarketCap, m.Symbol, err)
		}
	}
	if m.Volume24h != "" {
		if _, err := strconv.ParseFloat(m.Volume24h, 64); err != nil {
			return fmt.Errorf("market data: invalid volume_24h %q for %s: %w", m.Volume24h, m.Symbol, err)
		}
	}
	if m.Change24h != "" {
		if _, err := strconv.ParseFloat(m.Change24h, 64); err != nil {
			return fmt.Errorf("market data: invalid change_24h %q for %s: %w", m.Change24h, m.Symbol, err)
		}
	}
	if m.Provider == "" {
		return fmt.Errorf("market data: provider is required for %s", m.Symbol)
	}
	return nil
}
