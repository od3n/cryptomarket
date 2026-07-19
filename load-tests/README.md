# Load and Resilience Tests

k6-based load tests for the Crypto Market Data Platform.

## Prerequisites

- [k6](https://k6.io/docs/getting-started/installation/) installed
- Platform stack running (`make up`)

## Running Tests

```bash
# Run all scenarios
k6 run load-tests/resilience.js

# Run specific scenario
k6 run --scenario normal_load load-tests/resilience.js

# Custom API URL
k6 run --env API_URL=http://localhost:8080 load-tests/resilience.js

# With Makefile
make load-test-resilience
```

## Scenarios

| Scenario | Duration | VUs | Purpose |
|----------|----------|-----|---------|
| `normal_load` | 30s | 10 | Baseline performance under moderate traffic |
| `burst` | 20s | 0→50→10 | Spike handling and graceful degradation |
| `provider_failure` | 20s | 5 | Data availability during provider outage |
| `recovery` | 20s | 10 | Return to normal after failure |

## Metrics Captured

- **p50/p95/p99 latency** for API and markets endpoints
- **Error rate** across all scenarios
- **Recovery time** after provider failure
- **Fallback request count** during degraded mode

## Thresholds

| Metric | Threshold |
|--------|-----------|
| Error rate | < 10% |
| API p95 latency | < 500ms |
| Markets p95 latency | < 1000ms |
| Overall p99 latency | < 2000ms |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `API_URL` | `http://localhost:8080` | API base URL |
| `REALTIME_URL` | `http://localhost:8081` | Realtime gateway URL |

## Interpreting Results

- **errors rate > 0.1**: Platform is not handling load gracefully
- **api_latency p95 > 500ms**: API is slower than SLO target
- **fallback_requests > 0**: Fallback provider was used (expected during failure scenarios)
- **recovery_time**: Time for responses to return to normal after failure injection
