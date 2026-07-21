package api

import (
	"net/http"
	"net/http/pprof"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewRouter creates the HTTP router with all routes and middleware.
func NewRouter(h *Handler, mw *Middleware, authCfg *AuthConfig) http.Handler {
	r := chi.NewRouter()

	r.Get("/health", h.Health)
	r.Get("/ready", h.Ready)
	r.Get("/markets", h.Markets)
	r.Get("/coins", h.Coins)
	r.Get("/coins/{symbol}", h.CoinBySymbol)
	r.Get("/coins/{symbol}/history", h.CoinHistory)
	r.Get("/providers/status", h.ProviderStatus)
	r.Get("/operations/status", h.OperationsStatus)
	r.Method("GET", "/metrics", promhttp.Handler())

	// pprof debug endpoints (bound to localhost in production via config)
	r.Route("/debug/pprof", func(r chi.Router) {
		r.HandleFunc("/", pprof.Index)
		r.HandleFunc("/cmdline", pprof.Cmdline)
		r.HandleFunc("/profile", pprof.Profile)
		r.HandleFunc("/symbol", pprof.Symbol)
		r.HandleFunc("/trace", pprof.Trace)
		r.Handle("/allocs", pprof.Handler("allocs"))
		r.Handle("/block", pprof.Handler("block"))
		r.Handle("/goroutine", pprof.Handler("goroutine"))
		r.Handle("/heap", pprof.Handler("heap"))
		r.Handle("/mutex", pprof.Handler("mutex"))
		r.Handle("/threadcreate", pprof.Handler("threadcreate"))
	})

	return Chain(r,
		mw.Recover,
		mw.RequestID,
		mw.SecurityHeaders,
		mw.Authenticate(authCfg),
		mw.RateLimit(100, 200),
		mw.MaxBodySize(1<<20), // 1 MiB
		mw.Compress,
		mw.Logging,
		mw.Metrics,
	)
}
