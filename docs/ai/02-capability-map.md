# AI Capability Map

## Purpose

Classify every engineering and operational capability by its current automation maturity and its target state with AI augmentation.

**Maturity levels:**

| Level | Definition |
|-------|-----------|
| Manual | Human performs entirely, no tooling assistance |
| Assisted | Tooling exists; AI could augment but does not today |
| Partially Automated | Automation handles routine cases; human handles exceptions |
| Fully Automated with Approval | Automation executes after explicit human approval |
| Intentionally Manual | Deliberately kept human-driven (safety, judgment, governance) |

---

## Engineering

| Capability | Current | Target | Rationale |
|-----------|---------|--------|-----------|
| Feature implementation | Manual | Assisted | AI pair programming; human owns design decisions |
| Code review | Assisted (lint, CI checks) | Partially Automated | AI pre-review flags issues; human approves |
| PR summarization | Manual | Fully Automated with Approval | Low risk; auto-generated summaries posted to PRs |
| Architecture review | Manual (ADR process) | Assisted | AI checks consistency with ADRs 001–020; human decides |
| Dependency updates | Partially Automated (Dependabot) | Fully Automated with Approval | Dependabot PRs + AI impact analysis; human merges |
| Test authoring | Manual | Assisted | AI suggests tests for uncovered paths; human validates |
| Refactoring | Manual | Assisted | AI proposes; human reviews blast radius |
| Bug diagnosis | Manual | Assisted | AI correlates signals; human confirms root cause |

**Evidence:** `.github/dependabot.yml`, `.golangci.yml`, `ci.yml` (lint + test + vet), 15 Go test files.

---

## Operations

| Capability | Current | Target | Rationale |
|-----------|---------|--------|-----------|
| Alert triage | Manual (on-call reads alert) | Partially Automated | AI summarizes + correlates; human acknowledges |
| Incident response | Assisted (runbooks, severity matrix) | Assisted | AI recommends runbooks and hypotheses; human executes |
| Failure injection | Fully Automated with Approval (`inject_failures.py` with `ALLOW_FAILURE_INJECTION` guard) | Fully Automated with Approval | Maintain; AI suggests experiment scenarios |
| Backup execution | Partially Automated (`scripts/backup/backup.py`) | Fully Automated with Approval | Scheduled; human approves restore |
| Restore verification | Partially Automated (`scripts/restore/verify_restore.py`) | Fully Automated with Approval | Automated verification; human reviews results |
| Price reconciliation | Partially Automated (`sre-toolkit/reconcile_prices.py`) | Fully Automated with Approval | Automated detection; human decides action |
| On-call handover | Manual | Assisted | AI generates shift summaries from alert/incident history |
| Capacity management | Manual | Assisted | AI forecasts from metrics; human approves changes |

**Evidence:** `docs/operations/on-call.md`, `docs/operations/severity-matrix.md`, `sre-toolkit/`, `scripts/backup/`, `scripts/restore/`.

---

## Infrastructure

| Capability | Current | Target | Rationale |
|-----------|---------|--------|-----------|
| Terraform changes | Partially Automated (plan workflow + drift detection) | Fully Automated with Approval | AI reviews plans; human approves apply |
| Kubernetes manifests | Assisted (Helm, CI validation) | Partially Automated | AI reviews manifests against conventions; human merges |
| IAM changes | Manual (module-based) | Intentionally Manual → Assisted | AI flags privilege escalation; human always decides |
| Network/security groups | Manual (WAF module) | Intentionally Manual → Assisted | AI review only; human decides |
| Cost management | Manual (`docs/cost/estimates.md`) | Assisted | AI identifies waste; human approves changes |
| Secrets rotation | Partially Automated (`secrets-rotation` module) | Fully Automated with Approval | Automated rotation; human approves policy changes |
| Cluster upgrades | Manual | Intentionally Manual | High blast radius; AI provides upgrade checklist only |

**Evidence:** `.github/workflows/terraform-plan.yml`, `terraform-drift.yml`, `deploy/terraform/modules/iam/`, `deploy/terraform/modules/waf/`, `deploy/terraform/modules/secrets-rotation/`.

---

## Documentation

