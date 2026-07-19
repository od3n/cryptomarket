package market

import (
	"time"
)

// Coin represents a supported cryptocurrency.
type Coin struct {
	ID             int64     `json:"id"`
	Symbol         string    `json:"symbol"`
	Name           string    `json:"name"`
	ProviderSymbol string    `json:"provider_symbol"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// PriceSnapshot represents a point-in-time market data capture.
// Prices are stored as strings to avoid floating-point precision loss.
type PriceSnapshot struct {
	ID         int64     `json:"id"`
	CoinID     int64     `json:"coin_id"`
	PriceUSD   string    `json:"price_usd"`
	MarketCap  string    `json:"market_cap"`
	Volume24h  string    `json:"volume_24h"`
	Change24h  string    `json:"change_24h"`
	Provider   string    `json:"provider"`
	CapturedAt time.Time `json:"captured_at"`
}

// ProviderSyncLog records the outcome of a provider synchronization attempt.
type ProviderSyncLog struct {
	ID                int64      `json:"id"`
	Provider          string     `json:"provider"`
	RequestDurationMs int64      `json:"request_duration_ms"`
	Status            string     `json:"status"`
	ErrorMessage      *string    `json:"error_message,omitempty"`
	StartedAt         time.Time  `json:"started_at"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
}

// MarketData is the normalized representation of market data from any provider.
type MarketData struct {
	Symbol    string
	PriceUSD  string
	MarketCap string
	Volume24h string
	Change24h string
	Provider  string
	FetchedAt time.Time
}

// LatestMarketData combines coin info with its latest market values for API responses.
type LatestMarketData struct {
	Symbol     string    `json:"symbol"`
	Name       string    `json:"name"`
	PriceUSD   string    `json:"price_usd"`
	MarketCap  string    `json:"market_cap"`
	Volume24h  string    `json:"volume_24h"`
	Change24h  string    `json:"change_24h"`
	Provider   string    `json:"provider"`
	CapturedAt time.Time `json:"captured_at"`
}
