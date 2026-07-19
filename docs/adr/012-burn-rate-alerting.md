# ADR-012: Burn-Rate Alerting

## Status

Accepted

## Context

Traditional threshold alerts (e.g., "error rate > 1%") generate false positives during brief spikes and miss slow burns. Multi-window, multi-burn-rate alerting provides better signal-to-noise.

## Decision

Implement multi-window burn-rate alerts following the Google SRE Workbook pattern:

- **Fast burn** (critical): 1h long window + 5m short window at 14.4x burn rate
  - Consumes 2% of 30-day budget in 1 hour
  - Pages on-call immediately
- **Slow burn** (warning): 6h long window + 30m short window at 6x burn rate
  - Consumes 5% of budget in 6 hours
  - Creates ticket for next business day

Both windows must fire simultaneously to reduce false positives.

Implementation: `monitoring/prometheus/alerts.yml` with recording rules from `recording-rules.yml`.

## Consequences

- Fewer false positives than simple threshold alerts
- Catches both fast failures and slow degradation
- Clear severity mapping: fast burn = critical, slow burn = warning
- Requires recording rules for efficient evaluation
