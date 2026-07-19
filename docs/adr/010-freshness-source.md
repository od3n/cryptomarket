# ADR-010: Freshness Source of Truth

## Status

Accepted

## Context

Data freshness can be measured from multiple sources: provider timestamp, ingestion timestamp, cache write time, or API response time. The choice affects staleness detection accuracy.

## Decision

**Ingestion timestamp is the source of truth for freshness.** Specifically:

- Freshness is calculated from when data was last successfully ingested and written to cache
- The `FreshnessTracker` (`internal/market/freshness.go`) tracks per-symbol last-update times
- Thresholds: fresh (≤120s), delayed (≤300s), stale (>300s)
- Provider timestamps are validated but not used for freshness (providers may have clock skew)

## Consequences

- Freshness accurately reflects platform data pipeline health
- Independent of provider clock accuracy
- Per-symbol tracking enables granular staleness detection
- Configurable thresholds via `FRESHNESS_THRESHOLD` and `STALE_THRESHOLD`
- Metric `data_freshness_status` enables Prometheus alerting on staleness
