# Resilience Testing Guide

## Overview

This document describes the testing strategy for platform resilience, covering unit tests, integration tests, failure injection, and load testing.

## Test Layers

### 1. Unit Tests (Go)

```bash
# All resilience tests
make test-resilience

# Provider fallback tests
make test-provider-fallback

# All tests
CGO_ENABLED=0 go test -short -count=1 ./...
```

**Coverage areas:**
- Circuit breaker state transitions (open, half-open, close)
- Retry backoff calculation and error classification
- Rate limit tracking and Retry-After parsing
- Provider selector ordering and disablement
- Fallback orchestrator flow
- Data validation and freshness calculation
- Anomaly detection signals

### 2. Python SRE Toolkit Tests

```bash
make test-python
```

**Coverage areas:**
- Price reconciliation comparison logic
- Failure injection scenario configuration
- Environment guard behavior
- Cleanup procedures

### 3. Failure Injection (Manual/Scripted)

```bash
# Run incident demo
make incident-demo

# Manual injection
ALLOW_FAILURE_INJECTION=true python3 sre-toolkit/inject_failures.py --scenario provider_429

# Cleanup
ALLOW_FAILURE_INJECTION=true python3 sre-toolkit/inject_failures.py --scenario provider_429 --cleanup
```

**Scenarios:**
| Scenario | What it tests |
|----------|--------------|
| provider_429 | Rate limit handling, fallback activation |
| provider_500 | Error classification, circuit breaker |
| provider_timeout | Timeout handling, retry behavior |
| provider_malformed | Data validation, error handling |
| redis_failure | Cache fallback, stream recovery |
| stale_data | Freshness detection, staleness alerts |

### 4. Load Testing (k6)

```bash
make load-test-resilience
```

**Scenarios:**
- Normal load (10 VUs, 30s)
- Burst traffic (50 VUs spike)
- Provider failure during load
- Recovery verification

### 5. Incident Simulation

```bash
make incident-demo
```

Full lifecycle: healthy → inject failure → observe degradation → recovery → verify.

## Test Environment

| Component | URL |
|-----------|-----|
| API | http://localhost:8080 |
| Mock Provider | http://localhost:8082 |
| Prometheus | http://localhost:9090 |
| Alertmanager | http://localhost:9093 |
| Grafana | http://localhost:3001 |

## Safety

- Failure injection requires `ALLOW_FAILURE_INJECTION=true`
- Production mode (`APP_ENV=production`) disables injection
- All scenarios are reversible
- Use `make incident-reset` to clean up all injected failures
