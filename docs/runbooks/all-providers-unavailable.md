# Runbook: All Providers Unavailable

## Alert
`AllProvidersUnavailable` — All provider circuit breakers are open.

## Impact
No data ingestion possible. Data will become stale. Platform enters unavailable state.

## Diagnosis
1. Check all circuit breakers: `curl http://localhost:8080/operations/status | jq .provider.circuit_states`
2. Check network: `curl -s https://api.coingecko.com/api/v3/ping`
3. Check CoinCap: `curl -s https://api.coincap.io/v2/assets/bitcoin`
4. Check ingestor logs: `docker compose logs ingestor --tail=100`

## Resolution
1. If network issue: Resolve network connectivity
2. If providers genuinely down: Wait for recovery (circuit breakers probe every 30s)
3. If config issue: Verify `PROVIDER_BASE_URL` and `COINCAP_BASE_URL`
4. Force circuit reset: `docker compose restart ingestor`

## Escalation
Immediate escalation required. This is a critical incident.
