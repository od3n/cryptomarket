# Human-in-the-Loop Model

## Purpose

Define precisely which activities require human decision, human approval, human notification, or involve AI recommendation only. This model is the operational expression of the program's guiding principle: AI augments engineers — it never replaces operational decision-making.

**Absolute rule:** No action that could cause a production outage is automated without explicit human authorization. This is non-negotiable regardless of measured AI reliability.

---

## Interaction Tiers

| Tier | Definition | AI Role | Human Role |
|------|-----------|---------|-----------|
| **T1: Human Decision** | Human makes the decision with or without AI information | May provide data; no recommendation | Decides, acts, owns outcome |
| **T2: Human Approval** | AI prepares a recommendation; human must explicitly approve before any effect | Recommends with evidence | Reviews, approves/rejects, owns outcome |
| **T3: Human Notification** | AI acts within pre-approved bounds; human is notified and can veto | Executes within guardrails; reports | Monitors; vetoes if needed |
| **T4: AI Recommendation Only** | AI output is purely informational; no action pathway exists | Informs | Consumes at discretion |

---

## Activity Classification

### T1: Human Decision (AI provides data, no recommendation)

| Activity | Rationale | AI Support Allowed |
|----------|-----------|-------------------|
| Incident severity declaration | Business judgment per `docs/operations/severity-matrix.md` | Surface criteria document; never suggest severity |
| Incident declaration/closure | Operational authority; legal and communication implications | Timeline drafts, impact metrics |
| Architecture decisions (ADRs) | Engineering judgment; long-term consequences | Constraint checking, related ADR retrieval |
| SLO target changes | Business-reliability trade-off | Budget consumption data, forecasting |
| Error budget policy exceptions | Policy governance | Policy text, budget state |
| Stakeholder communication during incidents | Messaging judgment | Impact summary data (internal only) |
| Hiring/team decisions | Out of scope for operational AI | None |
| Postmortem lessons learned | Human reflection; cultural artifact | Draft suggestions marked as AI-generated |

### T2: Human Approval (AI recommends; human gates)

| Activity | AI Output | Approval Mechanism | Approver |
|----------|-----------|-------------------|----------|
| Production deployment | PROCEED/DELAY/INVESTIGATE/ROLLBACK recommendation | Workflow environment protection (existing) | Deployer via workflow_dispatch |
| SLO gate override | Budget analysis + risk statement | `SLO_GATE_OVERRIDE` with audit documentation | Platform lead |
| Terraform apply | Plan review findings | Manual apply step (existing workflow design) | Infrastructure reviewer |
| IAM policy changes | Least-privilege assessment | PR review + security approval | Security reviewer |
| Rollback execution | Rollback recommendation with evidence | Human triggers `helm rollback` | On-call / deployer |
| Alert rule changes | Suggested rule with rationale | PR review | SRE reviewer |
| Runbook changes | Suggested updates from incident learnings | PR review | Runbook owner |
| Documentation publication | Generated content | PR review | Doc reviewer |
| Failure injection experiments | Suggested scenarios | `ALLOW_FAILURE_INJECTION` guard + human trigger (ADR-014 pattern) | SRE engineer |
| Cost optimization changes | Savings estimate + risk assessment | Terraform PR workflow | Infrastructure reviewer |
| Assistant activation (shadow → active) | Eval results summary | Governance sign-off | Platform lead + SRE lead |
| Model changes | Comparison eval results | Governance sign-off | Platform lead + security |

### T3: Human Notification (AI acts within bounds; veto available)

| Activity | Pre-Approved Bounds | Notification Channel | Veto Mechanism |
|----------|--------------------|---------------------|----------------|
| Incident summary generation | Read-only queries; advisory output | On-call channel | Disable assistant |
| Alert correlation grouping | Informational grouping only | Incident channel | Uncorrelate manually |
| PR summary posting | Informational; from public diff | PR comment | Remove comment |
| Weekly drift report | Read-only analysis | Engineering channel | Suppress report |
| Eval suite execution | Test environment only | CI results | N/A (no production effect) |
| Knowledge graph refresh | Derived from committed artifacts | Changelog notification | Revert index |

