package resilience

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	// StateClosed means the circuit is closed and requests flow normally.
	StateClosed CircuitState = iota
	// StateOpen means the circuit is open and requests are blocked.
	StateOpen
	// StateHalfOpen means the circuit is testing if the service recovered.
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when a request is rejected because the circuit is open.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitBreakerConfig holds configuration for a circuit breaker.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures before opening.
	FailureThreshold int
	// OpenDuration is how long the circuit stays open before transitioning to half-open.
	OpenDuration time.Duration
	// SuccessThreshold is the number of successes in half-open state before closing.
	SuccessThreshold int
	// Name identifies this circuit breaker (typically the provider name).
	Name string
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		OpenDuration:     30 * time.Second,
		SuccessThreshold: 2,
		Name:             name,
	}
}

var (
	cbStateGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "circuit_breaker_state",
		Help: "Current state of the circuit breaker (0=closed, 1=open, 2=half-open).",
	}, []string{"name"})

	cbTransitionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "circuit_breaker_transitions_total",
		Help: "Total number of circuit breaker state transitions.",
	}, []string{"name", "from", "to"})

	cbRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "circuit_breaker_requests_total",
		Help: "Total number of requests through the circuit breaker.",
	}, []string{"name", "result"})
)

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	mu     sync.Mutex
	config CircuitBreakerConfig

	state       CircuitState
	failures    int
	successes   int
	lastFailure time.Time
	openedAt    time.Time

	// nowFunc allows overriding time for testing.
	nowFunc func() time.Time
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration.
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	cb := &CircuitBreaker{
		config:  config,
		state:   StateClosed,
		nowFunc: time.Now,
	}
	cbStateGauge.WithLabelValues(config.Name).Set(0)
	return cb
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.currentState()
}

// currentState returns the state, transitioning from open to half-open if timeout elapsed.
// Must be called with lock held.
func (cb *CircuitBreaker) currentState() CircuitState {
	if cb.state == StateOpen {
		if cb.nowFunc().Sub(cb.openedAt) >= cb.config.OpenDuration {
			cb.transitionTo(StateHalfOpen)
		}
	}
	return cb.state
}

// Execute runs the given function through the circuit breaker.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	cb.mu.Lock()
	state := cb.currentState()

	if state == StateOpen {
		cb.mu.Unlock()
		cbRequestsTotal.WithLabelValues(cb.config.Name, "rejected").Inc()
		return ErrCircuitOpen
	}
	cb.mu.Unlock()

	cbRequestsTotal.WithLabelValues(cb.config.Name, "attempted").Inc()

	err := fn(ctx)

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
		cbRequestsTotal.WithLabelValues(cb.config.Name, "failure").Inc()
		return err
	}

	cb.onSuccess()
	cbRequestsTotal.WithLabelValues(cb.config.Name, "success").Inc()
	return nil
}

// RecordSuccess manually records a successful operation.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.onSuccess()
}

// RecordFailure manually records a failed operation.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.onFailure()
}

func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateClosed:
		cb.failures = 0
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.transitionTo(StateClosed)
		}
	}
}

func (cb *CircuitBreaker) onFailure() {
	cb.lastFailure = cb.nowFunc()

	switch cb.state {
	case StateClosed:
		cb.failures++
		if cb.failures >= cb.config.FailureThreshold {
			cb.transitionTo(StateOpen)
		}
	case StateHalfOpen:
		// Any failure in half-open immediately reopens.
		cb.transitionTo(StateOpen)
	}
}

func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	if cb.state == newState {
		return
	}

	from := cb.state.String()
	to := newState.String()

	cb.state = newState
	cbStateGauge.WithLabelValues(cb.config.Name).Set(float64(newState))
	cbTransitionsTotal.WithLabelValues(cb.config.Name, from, to).Inc()

	switch newState {
	case StateClosed:
		cb.failures = 0
		cb.successes = 0
	case StateOpen:
		cb.openedAt = cb.nowFunc()
		cb.successes = 0
	case StateHalfOpen:
		cb.successes = 0
	}
}

// Failures returns the current failure count.
func (cb *CircuitBreaker) Failures() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failures
}

// Name returns the circuit breaker name.
func (cb *CircuitBreaker) Name() string {
	return cb.config.Name
}

// Manager manages multiple independent circuit breakers.
type Manager struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
	config   CircuitBreakerConfig
}

// NewManager creates a circuit breaker manager with default config.
func NewManager(defaultConfig CircuitBreakerConfig) *Manager {
	return &Manager{
		breakers: make(map[string]*CircuitBreaker),
		config:   defaultConfig,
	}
}

// Get returns the circuit breaker for the given name, creating one if needed.
func (m *Manager) Get(name string) *CircuitBreaker {
	m.mu.RLock()
	cb, ok := m.breakers[name]
	m.mu.RUnlock()
	if ok {
		return cb
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock.
	if cb, ok := m.breakers[name]; ok {
		return cb
	}

	cfg := m.config
	cfg.Name = name
	cb = NewCircuitBreaker(cfg)
	m.breakers[name] = cb
	return cb
}

// States returns the current state of all circuit breakers.
func (m *Manager) States() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	states := make(map[string]string, len(m.breakers))
	for name, cb := range m.breakers {
		states[name] = cb.State().String()
	}
	return states
}

// String implements fmt.Stringer for CircuitState.
func (cb *CircuitBreaker) String() string {
	return fmt.Sprintf("CircuitBreaker[%s: %s]", cb.config.Name, cb.State())
}
