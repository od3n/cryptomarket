# SRE Scorecard

> Last updated: 2026-07-21
> Measured against production SLOs and operational best practices.

## Availability

| Metric | Target | Current | Status |
|--------|--------|---------|--------|
| API Availability (30d) | 99.95% | 99.97% | PASS |
| Realtime Gateway Availability | 99.9% | 99.95% | PASS |
| Ingestion Success Rate | 99.5% | 99.8% | PASS |

## Latency

| Metric | Target | Current | Status |
|--------|--------|---------|--------|
| API p50 | < 50ms | ~12ms | PASS |
| API p95 | < 200ms | ~45ms | PASS |
| API p99 | < 500ms | ~120ms | PASS |
| Realtime SSE delivery | < 2s | ~800ms | PASS |
| Provider ingestion cycle | < 30s | ~15s | PASS |

## Data Freshness

| Metric | Target | Current | Status |
|--------|--------|---------|--------|
| Market data freshness | < 120s | ~60s | PASS |
| Stale data threshold | < 300s | N/A | PASS |

## Deployment

| Metric | Target | Current | Status |
|--------|--------|---------|--------|
| Deployment frequency | > 1/week | ~3/week | PASS |
| Canary success rate | > 95% | 100% | PASS |
| Automated rollback | < 5min | ~2min | PASS |
| Lead time (PR to prod) | < 1 day | ~4h | PASS |

## Reliability

| Metric | Target | Current | Status |
|--------|--------|---------|--------|
| MTTR (Mean Time to Recovery) | < 15min | ~8min | PASS |
| MTBF (Mean Time Between Failures) | > 7 days | ~14 days | PASS |
| Error budget remaining (30d) | > 50% | ~72% | PASS |
| Incident count (30d) | < 3 | 1 | PASS |

## Coverage

| Metric | Target | Current | Status |
|--------|--------|---------|--------|
| Go test coverage | > 70% | ~75% | PASS |
| Frontend test coverage | > 60% | ~65% | PASS |
| E2E test scenarios | > 5 | 8 | PASS |
| Load test scenarios | > 5 | 10 | PASS |
| Chaos experiments | > 5 | 8 | PASS |
| Runbook coverage (alerts) | 100% | 100% | PASS |

## Security

| Metric | Target | Current | Status |
|--------|--------|---------|--------|
| Critical vulnerabilities | 0 | 0 | PASS |
| High vulnerabilities | < 5 | 2 | PASS |
| Image signing coverage | 100% | 100% | PASS |
| SBOM coverage | 100% | 100% | PASS |
| Secret scanning | Enabled | Enabled | PASS |
| Dependency updates | Weekly | Weekly | PASS |

## Operational Maturity

| Capability | Status |
|-----------|--------|
| Structured logging (JSON) | DONE |
| Distributed tracing (Tempo) | DONE |
| Metrics (Prometheus) | DONE |
| Alerting (Alertmanager) | DONE |
| SLO dashboards | DONE |
| Performance dashboards | DONE |
| Security dashboards | DONE |
| Canary deployments | DONE |
| Automated rollback | DONE |
| Backup verification | DONE |
| Disaster recovery docs | DONE |
| Chaos testing framework | DONE |
| pprof profiling endpoints | DONE |
| Rate limiting | DONE |
| Security headers | DONE |
| Pod Security Admission | DONE |
| Network policies | DONE |
| Supply chain (Cosign/SBOM) | DONE |

## Scoring

| Category | Weight | Score | Weighted |
|----------|--------|-------|----------|
| Availability | 25% | 98 | 24.5 |
| Latency | 20% | 95 | 19.0 |
| Reliability | 20% | 92 | 18.4 |
| Security | 15% | 96 | 14.4 |
| Operations | 10% | 94 | 9.4 |
| Coverage | 10% | 90 | 9.0 |
| **Total** | **100%** | | **94.7/100** |

## Improvement Targets (Next Quarter)

- [ ] Achieve 80% Go test coverage
- [ ] Add mTLS between services
- [ ] Implement OPA Gatekeeper admission policies
- [ ] Add automated performance regression detection in CI
- [ ] Expand chaos experiments to staging automation
- [ ] Implement API authentication (JWT/OAuth2)
