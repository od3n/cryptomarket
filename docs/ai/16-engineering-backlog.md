# AI Engineering Backlog

## Purpose

Define the implementation backlog for AI-assisted capabilities as 10 epics (AI-1 through AI-10). Each epic includes objective, scope, dependencies, risks, acceptance criteria, and validation approach.

**Prioritization basis:** Use case portfolio rankings ([03-use-case-portfolio.md](03-use-case-portfolio.md)), readiness assessment scores ([01-readiness-assessment.md](01-readiness-assessment.md)), and the program execution rule: implement one capability, evaluate, document, then proceed.

---

## Priority Order

```
AI-10 (Governance foundation) → AI-9 (Evaluation) → AI-2 (Incident Assistant)
→ AI-3 (Deployment Advisor) → AI-1 (Repository Intelligence) → AI-8 (Retrieval)
→ AI-4 (Documentation) → AI-6 (Cost Advisor) → AI-5 (Capacity Planner) → AI-7 (full Incident depth)
```

Rationale: Governance and evaluation infrastructure must exist before any assistant ships (program principle: independently testable, reviewable, removable).

---

## AI-1: Repository Intelligence

| Attribute | Detail |
|-----------|--------|
| **Objective** | Enable semantic understanding of the repository: PR summaries, code review assistance, architecture consistency checking |
| **Scope** | PR summary generation from diffs; ADR-constraint checking on PRs; test completeness flagging; observability coverage checking (metrics/logs on new paths) |
| **Out of scope** | Automated code fixes; approval authority; style enforcement (golangci-lint owns that) |
| **Dependencies** | AI-10 (governance: audit logging for PR interactions); AI-9 (eval scenarios for review quality) |
| **Risks** | Noisy comments erode trust (mitigation: precision target >85%, dismissal tracking); stale ADR interpretation (mitigation: ADR status tracking) |
| **Acceptance criteria** | 1. PR summary posted within 2 min of PR open 2. >85% of review comments accepted or resolved without pushback over 20-PR sample 3. Zero comments contradicting active ADRs 4. All comments carry confidence + evidence 5. Disable switch verified (PRs function identically without assistant) |
| **Validation** | 4-week shadow period comparing AI comments to human review findings; precision/recall measurement against human-identified issues |
| **Use cases covered** | E-1, E-2, E-4 |
| **Estimated effort** | 3-4 weeks |

---

## AI-2: Operational Copilot (Incident Assistant — Core)

| Attribute | Detail |
|-----------|--------|
| **Objective** | Reduce incident time-to-understand: alert summarization, correlation, runbook recommendation |
| **Scope** | Alertmanager webhook integration; alert summary generation; causal-chain correlation using knowledge graph; runbook recommendation with pre-filled diagnostics; recent deployment identification |
| **Out of scope** | Root cause hypotheses (AI-7); timeline/postmortem drafting (AI-7); any remediation execution |
| **Dependencies** | AI-10 (governance); AI-9 (eval scenarios S-1, S-2); AI-8 (retrieval: runbook + postmortem indexing); knowledge graph (alert-runbook-service edges from [05-knowledge-model.md](05-knowledge-model.md)) |
| **Risks** | Wrong correlation misdirects investigation (mitigation: confidence levels, uncorrelated alerts reported separately); alert fatigue amplification (mitigation: summaries only on grouped incidents, not per-alert) |
| **Acceptance criteria** | 1. Summary delivered <30s from alert group firing 2. Correlation precision >90% on eval scenarios 3. Runbook recommendation correct in >95% of scenarios (ground truth: alert annotations) 4. 100% citation compliance 5. On-call can disable with single command 6. Zero impact on alerting when disabled |
| **Validation** | Scenario replay (S-1, S-2 + 3 new scenarios from failure injection); 2-week shadow during live alerts; on-call feedback survey |
| **Use cases covered** | O-1, O-2, O-4 |
| **Estimated effort** | 4-6 weeks |

---

## AI-3: Deployment Advisor

