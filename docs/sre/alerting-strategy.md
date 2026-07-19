# Alerting Strategy

## Philosophy

Alerts should be **actionable** — every alert must have a clear owner, runbook, and resolution path. We alert on symptoms (SLO burn) rather than causes (individual metrics) where possible.

## Alert Severity Levels

| Severity | Response Time | Channel | Example |
|----------|--------------|---------|---------|
| Critical | < 5 minutes | Page on-call | API unavailable, all providers down |
| Warning | < 1 hour | Slack/ticket | Primary provider down, elevated latency |
| Info | Next business day | Dashboard | Error budget consumption |

## Alert Categories

### 1. Availability Alerts (Critical)
- **APIUnavailable**: >10% 5xx for 2 minutes
- **IngestionFailing**: No successful ingestion for 5 minutes
- **AllProvidersUnavailable**: All circuit breakers open
- **PostgreSQLUnavailable**: Database connection failed
- **RedisUnavailable**: Cache/stream backend failed

### 2. Degradation Alerts (Warning)
- **PrimaryProviderDown**: Primary circuit breaker open (fallback active)
- **ProviderHighLatency**: p95 provider latency > 5s
- **HighRateLimitFrequency**: Frequent 429 responses
- **APIHighLatency**: p99 API latency > 500ms
- **RealtimeConsumerLag**: Stream lag > 1000 messages
- **SustainedStaleSymbols**: Stale data for > 10 minutes

### 3. Error Budget Alerts (Multi-Window Burn Rate)
- **Fast burn** (critical): 14.4x burn rate over 1h + 5m windows
- **Slow burn** (warning): 6x burn rate over 6h + 30m windows

## Alert Routing

```
Alert → Alertmanager → Route by severity
  ├── Critical → critical-webhook (immediate)
  ├── Warning → default-webhook (grouped, 30s wait)
  └── Burn Rate → critical-webhook (2m wait)
```

## Inhibition Rules

- AllProvidersUnavailable suppresses PrimaryProviderDown
- APIUnavailable suppresses APIHighLatency

## Runbook Requirements

Every alert annotation includes a `runbook` field pointing to `docs/runbooks/`. Each runbook must contain:
1. Alert description and trigger conditions
2. Impact assessment
3. Diagnostic steps
4. Resolution procedures
5. Escalation path

## Testing Alerts

```bash
# Send test alert
make alert-test

# Check active alerts
curl http://localhost:9090/api/v1/alerts

# View Alertmanager UI
open http://localhost:9093
```