| Capability | Current | Target | Rationale |
|-----------|---------|--------|-----------|
| Architecture docs | Manual | Assisted | AI detects drift between docs and manifests |
| ADR authoring | Manual (20 ADRs, consistent format) | Assisted | AI drafts from discussion; human owns decision |
| API documentation | Partially Automated (OpenAPI spec) | Fully Automated with Approval | Spec-driven generation; human reviews |
| Onboarding material | Manual (`docs/onboarding.md`) | Assisted | AI keeps current; human validates accuracy |
| Release notes | Partially Automated (release-please) | Fully Automated with Approval | Conventional commits → changelog; human edits |
| Runbook maintenance | Manual | Assisted | AI suggests updates after incidents; human approves |

**Evidence:** `api/openapi.yaml`, `docs/adr/`, `.release-please-config.json`, `docs/onboarding.md`, `docs/runbooks/`.

---

## Incident Management

| Capability | Current | Target | Rationale |
|-----------|---------|--------|-----------|
| Incident declaration | Intentionally Manual | Intentionally Manual | Human judgment per severity matrix |
| Alert correlation | Manual | Partially Automated | AI groups related alerts; human confirms |
| Timeline construction | Manual (postmortem 001 format) | Partially Automated | AI drafts from logs/metrics; human validates |
| Root cause analysis | Manual | Assisted | AI generates hypotheses with evidence; human confirms |
| Postmortem drafting | Manual | Assisted | AI drafts from timeline; human owns narrative |
| Stakeholder communication | Intentionally Manual | Intentionally Manual | Human judgment on messaging |
| Action item tracking | Manual | Assisted | AI extracts and tracks; human owns delivery |

**Evidence:** `docs/postmortems/001-primary-provider-rate-limit.md`, `docs/operations/severity-matrix.md`, `monitoring/alertmanager/alertmanager.yml`.

---

## Observability

| Capability | Current | Target | Rationale |
|-----------|---------|--------|-----------|
| Alert rule authoring | Manual (282-line alerts.yml) | Assisted | AI suggests rules from metric catalog; human approves |
| Dashboard creation | Manual (5 dashboards) | Assisted | AI proposes panels; human curates |
| Anomaly detection | Partially Automated (burn-rate alerts, freshness checks) | Partially Automated | Statistical detection exists; AI adds explanation |
| Alert tuning | Manual (`docs/sre/alert-tuning-review.md`) | Assisted | AI identifies noisy alerts from firing history |
| Metric onboarding | Manual | Assisted | AI validates naming, labels, dashboard coverage |
| Log analysis | Manual (Loki queries) | Assisted | AI summarizes error patterns; human investigates |

**Evidence:** `monitoring/prometheus/alerts.yml`, `monitoring/prometheus/recording-rules.yml`, `docs/sre/alert-tuning-review.md`, 5 Grafana dashboards.

---

## Developer Experience

| Capability | Current | Target | Rationale |
|-----------|---------|--------|-----------|
| Onboarding | Assisted (`docs/onboarding.md`, Makefile targets) | Assisted | AI Q&A over repo; docs remain source of truth |
| Local environment setup | Partially Automated (devcontainer, docker-compose, Makefile) | Fully Automated with Approval | Already strong; AI troubleshoots failures |
| Repository navigation | Manual | Assisted | AI semantic search over code and docs |
| Architecture Q&A | Manual (docs + ADRs) | Assisted | Retrieval-augmented answers citing ADRs |
| Debugging assistance | Manual | Assisted | AI correlates logs, metrics, traces |

**Evidence:** `.devcontainer/devcontainer.json`, `docker-compose.yml`, `Makefile` (30+ targets), `docs/onboarding.md`, `Tiltfile`.

---

## Knowledge Management

| Capability | Current | Target | Rationale |
|-----------|---------|--------|-----------|
| Decision records | Manual (20 ADRs) | Assisted | AI detects undocumented decisions |
| Cross-referencing | Manual (runbook links in alerts) | Partially Automated | AI maintains knowledge graph; human validates edges |
| Tribal knowledge capture | Manual | Assisted | AI extracts from PRs and incidents |
| Search | Manual (grep/IDE) | Assisted | Semantic retrieval over all artifacts |

