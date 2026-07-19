# Runbook: Provider Rate Limiting

## Alert
`HighRateLimitFrequency` — Provider frequently returning 429.

## Impact
Increased fallback usage. Potential data gaps if all providers rate-limited.

## Diagnosis
1. Check rate limit metrics: `rate(provider_rate_limited_total[5m])`
2. Check Retry-After headers in ingestor logs
3. Verify ingestion interval isn't too aggressive

## Resolution
1. Reduce ingestion frequency: Increase `INGESTION_INTERVAL`
2. Verify fallback is working: Check `provider_active` metric
3. If CoinGecko free tier: Consider upgrading API plan
4. Rate limiter automatically backs off using Retry-After header

## Escalation
If rate limiting persists > 1 hour, review API usage patterns.
