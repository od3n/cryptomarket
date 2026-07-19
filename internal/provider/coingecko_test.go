package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCoinGeckoProvider_FetchMarketData_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/coins/markets" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("vs_currency") != "usd" {
			t.Errorf("expected vs_currency=usd")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[
			{"id":"bitcoin","symbol":"btc","current_price":65000.5,"market_cap":1300000000000,"total_volume":50000000000,"price_change_percentage_24h":2.5},
			{"id":"ethereum","symbol":"eth","current_price":3500.25,"market_cap":420000000000,"total_volume":20000000000,"price_change_percentage_24h":-1.2}
		]`))
	}))
	defer server.Close()

	p := NewCoinGeckoProvider(server.URL, 5*time.Second)
	data, err := p.FetchMarketData(context.Background(), []string{"bitcoin", "ethereum"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(data) != 2 {
		t.Fatalf("expected 2 results, got %d", len(data))
	}

	if data[0].Symbol != "BTC" {
		t.Errorf("expected symbol BTC, got %s", data[0].Symbol)
	}
	if data[0].PriceUSD != "65000.5" {
		t.Errorf("expected price 65000.5, got %s", data[0].PriceUSD)
	}
	if data[0].Provider != "coingecko" {
		t.Errorf("expected provider coingecko, got %s", data[0].Provider)
	}
	if data[1].Symbol != "ETH" {
		t.Errorf("expected symbol ETH, got %s", data[1].Symbol)
	}
}

func TestCoinGeckoProvider_FetchMarketData_NonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer server.Close()

	p := NewCoinGeckoProvider(server.URL, 5*time.Second)
	_, err := p.FetchMarketData(context.Background(), []string{"bitcoin"})
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}

	provErr, ok := err.(*ProviderError)
	if !ok {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if provErr.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", provErr.StatusCode)
	}
}

func TestCoinGeckoProvider_FetchMarketData_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	p := NewCoinGeckoProvider(server.URL, 5*time.Second)
	_, err := p.FetchMarketData(context.Background(), []string{"bitcoin"})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestCoinGeckoProvider_FetchMarketData_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := NewCoinGeckoProvider(server.URL, 5*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := p.FetchMarketData(ctx, []string{"bitcoin"})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestCoinGeckoProvider_FetchMarketData_EmptySymbols(t *testing.T) {
	p := NewCoinGeckoProvider("http://unused", 5*time.Second)
	data, err := p.FetchMarketData(context.Background(), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil data for empty symbols, got %v", data)
	}
}

func TestCoinGeckoProvider_FetchMarketData_NilPrice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":"bitcoin","symbol":"btc","current_price":null}]`))
	}))
	defer server.Close()

	p := NewCoinGeckoProvider(server.URL, 5*time.Second)
	data, err := p.FetchMarketData(context.Background(), []string{"bitcoin"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected 0 results for nil price, got %d", len(data))
	}
}

func TestCoinGeckoProvider_Name(t *testing.T) {
	p := NewCoinGeckoProvider("http://unused", 5*time.Second)
	if p.Name() != "coingecko" {
		t.Errorf("expected name 'coingecko', got %s", p.Name())
	}
}
