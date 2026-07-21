# Continuous Improvement Program

## Purpose

Define recurring engineering reviews that keep the platform — and its AI capabilities — healthy, current, and improving. Each review has an owner, cadence, inputs, outputs, and follow-up mechanism.

**Principle:** Reviews produce actions, not reports. Every review outputs tracked action items with owners and deadlines.

---

## Weekly Reviews

### W-1: Dependency Review

| Attribute | Detail |
|-----------|--------|
| Owner | Rotating engineer |
| Duration | 30 minutes |
| Inputs | Dependabot PR queue; `go.mod`/`package-lock.json` change log; security advisories (GitHub Security tab); CodeQL/gitleaks findings from `security.yml` |
| Checklist | 1. Triage all open Dependabot PRs (merge, defer with reason, or reject) 2. Review new security advisories affecting dependencies 3. Check for deprecated dependencies 4. Verify CI security workflows ran clean this week |
| Outputs | Dependabot PRs actioned; deferrals documented with reasons and review dates |
| Escalation | Critical CVE → immediate patch PR + security reviewer ping |
| AI support | Dependency change summaries; CVE impact assessment against usage (T4 recommendation) |

### W-2: AI Recommendation Review

| Attribute | Detail |
|-----------|--------|
| Owner | Assistant owner (rotating per active assistant) |
| Duration | 30 minutes |
| Inputs | Weekly AI quality report (automated); operator feedback log; rejected recommendation details; audit log sample |
| Checklist | 1. Review acceptance rate trend (alert if >10% WoW drop) 2. Validate 20-item precision sample 3. Analyze all rejected recommendations: wrong, unconvincing, or context gap? 4. Check citation compliance (must be 100%) 5. Review any safety flags from automated scan |
| Outputs | Precision score recorded; rejected-rec patterns documented; prompt improvement tickets created |
| Escalation | Precision <85% or safety violation → governance review trigger |
| AI support | Automated quality report generation; pattern clustering of rejections |

### W-3: Error Budget Review

| Attribute | Detail |
|-----------|--------|
| Owner | SRE lead (delegable) |
| Duration | 15 minutes |
| Inputs | SLO dashboard (`slo-error-budget.json`); budget consumption rate; burn-rate alert history |
| Checklist | 1. Budget remaining for all 5 SLOs 2. Consumption trend vs. prior week 3. Any burn-rate alerts fired → verify resolution 4. Budget zone check per policy (>50% normal, 25-50% caution, <25% freeze) |
| Outputs | Budget state recorded; policy zone actions triggered if applicable |
| Escalation | Budget <25% → feature freeze communication per `docs/runbooks/error-budget-burn.md` |
| AI support | Budget forecasting; top-consumer attribution (T4) |

---

## Monthly Reviews

### M-1: Architecture Review

| Attribute | Detail |
|-----------|--------|
| Owner | Platform lead |
| Duration | 60 minutes |
| Inputs | ADR gap report (Documentation Assistant); significant PRs merged; new packages/modules added; technical debt signals |
| Checklist | 1. Review ADR gap suggestions — author ADRs or dismiss with reason 2. Assess architectural drift: does implementation match `docs/architecture/`? 3. Review any cross-service changes for boundary violations 4. Evaluate emerging patterns that need standardization 5. Multi-region plan progress check (`docs/multi-region/program-plan.md`) |
| Outputs | New ADR tickets (or dismissals recorded); architecture action items |
| Escalation | Undocumented significant decision discovered → ADR required within 2 weeks |
| AI support | ADR gap detection; consistency checking against ADR constraints (T4) |

### M-2: Performance Review

| Attribute | Detail |
|-----------|--------|
| Owner | Rotating engineer |
| Duration | 45 minutes |
| Inputs | `performance-engineering.json` dashboard; p99 latency trends; load test results if run; resource utilization |
| Checklist | 1. API p99 vs. 300ms SLO threshold — trend direction 2. Provider latency patterns (degradation signals) 3. Redis memory growth trend 4. PostgreSQL query performance (slow queries) 5. Frontend Core Web Vitals if measured 6. Compare against last month; investigate >10% regressions |
| Outputs | Regression tickets with evidence; performance budget status |
| Escalation | Sustained regression >20% → dedicated performance investigation |
| AI support | Trend analysis; regression detection with change correlation (T4) |

### M-3: Documentation Review

| Attribute | Detail |
|-----------|--------|
| Owner | Rotating engineer |
| Duration | 30 minutes |
| Inputs | Weekly drift reports (aggregated); doc freshness metrics; onboarding feedback from new members |
| Checklist | 1. Resolve all confirmed drift findings 2. Verify docs touched by recent changes are current 3. Review doc usage signals: are runbooks being accessed during incidents? 4. Update stale diagrams 5. Check onboarding doc against current developer experience |
| Outputs | Drift findings resolved; freshness metrics updated |
| Escalation | Critical doc (runbook, SLO) stale >30 days → owner assigned immediately |
| AI support | Drift detection; freshness tracking; update suggestions (T3 notification + T2 for publication) |

### M-4: Alert and Dashboard Review

