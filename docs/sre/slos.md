# Service Level Objectives (SLOs)

This document defines the Service Level Indicators (SLIs), Service Level Objectives (SLOs), and error budgets for the Crypto Market Data Platform.

## Overview

SLOs define the reliability targets for our platform. They are based on measurable SLIs and define acceptable error budgets.

## SLI Definitions

### 1. API Availability

**Formula:**
```
availability = successful_responses / total_responses
```

**Measurement:**
- Successful: HTTP 2xx, 3xx responses
- Excluded: HTTP 4xx (client errors) - these are not platform failures
- Failed: HTTP 5xx responses

**Metric:**
```promql
sum(rate(http_responses_total{status!~"5.."}[5m])) / sum(rate(http_responses_total[5m]))
```

### 2. API Latency

**Formula:**
```
latency_compliance = requests_below_threshold / total_requests
```

**Measurement:**
- Threshold: 300ms for 99th percentile
- Route classes: `/coins`, `/markets`, `/health`, `/metrics`
- Excluded: Long-polling SSE connections

**Metric:**
```promql
histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))
```

### 3. Ingestion Success

**Formula:**
```
ingestion_success = successful_cycles / attempted_cycles
```

**Measurement:**
- Successful: Ingestion cycle completes with valid data stored
- Failed: Ingestion cycle fails to store any data

**Metric:**
```promql
rate(ingestion_success_total[5m]) / (rate(ingestion_success_total[5m]) + rate(ingestion_failure_total[5m]))
```

### 4. Data Freshness

**Formula:**
```
freshness_compliance = fresh_symbols / total_tracked_symbols
```

**Measurement:**
- Fresh: Data age <= 120 seconds
- Delayed: Data age > 120s and <= 300s
- Stale: Data age > 300s

**Metric:**
```promql
data_freshness_status == 0  # 0 = fresh
```

### 5. Realtime Delivery

**Formula:**
```
delivery_success = events_broadcast / events_consumed
```

**Measurement:**
- Successful: Event broadcast to connected clients
- Failed: Events dropped due to slow clients or errors

**Metric:**
```promql
rate(realtime_events_broadcast_total[5m]) / rate(realtime_events_consumed_total[5m])
```

### 6. Provider Availability

**Formula:**
```
provider_availability = successful_requests / eligible_attempts
```

**Measurement:**
- Successful: HTTP 2xx from provider
- Excluded: Requests blocked by circuit breaker (not eligible)
- Failed: HTTP 5xx, timeouts, connection errors

**Metric:**
```promql
sum(rate(provider_request_duration_seconds_count{status="success"}[5m])) by (provider) /
sum(rate(provider_request_duration_seconds_count[5m])) by (provider)
```

## SLO Targets

| SLO | Target | Window | Error Budget |
|-----|--------|--------|--------------|
| API Availability | 99.9% | 30 days | 43.2 minutes |
| API Latency (p99 < 300ms) | 99% | 30 days | 7.2 hours |
| Ingestion Success | 99.5% | 30 days | 3.6 hours |
| Data Freshness | 99% | 30 days | 7.2 hours |
| Realtime Delivery | 99.5% | 30 days | 3.6 hours |

## Error Budgets

Error budget = (1 - SLO target) × measurement window

### API Availability (99.9% over 30 days)
- Allowed downtime: 0.1% × 30 days = 43.2 minutes per month
- Weekly budget: ~10.1 minutes
- Daily budget: ~1.44 minutes

### API Latency (99% over 30 days)
- Allowed slow requests: 1% of all requests
- For 1M requests/month: 10,000 slow requests allowed

### Ingestion Success (99.5% over 30 days)
- With 60s interval: 43,200 cycles/month
- Allowed failures: 216 cycles/month

## Error Budget Policy

### Budget Consumption Rates

| Burn Rate | Window | Action |
|-----------|--------|--------|
| 14.4x | 1 hour | Page on-call (critical) |
| 6x | 6 hours | Page on-call (high) |
| 1x | 30 days | Normal operations |

### Budget Exhaustion

When error budget is exhausted:
1. Freeze non-critical deployments
2. Focus engineering on reliability improvements
3. Post-incident review for major budget consumers
4. Re-evaluate SLO targets if consistently missed

## Exclusions

The following are excluded from SLO calculations:
- Scheduled maintenance windows (with 24h notice)
- Client-side errors (4xx responses)
- Failures in unsupported browsers/clients
- Rate-limited requests (429 responses)
- Requests during declared incidents

## Review Cadence

- SLO targets reviewed quarterly
- Error budget consumption reviewed weekly
- SLI formulas reviewed when metrics change

## Version History

| Date | Version | Changes |
|------|---------|---------|
| 2024-01-15 | 1.0 | Initial SLO definitions |