| Attribute | Detail |
|-----------|--------|
| **Objective** | Improve deployment safety decisions: pre-promotion risk assessment with PROCEED/DELAY/INVESTIGATE/ROLLBACK recommendations |
| **Scope** | Error rate and latency baseline comparison; SLO budget state analysis (beyond binary gate); canary stage health assessment; rollback history check; migration risk flagging; dependency change analysis; drift status integration |
| **Out of scope** | Blocking deployments (advisory only); canary automation changes; rollback execution |
| **Dependencies** | AI-10 (governance); AI-9 (eval scenarios S-3, S-4); Prometheus query access (read-only); GitHub API access (workflow history) |
| **Risks** | False DELAY recommendations slow delivery (mitigation: <10% false alarm target, tracked); over-trust in PROCEED (mitigation: explicit "does not replace SLO gate" in every output) |
| **Acceptance criteria** | 1. Assessment completes <60s 2. Zero missed deployment-caused SEV-1/2 with PROCEED recommendation (rolling quarter) 3. >90% of ROLLBACK/INVESTIGATE recommendations later confirmed valid 4. Every factor has evidence citation 5. Advisory-only: verified cannot block workflow 6. Backtest against 10 historical deployments produces sane results |
| **Validation** | Backtesting on historical deployment windows; 4-week shadow mode; comparison with human deployment decisions |
| **Use cases covered** | Deployment safety (Deliverable 7) |
| **Estimated effort** | 3-4 weeks |

---

## AI-4: Documentation Assistant

| Attribute | Detail |
|-----------|--------|
| **Objective** | Eliminate documentation drift: automated detection, ADR gap identification, generated reference docs |
| **Scope** | Drift detection (9 check categories from [09-documentation-assistant.md](09-documentation-assistant.md)); ADR gap suggestions; release summaries; onboarding doc validation |
| **Out of scope** | Direct doc edits (PR-only); ADR authorship (human-owned); architecture diagram generation (phase 2) |
| **Dependencies** | AI-10 (governance); AI-9 (eval scenario S-6); AI-8 (retrieval: doc indexing with section-level chunking) |
| **Risks** | False drift reports waste reviewer time (mitigation: >90% precision target, high-confidence-only reporting); change fatigue from weekly reports (mitigation: batch + deduplicate) |
| **Acceptance criteria** | 1. >90% of reported drift confirmed real 2. Known planted drift detected within 1 week 3. All output via PRs/issues (no direct pushes) 4. Weekly report processing time <10 min for reviewer 5. ADR gap suggestions include relevant context (related ADRs, code refs) |
| **Validation** | Drift injection exercise (plant 3 known drifts); 4-week production run with human validation of all findings |
| **Use cases covered** | E-3, D-3 (partial), Deliverable 13 |
| **Estimated effort** | 3 weeks |

---

## AI-5: Capacity Planner

| Attribute | Detail |
|-----------|--------|
| **Objective** | Forecast resource needs from historical metrics with confidence intervals |
| **Scope** | CPU/memory trend forecasting per service; Redis memory and stream growth projection; PostgreSQL storage growth; deployment frequency tracking; cost projection integration |
| **Out of scope** | Automatic scaling changes; infrastructure modifications (recommendations only) |
| **Dependencies** | AI-9 (evaluation: forecast accuracy measurement); metrics history retention >90 days; AI-6 (cost baseline data) |
| **Risks** | Forecast inaccuracy erodes trust (mitigation: confidence intervals always shown, accuracy tracked); low data volume for this platform's scale (mitigation: simple models, explicit assumption documentation) |
| **Acceptance criteria** | 1. Forecasts include confidence intervals and stated assumptions 2. 30-day forecasts within ±25% of actual (tracked quarterly) 3. Recommendations reference specific resources and thresholds 4. Quarterly capacity report generated with human review step |
| **Validation** | Backtest against available metrics history; comparison with naive baseline (linear extrapolation must be beaten or matched) |
| **Use cases covered** | I-4, Deliverable 11 |
| **Estimated effort** | 2-3 weeks |

---

## AI-6: Cost Advisor

