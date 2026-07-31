package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/crypto-market-platform/internal/market"
	"github.com/crypto-market-platform/internal/telemetry"
)

// mockCoinRepo implements repository.CoinRepository for testing.
type mockCoinRepo struct {
	coins []market.Coin
}

func (m *mockCoinRepo) GetActiveCoins(_ context.Context) ([]market.Coin, error) {
	return m.coins, nil
}

func (m *mockCoinRepo) GetBySymbol(_ context.Context, symbol string) (*market.Coin, error) {
	for _, c := range m.coins {
		if c.Symbol == symbol {
			return &c, nil
		}
	}
	return nil, nil
}

// mockSnapshotRepo implements repository.SnapshotRepository for testing.
type mockSnapshotRepo struct {
	snapshots []market.PriceSnapshot
}

func (m *mockSnapshotRepo) InsertBatch(_ context.Context, snapshots []market.PriceSnapshot) error {
	m.snapshots = append(m.snapshots, snapshots...)
	return nil
}

func (m *mockSnapshotRepo) GetLatestByCoin(_ context.Context, coinID int64) (*market.PriceSnapshot, error) {
	for _, s := range m.snapshots {
		if s.CoinID == coinID {
			return &s, nil
		}
	}
	return nil, nil
}

func (m *mockSnapshotRepo) GetHistory(_ context.Context, coinID int64, limit int, _ *time.Time, _ *time.Time, _ *time.Time) ([]market.PriceSnapshot, error) {
	var result []market.PriceSnapshot
	for _, s := range m.snapshots {
		if s.CoinID == coinID {
			result = append(result, s)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func newTestHandler() *Handler {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	coinRepo := &mockCoinRepo{
		coins: []market.Coin{
			{ID: 1, Symbol: "BTC", Name: "Bitcoin", ProviderSymbol: "bitcoin", IsActive: true},
			{ID: 2, Symbol: "ETH", Name: "Ethereum", ProviderSymbol: "ethereum", IsActive: true},
		},
	}
	snapshotRepo := &mockSnapshotRepo{
		snapshots: []market.PriceSnapshot{
			{ID: 1, CoinID: 1, PriceUSD: "65000", MarketCap: "1300000000000", Volume24h: "50000000000", Change24h: "2.5", Provider: "coingecko", CapturedAt: time.Now()},
		},
	}
	return NewHandler(nil, coinRepo, snapshotRepo, nil, nil, logger)
}

var testMetrics = telemetry.NewMetrics()

func newTestRouter(h *Handler) http.Handler {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	mw := NewMiddleware(logger, testMetrics)

	r := chi.NewRouter()
	r.Get("/health", h.Health)
	r.Get("/coins", h.Coins)
	r.Get("/coins/{symbol}", h.CoinBySymbol)
	r.Get("/coins/{symbol}/history", h.CoinHistory)

	return Chain(r, mw.Recover, mw.RequestID)
}

func TestHealth(t *testing.T) {
	h := newTestHandler()
	router := newTestRouter(h)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %s", resp["status"])
	}
}

func TestCoins(t *testing.T) {
	h := newTestHandler()
	router := newTestRouter(h)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/coins", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp PaginatedResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Count != 2 {
		t.Errorf("expected 2 coins, got %d", resp.Count)
	}
}

func TestCoinBySymbol_Found(t *testing.T) {
	h := newTestHandler()
	router := newTestRouter(h)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/coins/BTC", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp market.LatestMarketData
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Symbol != "BTC" {
		t.Errorf("expected symbol BTC, got %s", resp.Symbol)
	}
	if resp.PriceUSD != "65000" {
		t.Errorf("expected price 65000, got %s", resp.PriceUSD)
	}
}

func TestCoinBySymbol_NotFound(t *testing.T) {
	h := newTestHandler()
	router := newTestRouter(h)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/coins/INVALID", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestCoinHistory(t *testing.T) {
	h := newTestHandler()
	router := newTestRouter(h)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/coins/BTC/history?limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp PaginatedResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Count != 1 {
		t.Errorf("expected 1 snapshot, got %d", resp.Count)
	}
}

func TestCoinHistory_NotFound(t *testing.T) {
	h := newTestHandler()
	router := newTestRouter(h)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/coins/INVALID/history", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestRequestID_Header(t *testing.T) {
	h := newTestHandler()
	router := newTestRouter(h)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header to be set")
	}
}

func TestRequestID_PreservesExisting(t *testing.T) {
	h := newTestHandler()
	router := newTestRouter(h)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)
	req.Header.Set("X-Request-ID", "test-id-123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") != "test-id-123" {
		t.Errorf("expected X-Request-ID 'test-id-123', got %s", w.Header().Get("X-Request-ID"))
	}
}
