# On-Call Guide

> Everything an on-call engineer needs to handle incidents from page to resolution.

## On-Call Responsibilities

As the on-call engineer, you are the **first responder** for all production alerts and incidents. Your responsibilities:

1. **Acknowledge** alerts within the response time defined in the [Severity Matrix](severity-matrix.md)
2. **Triage** the issue and determine severity
3. **Mitigate** the impact (restore service, even if root cause isn't fixed)
4. **Escalate** when needed (don't hero-hero)
5. **Communicate** status to stakeholders
6. **Document** the incident timeline and resolution
7. **Hand off** cleanly to the next on-call engineer

## Rotation

| Detail | Value |
|--------|-------|
| Rotation length | 1 week (Monday 9am → Monday 9am) |
| Team size | Minimum 4 engineers |
| Coverage | 24/7 for SEV-1/2; business hours for SEV-3/4 |
| Shadow | First rotation paired with experienced on-call |
| Compensation | Time-off-in-lieu for overnight pages |

## When You Get Paged

### Step 1: Acknowledge (within 5 min)

- Acknowledge the alert in Alertmanager/PagerDuty
- Join #incidents channel
- State: "I'm looking into [alert name]"

### Step 2: Assess Severity

Use the [Severity Matrix](severity-matrix.md) classification guide:
- Check the alert details and linked runbook
- Check Grafana dashboards: [Platform Overview](http://localhost:3001)
- Check recent deployments: `kubectl rollout history`
- Determine: SEV-1, SEV-2, SEV-3, or SEV-4

### Step 3: Follow the Runbook

Every alert links to a runbook. Follow it:

| Alert | Runbook |
|-------|---------|
| APIUnavailable | [api-unavailable.md](../runbooks/api-unavailable.md) |
| IngestionFailing | [ingestion-failure.md](../runbooks/ingestion-failure.md) |
| AllProvidersUnavailable | [all-providers-unavailable.md](../runbooks/all-providers-unavailable.md) |
| DataStaleCritical | [data-freshness-alert.md](../runbooks/data-freshness-alert.md) |
| PostgreSQLUnavailable | [postgresql-unavailable.md](../runbooks/postgresql-unavailable.md) |
| RedisUnavailable | [redis-unavailable.md](../runbooks/redis-unavailable.md) |
| APIHighLatency | [high-api-latency.md](../runbooks/high-api-latency.md) |
| ErrorBudgetBurn | [error-budget-burn.md](../runbooks/error-budget-burn.md) |

### Step 4: Mitigate

Priority order:
1. **Rollback** if caused by recent deployment: `kubectl rollout undo deployment/<name>`
2. **Scale up** if resource-constrained: `kubectl scale deployment/<name> --replicas=N`
3. **Restart** if stuck: `kubectl rollout restart deployment/<name>`
4. **Failover** if dependency down: activate fallback provider, switch to cache-only mode

### Step 5: Escalate (if needed)

Escalate when:
- You can't mitigate within the severity time limit
- The issue is outside your domain expertise
- Multiple services are affected simultaneously
- You need a decision that impacts users (e.g., degrade gracefully vs. take offline)

**How to escalate**: Page Level 2 via on-call tool. State: severity, what you've tried, what you need.

### Step 6: Communicate

For SEV-1/SEV-2, post updates in #incidents every 15-30 minutes:

```
[HH:MM] Status: Investigating | Identified | Mitigating | Resolved
Impact: <what users experience>
Action: <what you're doing now>
ETA: <estimated resolution time or "unknown">
```

### Step 7: Resolve and Document

1. Verify the fix (check dashboards, run smoke tests)
2. Post resolution in #incidents
3. Create incident ticket with timeline
4. Schedule postmortem if SEV-1/SEV-2

## Quick Reference Commands

```bash
# Check service health
curl -s http://localhost:8080/health | jq .
curl -s http://localhost:8080/ready | jq .

# Check pod status
kubectl get pods -n cryptomarket-prod
kubectl describe pod <pod-name> -n cryptomarket-prod

# View logs
kubectl logs -l app.kubernetes.io/name=market-api -n cryptomarket-prod --tail=100 -f

# Rollback deployment
kubectl rollout undo deployment/market-api -n cryptomarket-prod

# Scale up
kubectl scale deployment/market-api --replicas=5 -n cryptomarket-prod

# Restart
kubectl rollout restart deployment/market-api -n cryptomarket-prod

# Check Prometheus alerts
curl -s 'http://localhost:9090/api/v1/alerts' | jq '.data.alerts[] | select(.state=="firing")'

# Reset injected failures (if chaos experiment gone wrong)
make incident-reset
```

## Dashboards

| Dashboard | URL | Use For |
|-----------|-----|---------|
| Platform Overview | Grafana → platform-overview | Overall health |
| SLO & Error Budget | Grafana → slo-error-budget | Budget consumption |
| Provider Reliability | Grafana → provider-reliability | Provider issues |
| Performance | Grafana → performance-engineering | Latency, throughput |
| Security Posture | Grafana → security-posture | Security metrics |

## Handoff Checklist

At rotation change (Monday 9am):

- [ ] Review any active or recent incidents
- [ ] Check for suppressed/silenced alerts
- [ ] Note any ongoing issues or watch items
- [ ] Verify alerting is functional (test alert)
- [ ] Brief incoming on-call verbally (15 min)
- [ ] Update #incidents with handoff summary

## Self-Care

- **You are not expected to know everything** — escalate freely
- **Sleep matters** — if paged overnight, take time off the next day
- **No blame** — incidents are system failures, not personal failures
- **Ask for help** — a second pair of eyes speeds resolution
- **Take breaks** — for long incidents, rotate the IC role

## Related Documents

- [Severity Matrix](severity-matrix.md)
- [SLOs](../sre/slos.md)
- [Alerting Strategy](../sre/alerting-strategy.md)
- [DR Strategy](../dr/strategy.md)
- [Postmortem Template](../postmortems/)
