package resilience

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var rateLimitedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "provider_rate_limited_total",
	Help: "Total number of rate-limit (HTTP 429) responses by provider.",
}, []string{"provider"})

// RateLimitTracker tracks rate-limit state per provider.
type RateLimitTracker struct {
	mu        sync.RWMutex
	providers map[string]*providerRateLimit
}

type providerRateLimit struct {
	lastRateLimit time.Time
	retryAfter    time.Duration
	count         int
}

// NewRateLimitTracker creates a new rate limit tracker.
func NewRateLimitTracker() *RateLimitTracker {
	return &RateLimitTracker{
		providers: make(map[string]*providerRateLimit),
	}
}

// RecordRateLimit records a rate-limit event for a provider.
func (t *RateLimitTracker) RecordRateLimit(provider string, retryAfter time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	rl, ok := t.providers[provider]
	if !ok {
		rl = &providerRateLimit{}
		t.providers[provider] = rl
	}

	rl.lastRateLimit = time.Now()
	rl.retryAfter = retryAfter
	rl.count++

	rateLimitedTotal.WithLabelValues(provider).Inc()
}

// ShouldWait returns whether requests to the provider should wait, and for how long.
func (t *RateLimitTracker) ShouldWait(provider string) (bool, time.Duration) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	rl, ok := t.providers[provider]
	if !ok {
		return false, 0
	}

	if rl.retryAfter <= 0 {
		return false, 0
	}

	elapsed := time.Since(rl.lastRateLimit)
	if elapsed >= rl.retryAfter {
		return false, 0
	}

	return true, rl.retryAfter - elapsed
}

// Count returns the total rate-limit count for a provider.
func (t *RateLimitTracker) Count(provider string) int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	rl, ok := t.providers[provider]
	if !ok {
		return 0
	}
	return rl.count
}

// Reset clears rate-limit state for a provider.
func (t *RateLimitTracker) Reset(provider string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.providers, provider)
}

// ParseRetryAfter extracts the retry-after duration from an HTTP response.
// Supports both numeric seconds and HTTP-date formats.
func ParseRetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}

	header := resp.Header.Get("Retry-After")
	if header == "" {
		return 0
	}

	// Try numeric seconds first.
	if seconds, err := strconv.Atoi(header); err == nil {
		if seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
		return 0
	}

	// Try HTTP-date format (RFC 1123).
	if t, err := time.Parse(time.RFC1123, header); err == nil {
		delay := time.Until(t)
		if delay > 0 {
			return delay
		}
		return 0
	}

	// Try RFC 850 format.
	if t, err := time.Parse(time.RFC850, header); err == nil {
		delay := time.Until(t)
		if delay > 0 {
			return delay
		}
		return 0
	}

	return 0
}

// DefaultRetryAfter returns a default retry-after duration when the header is missing.
func DefaultRetryAfter() time.Duration {
	return 60 * time.Second
}
