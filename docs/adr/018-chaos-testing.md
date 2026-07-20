# ADR-018: Chaos Testing Strategy

## Status

Accepted

## Context

Resilience mechanisms (circuit breakers, fallbacks, caching, graceful degradation) are only trustworthy if exercised under realistic failure conditions. Without periodic chaos experiments, drift and configuration errors can silently erode fault tolerance.

## Decision

### Safety Constraints

- Chaos experiments run ONLY in `local` and `staging` environments
- A hard safety gate (`CHAOS_ENV` must be explicitly set) prevents accidental production execution
- Each experiment has automatic cleanup via trap handlers
- Experiments are idempotent and reversible

### Experiment Catalog

| Experiment | Failure Injected | Expected Behavior |
|-----------|-----------------|-------------------|
| kill-api | Delete API pod | HPA replaces pod; brief 503s; recovery < 30s |
| kill-redis | Scale Redis to 0 | API serves from DB fallback; degraded status |
| kill-postgres | Scale PostgreSQL to 0 | API serves from Redis cache; stale data warning |
| slow-provider | 5s network delay on ingestor | Circuit breaker opens; fallback provider activates |
| network-latency | 200ms on API pod | Latency increase; no errors |
| packet-loss | 10% on API pod | Retries handle loss; minimal user impact |
| dns-failure | Block DNS on ingestor | Provider calls fail; circuit breaker; cached data served |
| disk-pressure | Fill /tmp | Service continues (read-only fs); alert fires |

### Execution Model

1. Record baseline metrics (health, latency)
2. Inject failure
3. Observe for configurable duration (default 30s)
4. Remove failure (automatic cleanup)
5. Verify recovery (health returns 200)
6. Report results

### Frequency

- Manual: on-demand via `scripts/chaos/run-experiment.sh`
- Automated (future): weekly in staging via GitHub Actions cron
- Post-deploy: after every production release (staging mirror)

### Tooling

- `tc netem` for network chaos (latency, loss)
- `iptables` for DNS/network blocking
- `kubectl scale` for pod/statefulset removal
- `dd` for disk pressure
- k6 for concurrent load during experiments

## Consequences

- Experiments require `NET_ADMIN` capability in staging pods (not in production)
- tc/iptables not available in distroless images — staging uses debug sidecar or ephemeral containers
- Results are observational, not pass/fail gated (future: integrate with SLO assertions)