**Evidence:** `docs/adr/`, runbook annotations in alerts.yml, `docs/architecture/overview.md`.

---

## Security

| Capability | Current | Target | Rationale |
|-----------|---------|--------|-----------|
| Dependency scanning | Fully Automated with Approval (security.yml, CodeQL, Gitleaks) | Fully Automated with Approval | Maintain current automation level |
| Container scanning | Fully Automated with Approval (Trivy in CI) | Fully Automated with Approval | Maintain |
| Threat model review | Intentionally Manual (`docs/security/threat-model.md`) | Intentionally Manual | Human judgment; AI assists with checklists |
| IAM review | Intentionally Manual | Assisted | AI flags changes; human always decides |
| Secrets detection | Fully Automated (Gitleaks, `.gitleaks.toml`) | Fully Automated with Approval | Maintain |
| Access control changes | Intentionally Manual | Intentionally Manual | Highest-risk area; no AI execution |

**Evidence:** `.github/workflows/security.yml`, `codeql.yml`, `.gitleaks.toml`, `docs/security/threat-model.md`, `docs/security/iam-least-privilege.md`.

---

## Release Engineering

| Capability | Current | Target | Rationale |
|-----------|---------|--------|-----------|
| Version management | Fully Automated (release-please, conventional commits) | Fully Automated with Approval | Maintain |
| Canary deployment | Fully Automated with Approval (5%→25%→50%→100% with gates) | Fully Automated with Approval | AI advises on canary health; automation proceeds per policy |
| SLO gating | Fully Automated (`check-slo-gate.sh` in deploy workflow) | Fully Automated with Approval | Maintain; override requires human approval |
| Rollback | Partially Automated (helm rollback) | Fully Automated with Approval | AI recommends; human triggers |
| Release notes | Fully Automated (release-please changelog) | Fully Automated with Approval | Maintain |

**Evidence:** `.github/workflows/deploy-production.yml`, `release-please.yml`, `scripts/check-slo-gate.sh`, `docs/deployment/canary-strategy.md`.

---

## Testing

| Capability | Current | Target | Rationale |
|-----------|---------|--------|-----------|
| Unit testing | Fully Automated (CI runs on PR) | Fully Automated with Approval | Maintain |
| Integration testing | Partially Automated (requires Docker) | Fully Automated with Approval | Run in CI with services |
| E2E testing | Partially Automated (Playwright, requires stack) | Fully Automated with Approval | Preview environments enable this |
| Load testing | Manual trigger (`load-tests/` k6 scripts) | Partially Automated | AI analyzes results; human decides thresholds |
| Chaos testing | Fully Automated with Approval (`chaos-staging.yml`, safeguards per ADR-014) | Fully Automated with Approval | Maintain safeguards |
| Test gap analysis | Manual | Assisted | AI identifies uncovered paths |

**Evidence:** `.github/workflows/ci.yml`, `chaos-staging.yml`, `performance-gate.yml`, `e2e/`, `load-tests/`.

---

## Architecture

| Capability | Current | Target | Rationale |
|-----------|---------|--------|-----------|
| Architecture evolution | Intentionally Manual (ADR process) | Intentionally Manual | Human owns decisions; AI provides analysis |
| Consistency checking | Manual | Partially Automated | AI verifies code against ADR constraints |
| Technical debt tracking | Manual | Assisted | AI identifies debt signals; human prioritizes |
| Multi-region planning | Manual (`docs/multi-region/` 6 documents) | Assisted | AI models scenarios; human decides |

**Evidence:** `docs/adr/`, `docs/architecture/`, `docs/multi-region/`.

---

## Summary

| Maturity Level | Count (Current) | Count (Target) |
|---------------|-----------------|----------------|
| Manual | 18 | 0 |
| Assisted | 16 | 30 |
| Partially Automated | 14 | 14 |
| Fully Automated with Approval | 12 | 16 |
| Intentionally Manual | 8 | 8 |

**Key principle:** No capability moves to "Fully Automated" without an approval gate. Capabilities marked "Intentionally Manual" (incident declaration, IAM changes, cluster upgrades, threat model, stakeholder communication, architecture decisions) remain human-owned regardless of AI capability improvements.
