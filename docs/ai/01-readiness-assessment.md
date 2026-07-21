# AI Readiness Assessment

## Purpose

Evaluate how prepared this repository is for AI-assisted engineering and operations. Each area is scored 0–5 based on evidence found in the repository, with blockers and improvement opportunities identified.

**Scoring scale:**

| Score | Meaning |
|-------|---------|
| 0 | Nothing exists |
| 1 | Ad hoc, inconsistent |
| 2 | Basic coverage, significant gaps |
| 3 | Solid foundation, usable by AI with caveats |
| 4 | Strong, structured, mostly machine-consumable |
| 5 | Exemplary, fully machine-consumable, continuously validated |

**Overall score: 3.9 / 5** — The platform is well-positioned for AI augmentation. Primary gaps are in incident history depth, dashboard metadata, and lack of machine-readable relationship data between artifacts.

---

## 1. Source Code — Score: 4

### Evidence

- Single Go module with clear package boundaries (`internal/api`, `internal/cache`, `internal/config`, `internal/market`, `internal/provider`, `internal/repository`, `internal/resilience`, `internal/realtime`, `internal/scheduler`, `internal/stream`, `internal/subscriber`, `internal/telemetry`, `internal/worker`)
- Interface-driven design (provider adapters implement a common interface)
- Lint configuration: `.golangci.yml`
- Frontend: Next.js with TypeScript, ESLint config, strict types in `frontend/types/`
- OpenAPI specification: `api/openapi.yaml`

### Blockers

- No inline architecture annotations or doc comments on all exported symbols
- No CODEOWNERS-enforced documentation coupling (code changes do not require doc updates)

### Improvement Opportunities

- Add structured doc comments to exported interfaces for AI code comprehension
- Generate package-level documentation automatically
- AI-assisted PR summaries can leverage the clear package structure immediately

---

## 2. Documentation — Score: 4

### Evidence

- 70+ markdown documents across: `docs/architecture/` (3), `docs/operations/` (4), `docs/security/` (4), `docs/sre/` (5), `docs/dr/` (3), `docs/multi-region/` (6), `docs/deployment/` (3), `docs/backup/` (2), `docs/testing/` (2), `docs/cost/` (1)
- Onboarding guide: `docs/onboarding.md`
- Consistent structure with headers, tables, and mermaid diagrams
- Architecture overview includes data flow, deployment view, and CI/CD pipeline diagrams

### Blockers

- No documentation freshness metadata (last-reviewed dates)
- No machine-readable index of documents and their relationships
- Some documents reference future state (multi-region) not yet implemented

### Improvement Opportunities

- Documentation drift detection (AI can compare docs against actual configs)
- Automated last-reviewed metadata via git history
- AI-generated documentation index with topic clustering

---

## 3. Observability — Score: 4

### Evidence

- Prometheus: `monitoring/prometheus/prometheus.yml`, 282-line alert rules with runbook annotations, 201-line recording rules
- Structured logging: `docs/operations/logging.md`, JSON structured logs with request IDs
- Distributed tracing: `docs/operations/tracing.md`, OTel Collector in stack
- Synthetic monitoring: `monitoring/synthetic/checks.yaml` (145 lines)
- Loki for log aggregation: `monitoring/loki/`
- SLO definitions with PromQL formulas: `docs/sre/slos.md`

### Blockers

- Metric metadata (unit, owner, dashboard links) not machine-readable
- No centralized metric catalog or data dictionary
- Trace-to-alert correlation not formalized

### Improvement Opportunities

- AI can build a metric catalog from recording rules and alert expressions
- Anomaly explanation using existing recording rules as semantic context
- Missing telemetry detection by comparing service endpoints against instrumented metrics

---

## 4. Infrastructure — Score: 4

### Evidence

- Docker Compose full-stack: `docker-compose.yml`
- Kubernetes manifests: `deploy/kubernetes/base/` (13 subdirectories)
- Helm chart with monitoring subchart: `deploy/helm/cryptomarket/`
- Kind local cluster: `deploy/kind/cluster.yaml`
- Tiltfile for dev orchestration
- Production Redis config: `deploy/redis/redis-prod.conf`

### Blockers

- No infrastructure documentation auto-generated from manifests
- Helm values spread across multiple files without unified schema documentation

### Improvement Opportunities

- AI-assisted manifest review (resource limits, security contexts, probes)
- Automated infrastructure documentation generation from Helm templates
- Configuration drift narrative generation

---

## 5. Runbooks — Score: 4

### Evidence

