package api

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/crypto-market-platform/internal/cache"
	"github.com/crypto-market-platform/internal/market"
	"github.com/crypto-market-platform/internal/repository"
)

const (
	defaultHistoryLimit = 50
	maxHistoryLimit     = 500
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	db           *sql.DB
	coinRepo     repository.CoinRepository
	snapshotRepo repository.SnapshotRepository
	cache        *cache.MarketCache
	logger       *slog.Logger
}

// NewHandler creates a new Handler.
func NewHandler(
	db *sql.DB,
	coinRepo repository.CoinRepository,
	snapshotRepo repository.SnapshotRepository,
	cache *cache.MarketCache,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		db:           db,
		coinRepo:     coinRepo,
		snapshotRepo: snapshotRepo,
		cache:        cache,
		logger:       logger,
	}
}

// Health returns a simple liveness check.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready checks required dependencies (PostgreSQL and Redis).
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := h.db.PingContext(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not_ready",
			"error":  "postgres unavailable",
		})
		return
	}

	if h.cache != nil {
		if err := h.cache.Ping(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "not_ready",
				"error":  "redis unavailable",
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// Markets returns the latest cached market values for all active coins.
func (h *Handler) Markets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	coins, err := h.coinRepo.GetActiveCoins(ctx)
	if err != nil {
		h.logger.Error("failed to get active coins", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to retrieve market data")
		return
	}

	symbols := make([]string, len(coins))
	for i, c := range coins {
		symbols[i] = c.Symbol
	}

	var data []market.LatestMarketData
	if h.cache != nil {
		data, err = h.cache.GetAllLatest(ctx, symbols)
		if err != nil {
			h.logger.Error("failed to get cached markets", slog.String("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "failed to retrieve market data")
			return
		}
	}

	if data == nil {
		data = []market.LatestMarketData{}
	}

	writeJSON(w, http.StatusOK, PaginatedResponse{Data: data, Count: len(data)})
}

// Coins returns all supported active coins.
func (h *Handler) Coins(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	coins, err := h.coinRepo.GetActiveCoins(ctx)
	if err != nil {
		h.logger.Error("failed to get coins", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to retrieve coins")
		return
	}

	if coins == nil {
		coins = []market.Coin{}
	}

	writeJSON(w, http.StatusOK, PaginatedResponse{Data: coins, Count: len(coins)})
}

// CoinBySymbol returns the latest market data for a specific coin.
func (h *Handler) CoinBySymbol(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	symbol := strings.ToUpper(chi.URLParam(r, "symbol"))

	if symbol == "" {
		writeError(w, http.StatusBadRequest, "symbol is required")
		return
	}

	coin, err := h.coinRepo.GetBySymbol(ctx, symbol)
	if err != nil {
		h.logger.Error("failed to get coin", slog.String("symbol", symbol), slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to retrieve coin")
		return
	}
	if coin == nil {
		writeError(w, http.StatusNotFound, "coin not found")
		return
	}

	// Try cache first.
	if h.cache != nil {
		latest, err := h.cache.GetLatest(ctx, symbol)
		if err == nil && latest != nil {
			writeJSON(w, http.StatusOK, latest)
			return
		}
	}

	// Fallback to database.
	snapshot, err := h.snapshotRepo.GetLatestByCoin(ctx, coin.ID)
	if err != nil {
		h.logger.Error("failed to get latest snapshot", slog.String("symbol", symbol), slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to retrieve market data")
		return
	}
	if snapshot == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"symbol": coin.Symbol,
			"name":   coin.Name,
			"data":   nil,
		})
		return
	}

	writeJSON(w, http.StatusOK, market.LatestMarketData{
		Symbol:     coin.Symbol,
		Name:       coin.Name,
		PriceUSD:   snapshot.PriceUSD,
		MarketCap:  snapshot.MarketCap,
		Volume24h:  snapshot.Volume24h,
		Change24h:  snapshot.Change24h,
		Provider:   snapshot.Provider,
		CapturedAt: snapshot.CapturedAt,
	})
}

// CoinHistory returns paginated historical snapshots for a coin.
func (h *Handler) CoinHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	symbol := strings.ToUpper(chi.URLParam(r, "symbol"))

	if symbol == "" {
		writeError(w, http.StatusBadRequest, "symbol is required")
		return
	}

	coin, err := h.coinRepo.GetBySymbol(ctx, symbol)
	if err != nil {
		h.logger.Error("failed to get coin", slog.String("symbol", symbol), slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to retrieve coin")
		return
	}
	if coin == nil {
		writeError(w, http.StatusNotFound, "coin not found")
		return
	}

	// Parse query parameters.
	limit := parseIntParam(r, "limit", defaultHistoryLimit)
	if limit > maxHistoryLimit {
		limit = maxHistoryLimit
	}
	if limit < 1 {
		limit = 1
	}

	var before, from, to *time.Time
	if v := r.URL.Query().Get("before"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'before' parameter, use RFC3339 format")
			return
		}
		before = &t
	}
	if v := r.URL.Query().Get("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'from' parameter, use RFC3339 format")
			return
		}
		from = &t
	}
	if v := r.URL.Query().Get("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'to' parameter, use RFC3339 format")
			return
		}
		to = &t
	}

	snapshots, err := h.snapshotRepo.GetHistory(ctx, coin.ID, limit, before, from, to)
	if err != nil {
		h.logger.Error("failed to get history", slog.String("symbol", symbol), slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "failed to retrieve history")
		return
	}

	if snapshots == nil {
		snapshots = []market.PriceSnapshot{}
	}

	writeJSON(w, http.StatusOK, PaginatedResponse{Data: snapshots, Count: len(snapshots)})
}

func parseIntParam(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}
