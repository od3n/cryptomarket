# Runbook: Redis Unavailable

## Alert
`RedisUnavailable` — Cannot connect to Redis.

## Impact
Cache unavailable (higher API latency). Realtime streams interrupted. SSE delivery stopped.

## Diagnosis
1. Check container: `docker compose ps redis`
2. Check logs: `docker compose logs redis --tail=30`
3. Test: `docker compose exec redis redis-cli ping`

## Resolution
1. Restart: `docker compose restart redis`
2. If OOM: Check `maxmemory` settings
3. Verify: `curl http://localhost:8080/ready`
4. Realtime gateway auto-reconnects on Redis recovery

## Escalation
If Redis data loss suspected, check AOF persistence status.