- 12 runbooks in `docs/runbooks/` covering: API unavailable, ingestion failure, all providers unavailable, data freshness, error budget burn, high API latency, PostgreSQL/Redis unavailable, provider issues (4 runbooks), realtime degraded
- Alerts directly reference runbooks via `runbook:` annotation in `monitoring/prometheus/alerts.yml`
- Consistent format: Alert, Impact, Diagnosis, Resolution, Escalation
- Detailed provider ingestion failures runbook (119 lines) with decision trees

### Blockers

- Some runbooks are brief (20-22 lines) with limited diagnostic depth
- No runbook execution history or effectiveness tracking
- No machine-readable runbook metadata (applicable alerts, required access, estimated resolution time)

### Improvement Opportunities

- AI can recommend runbooks during incidents using alert-to-runbook links
- Runbook enrichment from incident history
- AI-suggested diagnostic commands based on alert context

---

## 6. Dashboards — Score: 3

### Evidence

- 5 Grafana dashboards: `platform-overview.json`, `provider-reliability.json`, `slo-error-budget.json`, `performance-engineering.json`, `security-posture.json`
- Provisioned via `monitoring/grafana/provisioning/`
- SLO dashboard implements error budget burn visualization

### Blockers

- Dashboard JSON not annotated with purpose/audience metadata
- No dashboard-to-alert mapping documentation
- No dashboard usage analytics to identify unused panels

### Improvement Opportunities

- AI dashboard summarization (describe current state from panel queries)
- Dashboard improvement recommendations based on alert coverage gaps
- Natural-language descriptions of what each dashboard monitors

---

## 7. Incident History — Score: 2

### Evidence

- 1 postmortem: `docs/postmortems/001-primary-provider-rate-limit.md` with full timeline, root cause, lessons learned, action items
- Incident demo script: `scripts/incident-demo.sh`
- Failure injection toolkit: `sre-toolkit/inject_failures.py` (4 scenarios: provider_429, provider_500, redis_failure, stale_data)
- Severity matrix: `docs/operations/severity-matrix.md`
- On-call documentation: `docs/operations/on-call.md`

### Blockers

- Only one historical incident documented (limited training data for pattern matching)
- No structured incident metadata format (JSON/YAML frontmatter)
- No incident-to-alert-to-runbook traceability database

### Improvement Opportunities

- Generate synthetic incident scenarios from failure injection toolkit for AI training
- Structure postmortem metadata for machine consumption
- AI-assisted timeline generation from logs/metrics during live incidents

---

## 8. CI/CD — Score: 5

### Evidence

- 12 GitHub Actions workflows: `ci.yml`, `security.yml`, `codeql.yml`, `build-sign.yml`, `deploy-staging.yml`, `deploy-production.yml`, `chaos-staging.yml`, `performance-gate.yml`, `terraform-plan.yml`, `terraform-drift.yml`, `release-please.yml`, `preview-env.yml`
- SLO deployment gate: `scripts/check-slo-gate.sh` integrated into production deploy
- Canary deployment stages (5% → 25% → 50% → 100%) with verification
- OIDC-based AWS authentication (no static credentials)
- Conventional commits enforced via release-please
- Supply chain security: signed builds (ADR-017)

### Blockers

- Workflow logs not retained/analyzed for pattern detection
- No deployment frequency metrics dashboard

### Improvement Opportunities

- AI-assisted deployment review using workflow outputs and canary metrics
- CI failure pattern analysis and flaky test detection
- Deployment risk scoring from change size, affected services, and error budget state

---

## 9. Terraform — Score: 4

### Evidence

- 13 modules: `networking`, `eks`, `rds`, `elasticache`, `s3`, `iam`, `kms`, `dns`, `acm`, `monitoring`, `secrets`, `secrets-rotation`, `waf`
- 3 environments: `deploy/terraform/environments/` (dev, staging, prod)
- Drift detection workflow: `.github/workflows/terraform-drift.yml`
- Plan review workflow: `.github/workflows/terraform-plan.yml`

### Blockers

- No module documentation beyond code comments
- No cost annotations on resources
- Module interdependencies not documented as a graph

### Improvement Opportunities

- AI-assisted Terraform plan review (security, cost, blast radius)
- Automated module documentation generation
- Drift explanation narratives from plan diffs

---

## 10. Kubernetes — Score: 4

### Evidence

- Base manifests: `deploy/kubernetes/base/` (13 subdirectories including monitoring with Falco rules)
- Helm chart with values per environment
- HPA configuration documented in `docs/deployment/autoscaling-strategy.md`
- Canary strategy: `docs/deployment/canary-strategy.md`
- Ingress configuration: `docs/deployment/ingress-configuration.md`
- Runtime security: Falco rules (166 lines)

