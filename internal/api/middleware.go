package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/crypto-market-platform/internal/telemetry"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// Middleware provides HTTP middleware for the API.
type Middleware struct {
	logger  *slog.Logger
	metrics *telemetry.Metrics
}

// NewMiddleware creates a new Middleware.
func NewMiddleware(logger *slog.Logger, metrics *telemetry.Metrics) *Middleware {
	return &Middleware{logger: logger, metrics: metrics}
}

// RequestID generates and injects a request ID into the context and response header.
func (m *Middleware) RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = generateRequestID()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logging logs each request with structured fields.
func (m *Middleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		reqID, _ := r.Context().Value(requestIDKey).(string)
		m.logger.Info("request",
			slog.String("request_id", reqID),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", wrapped.statusCode),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

// Metrics records HTTP request duration and response status metrics.
func (m *Middleware) Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(wrapped.statusCode)
		path := normalizePath(r.URL.Path)

		m.metrics.HTTPRequestDuration.WithLabelValues(r.Method, path, status).Observe(duration)
		m.metrics.HTTPResponseTotal.WithLabelValues(r.Method, path, status).Inc()
	})
}

// Recover recovers from panics and returns a 500 response.
func (m *Middleware) Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				m.logger.Error("panic recovered",
					slog.Any("error", err),
					slog.String("path", r.URL.Path),
				)
				writeJSON(w, http.StatusInternalServerError, ErrorResponse{
					Error: "internal server error",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Chain applies middleware in order.
func Chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

func generateRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// normalizePath reduces cardinality by replacing path parameters.
func normalizePath(path string) string {
	// Simple normalization: /coins/BTC -> /coins/{symbol}
	// This avoids high-cardinality metric labels.
	parts := splitPath(path)
	if len(parts) >= 2 && parts[0] == "coins" {
		if len(parts) == 2 {
			return "/coins/{symbol}"
		}
		if len(parts) == 3 && parts[2] == "history" {
			return "/coins/{symbol}/history"
		}
	}
	return path
}

func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// GetRequestID extracts the request ID from context.
func GetRequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
