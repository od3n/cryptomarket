# Runbook: High API Latency

## Alert
`APIHighLatency` — 99th percentile API latency exceeds 500ms.

## Impact
Slow dashboard loading. Poor user experience.

## Diagnosis
1. Check latency: `api:latency_p99:5m` in Prometheus
2. Check Redis latency: `docker compose exec redis redis-cli --latency`
3. Check PostgreSQL: `docker compose exec postgres pg_stat_activity`
4. Check API container resources: `docker stats api`

## Resolution
1. If Redis slow: Check memory usage, restart if needed
2. If PostgreSQL slow: Check for long-running queries
3. If API overloaded: Scale or restart container
4. If network issue: Check Docker network

## Escalation
If p99 > 2s for > 10 minutes, escalate.
