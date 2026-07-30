package resilience

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// RetryConfig holds configuration for retry behavior.
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts (including the initial one).
	MaxAttempts int
	// BaseDelay is the initial delay before the first retry.
	BaseDelay time.Duration
	// MaxDelay caps the exponential backoff delay.
	MaxDelay time.Duration
	// RetryableFunc determines if an error is retryable.
	RetryableFunc func(error) bool
}

// DefaultRetryConfig returns sensible defaults for retry behavior.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    10 * time.Second,
		RetryableFunc: func(_ error) bool {
			return true // Default: retry all errors
		},
	}
}

var retryAttemptsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "retry_attempts_total",
	Help: "Total number of retry attempts.",
}, []string{"result"})

// Retry executes fn with bounded exponential backoff and jitter.
// It respects context cancellation and only retries errors deemed retryable.
//
// Retry ownership: The provider adapter layer owns retry responsibility.
// Higher layers (fallback orchestrator) should not add additional retries
// to avoid retry multiplication.
func Retry(ctx context.Context, cfg RetryConfig, fn func(ctx context.Context) error) error {
	var lastErr error

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		// Check context before attempting.
		if ctx.Err() != nil {
			retryAttemptsTotal.WithLabelValues("canceled").Inc()
			return fmt.Errorf("retry aborted: %w", ctx.Err())
		}

		lastErr = fn(ctx)
		if lastErr == nil {
			if attempt > 0 {
				retryAttemptsTotal.WithLabelValues("success_after_retry").Inc()
			}
			return nil
		}

		// Check if error is retryable.
		if cfg.RetryableFunc != nil && !cfg.RetryableFunc(lastErr) {
			retryAttemptsTotal.WithLabelValues("not_retryable").Inc()
			return lastErr
		}

		// Don't sleep after the last attempt.
		if attempt < cfg.MaxAttempts-1 {
			delay := calculateDelay(attempt, cfg.BaseDelay, cfg.MaxDelay)

			select {
			case <-ctx.Done():
				retryAttemptsTotal.WithLabelValues("canceled").Inc()
				return fmt.Errorf("retry aborted: %w", ctx.Err())
			case <-time.After(delay):
			}
		}
	}

	retryAttemptsTotal.WithLabelValues("exhausted").Inc()
	return lastErr
}

// calculateDelay computes exponential backoff with full jitter.
func calculateDelay(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	// Exponential backoff: baseDelay * 2^attempt
	backoff := float64(baseDelay) * math.Pow(2, float64(attempt))
	if backoff > float64(maxDelay) {
		backoff = float64(maxDelay)
	}

	// Full jitter: random value between 0 and backoff.
	jittered := time.Duration(rand.Float64() * backoff) //nolint:gosec // G404: jitter needs no cryptographic randomness
	if jittered < time.Millisecond {
		jittered = time.Millisecond
	}
	return jittered
}

// RetryWithResult executes fn with retry and returns the result.
func RetryWithResult[T any](ctx context.Context, cfg RetryConfig, fn func(ctx context.Context) (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if ctx.Err() != nil {
			retryAttemptsTotal.WithLabelValues("canceled").Inc()
			return result, fmt.Errorf("retry aborted: %w", ctx.Err())
		}

		result, lastErr = fn(ctx)
		if lastErr == nil {
			if attempt > 0 {
				retryAttemptsTotal.WithLabelValues("success_after_retry").Inc()
			}
			return result, nil
		}

		if cfg.RetryableFunc != nil && !cfg.RetryableFunc(lastErr) {
			retryAttemptsTotal.WithLabelValues("not_retryable").Inc()
			return result, lastErr
		}

		if attempt < cfg.MaxAttempts-1 {
			delay := calculateDelay(attempt, cfg.BaseDelay, cfg.MaxDelay)
			select {
			case <-ctx.Done():
				retryAttemptsTotal.WithLabelValues("canceled").Inc()
				return result, fmt.Errorf("retry aborted: %w", ctx.Err())
			case <-time.After(delay):
			}
		}
	}

	retryAttemptsTotal.WithLabelValues("exhausted").Inc()
	return result, lastErr
}
