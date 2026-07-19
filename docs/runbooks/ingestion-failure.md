# Runbook: Ingestion Failure

## Alert
`IngestionFailing` — No successful ingestion cycles in 5 minutes.

## Impact
Market data becomes stale. No new prices ingested.

## Diagnosis
1. Check ingestor logs: `docker compose logs ingestor --tail=50`
2. Check provider status: `curl http://localhost:8080/operations/status`
3. Check circuit breakers: Look for `circuit_breaker_state` in Prometheus
4. Verify provider connectivity: `curl -s https://api.coingecko.com/api/v3/ping`

## Resolution
1. If provider rate limited: Wait for circuit breaker recovery (30s)
2. If all providers down: Check network connectivity
3. If ingestor crashed: `docker compose restart ingestor`
4. If config error: Check environment variables

## Escalation
If all providers are down for > 10 minutes, declare incident.
