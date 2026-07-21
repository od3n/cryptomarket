# AI Quality Gates

## Purpose

Define the quality gates that every AI-assisted workflow must satisfy before shipping and during operation. These gates are the enforceable expression of the program's guiding principles: deterministic, observable, testable, explainable, secure, reversible.

**Rule:** A workflow that fails any gate does not ship. A workflow that fails a runtime gate is suspended until resolved.

---

## Gate Definitions

### Gate 1: Evidence-Backed Recommendations

> Every recommendation cites verifiable evidence.

| Check | Method | Pass Criteria |
|-------|--------|--------------|
| All factual claims have evidence citations | Automated output schema validation | 100% of claims carry `evidence[]` with source reference |
| Cited sources exist and contain claimed content | Citation validator (file existence + content match at cited commit) | 100% valid citations |
| Metric claims include query + timestamp + value | Schema validation | 100% compliance |
| Ungrounded claims explicitly marked | Output scan for unmarked claims without evidence | 0 unmarked ungrounded claims |

**Runtime enforcement:** Outputs failing citation validation are withheld and logged as gate failures.

---

### Gate 2: Explainable Output

> Every recommendation explains why, in terms a reviewer can verify.

| Check | Method | Pass Criteria |
|-------|--------|--------------|
| Recommendation includes reasoning chain | Schema: `reasoning` field required | 100% present |
| Reasoning references specific inputs (not generic) | Human sample review | >90% rated "verifiable" |
| Confidence level stated with justification | Schema: `confidence` + `confidence_basis` | 100% present |
| Alternative interpretations acknowledged where evidence conflicts | Schema: `alternatives` field when conflicts exist | Present when conflict detected |
| Output distinguishes facts from judgments | Format: observations vs. interpretations sections | Structural compliance |

**Anti-pattern (auto-fail):** "Based on my analysis..." without citing what was analyzed.

---

### Gate 3: Deterministic Inputs

> Given the same situation, the workflow produces gradeable-equivalent output.

| Check | Method | Pass Criteria |
|-------|--------|--------------|
| All inputs recorded (context chunks, queries, results) | Audit log completeness | 100% of interactions reproducible from log |
| Retrieval parameters pinned (top-k, threshold, index version) | Configuration audit | No floating parameters |
| Prompt version pinned (not latest-by-default) | Configuration audit | Explicit version reference |
| Temperature/sampling fixed per task | Model config audit | Deterministic settings documented |
| Repeated execution on recorded inputs produces equivalent grades | Reproducibility test (3 runs) | Same REQUIRED items pass |

**Note:** Exact token-level reproducibility is not required; gradeable equivalence (same facts, same recommendation, same citations) is the standard.

---

### Gate 4: Documented Limitations

> Every workflow states what it cannot do.

| Check | Method | Pass Criteria |
|-------|--------|--------------|
| Limitations section in workflow documentation | Doc review | Present and current |
| Output includes capability boundary reminder | Schema: footer disclaimer | 100% of outputs |
| Known failure modes documented | Doc review against eval results | Updated after each eval cycle |
| Degraded behavior documented (what happens when inputs missing) | Doc review + chaos test | Matches actual behavior |
| Human verification steps explicitly stated | Output schema: `verification` field | Present on action-relevant outputs |

**Required limitations disclosure (all assistants):**
- "AI-generated; verify against live systems"
- Data freshness boundary (index timestamp, query time)
- Scope boundary (what was NOT checked)

---

### Gate 5: Human Approval Points

> No production-affecting action without explicit human authorization.

| Check | Method | Pass Criteria |
|-------|--------|--------------|
| HITL tier assigned and documented | Review against [14-human-in-the-loop.md](14-human-in-the-loop.md) | Tier recorded per activity |
| T2 activities have approval records in audit log | Weekly audit scan | 100% have `human_decision` |
| No write credentials to production systems | Access audit | Zero write access verified |
| Kill switch exists and is tested | Quarterly exercise | Disable in <1 min, zero platform impact |
| Approval bypass impossible (no override flags without audit) | Code review + audit log | Overrides logged with identity |

