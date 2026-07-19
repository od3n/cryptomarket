# Postmortem 001: Primary Provider Rate Limit Incident

## Summary

| Field | Value |
|-------|-------|
| Date | 2024-01-10 |
| Duration | 47 minutes |
| Severity | SEV-2 (degraded service) |
| Impact | Data ingestion delayed, stale prices shown to users |
| Status | Resolved |

## Timeline

| Time (UTC) | Event |
|------------|-------|
| 14:03 | CoinGecko begins returning 429 responses |
| 14:05 | Ingestion failures increase, no fallback available |
| 14:08 | Data freshness degrades (delayed → stale) |
| 14:12 | Alert fires: DataStaleCritical |
| 14:15 | On-call acknowledges, begins investigation |
| 14:22 | Root cause identified: CoinGecko free tier rate limit |
| 14:30 | Mitigation: Reduced ingestion interval from 60s to 120s |
| 14:45 | CoinGecko rate limit resets, ingestion resumes |
| 14:50 | Data freshness returns to normal |

## Root Cause

CoinGecko free-tier API rate limit (10-30 calls/minute) was exceeded due to:
1. Ingestion interval too aggressive for free tier
2. No retry backoff on 429 responses
3. No fallback provider configured

## Impact

- 47 minutes of stale market data
- ~47 missed ingestion cycles
- No data loss (historical data intact)
- User-facing: Dashboard showed stale indicators (after Phase 3 implementation)

## Resolution

Immediate:
- Reduced ingestion interval to respect rate limits
- Waited for rate limit window to reset

Long-term (Phase 3 implementation):
- Added CoinCap as fallback provider
- Implemented circuit breaker to detect rate limiting
- Added Retry-After header parsing
- Implemented exponential backoff with jitter
- Added provider fallback orchestration
- Created alerting for rate limit frequency

## Lessons Learned

1. **Single provider is a single point of failure** — Always have fallback
2. **Rate limits are predictable** — Respect Retry-After headers
3. **Circuit breakers prevent cascade** — Stop calling failing providers
4. **Freshness monitoring catches issues** — Users need stale indicators
5. **Free tiers have limits** — Plan for production-grade API access

## Action Items

| Action | Owner | Status |
|--------|-------|--------|
| Implement provider fallback | Platform | Done (Phase 3) |
| Add circuit breaker | Platform | Done (Phase 3) |
| Add rate limit handling | Platform | Done (Phase 3) |
| Add freshness alerting | SRE | Done (Phase 3) |
| Evaluate paid API tier | Product | Pending |
| Add incident demo script | SRE | Done (Phase 3) |

## Prevention

- Multi-provider fallback prevents single-provider dependency
- Circuit breaker automatically isolates failing providers
- Rate limit tracker respects Retry-After headers
- Burn-rate alerting catches degradation early
- Failure injection toolkit tests resilience regularly