| Attribute | Detail |
|-----------|--------|
| **Objective** | Identify cost waste with actionable recommendations including savings estimates and operational risk |
| **Scope** | Resource utilization vs. requests analysis; idle workload detection; storage growth waste; log volume analysis; autoscaling efficiency; AI spend tracking (model API costs) |
| **Out of scope** | Cost changes (Terraform PR workflow required); billing access without governance approval; reserved instance purchases |
| **Dependencies** | AI-10 (governance: data access review); CloudWatch/Cost Explorer read access; `docs/cost/estimates.md` baseline |
| **Risks** | Under-provisioning recommendations cause incidents (mitigation: risk assessment mandatory per recommendation; human approval required); stale pricing data (mitigation: baseline refresh quarterly) |
| **Acceptance criteria** | 1. Recommendations include estimated savings (±30% stated uncertainty) 2. Recommendations include operational impact assessment 3. >50% of recommendations adopted within 60 days 4. Zero recommendations that caused under-provisioning incidents 5. Monthly report cadence maintained |
| **Validation** | First report validated against known utilization data; adopted recommendations' actual savings tracked vs. estimates |
| **Use cases covered** | I-3, Deliverable 12 |
| **Estimated effort** | 2-3 weeks |

---

## AI-7: Incident Assistant (Full Depth — RCA and Documentation)

| Attribute | Detail |
|-----------|--------|
| **Objective** | Extend AI-2 with root cause hypotheses, timeline generation, and postmortem drafting |
| **Scope** | Multi-hypothesis RCA with evidence ranking; counter-evidence identification; timeline auto-drafting from metrics/logs/deployments; postmortem draft generation in established format |
| **Out of scope** | Cause assertion without evidence; postmortem publication; severity assignment |
| **Dependencies** | AI-2 (core incident assistant operational); AI-8 (retrieval: postmortem corpus indexed); Loki query access; 2+ postmortems in corpus (currently 1 — may need synthetic scenarios) |
| **Risks** | Anchoring bias from early hypotheses (mitigation: always multiple hypotheses, counter-evidence required); timeline gaps filled with speculation (mitigation: explicit gap markers) |
| **Acceptance criteria** | 1. Top-3 hypotheses contain actual cause in >80% of eval scenarios 2. Every hypothesis has evidence + counter-evidence sections 3. Timelines cite data source per event; gaps explicitly marked 4. Postmortem drafts match established format 5. Human validation required before any draft used in actual postmortem |
| **Validation** | Scenario replay with all 4 failure injection scenarios; blind evaluation: 3 engineers rate hypothesis usefulness without knowing AI-generated |
| **Use cases covered** | O-3, Deliverables 6 and 15 |
| **Estimated effort** | 4 weeks (after AI-2 stabilizes) |

---

## AI-8: Knowledge Retrieval

| Attribute | Detail |
|-----------|--------|
| **Objective** | Provide grounded context to all assistants via indexed retrieval with mandatory citation |
| **Scope** | Source parsers for all artifact types ([10-retrieval-strategy.md](10-retrieval-strategy.md)); chunking with provenance metadata; semantic + keyword hybrid search; citation attachment; secrets scrubbing; index freshness management |
| **Out of scope** | Live metric queries (direct API); knowledge graph construction (separate work); cross-repository indexing |
| **Dependencies** | AI-10 (governance: data classification review); AI-9 (retrieval quality eval) |
| **Risks** | Stale index serves outdated context (mitigation: freshness metadata, push-triggered re-index); secrets leak into index (mitigation: pre-index filtering + CI validation against gitleaks patterns) |
| **Acceptance criteria** | 1. Every retrieved chunk carries file path + section + line range + commit 2. Zero secrets in index (validated by automated scan) 3. Re-index completes <5 min on push to main 4. Retrieval relevance: correct source in top-3 for >90% of eval queries 5. Index failure degrades gracefully (structured lookup continues) |
| **Validation** | Retrieval eval set: 50 query-source pairs curated from real assistant needs; secrets scan on full index; chaos test (index unavailable → verify degradation) |
| **Use cases covered** | Deliverable 18; foundation for AI-2, AI-4, AI-7 |
| **Estimated effort** | 3-4 weeks |

---

## AI-9: Evaluation Framework

