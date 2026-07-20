# ADR-016: Performance Strategy

## Status

Accepted

## Context

The platform serves real-time market data to potentially thousands of concurrent users. Performance must be characterized, monitored, and protected against regression as the system evolves.

## Decision

### Profiling

- pprof endpoints exposed at `/debug/pprof/*` on all Go services
- Profiles collected: CPU, heap, goroutines, mutex, blocking, allocations
- Production profiling gated by network policy (internal access only)
- Baseline profiles captured during load tests for comparison

### Optimization Techniques

1. **HTTP Compression**: gzip middleware with sync.Pool for writer reuse; skipped for SSE streams
2. **Redis Pipelining**: Batch writes via `SetLatestBatch()` to reduce round-trips during ingestion bursts
3. **Connection Pooling**: PostgreSQL pool tuned (25 max open, 5 idle, 5min lifetime); Redis connection reuse via go-redis pool
4. **Query Optimization**: Indexes on `(coin_id, captured_at DESC)` for history queries; LIMIT-based pagination (no OFFSET)
5. **Caching**: Redis as L1 cache (5min TTL); MGET for batch reads; cache-aside pattern with DB fallback
6. **Memory**: `-trimpath` and `-ldflags="-s -w"` for smaller binaries; sync.Pool for gzip writers

### Benchmarking

- k6 benchmark suite at 1 / 100 / 1000 / 5000 concurrent users
- Metrics: p50, p95, p99 latency; throughput (RPS); error rate
- Results documented in `docs/performance/benchmarks.md`
- Regression threshold: p95 must not increase >20% between releases

### Monitoring

- Performance Grafana dashboard with latency, CPU, memory, GC, goroutines
- Prometheus recording rules for SLO-relevant percentiles
- Alert on p99 > 500ms for 5 minutes

## Consequences

- pprof adds minimal overhead (~1-2% CPU when idle) but must be network-restricted
- gzip compression adds ~0.5ms per response but saves 60-80% bandwidth on JSON payloads
- Redis pipelining trades individual error visibility for throughput — batch errors reported together
