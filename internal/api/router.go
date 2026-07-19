package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewRouter creates the HTTP router with all routes and middleware.
func NewRouter(h *Handler, mw *Middleware) http.Handler {
	r := chi.NewRouter()

	r.Get("/health", h.Health)
	r.Get("/ready", h.Ready)
	r.Get("/markets", h.Markets)
	r.Get("/coins", h.Coins)
	r.Get("/coins/{symbol}", h.CoinBySymbol)
	r.Get("/coins/{symbol}/history", h.CoinHistory)
	r.Get("/providers/status", h.ProviderStatus)
	r.Method("GET", "/metrics", promhttp.Handler())

	return Chain(r,
		mw.Recover,
		mw.RequestID,
		mw.Logging,
		mw.Metrics,
	)
}
