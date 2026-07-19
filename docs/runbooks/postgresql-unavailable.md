# Runbook: PostgreSQL Unavailable

## Alert
`PostgreSQLUnavailable` — Cannot connect to PostgreSQL database.

## Impact
No data persistence. API falls back to cache-only mode. Historical queries fail.

## Diagnosis
1. Check container: `docker compose ps postgres`
2. Check logs: `docker compose logs postgres --tail=30`
3. Test connection: `docker compose exec postgres pg_isready -U cryptouser`

## Resolution
1. Restart: `docker compose restart postgres`
2. If disk full: Free disk space on Docker volume
3. If corrupted: Check PostgreSQL logs for recovery information
4. Verify after restart: `curl http://localhost:8080/ready`

## Escalation
If data corruption suspected, escalate immediately. Do not restart repeatedly.
