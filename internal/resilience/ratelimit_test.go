package resilience

import (
	"net/http"
	"testing"
	"time"
)

func TestRateLimitTracker_RecordAndCount(t *testing.T) {
	tracker := NewRateLimitTracker()

	tracker.RecordRateLimit("coingecko", 30*time.Second)
	tracker.RecordRateLimit("coingecko", 30*time.Second)
	tracker.RecordRateLimit("coincap", 60*time.Second)

	if tracker.Count("coingecko") != 2 {
		t.Errorf("expected 2 rate limits for coingecko, got %d", tracker.Count("coingecko"))
	}
	if tracker.Count("coincap") != 1 {
		t.Errorf("expected 1 rate limit for coincap, got %d", tracker.Count("coincap"))
	}
	if tracker.Count("unknown") != 0 {
		t.Errorf("expected 0 rate limits for unknown, got %d", tracker.Count("unknown"))
	}
}

func TestRateLimitTracker_ShouldWait(t *testing.T) {
	tracker := NewRateLimitTracker()

	// No rate limit recorded.
	wait, _ := tracker.ShouldWait("coingecko")
	if wait {
		t.Error("expected no wait for provider without rate limit")
	}

	// Record rate limit with 100ms retry-after.
	tracker.RecordRateLimit("coingecko", 100*time.Millisecond)

	wait, remaining := tracker.ShouldWait("coingecko")
	if !wait {
		t.Error("expected wait after rate limit")
	}
	if remaining <= 0 || remaining > 100*time.Millisecond {
		t.Errorf("unexpected remaining time: %v", remaining)
	}

	// Wait for retry-after to elapse.
	time.Sleep(110 * time.Millisecond)

	wait, _ = tracker.ShouldWait("coingecko")
	if wait {
		t.Error("expected no wait after retry-after elapsed")
	}
}

func TestRateLimitTracker_Reset(t *testing.T) {
	tracker := NewRateLimitTracker()

	tracker.RecordRateLimit("coingecko", time.Hour)
	if tracker.Count("coingecko") != 1 {
		t.Error("expected 1 rate limit before reset")
	}

	tracker.Reset("coingecko")
	if tracker.Count("coingecko") != 0 {
		t.Error("expected 0 rate limits after reset")
	}
}

func TestParseRetryAfter_Numeric(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}
	resp.Header.Set("Retry-After", "120")

	delay := ParseRetryAfter(resp)
	if delay != 120*time.Second {
		t.Errorf("expected 120s, got %v", delay)
	}
}

func TestParseRetryAfter_NumericZero(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}
	resp.Header.Set("Retry-After", "0")

	delay := ParseRetryAfter(resp)
	if delay != 0 {
		t.Errorf("expected 0 for zero value, got %v", delay)
	}
}

func TestParseRetryAfter_HTTPDate(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}
	// Set a date 60 seconds in the future.
	future := time.Now().Add(60 * time.Second).UTC()
	resp.Header.Set("Retry-After", future.Format(time.RFC1123))

	delay := ParseRetryAfter(resp)
	// Allow some tolerance for test execution time.
	if delay < 55*time.Second || delay > 65*time.Second {
		t.Errorf("expected ~60s, got %v", delay)
	}
}

func TestParseRetryAfter_PastDate(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}
	// Set a date in the past.
	past := time.Now().Add(-60 * time.Second).UTC()
	resp.Header.Set("Retry-After", past.Format(time.RFC1123))

	delay := ParseRetryAfter(resp)
	if delay != 0 {
		t.Errorf("expected 0 for past date, got %v", delay)
	}
}

func TestParseRetryAfter_Missing(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}

	delay := ParseRetryAfter(resp)
	if delay != 0 {
		t.Errorf("expected 0 for missing header, got %v", delay)
	}
}

func TestParseRetryAfter_NilResponse(t *testing.T) {
	delay := ParseRetryAfter(nil)
	if delay != 0 {
		t.Errorf("expected 0 for nil response, got %v", delay)
	}
}

func TestParseRetryAfter_InvalidFormat(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}
	resp.Header.Set("Retry-After", "not-a-valid-value")

	delay := ParseRetryAfter(resp)
	if delay != 0 {
		t.Errorf("expected 0 for invalid format, got %v", delay)
	}
}

func TestDefaultRetryAfter(t *testing.T) {
	delay := DefaultRetryAfter()
	if delay != 60*time.Second {
		t.Errorf("expected 60s default, got %v", delay)
	}
}
