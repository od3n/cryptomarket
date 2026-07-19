# Runbook: Error Budget Burn

## Alert
`APIAvailabilityFastBurn` / `APIAvailabilitySlowBurn` — SLO error budget being consumed.

## Impact
Reliability target at risk. Feature freeze may be triggered per error budget policy.

## Diagnosis
1. Check burn rate: `slo:api_availability:burn_rate1h` in Prometheus
2. Check budget remaining: `slo:api_availability:error_budget_remaining`
3. Identify error source: `sum(rate(http_responses_total{status=~"5.."}[5m])) by (handler)`
4. Check Grafana SLO dashboard: http://localhost:3001

## Resolution
1. **Fast burn (>14.4x)**: Immediate investigation required
   - Identify failing endpoint
   - Check dependencies (PostgreSQL, Redis, providers)
   - Consider rolling back recent changes
2. **Slow burn (>6x)**: Schedule investigation
   - Review error patterns over 6h window
   - Check for intermittent failures
   - Review recent deployments

## Error Budget Policy
- Budget > 50%: Normal operations
- Budget 25-50%: Increased monitoring, no risky deploys
- Budget < 25%: Feature freeze, reliability work only
- Budget exhausted: Incident review required

## Escalation
Fast burn alerts page on-call. Slow burn creates ticket for next business day.
