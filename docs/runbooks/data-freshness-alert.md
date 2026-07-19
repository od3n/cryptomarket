# Runbook: Data Freshness Alert

## Alert
`DataStaleCritical` / `SustainedStaleSymbols` — Market data is stale.

## Impact
Users see outdated prices. Dashboard shows stale indicators.

## Diagnosis
1. Check freshness: `curl http://localhost:8080/operations/status | jq .freshness`
2. Check ingestor: `docker compose logs ingestor --tail=30`
3. Check provider status: `curl http://localhost:8080/operations/status | jq .provider`

## Resolution
1. If ingestor stopped: `docker compose restart ingestor`
2. If provider failing: Check circuit breakers, wait for fallback
3. If Redis down (cache stale): `docker compose restart redis`

## Escalation
If stale > 10 minutes with no recovery, escalate.
