# AI Use Case Portfolio

## Purpose

Identify, evaluate, and prioritize AI opportunities across the platform. Each use case documents expected value, operational risk, and implementation readiness based on repository evidence.

**Value rating:** Impact on toil reduction, incident response time, or engineering velocity (H/M/L).
**Risk rating:** Potential for harm if AI output is wrong or misused (H/M/L).
**Priority:** Value/Risk ratio adjusted for implementation readiness.

---

## Engineering Use Cases

### E-1: PR Summaries

| Attribute | Detail |
|-----------|--------|
| Description | Auto-generate PR descriptions from diffs: what changed, why (from commit messages), affected services, risk areas |
| Expected value | M — Saves 10-15 min per PR; improves review quality for reviewers |
| Operational risk | L — Informational only; no production impact |
| Readiness | High — Conventional commits enforced (release-please), clear package structure |
| Evidence | `.release-please-config.json`, 12 services/packages in `internal/` |
| Priority | **P1** |

### E-2: Code Review Assistance

| Attribute | Detail |
|-----------|--------|
| Description | AI pre-review checking: architecture consistency with ADRs, coding standards (golangci rules), security patterns, observability coverage (metrics/logs on new code paths), test completeness |
| Expected value | H — Catches issues before human review; enforces 20 ADR constraints consistently |
| Operational risk | L — Advisory comments only; human reviewer retains approval authority |
| Readiness | High — `.golangci.yml`, 20 ADRs as constraint corpus, 15 test files as patterns |
| Evidence | `docs/adr/`, `.golangci.yml`, `ci.yml` |
| Priority | **P1** |

### E-3: Documentation Generation

| Attribute | Detail |
|-----------|--------|
| Description | Generate/update package docs, API docs from OpenAPI spec, architecture diagram updates when topology changes |
| Expected value | M — Keeps 70+ docs current; reduces drift |
| Operational risk | L — Review before publication gate |
| Readiness | High — `api/openapi.yaml`, consistent doc format, mermaid diagrams in docs |
| Evidence | `api/openapi.yaml`, `docs/architecture/overview.md` |
| Priority | **P2** |

### E-4: Architecture Review

| Attribute | Detail |
|-----------|--------|
| Description | Check proposed changes against ADR constraints (e.g., "prices must be string-based" per overview, "SSE over WebSockets" per ADR-003, "retry ownership" per ADR-008) |
| Expected value | M — Prevents architecture erosion as team grows |
| Operational risk | L — Advisory; ADR process remains human-owned |
| Readiness | High — 20 well-structured ADRs |
| Evidence | `docs/adr/001-020` |
| Priority | **P2** |

---

## Operations Use Cases

### O-1: Incident Summaries

| Attribute | Detail |
|-----------|--------|
| Description | When alerts fire, generate a situational summary: what's alerting, affected services, current SLO budget state, recent deployments, similar historical incidents |
| Expected value | H — Reduces time-to-understand from ~10 min to <2 min (based on postmortem 001 timeline: 9 min from alert to root cause identification) |
| Operational risk | M — Wrong summary could misdirect investigation; mitigated by evidence citations |
| Readiness | Medium — Alerts have runbook links; needs metric catalog and incident history structure |
| Evidence | `monitoring/prometheus/alerts.yml` (18 alerts), `docs/postmortems/001` |
| Priority | **P1** |

### O-2: Alert Correlation

| Attribute | Detail |
|-----------|--------|
| Description | Group related alerts into a single incident view (e.g., `HighRateLimitFrequency` → `PrimaryProviderDown` → `DataStaleCritical` is one causal chain, not three incidents) |
| Expected value | H — Reduces alert fatigue; 18 alerts across 4 groups with known causal relationships |
| Operational risk | M — Incorrect grouping could hide independent failures |
| Readiness | High — Causal chains documented in runbooks and postmortem 001 |
| Evidence | Alert groups in `alerts.yml`, postmortem 001 timeline showing cascade |
| Priority | **P1** |

### O-3: Root Cause Hypotheses

| Attribute | Detail |
|-----------|--------|
| Description | Given an active alert set, generate ranked hypotheses with supporting evidence (metrics, logs, recent changes) and confidence scores |
| Expected value | H — Postmortem 001 shows 19 min from alert to root cause; AI could suggest "provider rate limit" in seconds based on `provider_rate_limited_total` |
| Operational risk | M — Anchoring bias risk; mitigated by presenting multiple hypotheses with evidence |
| Readiness | Medium — Requires structured metric queries and deployment history |
| Evidence | `docs/postmortems/001`, `monitoring/prometheus/recording-rules.yml` |
| Priority | **P2** |

