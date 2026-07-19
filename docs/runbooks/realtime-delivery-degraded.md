# Runbook: Realtime Delivery Degraded

## Alert
`RealtimeConsumerLag` — Stream consumer lag exceeds 1000 messages.

## Impact
Realtime price updates delayed. Dashboard shows stale SSE data.

## Diagnosis
1. Check realtime logs: `docker compose logs realtime --tail=30`
2. Check Redis streams: `docker compose exec redis redis-cli XLEN market:events`
3. Check consumer group: `docker compose exec redis redis-cli XINFO GROUPS market:events`
4. Check active connections: `realtime_active_connections` metric

## Resolution
1. If consumer stuck: `docker compose restart realtime`
2. If Redis overloaded: Check memory, consider trimming streams
3. If too many connections: Review client reconnection patterns
4. Consumer auto-recovers on restart

## Escalation
If lag > 10000 for > 5 minutes, escalate.