### Blockers

- No manifest schema validation documentation
- Resource request/limit rationale not documented

### Improvement Opportunities

- AI manifest review against platform conventions
- Autoscaling recommendation analysis from metrics history
- Security context compliance checking

---

## 11. ADRs — Score: 5

### Evidence

- 20 ADRs in `docs/adr/` covering: language choice, data stores, realtime delivery, consumer groups, delivery policy, frontend, provider fallback, retry ownership, circuit breaker, freshness, SLOs, burn-rate alerting, degraded mode, failure injection, security model, performance, supply chain, chaos testing, release strategy, multi-region
- Consistent format with context, decision, consequences
- Decisions are specific, bounded, and referenceable

### Blockers

- No ADR supersession tracking
- No machine-readable ADR index with status (accepted/superseded/deprecated)

### Improvement Opportunities

- AI architecture review grounded in ADR constraints
- ADR gap detection (decisions made in code but not documented)
- AI-assisted ADR drafting for new decisions

---

## 12. Tests — Score: 4

### Evidence

- 15 Go test files covering: API handlers, config, validation, providers (coincap, coingecko, contract, fallback, selector), realtime, resilience (circuit breaker, rate limit, retry), streams (consumer, event), subscriber hub
- Frontend: 4 test suites (ConnectionStatus, FreshnessBadge, MarketTable, freshness logic)
- E2E: Playwright (`e2e/dashboard.spec.ts`)
- Load tests: k6 (`load-tests/benchmark.js`, `resilience.js`, `scale.js`)
- Python toolkit tests: `sre-toolkit/tests/`
- Chaos experiments: `scripts/chaos/run-experiment.sh`
- Contract tests for provider adapters

### Blockers

- No test coverage reporting in CI
- No mutation testing
- Integration tests require Docker (not always run)

### Improvement Opportunities

- AI-assisted test gap analysis (compare code paths against test coverage)
- Test generation for uncovered error paths
- AI review of test completeness on PRs

---

## 13. Operational Tooling — Score: 4

### Evidence

- SRE toolkit: `sre-toolkit/inject_failures.py` (failure injection with safeguards), `sre-toolkit/reconcile_prices.py` (cross-provider price reconciliation)
- Backup automation: `scripts/backup/backup.py`
- Restore verification: `scripts/restore/verify_restore.py`
- Operations automation: `scripts/ops/automate.sh`
- SLO gate: `scripts/check-slo-gate.sh`
- Incident demo: `scripts/incident-demo.sh`
- Makefile with 30+ operational targets

### Blockers

- Tool outputs are human-readable text, not structured JSON
- No unified operational CLI (tools are scattered scripts)
- No tool execution audit log

### Improvement Opportunities

- Structured output modes for all operational tools (AI-consumable)
- AI orchestration layer that composes existing tools
- Tool recommendation engine based on alert context

---

## Summary Matrix

| Area | Score | Key Strength | Primary Gap |
|------|-------|--------------|-------------|
| Source Code | 4 | Clear package boundaries | Missing doc annotations |
| Documentation | 4 | Breadth and consistency | No freshness tracking |
| Observability | 4 | Alerts with runbook links | No metric catalog |
| Infrastructure | 4 | Multi-layer (compose/kind/k8s/helm) | No generated docs |
| Runbooks | 4 | Alert-linked, consistent format | Limited depth in some |
| Dashboards | 3 | SLO error budget dashboard | No metadata/annotations |
| Incident History | 2 | Structured postmortem format | Only 1 incident recorded |
| CI/CD | 5 | SLO gates, canary, signing | No failure analytics |
| Terraform | 4 | Drift detection, 13 modules | No cost annotations |
| Kubernetes | 4 | Falco, HPA, canary docs | No rationale docs |
| ADRs | 5 | 20 decisions, consistent | No status tracking |
| Tests | 4 | Multi-level (unit→chaos) | No coverage reporting |
| Operational Tooling | 4 | Failure injection, reconciliation | Unstructured outputs |

## Recommended Priority Actions

1. **Add structured metadata to incidents and runbooks** — enables AI incident correlation (addresses Incident History gap)
2. **Create machine-readable metric catalog** — enables AI observability assistants (addresses Observability/Dashboard gaps)
3. **Add structured JSON output to operational tools** — enables AI tool orchestration (addresses Tooling gap)
4. **Track documentation freshness** — enables AI drift detection (addresses Documentation gap)
