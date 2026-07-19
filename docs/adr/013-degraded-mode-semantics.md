# ADR-013: Degraded Mode Semantics

## Status

Accepted

## Context

The platform needs clear operational states to communicate health to operators, users, and monitoring systems. Binary healthy/unhealthy is insufficient for a system with fallback capabilities.

## Decision

Define four operational states:

| State | Meaning | `/ready` Response | Frontend Indicator |
|-------|---------|-------------------|-------------------|
| `healthy` | All systems normal, primary provider active | 200 `{"status":"ready"}` | None |
| `degraded` | Fallback provider active, data flowing | 200 `{"status":"ready","mode":"degraded"}` | Yellow banner |
| `stale` | Data is stale but service available | 200 `{"status":"ready"}` | Orange banner |
| `unavailable` | Cannot serve data (all providers down or deps failed) | 503 | Red banner |

State determination logic in `internal/api/status.go`:
1. Check dependencies (PostgreSQL, Redis) → unavailable if down
2. Check circuit breakers → unavailable if all open
3. Check provider degradation → degraded if on fallback
4. Check freshness → stale if overall state is stale

## Consequences

- Operators can distinguish between "working but degraded" and "fully down"
- `/ready` stays 200 during degraded mode (load balancer keeps routing)
- Frontend shows appropriate user-facing indicators
- `/operations/status` provides full detail for automation