**Auto-fail conditions:**
- Any assistant credential with production write access
- T2 interaction without approval record
- Kill switch failure during exercise

---

### Gate 6: Reproducible Evaluation

> Quality is measured by replayable scenarios, not vibes.

| Check | Method | Pass Criteria |
|-------|--------|--------------|
| Eval scenarios exist for the workflow | Scenario library audit | Minimum 3 scenarios per assistant task |
| Scenarios derived from real platform behavior | Scenario provenance field | Source documented (postmortem, injection test, production event) |
| Eval passes before ship | CI gate | All REQUIRED items pass |
| Eval results versioned and retained | Results log audit | Complete history available |
| Regression suite covers all assistants | CI configuration | Full suite on any assistant change |
| Scenario library refreshed quarterly | Governance review | New incidents → new scenarios within 2 weeks |

---

### Gate 7: Auditability

> Complete record of AI influence on operational decisions.

| Check | Method | Pass Criteria |
|-------|--------|--------------|
| Every interaction logged (input summary, output, versions, decision) | Audit log schema validation | 100% complete records |
| Audit log is append-only | Infrastructure verification | No edit/delete capability exists |
| Audit log failure suspends assistant (fail-closed) | Chaos test | Verified suspension behavior |
| Records retained 12 months | Retention policy verification | Oldest record ≥12 months available |
| Audit trail supports incident investigation | Exercise: reconstruct AI involvement in test incident | Complete reconstruction <30 min |
| Prompt/model versions traceable per interaction | Record schema: version fields | 100% populated |

---

## Gate Application by Lifecycle Stage

### Pre-Ship Gates (all must pass)

```
┌─────────────────────────────────────────────────────────┐
│  Gate 1: Evidence     ─── Eval scenario citation audit  │
│  Gate 2: Explainable  ─── Human review of sample output │
│  Gate 3: Determinism  ─── Reproducibility test (3 runs) │
│  Gate 4: Limitations  ─── Documentation review          │
│  Gate 5: HITL         ─── Access audit + tier assignment │
│  Gate 6: Evaluation   ─── Scenario suite pass in CI     │
│  Gate 7: Auditability ─── Audit log integration test    │
└─────────────────────────────────────────────────────────┘
                        ↓ ALL PASS
              Shadow period (min 1 week)
                        ↓ NO VIOLATIONS
              Governance sign-off → ACTIVE
```

### Runtime Gates (continuous)

| Gate | Runtime Check | Frequency | Failure Action |
|------|--------------|-----------|----------------|
| Evidence | Citation validation on every output | Per interaction | Output withheld + logged |
| Explainable | Format compliance scan | Per interaction | Output flagged for review |
| Determinism | Input recording completeness | Per interaction | Interaction quarantined |
| Limitations | Disclaimer presence | Per interaction | Output blocked |
| HITL | Approval records for T2 | Daily scan | Process violation ticket |
| Evaluation | Precision/acceptance trends | Weekly | Tier downgrade if sustained |
| Auditability | Log completeness + availability | Continuous | Assistant suspended (fail-closed) |

---

## Gate Compliance Dashboard

Tracked metrics (extends Grafana platform-overview pattern):

| Panel | Content |
|-------|---------|
| Gate failures (7d) | Count by gate, by assistant |
| Citation compliance | % outputs with valid citations (target: 100%) |
| Audit completeness | % interactions with full records (target: 100%) |
| Eval status | Last scenario suite result per assistant |
| Kill switch readiness | Last exercise date + result |
| Approval compliance | T2 interactions with approval records (target: 100%) |

---

## Gate Exception Process

| Step | Detail |
|------|--------|
| Request | Written exception request with: gate, reason, compensating control, duration |
| Review | Platform lead + security (for Gates 5, 7); platform lead alone (others) |
| Approval | Time-boxed (max 30 days); recorded in governance log |
| Tracking | Exception register reviewed at quarterly governance |
| Expiry | Gate re-validated before renewal; max 2 renewals |

**Non-exceptable gates:** Gate 5 (HITL) and Gate 7 (auditability) have no exception path. These are the program's safety floor.