| Attribute | Detail |
|-----------|--------|
| **Objective** | Make assistant quality measurable and regression-testable before any assistant ships |
| **Scope** | Scenario format and library (initial: S-1 through S-6 from [11-evaluation-framework.md](11-evaluation-framework.md)); scenario runner (pytest-compatible); grading automation for REQUIRED items; results logging with version pinning; CI integration; quality report generation |
| **Out of scope** | Human feedback collection UI (phase 2); A/B testing infrastructure (phase 2) |
| **Dependencies** | AI-10 (governance defines targets); failure injection toolkit (scenario ground truth via `sre-toolkit/inject_failures.py`) |
| **Risks** | Scenarios don't represent reality (mitigation: scenarios derived from real postmortem + quarterly refresh); over-fitting to scenarios (mitigation: hold-out scenarios never shown during prompt development) |
| **Acceptance criteria** | 1. All 6 initial scenarios implemented and passing with reference implementation 2. Scenario run completes <5 min in CI 3. Results reproducible (same input → same grade, 3 consecutive runs) 4. New scenario creation documented (template + process) 5. CI blocks assistant changes that fail scenarios |
| **Validation** | Mutation testing: intentionally degrade a reference assistant → verify scenarios detect the regression |
| **Use cases covered** | Deliverable 16 |
| **Estimated effort** | 3 weeks |

---

## AI-10: Governance

| Attribute | Detail |
|-----------|--------|
| **Objective** | Establish governance infrastructure: audit logging, feature flags, access controls, compliance automation |
| **Scope** | Audit log implementation (append-only, all interactions); feature flag integration (`internal/featureflags` pattern); kill switch implementation (`make ai-disable`); access credential scoping (read-only per assistant); secrets scrubbing pipeline; compliance scan automation |
| **Out of scope** | Governance policy authoring (done: [13-governance.md](13-governance.md)); legal/compliance certification |
| **Dependencies** | None (foundation epic); `internal/featureflags/` existing package |
| **Risks** | Audit log becomes bottleneck (mitigation: async write, local buffer); flag proliferation (mitigation: single master + per-assistant flags only) |
| **Acceptance criteria** | 1. Every assistant interaction produces complete audit record 2. Audit log failure suspends assistant operation (fail-closed verified) 3. Kill switch disables all assistants <1 min (timed exercise) 4. All assistant credentials verified read-only (access audit) 5. Secrets scrubbing blocks credential patterns (test suite) 6. Platform fully functional with all assistants disabled |
| **Validation** | Chaos test: disable all assistants during staging load test → zero behavioral difference; audit completeness verified via interaction injection |
| **Use cases covered** | Deliverables 19, 23 (infrastructure portion) |
| **Estimated effort** | 2-3 weeks |

---

## Backlog Summary

| Epic | Priority | Effort | Depends On | Status |
|------|----------|--------|-----------|--------|
| AI-10 Governance | 1 | 2-3w | — | Ready |
| AI-9 Evaluation | 2 | 3w | AI-10 | Ready |
| AI-2 Operational Copilot | 3 | 4-6w | AI-10, AI-9, AI-8 | Ready |
| AI-8 Knowledge Retrieval | 4 | 3-4w | AI-10, AI-9 | Ready |
| AI-3 Deployment Advisor | 5 | 3-4w | AI-10, AI-9 | Ready |
| AI-1 Repository Intelligence | 6 | 3-4w | AI-10, AI-9 | Ready |
| AI-4 Documentation Assistant | 7 | 3w | AI-10, AI-9, AI-8 | Ready |
| AI-6 Cost Advisor | 8 | 2-3w | AI-10 | Ready |
| AI-5 Capacity Planner | 9 | 2-3w | AI-9, AI-6 | Ready |
| AI-7 Incident (Full Depth) | 10 | 4w | AI-2, AI-8 | Blocked (needs AI-2) |

**Total estimated effort:** 29-40 weeks of focused work (parallelizable across 2 engineers after foundations).

**Execution rule (from program spec):** Complete one epic → evaluate → document results → update governance → begin next. No parallel assistant development until AI-10 + AI-9 are stable.