### O-4: Runbook Recommendations

| Attribute | Detail |
|-----------|--------|
| Description | Given alert context, recommend the most relevant runbook and pre-fill diagnostic command outputs |
| Expected value | M — Alerts already link runbooks; AI adds context-aware selection when multiple apply |
| Operational risk | L — Runbooks remain human-executed; recommendation is informational |
| Readiness | High — `runbook:` annotations already map all 18 alerts to 12 runbooks |
| Evidence | `alerts.yml` annotations, `docs/runbooks/` |
| Priority | **P2** |

---

## Infrastructure Use Cases

### I-1: Terraform Plan Review

| Attribute | Detail |
|-----------|--------|
| Description | Analyze `terraform plan` output: security implications (IAM, security groups, encryption), cost impact, blast radius, drift from documented architecture |
| Expected value | H — 13 modules across 3 environments; plans are complex and errors are expensive |
| Operational risk | M — Missed issue in review; mitigated by human approval requirement (plan workflow already requires manual apply) |
| Readiness | High — `terraform-plan.yml` workflow produces plan output; module structure documented |
| Evidence | `.github/workflows/terraform-plan.yml`, `deploy/terraform/modules/` (13 modules) |
| Priority | **P1** |

### I-2: Kubernetes Manifest Review

| Attribute | Detail |
|-----------|--------|
| Description | Review manifest changes: resource limits, probe configuration, security contexts, HPA bounds, consistency with `deploy/kubernetes/base/` conventions |
| Expected value | M — Prevents misconfigurations that cause outages |
| Operational risk | L — Advisory; CI and canary stages catch runtime issues |
| Readiness | High — Existing manifests as convention corpus |
| Evidence | `deploy/kubernetes/base/` (13 dirs), `deploy/helm/cryptomarket/` |
| Priority | **P2** |

### I-3: Cost Optimization

| Attribute | Detail |
|-----------|--------|
| Description | Analyze resource utilization vs. requests: oversized RDS instances, over-provisioned ElastiCache, unnecessary NAT throughput, idle load balancers |
| Expected value | M — `docs/cost/estimates.md` provides baseline; typical waste is 20-40% |
| Operational risk | M — Under-provisioning risk; mitigated by human approval and gradual changes |
| Readiness | Medium — Needs CloudWatch/Cost Explorer access patterns |
| Evidence | `docs/cost/estimates.md`, `deploy/terraform/modules/` |
| Priority | **P3** |

### I-4: Capacity Recommendations

| Attribute | Detail |
|-----------|--------|
| Description | Forecast resource needs from Prometheus history: CPU/memory trends, Redis stream growth, PostgreSQL storage, network throughput |
| Expected value | M — Prevents capacity surprises; supports quarterly planning |
| Operational risk | L — Recommendations only; human approves changes |
| Readiness | Medium — Requires metrics history retention |
| Evidence | `monitoring/prometheus/recording-rules.yml`, `docs/deployment/autoscaling-strategy.md` |
| Priority | **P3** |

---

## Reliability Use Cases

### R-1: Anomaly Detection and Explanation

| Attribute | Detail |
|-----------|--------|
| Description | Detect statistical anomalies in metrics and explain them in context (distinguish "unusual but benign" from "confirmed incident") |
| Expected value | M — Recording rules already compute rates; AI adds explanation layer |
| Operational risk | M — False positives cause alert fatigue; false negatives miss incidents |
| Readiness | Medium — 201-line recording rules provide semantic context |
| Evidence | `recording-rules.yml`, `docs/sre/alert-tuning-review.md` |
| Priority | **P3** |

### R-2: SLO Forecasting

| Attribute | Detail |
|-----------|--------|
| Description | Project error budget consumption: "At current burn rate, API availability budget exhausts in N days" with confidence intervals |
| Expected value | M — Enables proactive reliability investment |
| Operational risk | L — Informational; existing burn-rate alerts handle reactive cases |
| Readiness | High — SLO formulas defined in `docs/sre/slos.md`, recording rules compute burn rates |
| Evidence | `docs/sre/slos.md`, `slo-error-budget.json` dashboard |
| Priority | **P2** |

### R-3: Error Budget Analysis

