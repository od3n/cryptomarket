# Recommended First Implementation Work Package

## WP-1: Incident Assistant Core with Governance and Evaluation Foundation

---

## Rationale

### Why this package

The program execution rule requires: assess → prioritize highest-value use case → implement one capability → evaluate → document → update governance → repeat. This package follows that sequence exactly.

**Assessment findings that drive this choice:**

1. **Readiness** ([01-readiness-assessment.md](01-readiness-assessment.md)): Overall 3.9/5. Strongest areas — CI/CD (5), ADRs (5), runbooks (4), observability (4) — are precisely the inputs an incident assistant needs. Weakest area — incident history (2) — is addressed by deriving scenarios from the failure injection toolkit and postmortem 001.

2. **Use case priority** ([03-use-case-portfolio.md](03-use-case-portfolio.md)): O-2 (alert correlation) and O-1 (incident summaries) rank #1 and #4 by value/risk. They share infrastructure, and O-2 is the foundation for all later incident capabilities (O-3 RCA, timeline drafting).

3. **Evidence of value** (postmortem 001): 12 minutes from alert fire to on-call acknowledgment; 19 minutes to root cause identification. The causal chain (rate limit → circuit breaker → stale data) is documented and replayable. An assistant that correlates this chain and pre-fills runbook diagnostics directly compresses both gaps.

4. **Existing surface**: All 18 alerts already carry `runbook:` annotations. The knowledge graph's alert→runbook→service edges are derivable today from `alerts.yml` parsing. Failure injection (`inject_failures.py`) provides ground-truth scenarios. `make incident-demo` provides the demonstration path.

### Why not the alternatives first

| Alternative | Reason deferred |
|-------------|----------------|
| E-2 Code review (P1-ranked) | Lower operational urgency; requires ADR-constraint formalization first; benefits from evaluation infrastructure built here |
| I-1 Terraform review (P1-ranked) | Requires plan-output integration work in CI; value realized per-PR (lower frequency than alerts) |
| E-1 PR summaries | Quick win but low value density; better as a "second capability" to validate the pattern cheaply |

### Why governance and evaluation are inside this package (not separate)

Quality gates ([17-quality-gates.md](17-quality-gates.md)) require audit logging, kill switches, and eval scenarios before any assistant ships. Building AI-10 and AI-9 as separate prior phases would delay first value by 5+ weeks with no operational output. Instead, this package builds the *minimum* governance and evaluation needed to ship one assistant safely — then extends in subsequent packages.

---

## Scope

### In scope

| Component | Deliverable | Epic ref |
|-----------|-------------|----------|
| **Governance minimum** | Audit log (append-only, all interactions); feature flags (`ai.assistants.enabled`, `ai.incident-assistant.enabled`); kill switch (`make ai-disable`); read-only credential scoping | AI-10 (subset) |
| **Evaluation minimum** | Scenario format + 4 scenarios (S-1 rate-limit cascade, S-2 redis isolated, S-3 healthy deploy baseline, S-4 budget-constrained); pytest-compatible runner; CI gate on assistant changes | AI-9 (subset) |
| **Knowledge graph minimum** | Parser for `alerts.yml` → alert entities + runbook edges + service labels; causal chain definitions (2 chains: rate-limit cascade, redis failure); JSON output | AI-2 dependency |
| **Incident assistant core** | Alert summarization; alert correlation (graph-based); runbook recommendation with pre-filled diagnostics; recent deployment identification (GitHub releases) | AI-2 (core) |
| **Prompts** | System base prompt (safety constraints per [12-prompt-standards.md](12-prompt-standards.md)); 3 task templates (summarize, correlate, recommend) with versioning | AI-2 |
| **Integration** | Alertmanager webhook receiver; output to on-call channel (markdown) + CLI query mode | AI-2 |
| **Documentation** | Operational runbook for the assistant itself; limitations doc; demo script (Demo 1 from [18-demonstration-plan.md](18-demonstration-plan.md)) | — |

### Out of scope (explicitly deferred)

- Root cause hypotheses (AI-7 — requires larger incident corpus)
- Timeline/postmortem drafting (AI-7)
- Semantic retrieval / embedding index (AI-8 — structured lookup suffices for v1; semantic search is a later enhancement)
- Deployment advisor, code review, documentation assistant (separate packages)
- Grafana dashboard for AI quality (phase 2; v1 uses CLI report)
- Human feedback collection UI (v1: manual logging)

---

## Work Breakdown

### Week 1-2: Foundation

| Task | Output | Verification |
|------|--------|-------------|
| 1.1 Audit log implementation | Append-only interaction log in `sre-toolkit/ai/audit/`; schema per [13-governance.md](13-governance.md) | Test: interaction produces complete record; log failure suspends assistant (fail-closed test) |
| 1.2 Feature flags + kill switch | Flags via `internal/featureflags` pattern (or sre-toolkit config); `make ai-disable` target | Timed exercise: full disable <1 min; platform unaffected |
| 1.3 Alert graph parser | Parse `monitoring/prometheus/alerts.yml` → entities + edges (JSON); causal chain definitions from postmortem 001 + runbooks | Test: all 18 alerts parsed; all runbook links validated against file existence |
| 1.4 Scenario format + S-1, S-2 | YAML scenarios per [11-evaluation-framework.md](11-evaluation-framework.md) format in `sre-toolkit/ai/eval/scenarios/` | Schema validation; ground truth verified against postmortem 001 and alerts.yml |

### Week 3-4: Assistant Core