| Attribute | Detail |
|-----------|--------|
| Owner | SRE lead (delegable) |
| Duration | 30 minutes |
| Inputs | Alert firing history; `docs/sre/alert-tuning-review.md` format; dashboard usage |
| Checklist | 1. Alerts fired >3 times without action → tune or remove 2. Alerts that never fire → verify thresholds still valid 3. Dashboard panels not viewed → consider removal 4. Missing coverage: incidents/alerts without dashboard panels 5. Runbook coverage: all alerts still have valid runbook links |
| Outputs | Tuning PRs; coverage gap tickets; tuning review doc updated |
| Escalation | Alert fired during incident without runbook → runbook required within 1 week |
| AI support | Noise identification; coverage gap detection (T4) |

---

## Quarterly Reviews

### Q-1: Disaster Recovery Exercise

| Attribute | Detail |
|-----------|--------|
| Owner | SRE lead |
| Duration | Half-day exercise |
| Inputs | `docs/dr/strategy.md`, `docs/dr/recovery-procedures.md`, `docs/dr/tabletop-simulations.md`, `docs/testing/game-day-template.md` |
| Checklist | 1. Execute restore verification (`scripts/restore/verify_restore.py`) 2. Run tabletop simulation (rotate scenario) 3. Validate RTO/RPO against `docs/sre/recovery-objectives.md` 4. Test failover procedures 5. Verify backup integrity (`scripts/backup/backup.py` outputs) |
| Outputs | Exercise report; procedure updates; gap tickets |
| Escalation | RTO/RPO miss → remediation plan within 2 weeks |
| AI support | Scenario suggestions from incident history; timeline documentation (T4) |

### Q-2: AI Evaluation (Governance Review)

| Attribute | Detail |
|-----------|--------|
| Owner | Platform lead (chair) + SRE lead + security |
| Duration | 90 minutes |
| Inputs | Full governance checklist from `docs/ai/13-governance.md`; quarterly eval metrics; incident cross-references; cost reports; model currency |
| Checklist | 1. Complete governance checklist (all items) 2. Review eval metrics vs. targets for all assistants 3. Scenario library update (new incidents → new scenarios) 4. Model/provider review: deprecations, new capabilities, cost 5. Tier compliance audit (HITL model) 6. Kill switch exercise results 7. Prompt review cadence compliance |
| Outputs | Governance report; remediation actions; tier changes if warranted; program direction decisions |
| Escalation | Any compliance failure → assistant frozen until resolved |
| AI support | Automated compliance evidence collection (T3) |

### Q-3: Threat Model Review

| Attribute | Detail |
|-----------|--------|
| Owner | Security reviewer |
| Duration | 90 minutes |
| Inputs | `docs/security/threat-model.md`; changes since last review (new services, endpoints, dependencies, AI surface); incident learnings |
| Checklist | 1. Update threat model for architectural changes 2. Review AI-specific surface: prompt injection vectors, data exposure, audit completeness 3. Verify secrets handling unchanged (zero AI access) 4. Review IAM changes from quarter 5. Container security posture (`docs/security/container-security.md`) 6. Supply chain verification (ADR-017 compliance) |
| Outputs | Updated threat model; new mitigation tickets |
| Escalation | New critical threat → mitigation plan within 2 weeks |
| AI support | Change surface enumeration; checklist pre-fill (T4) |

### Q-4: Cost Optimization Review

| Attribute | Detail |
|-----------|--------|
| Owner | Platform lead |
| Duration | 60 minutes |
| Inputs | Cloud cost reports; `docs/cost/estimates.md` baseline; utilization data; AI cost (model API spend) |
| Checklist | 1. Actual vs. estimated cost variance (>15% requires explanation) 2. Utilization review: oversized instances, idle resources 3. Storage growth: PostgreSQL, S3, logs retention 4. AI spend vs. budget; cost per interaction trend 5. Reserved instance / savings plan evaluation 6. Update cost estimates doc if baseline shifted |
| Outputs | Optimization tickets with savings estimates; updated baseline |
| Escalation | Cost overrun >25% → immediate optimization sprint |
| AI support | Utilization analysis; savings identification (T4; changes are T2) |

---

## Annual Reviews

### A-1: AI Program Review

| Attribute | Detail |
|-----------|--------|
| Owner | Platform lead + engineering leadership |
| Inputs | Full year of eval metrics; operator surveys; toil measurements; cost/benefit analysis |
| Questions | 1. Did AI assistance reduce operational toil measurably? 2. Which assistants delivered value? Which should be retired? 3. Are safety guarantees holding (audit evidence)? 4. What should next year's priorities be? 5. Is the governance model right-sized? |
| Outputs | Program direction paper; backlog re-prioritization; governance updates |

### A-2: SLO and Reliability Target Review

| Attribute | Detail |
|-----------|--------|
| Owner | SRE lead + product |
| Inputs | 12 months of SLO attainment; user impact data; business requirements |
| Outputs | Updated SLO targets if warranted (ADR process); error budget policy adjustments |

---

## Review Operations

### Scheduling

- All reviews on shared calendar with standing invites
- Owner rotation schedule published quarterly
- Reviews occur even if owner must delegate (never silently skipped)

### Action Tracking

| Rule | Detail |
|------|--------|
| Every review produces actions | Even "no changes needed" is recorded as outcome |
| Actions have owners and deadlines | Tracked in engineering backlog |
| Overdue actions escalate | 1 week overdue → platform lead notified |
| Actions from previous review checked first | Standing agenda item |

### Review Health Metrics

| Metric | Target |
|--------|--------|
| Reviews held on schedule | >95% |
| Actions closed by deadline | >80% |
| Repeat findings (same issue across reviews) | <10% (indicates blocked action) |
| Time-to-resolution for review findings | <30 days median |
