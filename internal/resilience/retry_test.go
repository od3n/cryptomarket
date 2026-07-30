package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry_SuccessFirstAttempt(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}

	attempts := 0
	err := Retry(context.Background(), cfg, func(_ context.Context) error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetry_SuccessAfterRetries(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}

	attempts := 0
	err := Retry(context.Background(), cfg, func(_ context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("transient error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetry_ExhaustsAttempts(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}

	attempts := 0
	err := Retry(context.Background(), cfg, func(_ context.Context) error {
		attempts++
		return errors.New("persistent error")
	})

	if err == nil {
		t.Error("expected error after exhausting attempts")
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetry_RespectsContextCancellation(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 10,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := Retry(ctx, cfg, func(_ context.Context) error {
		attempts++
		return errors.New("error")
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if attempts >= 10 {
		t.Errorf("expected early termination, got %d attempts", attempts)
	}
}

func TestRetry_NonRetryableError(t *testing.T) {
	permanentErr := errors.New("permanent error")
	cfg := RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		RetryableFunc: func(err error) bool {
			return !errors.Is(err, permanentErr)
		},
	}

	attempts := 0
	err := Retry(context.Background(), cfg, func(_ context.Context) error {
		attempts++
		return permanentErr
	})

	if !errors.Is(err, permanentErr) {
		t.Errorf("expected permanent error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt for non-retryable error, got %d", attempts)
	}
}

func TestRetry_ExponentialBackoff(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 4,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    time.Second,
	}

	var delays []time.Duration
	var lastTime time.Time
	first := true

	_ = Retry(context.Background(), cfg, func(_ context.Context) error {
		now := time.Now()
		if !first {
			delays = append(delays, now.Sub(lastTime))
		}
		first = false
		lastTime = now
		return errors.New("error")
	})

	// We should have 3 delays (between 4 attempts).
	if len(delays) != 3 {
		t.Fatalf("expected 3 delays, got %d", len(delays))
	}

	// Delays should generally increase (with jitter, not strictly).
	// Just verify they're all positive and bounded.
	for i, d := range delays {
		if d < 0 {
			t.Errorf("delay %d is negative: %v", i, d)
		}
		if d > cfg.MaxDelay+50*time.Millisecond {
			t.Errorf("delay %d exceeds max: %v", i, d)
		}
	}
}

func TestRetryWithResult_Success(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}

	attempts := 0
	result, err := RetryWithResult(context.Background(), cfg, func(_ context.Context) (string, error) {
		attempts++
		if attempts < 2 {
			return "", errors.New("transient")
		}
		return "success", nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "success" {
		t.Errorf("expected 'success', got %q", result)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestRetryWithResult_Error(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 2,
		BaseDelay:   time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}

	result, err := RetryWithResult(context.Background(), cfg, func(_ context.Context) (int, error) {
		return 0, errors.New("always fails")
	})

	if err == nil {
		t.Error("expected error")
	}
	if result != 0 {
		t.Errorf("expected zero value, got %d", result)
	}
}

func TestCalculateDelay(t *testing.T) {
	baseDelay := 100 * time.Millisecond
	maxDelay := time.Second

	// Test multiple attempts to verify bounds.
	for attempt := 0; attempt < 10; attempt++ {
		delay := calculateDelay(attempt, baseDelay, maxDelay)
		if delay < time.Millisecond {
			t.Errorf("attempt %d: delay too small: %v", attempt, delay)
		}
		if delay > maxDelay {
			t.Errorf("attempt %d: delay exceeds max: %v > %v", attempt, delay, maxDelay)
		}
	}
}
