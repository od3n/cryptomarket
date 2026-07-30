package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func testConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 3,
		OpenDuration:     100 * time.Millisecond,
		SuccessThreshold: 2,
		Name:             name,
	}
}

func TestCircuitBreaker_StartsClosed(t *testing.T) {
	cb := NewCircuitBreaker(testConfig("test"))
	if cb.State() != StateClosed {
		t.Errorf("expected closed state, got %s", cb.State())
	}
}

func TestCircuitBreaker_OpensOnThreshold(t *testing.T) {
	cb := NewCircuitBreaker(testConfig("test"))
	ctx := context.Background()
	errFail := errors.New("fail")

	// Record failures up to threshold.
	for i := 0; i < 3; i++ {
		_ = cb.Execute(ctx, func(_ context.Context) error {
			return errFail
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("expected open state after %d failures, got %s", 3, cb.State())
	}
}

func TestCircuitBreaker_BlocksWhileOpen(t *testing.T) {
	cb := NewCircuitBreaker(testConfig("test"))
	ctx := context.Background()
	errFail := errors.New("fail")

	// Open the circuit.
	for i := 0; i < 3; i++ {
		_ = cb.Execute(ctx, func(_ context.Context) error {
			return errFail
		})
	}

	// Requests should be rejected.
	err := cb.Execute(ctx, func(_ context.Context) error {
		return nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	cfg := testConfig("test")
	cfg.OpenDuration = 50 * time.Millisecond
	cb := NewCircuitBreaker(cfg)
	ctx := context.Background()
	errFail := errors.New("fail")

	// Open the circuit.
	for i := 0; i < 3; i++ {
		_ = cb.Execute(ctx, func(_ context.Context) error {
			return errFail
		})
	}

	if cb.State() != StateOpen {
		t.Fatalf("expected open state, got %s", cb.State())
	}

	// Wait for open duration to elapse.
	time.Sleep(60 * time.Millisecond)

	// Should transition to half-open.
	if cb.State() != StateHalfOpen {
		t.Errorf("expected half-open state after timeout, got %s", cb.State())
	}
}

func TestCircuitBreaker_ClosesAfterSuccessThreshold(t *testing.T) {
	cfg := testConfig("test")
	cfg.OpenDuration = 50 * time.Millisecond
	cb := NewCircuitBreaker(cfg)
	ctx := context.Background()
	errFail := errors.New("fail")

	// Open the circuit.
	for i := 0; i < 3; i++ {
		_ = cb.Execute(ctx, func(_ context.Context) error {
			return errFail
		})
	}

	// Wait for half-open.
	time.Sleep(60 * time.Millisecond)

	// Record successes in half-open.
	for i := 0; i < 2; i++ {
		err := cb.Execute(ctx, func(_ context.Context) error {
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error in half-open: %v", err)
		}
	}

	if cb.State() != StateClosed {
		t.Errorf("expected closed state after success threshold, got %s", cb.State())
	}
}

func TestCircuitBreaker_ReopensOnHalfOpenFailure(t *testing.T) {
	cfg := testConfig("test")
	cfg.OpenDuration = 50 * time.Millisecond
	cb := NewCircuitBreaker(cfg)
	ctx := context.Background()
	errFail := errors.New("fail")

	// Open the circuit.
	for i := 0; i < 3; i++ {
		_ = cb.Execute(ctx, func(_ context.Context) error {
			return errFail
		})
	}

	// Wait for half-open.
	time.Sleep(60 * time.Millisecond)

	if cb.State() != StateHalfOpen {
		t.Fatalf("expected half-open, got %s", cb.State())
	}

	// Fail in half-open.
	_ = cb.Execute(ctx, func(_ context.Context) error {
		return errFail
	})

	if cb.State() != StateOpen {
		t.Errorf("expected open state after half-open failure, got %s", cb.State())
	}
}

func TestCircuitBreaker_SuccessResetsFailures(t *testing.T) {
	cb := NewCircuitBreaker(testConfig("test"))
	ctx := context.Background()
	errFail := errors.New("fail")

	// Record 2 failures (below threshold of 3).
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, func(_ context.Context) error {
			return errFail
		})
	}

	// Success resets counter.
	_ = cb.Execute(ctx, func(_ context.Context) error {
		return nil
	})

	// 2 more failures should not open (counter was reset).
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, func(_ context.Context) error {
			return errFail
		})
	}

	if cb.State() != StateClosed {
		t.Errorf("expected closed state (failures reset), got %s", cb.State())
	}
}

func TestCircuitBreaker_IndependentPerProvider(t *testing.T) {
	manager := NewManager(CircuitBreakerConfig{
		FailureThreshold: 2,
		OpenDuration:     100 * time.Millisecond,
		SuccessThreshold: 1,
	})

	cb1 := manager.Get("provider-a")
	cb2 := manager.Get("provider-b")
	ctx := context.Background()
	errFail := errors.New("fail")

	// Open provider-a's circuit.
	for i := 0; i < 2; i++ {
		_ = cb1.Execute(ctx, func(_ context.Context) error {
			return errFail
		})
	}

	if cb1.State() != StateOpen {
		t.Errorf("expected provider-a open, got %s", cb1.State())
	}
	if cb2.State() != StateClosed {
		t.Errorf("expected provider-b closed (independent), got %s", cb2.State())
	}
}

func TestCircuitBreaker_ManagerStates(t *testing.T) {
	manager := NewManager(testConfig("default"))

	manager.Get("a")
	manager.Get("b")

	states := manager.States()
	if len(states) != 2 {
		t.Errorf("expected 2 states, got %d", len(states))
	}
	if states["a"] != "closed" || states["b"] != "closed" {
		t.Errorf("unexpected states: %v", states)
	}
}

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state CircuitState
		want  string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{CircuitState(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("State(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}
