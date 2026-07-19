# ADR-008: Retry Ownership

## Status

Accepted

## Context

Retries can be implemented at multiple layers: HTTP client, provider adapter, orchestrator, or worker. Multiple retry layers compound delay and can cause thundering herd problems.

## Decision

**The provider adapter layer owns retries.** Specifically:

- The fallback orchestrator calls `resilience.RetryWithResult()` around each provider's `FetchMarketData()`
- Retries use exponential backoff with full jitter
- Only transient errors are retried (network timeouts, 5xx, connection refused)
- Permanent errors (4xx except 429, validation failures) are NOT retried
- Context cancellation immediately stops retries
- Maximum 3 attempts by default (configurable via `RETRY_MAX_ATTEMPTS`)

The worker/scheduler layer does NOT retry — it relies on the next scheduled cycle.

## Consequences

- Single retry boundary prevents compounding delays
- Clear ownership: adapter handles transient failures, scheduler handles cycle-level recovery
- Retry metrics (`retry_attempts_total`) provide observability
- Configuration via environment variables keeps deployment flexible
