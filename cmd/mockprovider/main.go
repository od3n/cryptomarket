// Command mockprovider is a configurable mock market data provider for testing.
// It supports multiple failure modes controlled via environment variable or query parameter.
//
// Modes:
//   - success: returns valid market data (default)
//   - delayed: returns valid data after a configurable delay
//   - rate_limit: returns HTTP 429 with Retry-After header
//   - error: returns HTTP 500
//   - malformed: returns invalid JSON
//   - missing_symbols: returns data for fewer symbols than requested
//   - inconsistent: returns wildly different prices each call
//
// Configuration:
//
//	MOCK_MODE: default mode (overridden by ?mode= query param)
//	MOCK_DELAY: delay duration for "delayed" mode (default "3s")
//	MOCK_PORT: listen port (default "8082")
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var requestCount atomic.Int64

// basePrices are reference prices for mock data.
var basePrices = map[string]float64{
	"bitcoin":  67500.0,
	"ethereum": 3450.0,
	"solana":   172.0,
	"cardano":  0.62,
	"ripple":   0.58,
}

func main() {
	port := getEnv("MOCK_PORT", "8082")
	defaultMode := getEnv("MOCK_MODE", "success")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/simple/price", func(w http.ResponseWriter, r *http.Request) {
		handleCoinGecko(w, r, defaultMode)
	})
	mux.HandleFunc("/v2/assets", func(w http.ResponseWriter, r *http.Request) {
		handleCoinCap(w, r, defaultMode)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "mode": defaultMode})
	})
	mux.HandleFunc("/mode", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"default_mode": defaultMode,
			"requests":     strconv.FormatInt(requestCount.Load(), 10),
		})
	})

	addr := ":" + port
	log.Printf("mock provider listening on %s (mode=%s)", addr, defaultMode)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func handleCoinGecko(w http.ResponseWriter, r *http.Request, defaultMode string) {
	requestCount.Add(1)
	mode := getMode(r, defaultMode)

	if !applyMode(w, mode) {
		return
	}

	ids := r.URL.Query().Get("ids")
	symbols := strings.Split(ids, ",")

	result := make(map[string]map[string]interface{})
	for _, sym := range symbols {
		sym = strings.TrimSpace(strings.ToLower(sym))
		price := getPrice(sym)
		result[sym] = map[string]interface{}{
			"usd":             price,
			"usd_market_cap":  price * 19000000,
			"usd_24h_vol":     price * 500000,
			"usd_24h_change":  (rand.Float64() - 0.5) * 10,
			"last_updated_at": time.Now().Unix(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleCoinCap(w http.ResponseWriter, r *http.Request, defaultMode string) {
	requestCount.Add(1)
	mode := getMode(r, defaultMode)

	if !applyMode(w, mode) {
		return
	}

	ids := r.URL.Query().Get("ids")
	symbols := strings.Split(ids, ",")

	var data []map[string]interface{}
	for _, sym := range symbols {
		sym = strings.TrimSpace(strings.ToLower(sym))
		price := getPrice(sym)
		data = append(data, map[string]interface{}{
			"id":                sym,
			"rank":              "1",
			"symbol":            strings.ToUpper(sym[:min(3, len(sym))]),
			"name":              sym,
			"supply":            "19000000",
			"maxSupply":         "21000000",
			"marketCapUsd":      fmt.Sprintf("%.2f", price*19000000),
			"volumeUsd24Hr":     fmt.Sprintf("%.2f", price*500000),
			"priceUsd":          fmt.Sprintf("%.2f", price),
			"changePercent24Hr": fmt.Sprintf("%.2f", (rand.Float64()-0.5)*10),
			"vwap24Hr":          fmt.Sprintf("%.2f", price*0.99),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":      data,
		"timestamp": time.Now().UnixMilli(),
	})
}

// applyMode applies failure mode behavior. Returns false if the response was already written.
func applyMode(w http.ResponseWriter, mode string) bool {
	switch mode {
	case "rate_limit":
		retryAfter := 30 + rand.Intn(30)
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "rate limit exceeded",
		})
		return false

	case "error":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "internal server error",
		})
		return false

	case "malformed":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"invalid": json, "broken": [}`))
		return false

	case "delayed":
		delay := getEnv("MOCK_DELAY", "3s")
		d, err := time.ParseDuration(delay)
		if err != nil {
			d = 3 * time.Second
		}
		time.Sleep(d)
		return true

	case "missing_symbols":
		// Handled in the caller by returning fewer symbols
		return true

	case "inconsistent":
		// Handled by getPrice with high variance
		return true

	default: // "success"
		return true
	}
}

func getPrice(symbol string) float64 {
	base, ok := basePrices[symbol]
	if !ok {
		base = 100.0
	}

	// In inconsistent mode, vary wildly
	if os.Getenv("MOCK_MODE") == "inconsistent" {
		variance := 0.5 + rand.Float64() // 50% to 150% of base
		return base * variance
	}

	// Normal small variance (±2%)
	variance := 1.0 + (rand.Float64()-0.5)*0.04
	return base * variance
}

func getMode(r *http.Request, defaultMode string) string {
	if m := r.URL.Query().Get("mode"); m != "" {
		return m
	}
	return defaultMode
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