| Attribute | Detail |
|-----------|--------|
| Description | Attribute budget consumption to services, deployments, and incidents; identify top budget consumers |
| Expected value | M — Supports error budget policy decisions (feature freeze triggers) |
| Operational risk | L — Analytical; human owns policy decisions |
| Readiness | High — Budget formulas and policy documented |
| Evidence | `docs/sre/slos.md` (Error Budget Policy), `check-slo-gate.sh` |
| Priority | **P3** |

---

## Developer Experience Use Cases

### D-1: Onboarding Assistant

| Attribute | Detail |
|-----------|--------|
| Description | Answer new-engineer questions grounded in repo docs: "How does data flow from providers to the dashboard?" citing `docs/architecture/overview.md` |
| Expected value | M — 145-line onboarding doc + 70 docs to navigate; AI reduces time-to-first-contribution |
| Operational risk | L — Read-only Q&A with citations |
| Readiness | High — Rich documentation corpus |
| Evidence | `docs/onboarding.md`, `docs/architecture/` |
| Priority | **P2** |

### D-2: Architecture Q&A

| Attribute | Detail |
|-----------|--------|
| Description | Answer "why" questions grounded in ADRs: "Why SSE instead of WebSockets?" → cites ADR-003 with context and consequences |
| Expected value | M — Preserves decision context; prevents re-litigating settled decisions |
| Operational risk | L — Citation-backed answers; "I don't know" when no ADR exists |
| Readiness | High — 20 ADRs with consistent structure |
| Evidence | `docs/adr/` |
| Priority | **P2** |

### D-3: Repository Search

| Attribute | Detail |
|-----------|--------|
| Description | Semantic search across code, docs, configs: "Where is rate limiting handled?" → `internal/resilience/ratelimit.go`, ADR-008, provider-rate-limiting runbook |
| Expected value | M — Cross-artifact search currently requires knowing where to look |
| Operational risk | L — Read-only retrieval |
| Readiness | High — Well-organized repository structure |
| Evidence | Repository structure, `internal/` packages |
| Priority | **P3** |

---

## Prioritized Portfolio

| Rank | Use Case | Value | Risk | Rationale |
|------|----------|-------|------|-----------|
| 1 | O-2: Alert Correlation | H | M | Known causal chains, immediate toil reduction, builds foundation for O-1/O-3 |
| 2 | E-2: Code Review Assistance | H | L | Low risk, high leverage, ADR corpus ready |
| 3 | I-1: Terraform Plan Review | H | M | Expensive failure mode, plan output already available in CI |
| 4 | O-1: Incident Summaries | H | M | Highest operational impact; depends on O-2 foundation |
| 5 | E-1: PR Summaries | M | L | Quick win, builds AI-in-CI infrastructure |
| 6 | O-4: Runbook Recommendations | M | L | Alert-runbook links already exist |
| 7 | R-2: SLO Forecasting | M | L | Formulas ready; extends existing gate |
| 8 | D-2: Architecture Q&A | M | L | ADR corpus ready; demo-friendly |
| 9 | E-3: Documentation Generation | M | L | OpenAPI spec provides structure |
| 10 | O-3: Root Cause Hypotheses | H | M | Requires O-1/O-2 maturity first |
| 11 | I-2: K8s Manifest Review | M | L | Convention corpus exists |
| 12 | D-1: Onboarding Assistant | M | L | Doc corpus ready |
| 13 | E-4: Architecture Review | M | L | Extends E-2 |
| 14 | R-3: Error Budget Analysis | M | L | Analytical, lower urgency |
| 15 | I-3: Cost Optimization | M | M | Needs cost data access |
| 16 | R-1: Anomaly Detection | M | M | False positive risk; tune alerts first |
| 17 | I-4: Capacity Recommendations | M | L | Needs metrics retention |
| 18 | D-3: Repository Search | M | L | Generic tooling may suffice |

---

## Implementation Sequencing

```
Phase A (Quick wins):  E-1 PR Summaries + O-4 Runbook Recommendations
Phase B (Core ops):    O-2 Alert Correlation → O-1 Incident Summaries
Phase C (Review):      E-2 Code Review + I-1 Terraform Review
Phase D (Advanced):    O-3 RCA Hypotheses + R-2 SLO Forecasting
Phase E (Knowledge):   D-1/D-2 Onboarding + Architecture Q&A
```

Each phase is independently valuable and removable. No phase requires the previous phase to remain operational.
