# Alert Tuning Review

> Last reviewed: 2026-07-21
> Next review: 2026-10-21 (quarterly)

## Tuning Principles

1. **Actionable**: Every alert must require human action. No "FYI" alerts.
2. **Specific**: Alert name + description must identify the exact problem.
3. **Linked**: Every alert links to a runbook with resolution steps.
4. **Calibrated**: Thresholds based on SLO targets, not arbitrary values.
5. **Quiet**: < 5% false positive rate. Tune or remove noisy alerts.

## Current Alert Inventory

| Alert | Severity | Threshold | False Positive Rate | Action |
|-------|----------|-----------|--------------------|---------| 
| APIUnavailable | Critical | >10% 5xx for 2m | ~2% | Keep — well calibrated |
| IngestionFailing | Critical | 0 success for 5m | ~1% | Keep — critical path |
| AllProvidersUnavailable | Critical | All CBs open 1m | 0% | Keep — rare but critical |
| DataStaleCritical | Critical | >600s stale for 2m | ~3% | Keep — user-facing |
| PostgreSQLUnavailable | Critical | pg_up==0 for 1m | 0% | Keep — infrastructure |
| RedisUnavailable | Critical | redis_up==0 for 1m | 0% | Keep — infrastructure |
| PrimaryProviderDown | Warning | CB open 2m | ~8% | **Tune**: increase to 5m |
| ProviderHighLatency | Warning | p95 >5s for 5m | ~5% | Keep — at threshold |
| HighRateLimitFrequency | Warning | >0.1/s for 5m | ~4% | Keep — indicates pressure |
| APIHighLatency | Warning | p99 >500ms for 5m | ~6% | **Tune**: increase to 10m |
| RealtimeConsumerLag | Warning | >1000 for 5m | ~3% | Keep — early warning |
| SustainedStaleSymbols | Warning | Any stale 10m | ~7% | **Tune**: increase to 15m |
| ProviderDataMismatch | Warning | Divergence 5m | ~10% | **Tune**: increase to 10m |
| APIAvailabilityFastBurn | Critical | 14.4x burn 2m | ~1% | Keep — SLO protection |
| APIAvailabilitySlowBurn | Warning | 6x burn 15m | ~4% | Keep — SLO protection |
| IngestionErrorBudgetBurn | Warning | 6x burn 10m | ~3% | Keep — SLO protection |

## Tuning Changes Applied

### 1. PrimaryProviderDown: 2m → 5m
**Reason**: Circuit breaker opens briefly during normal provider hiccups. 2m causes false pages when the CB recovers within 3-4 minutes (normal behavior).

### 2. APIHighLatency: 5m → 10m
**Reason**: Brief latency spikes during ingestion cycles (Redis MGET contention) resolve within 5-8 minutes. 10m window filters transient spikes.

### 3. SustainedStaleSymbols: 10m → 15m
**Reason**: Individual symbols can be stale during provider rate limiting. 15m ensures we only alert on sustained issues, not transient per-symbol delays.

### 4. ProviderDataMismatch: 5m → 10m
**Reason**: Price divergence between providers is normal during high-volatility periods. 10m window reduces noise during market events.

## Missing Alerts (To Add)

| Alert | Severity | Condition | Rationale |
|-------|----------|-----------|-----------|
| PodRestartLoop | Warning | restarts > 3 in 10m | Detect crash loops early |
| HPAMaxedOut | Warning | current == max for 15m | Capacity planning signal |
| CertificateExpiring | Warning | cert expires < 14d | Prevent TLS outage |
| DiskPressure | Warning | PVC > 85% full | Prevent storage exhaustion |
| StreamGrowth | Warning | stream length > 100k | Detect consumer failure |

## Alert Routing

| Severity | Route | Receiver | Repeat Interval |
|----------|-------|----------|-----------------|
| Critical | Page on-call | PagerDuty | 5m |
| Warning | Slack #incidents | Slack | 4h |
| Info | Slack #monitoring | Slack | 24h |

## Suppression Rules

| Suppress | When | Duration |
|----------|------|----------|
| All warnings | During SEV-1 incident | Until resolved |
| Provider alerts | During scheduled maintenance | Maintenance window |
| Latency alerts | During deployment canary | 10 min post-deploy |

## Review Checklist

- [ ] All alerts have runbook links
- [ ] No alert fires more than once per week without action
- [ ] All critical alerts page (not just Slack)
- [ ] Thresholds align with SLO targets
- [ ] No duplicate/overlapping alerts
- [ ] Suppression rules tested
