# Runbook: Provider Ingestion Failures

## Overview

This runbook covers diagnosis and resolution of provider ingestion failures in the market-ingestor service.

## Symptoms

- `ingestion_failure_total` metric increasing
- `data_freshness_seconds` metric growing beyond expected interval
- No new entries in `price_snapshots` table
- Error logs from ingestor service containing "cycle failed"
- `provider_sync_logs` table showing `status = 'failure'`

## Diagnosis Steps

### 1. Check ingestor logs

```bash
docker compose logs ingestor --tail=50
```

Look for:
- Provider HTTP errors (429 rate limit, 5xx server errors)
- Context deadline exceeded (timeout)
- Connection refused (network issues)

### 2. Check provider sync logs

```sql
SELECT provider, status, error_message, request_duration_ms, started_at
FROM provider_sync_logs
ORDER BY started_at DESC
LIMIT 10;
```

### 3. Check data freshness

```bash
curl -s http://localhost:8080/metrics | grep data_freshness
```

If `data_freshness_seconds` exceeds 2x the ingestion interval, data is stale.

### 4. Verify provider availability

```bash
curl -s "https://api.coingecko.com/api/v3/ping"
```

## Common Causes and Resolutions

### Rate Limiting (HTTP 429)

**Cause**: CoinGecko free tier allows ~10-30 requests/minute.

**Resolution**:
- Increase `INGESTION_INTERVAL` to 120s or higher
- Consider upgrading to a paid provider tier
- Future: implement exponential backoff (planned for resilience phase)

### Timeout

**Cause**: Provider response exceeds `PROVIDER_TIMEOUT`.

**Resolution**:
- Increase `PROVIDER_TIMEOUT` (e.g., 30s)
- Check network connectivity from the container
- Verify DNS resolution inside Docker network

### Provider API Down (HTTP 5xx)

**Cause**: External provider outage.

**Resolution**:
- Wait for provider recovery (check their status page)
- Cached data remains available via API until TTL expires (5 minutes)
- Historical data in PostgreSQL remains accessible
- Future: multi-provider fallback will mitigate this

### Database Connection Lost

**Cause**: PostgreSQL restart or network partition.

**Resolution**:
```bash
docker compose ps postgres
docker compose logs postgres --tail=20
docker compose restart postgres
```

The ingestor will automatically retry on the next cycle.

### Redis Unavailable

**Cause**: Redis restart or memory exhaustion.

**Resolution**:
```bash
docker compose ps redis
docker compose logs redis --tail=20
docker compose restart redis
```

Note: Ingestion still persists to PostgreSQL even if Redis writes fail (errors are logged but non-fatal).

## Escalation

If failures persist beyond 30 minutes:
1. Verify the provider API is responding externally
2. Check Docker resource limits (`docker stats`)
3. Review recent deployments or configuration changes
4. Consider switching `PROVIDER_BASE_URL` to an alternative provider (when multi-provider support is available)

## Prevention

- Monitor `ingestion_failure_total` with alerting threshold
- Alert when `data_freshness_seconds > 180`
- Track provider response times via `provider_request_duration_seconds`
