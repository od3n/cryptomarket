# ADR-014: Failure Injection Safeguards

## Status

Accepted

## Context

Failure injection is essential for resilience testing but dangerous if accidentally enabled in production. Safeguards are needed to prevent unintended disruption.

## Decision

Implement multiple safeguards for failure injection:

1. **Environment guard**: `ALLOW_FAILURE_INJECTION=true` must be explicitly set
2. **Production mode**: When `APP_ENV=production`, injection is always disabled regardless of env var
3. **Explicit scenario**: Must specify exact scenario name (no wildcard injection)
4. **Reversible by default**: All scenarios have a defined cleanup path
5. **Audit trail**: All injection events logged with timestamp and scenario
6. **Auto-cleanup**: Optional `--duration` flag for automatic reversion

Implementation: `sre-toolkit/inject_failures.py` with guard check before any action.

## Consequences

- Cannot accidentally inject failures in production
- Clear audit trail for compliance and postmortems
- All scenarios are reversible (no permanent state changes)
- CI/CD pipeline does not set `ALLOW_FAILURE_INJECTION`
- Docker Compose development environment can enable it explicitly
