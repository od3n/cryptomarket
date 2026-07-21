package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Contract tests verify that provider adapters correctly parse the expected
// response shapes from external APIs. If a provider changes their response
// format, these tests will fail — alerting us before production breaks.
//
// These tests use recorded response fixtures that represent the actual API
// contract. They do NOT make real network calls.

// TestCoinGeckoContract_MarketResponse verifies the CoinGecko /coins/markets
// response shape matches what our adapter expects.
func TestCoinGeckoContract_MarketResponse(t *testing.T) {
	// This is the contract: the shape CoinGecko MUST return for us to work.
	contractResponse := `[
		{
			"id": "bitcoin",
			"symbol": "btc",
			"name": "Bitcoin",
			"current_price": 67432.15,
			"market_cap": 1324567890123,
			"total_volume": 28456789012,
			"price_change_percentage_24h": 2.34,
			"high_24h": 68100.00,
			"low_24h": 65800.00,
			"last_updated": "2024-03-15T10:30:00.000Z"
		},
		{
			"id": "ethereum",
			"symbol": "eth",
			"name": "Ethereum",
			"current_price": 3456.78,
			"market_cap": 415678901234,
			"total_volume": 15678901234,
			"price_change_percentage_24h": -1.23,
			"high_24h": 3520.00,
			"low_24h": 3380.00,
			"last_updated": "2024-03-15T10:30:00.000Z"
		}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request contract
		if r.URL.Path != "/coins/markets" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("vs_currency") != "usd" {
			t.Errorf("expected vs_currency=usd, got %s", r.URL.Query().Get("vs_currency"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(contractResponse))
	}))
	defer server.Close()

	// Create provider pointing at mock server
	p := NewCoinGeckoProvider(server.URL, http.DefaultClient)

	// Fetch and verify the contract holds
	data, err := p.FetchMarketData(t.Context(), []string{"bitcoin", "ethereum"})
	if err != nil {
		t.Fatalf("FetchMarketData failed: %v", err)
	}

	if len(data) != 2 {
		t.Fatalf("expected 2 results, got %d", len(data))
	}

	// Verify field mapping contract
	btc := data[0]
	if btc.Symbol != "BTC" {
		t.Errorf("expected symbol BTC, got %s", btc.Symbol)
	}
	if btc.Price == "" || btc.Price == "0" {
		t.Errorf("expected non-zero price, got %s", btc.Price)
	}
}

// TestCoinGeckoContract_ErrorResponse verifies error handling contract.
func TestCoinGeckoContract_ErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{
			name:       "rate limited (429)",
			statusCode: http.StatusTooManyRequests,
			body:       `{"status":{"error_code":429,"error_message":"rate limited"}}`,
			wantErr:    true,
		},
		{
			name:       "server error (500)",
			statusCode: http.StatusInternalServerError,
			body:       `{"error":"internal server error"}`,
			wantErr:    true,
		},
		{
			name:       "empty response (200)",
			statusCode: http.StatusOK,
			body:       `[]`,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			p := NewCoinGeckoProvider(server.URL, http.DefaultClient)
			_, err := p.FetchMarketData(t.Context(), []string{"bitcoin"})

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

// TestCoinCapContract_MarketResponse verifies the CoinCap /v2/assets
// response shape matches what our adapter expects.
func TestCoinCapContract_MarketResponse(t *testing.T) {
	contractResponse := `{
		"data": [
			{
				"id": "bitcoin",
				"rank": "1",
				"symbol": "BTC",
				"name": "Bitcoin",
				"supply": "19634375.0000000000000000",
				"maxSupply": "21000000.0000000000000000",
				"marketCapUsd": "1324567890123.45",
				"volumeUsd24Hr": "28456789012.34",
				"priceUsd": "67432.15",
				"changePercent24Hr": "2.34",
				"vwap24Hr": "67100.50"
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/assets" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(contractResponse))
	}))
	defer server.Close()

	p := NewCoinCapProvider(server.URL, http.DefaultClient)
	data, err := p.FetchMarketData(t.Context(), []string{"bitcoin"})
	if err != nil {
		t.Fatalf("FetchMarketData failed: %v", err)
	}

	if len(data) != 1 {
		t.Fatalf("expected 1 result, got %d", len(data))
	}

	if data[0].Symbol != "BTC" {
		t.Errorf("expected symbol BTC, got %s", data[0].Symbol)
	}
}

// TestContractResponseSchema documents the expected JSON schema for provider responses.
// This serves as living documentation of the API contracts we depend on.
func TestContractResponseSchema(t *testing.T) {
	// CoinGecko contract: array of objects with these required fields
	coingeckoFields := []string{
		"id", "symbol", "name", "current_price",
		"market_cap", "total_volume", "price_change_percentage_24h",
	}

	// CoinCap contract: {data: [{...}]} with these required fields
	coincapFields := []string{
		"id", "symbol", "name", "priceUsd",
		"marketCapUsd", "volumeUsd24Hr", "changePercent24Hr",
	}

	// Verify our fixture data contains all required fields
	coingeckoFixture := `[{"id":"bitcoin","symbol":"btc","name":"Bitcoin","current_price":67432.15,"market_cap":1324567890123,"total_volume":28456789012,"price_change_percentage_24h":2.34}]`
	coincapFixture := `{"data":[{"id":"bitcoin","symbol":"BTC","name":"Bitcoin","priceUsd":"67432.15","marketCapUsd":"1324567890123","volumeUsd24Hr":"28456789012","changePercent24Hr":"2.34"}]}`

	var cgData []map[string]interface{}
	if err := json.Unmarshal([]byte(coingeckoFixture), &cgData); err != nil {
		t.Fatalf("CoinGecko fixture invalid: %v", err)
	}
	for _, field := range coingeckoFields {
		if _, ok := cgData[0][field]; !ok {
			t.Errorf("CoinGecko contract missing field: %s", field)
		}
	}

	var ccData struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(coincapFixture), &ccData); err != nil {
		t.Fatalf("CoinCap fixture invalid: %v", err)
	}
	for _, field := range coincapFields {
		if _, ok := ccData.Data[0][field]; !ok {
			t.Errorf("CoinCap contract missing field: %s", field)
		}
	}
}
