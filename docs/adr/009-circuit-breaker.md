# ADR-009: Circuit Breaker Pattern

## Status

Accepted

## Context

When a provider is failing, continued requests waste resources, increase latency, and delay fallback activation. A mechanism is needed to stop calling failing providers and allow recovery probing.

## Decision

Implement a per-provider circuit breaker with three states:

- **Closed** (normal): Requests flow normally. Failures are counted.
- **Open** (tripped): Requests are blocked immediately. After `OpenDuration` (default 30s), transitions to Half-Open.
- **Half-Open** (probing): Limited requests allowed. `SuccessThreshold` (default 2) consecutive successes close the circuit. Any failure re-opens it.

Configuration:
- `CIRCUIT_BREAKER_FAILURE_THRESHOLD`: 5 (failures to open)
- `CIRCUIT_BREAKER_OPEN_DURATION`: 30s
- `CIRCUIT_BREAKER_SUCCESS_THRESHOLD`: 2 (successes to close from half-open)

Implementation: `internal/resilience/circuitbreaker.go` with a `Manager` for per-provider instances.

## Consequences

- Failing providers are quickly isolated (< 5 failures)
- Recovery is automatic via half-open probing
- Independent per-provider state prevents one failure from affecting others
- Metrics: `circuit_breaker_state`, `circuit_breaker_transitions_total`
- Thread-safe implementation supports concurrent access
