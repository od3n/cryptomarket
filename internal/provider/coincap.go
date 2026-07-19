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

const coincapProviderName = "coincap"

// CoinCapProvider implements the Provider interface using the CoinCap API.
type CoinCapProvider struct {
	baseURL    string
	httpClient *http.Client
}

// NewCoinCapProvider creates a new CoinCap provider adapter.
func NewCoinCapProvider(baseURL string, timeout time.Duration) *CoinCapProvider {
	return &CoinCapProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (p *CoinCapProvider) Name() string {
	return coincapProviderName
}

// coincapAssetResponse represents a single asset from the CoinCap /v2/assets endpoint.
type coincapAssetResponse struct {
	ID                string `json:"id"`
	Symbol            string `json:"symbol"`
	PriceUSD          string `json:"priceUsd"`
	MarketCapUSD      string `json:"marketCapUsd"`
	VolumeUsd24Hr     string `json:"volumeUsd24Hr"`
	ChangePercent24Hr string `json:"changePercent24Hr"`
}

// coincapListResponse wraps the CoinCap API list response.
type coincapListResponse struct {
	Data []coincapAssetResponse `json:"data"`
}

func (p *CoinCapProvider) FetchMarketData(ctx context.Context, providerSymbols []string) ([]market.MarketData, error) {
	if len(providerSymbols) == 0 {
		return nil, nil
	}

	endpoint := fmt.Sprintf("%s/v2/assets", p.baseURL)
	params := url.Values{}
	params.Set("ids", strings.Join(providerSymbols, ","))

	reqURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, &ProviderError{
			Provider:  coincapProviderName,
			Message:   "failed to create request",
			Err:       err,
			Permanent: true,
		}
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, &ProviderError{
			Provider:  coincapProviderName,
			Message:   "request failed",
			Err:       err,
			Transient: true,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, &ProviderError{
			Provider:    coincapProviderName,
			StatusCode:  resp.StatusCode,
			Message:     "rate limited",
			RateLimited: true,
		}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		transient := resp.StatusCode >= 500
		return nil, &ProviderError{
			Provider:   coincapProviderName,
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("unexpected status: %s", string(body)),
			Transient:  transient,
			Permanent:  !transient,
		}
	}

	var listResp coincapListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, &ProviderError{
			Provider:  coincapProviderName,
			Message:   "failed to decode response",
			Err:       err,
			Permanent: true,
		}
	}

	now := time.Now().UTC()
	results := make([]market.MarketData, 0, len(listResp.Data))
	for _, item := range listResp.Data {
		if item.PriceUSD == "" {
			continue
		}

		// Validate price is a valid number.
		if _, err := strconv.ParseFloat(item.PriceUSD, 64); err != nil {
			continue
		}

		md := market.MarketData{
			Symbol:    strings.ToUpper(item.Symbol),
			PriceUSD:  item.PriceUSD,
			Provider:  coincapProviderName,
			FetchedAt: now,
		}
		if item.MarketCapUSD != "" {
			md.MarketCap = item.MarketCapUSD
		}
		if item.VolumeUsd24Hr != "" {
			md.Volume24h = item.VolumeUsd24Hr
		}
		if item.ChangePercent24Hr != "" {
			md.Change24h = item.ChangePercent24Hr
		}

		results = append(results, md)
	}

	return results, nil
}