| Task | Output | Verification |
|------|--------|-------------|
| 2.1 Alert summarizer | Webhook receiver → summary (alert meaning, SLO impact, runbook link, current metric state) | S-1/S-2 summary sections pass; citation compliance 100% |
| 2.2 Alert correlator | Graph-based correlation: match firing set against causal chains; temporal ordering check; metric evidence verification | S-1: 3 alerts grouped correctly; S-2: redis pair grouped, NOT merged with provider chain |
| 2.3 Runbook recommender | Recommend runbook from graph edges; pre-fill diagnostic results from live metrics; mark steps CONFIRMED/PENDING | Recommendation matches alert annotation in >95% of test cases |
| 2.4 Deployment identifier | GitHub API: releases + workflow runs in 24h window; changed-file → service mapping | Test with synthetic deployment history |
| 2.5 Prompt templates | System base + 3 task templates, versioned per standards | Adversarial test cases pass (injection, authority escalation) |

### Week 5-6: Integration and Shadow

| Task | Output | Verification |
|------|--------|-------------|
| 3.1 Alertmanager webhook integration | Webhook config in `monitoring/alertmanager/alertmanager.yml` (additive, removable); assistant gateway service | Local stack test: injected failure → summary delivered <30s |
| 3.2 CLI query mode | `sre-toolkit/ai/query.py --alert <name>` for on-demand assessment | Smoke test on running stack |
| 3.3 Scenarios S-3, S-4 + full suite | Complete 4-scenario suite; CI integration (pytest) | CI blocks failing assistant changes; 3 consecutive reproducible runs |
| 3.4 Shadow deployment | Assistant runs against local/staging alerts; outputs logged, not delivered | 2-week shadow: zero safety violations; precision measured on all outputs |
| 3.5 Demo script | `make ai-demo` executing Demo 1 end-to-end | Demo passes all validation steps from [18-demonstration-plan.md](18-demonstration-plan.md) |
| 3.6 Operational documentation | Assistant runbook (troubleshooting, disable procedure, limitations); governance record (sign-off) | Review by SRE lead + platform lead |

---

## Acceptance Criteria (Package-Level)

| # | Criterion | Measurement |
|---|-----------|-------------|
| 1 | Alert summary delivered <30s from alert group firing | Gateway latency metric during demo |
| 2 | Correlation precision >90% on eval scenarios | Scenario suite results |
| 3 | Runbook recommendation correct in 100% of known-alert cases | Graph edge validation (ground truth: alerts.yml annotations) |
| 4 | Citation compliance 100% (every claim sourced) | Automated output validation |
| 5 | Kill switch disables assistant <1 min with zero platform impact | Timed exercise |
| 6 | Audit log complete for all interactions; fail-closed verified | Chaos test |
| 7 | All 4 eval scenarios pass in CI; reproducible across 3 runs | CI results |
| 8 | Shadow period (2 weeks) with zero safety violations | Shadow log review |
| 9 | Platform fully functional with assistant disabled | `make smoke` + `make smoke-realtime` pass with flags off |
| 10 | Demo 1 passes all validation steps | Demo execution record |

---

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Local stack alert timing differs from production thresholds | Medium | Demo flakiness | Demo script waits for alert state explicitly; documents timing assumptions |
| Single postmortem limits causal chain coverage | High | Correlation misses novel patterns | Assistant reports uncorrelated alerts separately (never forces grouping); S-2 tests independence |
| Model output variance affects reproducibility | Medium | Eval flakiness | Fixed temperature; gradeable-equivalence standard (not token equality); 3-run requirement |
| Webhook integration adds Alertmanager failure mode | Low | Alert delivery affected | Webhook is additive receiver (existing routes unchanged); assistant failure cannot block alerts |
| Scope creep toward RCA features | Medium | Timeline slip | Out-of-scope list enforced at review; AI-7 explicitly separate package |

---

## Dependencies and Prerequisites

| Dependency | Status | Action |
|-----------|--------|--------|
| Alertmanager running in stack | Available (`docker-compose.yml`) | None |
| Failure injection toolkit | Available (`sre-toolkit/inject_failures.py`) | None |
| GitHub API access (releases) | Requires token with `contents:read` | Create scoped token; audit scope |
| Model API access | Requires provider selection per [13-governance.md](13-governance.md) criteria | Select before Week 3; record decision |
| Python environment (sre-toolkit) | Available (`sre-toolkit/requirements.txt`) | Add AI dependencies to requirements |

---

## Success Definition and Next Steps

**This package succeeds when:** An on-call engineer, paged by `DataStaleCritical`, receives a summary within 30 seconds that correctly identifies the rate-limit cascade, links the right runbook with pre-filled diagnostics, and cites postmortem 001 as precedent — with every claim verifiable, every interaction audited, and a kill switch one command away.

**After this package (per program execution rule):**

1. **Evaluate**: 4-week production run → precision, acceptance, time-saved metrics
2. **Document**: Results report; scenario library update; lessons learned
3. **Update governance**: Refine thresholds based on measured baselines
4. **Repeat**: Next package = AI-3 (Deployment Advisor) or AI-1 (Repository Intelligence), based on measured results and team capacity — decision made with evidence from this package's operation.

---

## Estimated Effort

| Component | Effort |
|-----------|--------|
| Governance minimum (1.1-1.2) | 1 week |
| Knowledge graph minimum (1.3) | 0.5 week |
| Evaluation minimum (1.4, 3.3) | 1 week |
| Assistant core (2.1-2.5) | 2 weeks |
| Integration + shadow (3.1-3.6) | 1.5 weeks |
| **Total** | **6 weeks** (1 engineer) |

Shadow period (2 weeks) overlaps with integration work — calendar time: ~7 weeks.
