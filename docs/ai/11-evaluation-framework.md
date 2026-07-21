# AI Evaluation Framework

## Purpose

Define how AI assistant recommendations are measured, tested, and improved. No assistant operates in production without passing evaluation scenarios, and all assistants are continuously measured against these metrics.

---

## Evaluation Metrics

### Core Metrics (All Assistants)

| Metric | Definition | Target | Measurement Method |
|--------|-----------|--------|-------------------|
| **Precision** | Fraction of AI findings/recommendations that are correct | >85% | Human validation sample (weekly, min 20 items) |
| **Recall** | Fraction of true issues that AI identified | >70% | Retrospective analysis: incidents/issues AI missed |
| **Usefulness** | Operator rating: did this save time or improve decision quality? | >4.0/5.0 | Post-interaction survey (optional, 1-click) |
| **False positive rate** | Fraction of alerts/findings that required no action | <20% | Dismissal/no-action tracking |
| **Operator acceptance** | Fraction of recommendations followed | >60% | Decision log comparison (AI rec vs. human action) |
| **Time saved** | Estimated minutes saved per interaction | >5 min/incident, >10 min/review | Before/after time tracking on matched scenarios |
| **Grounding compliance** | Fraction of claims with valid evidence citations | 100% | Automated citation validation |

### Assistant-Specific Metrics

| Assistant | Additional Metric | Target |
|-----------|------------------|--------|
| Incident Assistant | Top-3 hypothesis contains actual cause | >80% |
| Incident Assistant | Correlation accuracy (grouped alerts truly related) | >90% |
| Incident Assistant | Time-to-summary from alert firing | <30s |
| Deployment Advisor | Zero missed deployment-caused SEV-1/2 with PROCEED rec | 100% |
| Deployment Advisor | ROLLBACK/INVESTIGATE recs later confirmed correct | >90% |
| Infrastructure Review | Findings dismissed as invalid | <15% |
| Code Review | Comments resolved without reply (accepted) | >50% |
| Documentation Assistant | Reported drift confirmed real | >90% |

---

## Evaluation Scenarios

### Scenario Library

Scenarios are replayable test cases derived from real platform behavior. Each scenario defines: inputs, expected outputs, and grading criteria.

#### S-1: Provider Rate Limit Cascade (from postmortem 001)

```yaml
scenario:
  id: S-1
  name: provider-rate-limit-cascade
  source: docs/postmortems/001-primary-provider-rate-limit.md
  target_assistant: incident-assistant

  inputs:
    alerts_firing:
      - HighRateLimitFrequency (t+0m)
      - PrimaryProviderDown (t+2m)
      - DataStaleCritical (t+9m)
    metrics:
      provider_rate_limited_total{provider="coingecko"}: 0.8/s
      circuit_breaker_state{name="coingecko"}: 1
      circuit_breaker_state{name="coincap"}: 1
      data_freshness_seconds: 743
    deployments:
      - v1.14.2 at t-2h16m (changed polling interval)

  expected_outputs:
    correlation:
      - alerts grouped as single incident (REQUIRED)
      - causal chain: rate limit → breaker → stale data (REQUIRED)
    hypotheses:
      - rank 1: provider rate limit exhaustion (REQUIRED in top-2)
      - evidence cites provider_rate_limited_total (REQUIRED)
    runbook:
      - recommends provider-rate-limiting or data-freshness-alert (REQUIRED)
    historical:
      - references postmortem 001 pattern (BONUS)

  grading:
    pass: all REQUIRED items satisfied
    excellent: REQUIRED + BONUS items
    fail: any REQUIRED item missing or incorrect correlation
```

#### S-2: Redis Unavailable (Independent Failure)

```yaml
scenario:
  id: S-2
  name: redis-unavailable-isolated
  target_assistant: incident-assistant

  inputs:
    alerts_firing:
      - RedisUnavailable (t+0m)
      - RealtimeConsumerLag (t+3m)
    metrics:
      redis_up: 0
      realtime_consumer_lag: 4500
    deployments: []

  expected_outputs:
    correlation:
      - RedisUnavailable + RealtimeConsumerLag grouped (REQUIRED)
      - NOT attributed to provider issues (REQUIRED)
    hypotheses:
      - rank 1: Redis/ElastiCache failure (REQUIRED)
      - does NOT suggest rate limiting (REQUIRED absence)
    runbook:
      - recommends redis-unavailable (REQUIRED)

  anti_patterns:
    - merging with provider alert chains without evidence
    - inventing deployment correlation when none exists
```

#### S-3: Healthy Deployment (No Action Needed)

```yaml
scenario:
  id: S-3
  name: healthy-deployment-proceed
  target_assistant: deployment-advisor

  inputs:
    error_rate: 0.0002 (baseline: 0.0003)
    p99_latency_ms: 187
    slo_budget_pct: 78
    rollbacks_30d: 0
    migrations: none
    dependency_changes: 1 patch bump
    drift: none

  expected_outputs:
    recommendation: PROCEED (REQUIRED)
    confidence: HIGH (REQUIRED)
    false_findings: none (REQUIRED — no spurious warnings)

  anti_patterns:
    - recommending DELAY/INVESTIGATE without evidence
    - fabricating risk factors
```

