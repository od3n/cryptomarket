# Runbook: Provider Unavailable

## Alert
`PrimaryProviderDown` — Primary provider circuit breaker open, operating on fallback.

## Impact
Platform degraded. Data may have slight differences from primary source.

## Diagnosis
1. Check status: `curl http://localhost:8080/operations/status | jq .provider`
2. Check CoinGecko: `curl -s https://api.coingecko.com/api/v3/ping`
3. Check Prometheus: `circuit_breaker_state{name="coingecko"}`

## Resolution
1. Usually self-resolves: Circuit breaker probes every 30s
2. If CoinGecko API key issue: Check `PROVIDER_BASE_URL` configuration
3. If sustained: Consider adjusting `CIRCUIT_BREAKER_OPEN_DURATION`
4. Data continues flowing via fallback (CoinCap)

## Escalation
If fallback also fails, see `all-providers-unavailable.md`.
