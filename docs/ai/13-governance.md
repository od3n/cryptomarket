# AI Governance

## Purpose

Define the governance framework for all AI-assisted capabilities: approval requirements, responsibilities, audit logging, model selection, prompt ownership, evaluation cadence, rollback procedures, and privacy considerations.

**Governance principle:** AI capabilities are treated like production services — they have owners, SLOs, change management, audit trails, and kill switches.

---

## Approval Requirements

### Change Approval Matrix

| Change Type | Approval Required | Approvers |
|-------------|------------------|-----------|
| New assistant introduced to production | Design review + shadow period + explicit sign-off | Platform lead + SRE lead |
| Prompt major version change | Eval suite pass + human review | Assistant owner + 1 reviewer |
| Prompt minor/patch version | Eval suite pass | Assistant owner |
| Model change (provider or version) | Full eval suite + security review + governance review | Platform lead + security |
| Retrieval source addition | Data classification review | Data owner + assistant owner |
| New data source access (metrics, logs, APIs) | Access review against least-privilege | Security + SRE lead |
| Evaluation threshold change | Documented justification | SRE lead |
| Assistant removal/disabling | None (any engineer can disable; no approval needed to make safe) | — |

### Deployment Approval Flow

```
Development → Eval Gate → Shadow (1 week min) → Production Review → Active
                                                    │
                                              Sign-off recorded
                                              in audit log
```

**Shadow period requirements:**
- Minimum 1 week in shadow mode before activation
- All shadow outputs reviewed for safety violations (automated scan + human sample)
- Disagreements between shadow and active versions analyzed
- No safety violations found during shadow period

---

## Operator Responsibilities

### Roles

| Role | Responsibilities |
|------|-----------------|
| **Platform Lead** | Final authority on assistant activation/deactivation; quarterly governance review chair |
| **SRE Lead** | Eval scenario ownership; incident cross-reference; weekly quality review |
| **Assistant Owner** | Prompt maintenance; eval compliance; responding to quality alerts; monthly accuracy report |
| **Security Reviewer** | Model/data access reviews; adversarial testing; quarterly threat assessment of AI surface |
| **On-Call Engineer** | Validates AI summaries during incidents; reports inaccuracies immediately; retains full decision authority |
| **All Engineers** | Report AI issues via feedback mechanism; never bypass human review steps; challenge incorrect AI output |

### On-Call Specific Rules

1. AI incident summaries are **starting points**, not conclusions — verify key claims against live systems
2. If AI output conflicts with observed reality, **reality wins** — report the discrepancy
3. On-call may disable any assistant during an incident without approval (kill switch: `make ai-disable` or feature flag)
4. Incident timelines drafted by AI require human validation before postmortem publication

---

## Audit Logging

### What Is Logged

Every AI interaction produces an audit record:

```yaml
audit_record:
  id: uuid
  timestamp: ISO-8601
  assistant: incident-assistant
  task: correlate-alerts
  prompt_version: "incident-assistant/correlate-alerts/v3"
  model: {provider}/{model-id}/{version}

  input_summary:
    trigger: alertmanager-webhook
    alerts: [DataStaleCritical, PrimaryProviderDown]
    context_chunks: 6
    context_sources: [SRC-1..SRC-6]  # file paths logged separately

  output_summary:
    recommendation: "single incident group, rate-limit cascade"
    confidence: HIGH
    citations_count: 8
    ungrounded_claims: 0

  human_decision:
    action: accepted | rejected | modified | not-reviewed
    reviewer: {identity or role}
    timestamp: ISO-8601
    note: optional

  metadata:
    latency_ms: 4200
    tokens_input: 3400
    tokens_output: 890
    eval_flags: []  # safety violations, format failures
```

### Retention and Access

| Attribute | Policy |
|-----------|--------|
| Retention | 12 months (matches incident review cycle) |
| Access | SRE team + platform lead; security for investigations |
| Immutability | Append-only storage; no edit/delete capability |
| PII | No user PII in assistant interactions (operational data only) |
| Failure behavior | Audit log unavailable → assistant operations suspended (fail-closed) |

### Audit Review Cadence

| Cadence | Review |
|---------|--------|
| Weekly | Sample 10 interactions; verify citation compliance and decision logging |
| Monthly | Full scan for safety violations (automated) + trend analysis |
| Quarterly | Governance review: are assistants operating within approved scope? |
| Per incident | All assistant interactions during incident window reviewed |

---

## Model Selection

### Selection Criteria

| Criterion | Requirement |
|-----------|-------------|
| Data handling | Provider must not train on our inputs (contractual + API configuration) |
| Availability | >99% monthly uptime OR graceful degradation path exists |
| Latency | p95 < 10s for incident assistant path (30s budget total) |
| Structured output | Reliable JSON schema adherence (validated in eval) |
| Context window | Minimum 128k tokens (retrieval context + input data) |
| Security review | Provider security posture reviewed by security before adoption |
| Cost | Budgeted per assistant; monthly cost tracked against estimate |

### Model Change Process

