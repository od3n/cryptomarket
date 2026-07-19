# ADR-011: SLO Definitions

## Status

Accepted

## Context

The platform needs measurable reliability targets to guide engineering decisions, alerting thresholds, and error budget policies.

## Decision

Define the following SLOs (documented in `docs/sre/slos.md`):

| SLI | Target | Window |
|-----|--------|--------|
| API Availability | 99.9% | 30 days |
| API Latency (p99 < 300ms) | 99% | 30 days |
| Ingestion Success | 99.5% | 30 days |
| Data Freshness | 99% | 30 days |
| Realtime Delivery | 99.5% | 30 days |

Error budgets are calculated as `1 - target`. Burn-rate alerting uses multi-window patterns from the Google SRE Workbook.

## Consequences

- Quantified reliability targets enable objective decision-making
- Error budget policy: freeze features when budget exhausted
- Recording rules pre-compute SLIs for efficient dashboarding
- Burn-rate alerts reduce false positives vs. threshold alerts
