# Runbook: API Unavailable

## Alert
`APIUnavailable` — More than 10% of API requests failing with 5xx for 2 minutes.

## Impact
Users cannot access market data. Dashboard shows errors.

## Diagnosis
1. Check API container: `docker compose logs api --tail=50`
2. Check dependencies: `curl http://localhost:8080/operations/status`
3. Check PostgreSQL: `docker compose exec postgres pg_isready`
4. Check Redis: `docker compose exec redis redis-cli ping`

## Resolution
1. If PostgreSQL down: `docker compose restart postgres`
2. If Redis down: `docker compose restart redis`
3. If API crashed: `docker compose restart api`
4. Check for OOM: `docker inspect api --format='{{.State.OOMKilled}}'`

## Escalation
If not resolved in 5 minutes, escalate to platform team lead.