#### S-4: Budget-Constrained Deployment

```yaml
scenario:
  id: S-4
  name: low-budget-deployment
  target_assistant: deployment-advisor

  inputs:
    error_rate: 0.0004 (baseline: 0.0003)
    p99_latency_ms: 210
    slo_budget_pct: 18
    budget_trend: -3%/day
    rollbacks_30d: 1 (same service, 5 days ago)

  expected_outputs:
    recommendation: DELAY or INVESTIGATE (REQUIRED — not PROCEED)
    factors:
      - budget below 25% policy zone cited (REQUIRED)
      - recent rollback history cited (REQUIRED)
    policy_reference: docs/runbooks/error-budget-burn.md budget zones (BONUS)
```

#### S-5: Terraform Security Regression

```yaml
scenario:
  id: S-5
  name: iam-policy-widening
  target_assistant: infrastructure-review

  inputs:
    plan_diff: |
      ~ aws_iam_role_policy.ingestor
        Action: ["s3:GetObject"] → ["s3:*"]
        Resource: ["arn:aws:s3:::market-data/*"] → ["*"]

  expected_outputs:
    findings:
      - severity CRITICAL (REQUIRED)
      - identifies both Action and Resource widening (REQUIRED)
      - references least-privilege policy (REQUIRED)
    recommendation:
      - does NOT approve; flags for human (REQUIRED)
```

#### S-6: Documentation Drift

```yaml
scenario:
  id: S-6
  name: port-drift-detection
  target_assistant: documentation-assistant

  inputs:
    doc_content: "Grafana at http://localhost:3001"
    actual_config: "docker-compose.yml: grafana ports: 3002:3000"

  expected_outputs:
    drift_detected: true (REQUIRED)
    location: correct file and section (REQUIRED)
    suggested_fix: update to 3002 (REQUIRED)
    false_positive: none on consistent docs (REQUIRED)
```

---

## Evaluation Process

### Pre-Deployment Gate

| Stage | Requirement |
|-------|-------------|
| Unit evaluation | All scenarios for target assistant pass (REQUIRED items) |
| Regression suite | All scenarios for ALL assistants still pass (no cross-assistant regression) |
| Grounding audit | 100% citation compliance on scenario outputs |
| Human review | 2 engineers review sample outputs for tone, clarity, safety |

**Rule:** Assistant version that fails any REQUIRED item does not ship.

### Continuous Evaluation (Production)

| Cadence | Activity |
|---------|----------|
| Per interaction | Automated: citation validation, latency, format compliance |
| Daily | Automated: anomaly check on acceptance rate (sudden drop = investigate) |
| Weekly | Human: validate 20-item sample; record precision scores |
| Monthly | Full scenario suite re-run against current assistant version |
| Quarterly | Scenario library review: add scenarios from new incidents; retire stale ones |

### A/B Evaluation (Assistant Changes)

When updating prompts or retrieval:
1. Run both versions against full scenario suite
2. Compare precision/recall/usefulness metrics
3. Shadow mode: new version runs silently for 1 week; outputs compared to production version
4. Human review of disagreements between versions
5. Promote only if new version wins or ties on all core metrics

---

## Feedback Loops

### Operator Feedback

```yaml
# Attached to every assistant output
feedback:
  verdict: accept | reject | modify
  reason: optional free text
  time_saved_minutes: optional estimate
```

- Acceptance rate tracked per assistant, per scenario type
- Rejected recommendations trigger root-cause analysis (was the rec wrong, or right-but-unconvincing?)
- Feedback data feeds quarterly evaluation review

### Incident Cross-Reference

After every SEV-1/SEV-2 incident:
1. Was the assistant active during this incident?
2. Did it correlate correctly? Did hypotheses include the actual cause?
3. Did it miss anything the human found?
4. New scenario added to library from this incident (within 2 weeks)

---

## Reporting

### Weekly AI Quality Report (Automated)

```markdown
## AI Assistant Quality — Week of 2024-03-11

| Assistant | Interactions | Acceptance | Precision (sample) | Citations Valid |
|-----------|-------------|-----------|-------------------|-----------------|
| Incident | 14 | 79% | 88% (17/19 claims) | 100% |
| Deploy Advisor | 6 | 83% | 92% | 100% |
| Infra Review | 9 | 71% | 85% | 100% |

### Alerts
- Infrastructure Review acceptance dropped 12% WoW — investigation opened
- New scenario S-7 added from incident 002

### Actions
- [ ] Review 3 rejected infra findings for pattern
```

---

## Evaluation Infrastructure

| Component | Implementation |
|-----------|---------------|
| Scenario storage | `sre-toolkit/ai/eval/scenarios/*.yaml` (version-controlled) |
| Scenario runner | `sre-toolkit/ai/eval/run_eval.py` (pytest-compatible) |
| Results storage | Append-only log with scenario version + assistant version |
| CI integration | Evaluation suite runs on assistant code changes (like unit tests) |
| Dashboard | Quality metrics panel in Grafana (extends platform-overview pattern) |

**Reproducibility:** Every evaluation run records: scenario version (git SHA), assistant version, prompt template version, model identifier, retrieval index timestamp. Identical inputs must produce gradeable-equivalent outputs.
