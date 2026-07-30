package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCoinCapProvider_Name(t *testing.T) {
	p := NewCoinCapProvider("http://localhost", 5*time.Second)
	if p.Name() != "coincap" {
		t.Errorf("expected name 'coincap', got %q", p.Name())
	}
}

func TestCoinCapProvider_FetchMarketData_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/assets" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ids := r.URL.Query().Get("ids")
		if ids != "bitcoin,ethereum" {
			t.Errorf("unexpected ids: %s", ids)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": [
				{
					"id": "bitcoin",
					"symbol": "BTC",
					"priceUsd": "65000.50",
					"marketCapUsd": "1300000000000",
					"volumeUsd24Hr": "50000000000",
					"changePercent24Hr": "2.5"
				},
				{
					"id": "ethereum",
					"symbol": "ETH",
					"priceUsd": "3500.25",
					"marketCapUsd": "420000000000",
					"volumeUsd24Hr": "25000000000",
					"changePercent24Hr": "-1.2"
				}
			]
		}`))
	}))
	defer server.Close()

	p := NewCoinCapProvider(server.URL, 5*time.Second)
	data, err := p.FetchMarketData(context.Background(), []string{"bitcoin", "ethereum"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 2 {
		t.Fatalf("expected 2 results, got %d", len(data))
	}

	// Check BTC
	if data[0].Symbol != "BTC" {
		t.Errorf("expected symbol BTC, got %s", data[0].Symbol)
	}
	if data[0].PriceUSD != "65000.50" {
		t.Errorf("expected price 65000.50, got %s", data[0].PriceUSD)
	}
	if data[0].Provider != "coincap" {
		t.Errorf("expected provider coincap, got %s", data[0].Provider)
	}

	// Check ETH
	if data[1].Symbol != "ETH" {
		t.Errorf("expected symbol ETH, got %s", data[1].Symbol)
	}
}

func TestCoinCapProvider_FetchMarketData_EmptySymbols(t *testing.T) {
	p := NewCoinCapProvider("http://localhost", 5*time.Second)
	data, err := p.FetchMarketData(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil data, got %v", data)
	}
}

func TestCoinCapProvider_FetchMarketData_RateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "rate limit exceeded"}`))
	}))
	defer server.Close()

	p := NewCoinCapProvider(server.URL, 5*time.Second)
	_, err := p.FetchMarketData(context.Background(), []string{"bitcoin"})
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !IsRateLimited(err) {
		t.Errorf("expected rate-limited error, got: %v", err)
	}
	if !IsTransient(err) {
		t.Errorf("rate limit should be transient")
	}
}

func TestCoinCapProvider_FetchMarketData_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`internal error`))
	}))
	defer server.Close()

	p := NewCoinCapProvider(server.URL, 5*time.Second)
	_, err := p.FetchMarketData(context.Background(), []string{"bitcoin"})
	if err == nil {
		t.Fatal("expected error for server error")
	}
	if !IsTransient(err) {
		t.Errorf("5xx should be transient, got: %v", err)
	}
	if IsPermanent(err) {
		t.Errorf("5xx should not be permanent")
	}
}

func TestCoinCapProvider_FetchMarketData_ClientError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`bad request`))
	}))
	defer server.Close()

	p := NewCoinCapProvider(server.URL, 5*time.Second)
	_, err := p.FetchMarketData(context.Background(), []string{"bitcoin"})
	if err == nil {
		t.Fatal("expected error for client error")
	}
	if !IsPermanent(err) {
		t.Errorf("4xx should be permanent, got: %v", err)
	}
	if IsTransient(err) {
		t.Errorf("4xx should not be transient")
	}
}

func TestCoinCapProvider_FetchMarketData_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	p := NewCoinCapProvider(server.URL, 5*time.Second)
	_, err := p.FetchMarketData(context.Background(), []string{"bitcoin"})
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !IsPermanent(err) {
		t.Errorf("malformed JSON should be permanent, got: %v", err)
	}
}

func TestCoinCapProvider_FetchMarketData_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	p := NewCoinCapProvider(server.URL, 5*time.Second)
	_, err := p.FetchMarketData(ctx, []string{"bitcoin"})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !IsTransient(err) {
		t.Errorf("timeout should be transient, got: %v", err)
	}
}

func TestCoinCapProvider_FetchMarketData_SkipsInvalidPrice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": [
				{"id": "bitcoin", "symbol": "BTC", "priceUsd": "65000"},
				{"id": "invalid", "symbol": "INV", "priceUsd": "not-a-number"},
				{"id": "empty", "symbol": "EMP", "priceUsd": ""}
			]
		}`))
	}))
	defer server.Close()

	p := NewCoinCapProvider(server.URL, 5*time.Second)
	data, err := p.FetchMarketData(context.Background(), []string{"bitcoin", "invalid", "empty"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 1 {
		t.Fatalf("expected 1 valid result, got %d", len(data))
	}
	if data[0].Symbol != "BTC" {
		t.Errorf("expected BTC, got %s", data[0].Symbol)
	}
}
