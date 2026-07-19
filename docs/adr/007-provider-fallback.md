# ADR-007: Provider Fallback Strategy

## Status

Accepted

## Context

The platform initially relied on a single data provider (CoinGecko). Single-provider dependency creates a single point of failure: rate limits, outages, or API changes can halt all data ingestion.

## Decision

Implement an ordered provider fallback strategy:

1. **Primary provider** (CoinGecko) is attempted first
2. **Fallback providers** (CoinCap) are tried in configured order
3. **Provider selection** respects disablement configuration
4. **Circuit breakers** prevent repeated calls to failing providers
5. **Rate limit tracking** skips providers that recently returned 429

The fallback orchestrator (`internal/provider/fallback.go`) coordinates selection, circuit breakers, retries, and rate limit tracking.

## Consequences

- Data ingestion continues during single-provider outages
- Slight complexity increase in the ingestion path
- Provider-specific normalization required (each adapter maps to common `MarketData`)
- Active provider recorded in snapshots for auditability
- Metrics emitted for fallback events and provider switches
