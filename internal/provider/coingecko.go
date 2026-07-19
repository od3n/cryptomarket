package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/crypto-market-platform/internal/market"
)

const providerName = "coingecko"

// CoinGeckoProvider implements the Provider interface using the CoinGecko API.
type CoinGeckoProvider struct {
	baseURL    string
	httpClient *http.Client
}

// NewCoinGeckoProvider creates a new CoinGecko provider adapter.
func NewCoinGeckoProvider(baseURL string, timeout time.Duration) *CoinGeckoProvider {
	return &CoinGeckoProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (p *CoinGeckoProvider) Name() string {
	return providerName
}

// coingeckoMarketResponse represents a single item from the CoinGecko /coins/markets endpoint.
type coingeckoMarketResponse struct {
	ID                string   `json:"id"`
	Symbol            string   `json:"symbol"`
	CurrentPrice      *float64 `json:"current_price"`
	MarketCap         *float64 `json:"market_cap"`
	TotalVolume       *float64 `json:"total_volume"`
	PriceChangePct24h *float64 `json:"price_change_percentage_24h"`
}

func (p *CoinGeckoProvider) FetchMarketData(ctx context.Context, providerSymbols []string) ([]market.MarketData, error) {
	if len(providerSymbols) == 0 {
		return nil, nil
	}

	endpoint := fmt.Sprintf("%s/coins/markets", p.baseURL)
	params := url.Values{}
	params.Set("vs_currency", "usd")
	params.Set("ids", strings.Join(providerSymbols, ","))
	params.Set("order", "market_cap_desc")
	params.Set("per_page", strconv.Itoa(len(providerSymbols)))
	params.Set("page", "1")

	reqURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, &ProviderError{
			Provider: providerName,
			Message:  "failed to create request",
			Err:      err,
		}
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, &ProviderError{
			Provider: providerName,
			Message:  "request failed",
			Err:      err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, &ProviderError{
			Provider:   providerName,
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("unexpected status: %s", string(body)),
		}
	}

	var items []coingeckoMarketResponse
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, &ProviderError{
			Provider: providerName,
			Message:  "failed to decode response",
			Err:      err,
		}
	}

	now := time.Now().UTC()
	results := make([]market.MarketData, 0, len(items))
	for _, item := range items {
		if item.CurrentPrice == nil {
			continue
		}

		md := market.MarketData{
			Symbol:    strings.ToUpper(item.Symbol),
			PriceUSD:  formatFloat(*item.CurrentPrice),
			Provider:  providerName,
			FetchedAt: now,
		}
		if item.MarketCap != nil {
			md.MarketCap = formatFloat(*item.MarketCap)
		}
		if item.TotalVolume != nil {
			md.Volume24h = formatFloat(*item.TotalVolume)
		}
		if item.PriceChangePct24h != nil {
			md.Change24h = formatFloat(*item.PriceChangePct24h)
		}

		results = append(results, md)
	}

	return results, nil
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
