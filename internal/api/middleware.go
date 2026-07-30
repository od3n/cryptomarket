package api

import (
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
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

// SecurityHeaders adds OWASP-recommended security headers to every response.
func (m *Middleware) SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-XSS-Protection", "0")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Cache-Control", "no-store")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

// MaxBodySize limits the request body to prevent memory exhaustion attacks.
func (m *Middleware) MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				writeJSON(w, http.StatusRequestEntityTooLarge, ErrorResponse{
					Error: "request body too large",
				})
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit implements a token-bucket rate limiter per client IP.
func (m *Middleware) RateLimit(requestsPerSecond int, burst int) func(http.Handler) http.Handler {
	type client struct {
		tokens   float64
		lastSeen time.Time
	}
	var mu sync.Mutex
	clients := make(map[string]*client)

	// Background cleanup of stale entries.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			now := time.Now()
			for ip, c := range clients {
				if now.Sub(c.lastSeen) > 10*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractClientIP(r)

			mu.Lock()
			now := time.Now()
			c, exists := clients[ip]
			if !exists {
				c = &client{tokens: float64(burst), lastSeen: now}
				clients[ip] = c
			}

			// Refill tokens.
			elapsed := now.Sub(c.lastSeen).Seconds()
			c.tokens += elapsed * float64(requestsPerSecond)
			if c.tokens > float64(burst) {
				c.tokens = float64(burst)
			}
			c.lastSeen = now

			if c.tokens < 1 {
				mu.Unlock()
				w.Header().Set("Retry-After", "1")
				writeJSON(w, http.StatusTooManyRequests, ErrorResponse{
					Error: "rate limit exceeded",
				})
				return
			}
			c.tokens--
			mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain.
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

// gzipResponseWriter wraps http.ResponseWriter with gzip compression.
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	n, err := w.Writer.Write(b)
	if err != nil {
		return n, fmt.Errorf("gzip write: %w", err)
	}
	return n, nil
}

var gzipPool = sync.Pool{
	New: func() interface{} {
		gz, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		return gz
	},
}

// Compress applies gzip compression to responses when the client supports it.
func (m *Middleware) Compress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// Skip compression for small responses and SSE streams.
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")

		gz, _ := gzipPool.Get().(*gzip.Writer)
		defer gzipPool.Put(gz)
		gz.Reset(w)
		defer gz.Close()

		next.ServeHTTP(&gzipResponseWriter{Writer: gz, ResponseWriter: w}, r)
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
	// crypto/rand.Read never fails on supported platforms.
	_, _ = rand.Read(b)
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