**T3 constraints:**
- Every T3 activity must have zero production write access (verified by access audit)
- Notification must include what was done and how to veto
- T3 activities that gain any recommendation capability are reclassified to T2/T4 review

### T4: AI Recommendation Only (informational, no action path)

| Activity | Output | Why Not Higher |
|----------|--------|---------------|
| Root cause hypotheses | Ranked hypotheses with evidence | Investigation guidance; human confirms |
| SLO forecasting | Budget projections with confidence | Planning input; no automatic action |
| Cost observations | Utilization analysis | Context for human decisions |
| Anomaly explanations | Statistical vs. incident assessment | Triage aid; human validates |
| Test gap analysis | Uncovered path suggestions | Author prioritizes work |
| Onboarding Q&A | Citation-backed answers | Informational; docs are source of truth |
| Architecture Q&A | ADR-grounded answers | Informational; ADRs are source of truth |
| Noisy alert identification | Firing frequency analysis | Tuning input; human changes rules (T2) |

---

## Tier Assignment Rules

### Classification Decision Tree

```
Does the activity write to production or change production behavior?
├── YES → Does it require judgment about business impact?
│   ├── YES → T1 (Human Decision)
│   └── NO → T2 (Human Approval)
└── NO → Does it produce recommendations that others might act on?
    ├── YES → T4 (Recommendation Only) unless acting on it requires approval (then T2)
    └── NO → T3 (Notification) if automated, T4 if passive
```

### Escalation Rules

| Trigger | Action |
|---------|--------|
| T3 activity produces output that humans start treating as decisions | Reclassify to T2; add approval gate |
| T4 recommendation is acted on repeatedly without review | Add explicit acknowledgment step; track acceptance |
| Any tier's AI output is involved in an incident | Freeze at current tier; governance review before any upgrade |
| Team requests tier upgrade (more automation) | Requires: 90 days clean operation + eval evidence + governance approval |

### Downgrade Rules (fast path to more human control)

| Trigger | Action |
|---------|--------|
| Safety violation detected | Immediate downgrade to T4 or disable |
| Precision drops below target for 2 consecutive weeks | Downgrade one tier; investigate |
| Operator feedback: "AI output is slowing me down" | Downgrade or disable; optimize before restore |
| Any engineer requests downgrade during incident | Immediate compliance; review within 48h |

---

## Existing Controls Preserved

This model layers onto — never replaces — existing operational controls:

| Existing Control | Maintained By | AI Relationship |
|-----------------|--------------|-----------------|
| SLO deployment gate (`check-slo-gate.sh`) | `deploy-production.yml` | AI advises; gate blocks independently |
| Canary stages (5%→25%→50%→100%) | `deploy-production.yml` | AI assesses stage health; stages proceed per policy |
| Failure injection safeguard (`ALLOW_FAILURE_INJECTION`) | ADR-014 pattern | AI suggests experiments; guard remains |
| Branch protection + CODEOWNERS | GitHub settings | AI comments; humans approve |
| Environment protection rules | GitHub environments | Unchanged; AI cannot satisfy required reviewers |
| Secrets access controls | IAM, Secrets Manager | AI has zero secrets access |
| On-call authority structure | `docs/operations/on-call.md` | AI supports on-call; never substitutes |

---

## Compliance Verification

| Check | Frequency | Method |
|-------|-----------|--------|
| No T2 activity executed without approval record | Weekly | Audit log scan: every T2 interaction has human_decision field |
| No T3 activity has write access | Quarterly | Access audit of assistant credentials |
| Tier assignments match actual behavior | Quarterly | Governance review with interaction sampling |
| Kill switches functional | Quarterly | Staging exercise: disable all assistants; verify zero impact |
| No tier upgrades without 90-day clean record | Per upgrade request | Governance checklist |

**Audit finding severity:**
- T2 activity without approval record → SEV-2 equivalent (process violation)
- T3 activity with unexpected write access → SEV-1 equivalent (safety violation)
- Missing audit records → assistant suspended until resolved