1. Candidate model runs full eval suite against all scenarios
2. Results compared to current model (must win or tie on all core metrics)
3. Shadow period: 2 weeks (longer than prompt changes — model behavior differs subtly)
4. Security review of provider changes (if new provider)
5. Governance sign-off recorded in audit log
6. Rollback plan verified before switch (previous model config retained)

### Multi-Model Strategy

- Single model per assistant at any time (simplifies evaluation and audit)
- No model ensembling in production (determinism requirement)
- Model fallback: if primary model unavailable, assistant degrades to structured-only output (no LLM) — never falls back to a weaker model silently

---

## Prompt Ownership

| Artifact | Owner | Review Cadence |
|----------|-------|---------------|
| System base prompt (safety) | Platform lead | Quarterly + on any safety incident |
| Task templates | Assistant owner (per epic in [16-engineering-backlog.md](16-engineering-backlog.md)) | Monthly |
| Eval scenarios | SRE lead | Quarterly + post-incident |
| Retrieval configuration | Assistant owner | On source changes |
| Output schemas | Assistant owner + consumers | On consumer changes |

**Rules:**
- No prompt changes without owner sign-off (enforced via CODEOWNERS on `prompts/` directory)
- Safety constraints in system base prompt can only be modified with platform lead + security approval
- Orphaned prompts (owner left team) must be re-assigned within 2 weeks

---

## Evaluation Cadence

| Cadence | Activity | Owner |
|---------|----------|-------|
| Per change | Eval suite on prompt/model/assistant changes | Automated (CI) |
| Weekly | Quality metrics review; 20-item human validation sample | Assistant owner |
| Monthly | Full scenario suite re-run; acceptance trend analysis | SRE lead |
| Quarterly | Governance review: scope compliance, cost, model currency, threat assessment | Platform lead (chair) |
| Post-incident | Incident cross-reference; new scenario creation | SRE lead |
| Annual | Full program review: is AI augmentation meeting objectives? Renew governance. | Platform lead + engineering |

See [11-evaluation-framework.md](11-evaluation-framework.md) for metric definitions and scenario format.

---

## Rollback Procedures

### Assistant-Level Rollback

| Situation | Action | Time to Effect |
|-----------|--------|---------------|
| Prompt regression detected | Repin previous prompt version in config | <5 minutes |
| Model regression detected | Repin previous model config | <5 minutes |
| Assistant producing harmful output | Disable assistant via feature flag | <1 minute |
| Audit log failure | Automatic suspension (fail-closed) | Immediate |
| Retrieval index corruption | Rebuild index; assistant degrades to structured-only | <30 minutes |

### Kill Switches

```yaml
# Feature flags (internal/featureflags pattern)
ai.assistants.enabled: true          # Master switch
ai.incident-assistant.enabled: true  # Per-assistant switches
ai.deployment-advisor.enabled: true
ai.infra-review.enabled: true
ai.shadow-mode: false                # Force all assistants to shadow (no visible output)
```

**Rules:**
- Any engineer can flip kill switches during incidents (no approval needed)
- Switch state changes are audit-logged
- Post-disable review required within 48h before re-enablement
- All platform workflows function identically with all assistants disabled (zero-dependency guarantee)

---

## Privacy Considerations

### Data Classification for AI Context

| Data Type | Classification | Allowed in AI Context? |
|-----------|---------------|----------------------|
| Alert names, thresholds, metric values | Internal | Yes |
| Runbooks, ADRs, architecture docs | Internal | Yes |
| Terraform plans (resource configs) | Internal | Yes |
| Log content (application) | Internal | Yes, with scrubbing |
| Connection strings, credentials | Secret | Never (scrubbed pre-injection) |
| User PII (if ever in logs) | Restricted | Never (scrubbed pre-injection) |
| Business metrics (revenue, users) | Confidential | Not applicable to this platform |
| Provider API keys | Secret | Never |

### Scrubbing Pipeline

1. Pre-index: files matching secret patterns (`.gitleaks.toml` rules) excluded from indexing
2. Pre-injection: regex scrub of credential patterns from retrieved chunks
3. Runtime: output scan for accidental secret echo (blocks output if detected)
4. Audit: scrub events logged (what was removed, not the content)

### Provider Data Handling

- Contractual requirement: no training on customer data
- API configuration: zero-retention mode where available
- No data residency requirements triggered (operational metrics only, no user data)
- Quarterly review of provider data handling certifications

---

## Governance Review Checklist (Quarterly)

- [ ] All assistants operating within approved scope (no capability creep)
- [ ] Audit logs complete (no gaps); sample reviewed
- [ ] Eval metrics meeting targets; regressions explained
- [ ] Model/provider currency: no deprecated models in use
- [ ] Prompt versions current: no orphaned owners, reviews on schedule
- [ ] Kill switches tested (exercised in staging this quarter)
- [ ] Cost within budget; variance explained
- [ ] New incidents added to scenario library
- [ ] Security: adversarial suite passing; no new attack surface
- [ ] Human-in-the-loop compliance: no automated actions detected outside approved scope
