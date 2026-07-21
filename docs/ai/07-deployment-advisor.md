# Deployment Advisor Design

## Purpose

Design an AI assistant that reviews deployments before promotion, evaluating operational readiness and recommending PROCEED / DELAY / INVESTIGATE / ROLLBACK with full explanation.

**Design basis:** The platform already enforces an SLO gate (`scripts/check-slo-gate.sh` in `deploy-production.yml`) and staged canaries (5%→25%→50%→100%). This advisor augments those gates with contextual analysis — it never replaces them.

---

## Decision Framework

| Recommendation | Meaning | When |
|---------------|---------|------|
| **PROCEED** | No elevated risk detected | All checks pass; normal canary process applies |
| **DELAY** | Temporary condition suggests waiting | Budget low but recovering; ongoing unrelated incident; freeze window |
| **INVESTIGATE** | Signals conflict or are unclear | Canary metrics ambiguous; drift detected; unusual error pattern |
| **ROLLBACK** | Evidence of active harm from this deployment | Error rate spike correlated with canary; SLO violation began at deploy time |

Every recommendation includes:
1. The decision
2. Contributing factors (ranked by weight)
3. Evidence for each factor (metric queries, timestamps, values)
4. What would change the recommendation
5. Confidence level

---

## Evaluation Checks

### 1. Error Rate Analysis

**Queries:**
```promql
# Current 5xx rate vs. 24h baseline
sum(rate(http_responses_total{status=~"5.."}[5m])) / sum(rate(http_responses_total[5m]))
# Baseline
avg_over_time((sum(rate(http_responses_total{status=~"5.."}[1h])) / sum(rate(http_responses_total[1h])))[24h:1h])
```

**Rules:**
- Current rate > 2x baseline → INVESTIGATE
- Current rate > 5x baseline → DELAY (if pre-deploy) or ROLLBACK (if canary active)
- Rate within baseline variance → PASS

### 2. Latency Analysis

**Queries:**
```promql
histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))
# vs. SLO threshold: 300ms (docs/sre/slos.md)
```

**Rules:**
- p99 > 300ms sustained 10m → INVESTIGATE
- p99 > 500ms (alert threshold from `APIHighLatency`) → DELAY
- Regression > 20% vs. previous release at same traffic level → INVESTIGATE

### 3. SLO Budget State

**Source:** `check-slo-gate.sh` logic (budget < 10% blocks deployment)

**Advisory layer beyond the gate:**
- Budget 10-25%: PROCEED with caution note ("budget low — canary verification especially important")
- Budget 25-50%: Note per error budget policy ("no risky deploys" zone from `docs/runbooks/error-budget-burn.md`)
- Budget trend: falling > 5%/day → DELAY recommendation with trend evidence

### 4. Canary Metrics (during staged rollout)

**At each canary stage (5%, 25%, 50%):**
```promql
# Canary vs. stable comparison
sum(rate(http_responses_total{status=~"5..",version="canary"}[5m])) / sum(rate(http_responses_total{version="canary"}[5m]))
# vs.
sum(rate(http_responses_total{status=~"5..",version="stable"}[5m])) / sum(rate(http_responses_total{version="stable"}[5m]))
```

**Rules:**
- Canary error rate > 3x stable → ROLLBACK recommendation
- Canary latency p99 > 1.5x stable → INVESTIGATE
- Canary healthy for stage duration → PROCEED to next stage

### 5. Rollback History

**Input:** GitHub workflow runs, Helm release history

**Rules:**
- Same service rolled back within 7 days → INVESTIGATE ("repeat deployment of recently-rolled-back change")
- 2+ rollbacks in 30 days for same service → DELAY with recommendation for root-cause review first
- Rollback of identical image tag → ROLLBACK recommendation (known-bad artifact)

### 6. Database Migrations

**Input:** PR changeset — files matching `migrations/*.up.sql`

**Rules:**
- Migration present → flag for review:
  - Destructive operations (DROP, TRUNCATE, ALTER TYPE narrowing) → INVESTIGATE with specific warning
  - Index creation on large tables → note lock risk
  - No down migration counterpart → INVESTIGATE
- Migration + canary deploy → note irreversibility window

### 7. Dependency Changes

**Input:** `go.mod`/`go.sum` or `package-lock.json` diffs; Dependabot PR metadata

**Rules:**
- Major version bump → INVESTIGATE with changelog summary
- Security advisory on new version → flag severity
- >10 transitive dependency changes → note increased uncertainty
- Known CVE in removed version → positive note (security improvement)

### 8. Infrastructure Drift

**Source:** `terraform-drift.yml` latest results

**Rules:**
- Active drift on modules related to deploying service → INVESTIGATE
- Drift on unrelated modules → informational note
- No drift check in last 24h → note staleness

---

## Output Format

```markdown
## Deployment Advisory: market-api v1.15.0

### Recommendation: PROCEED (confidence: HIGH)

### Factor Summary

| Check | Status | Detail |
|-------|--------|--------|
| Error rate | PASS | 0.02% (baseline: 0.03%) |
| Latency p99 | PASS | 187ms (SLO: 300ms) |
| SLO budget | CAUTION | API availability: 34% remaining (policy: no risky deploys below 25%) |
| Rollback history | PASS | No rollbacks in 30 days |
| Migrations | PASS | None in changeset |
| Dependencies | NOTE | 2 minor bumps (Dependabot, security patches) |
| Infra drift | PASS | Last check 6h ago, no drift |

### Evidence
- Error rate query: [PromQL] → 0.0002 at 2024-03-15T10:00Z
- Budget query: [PromQL] → 34.2% and stable (Δ +0.1%/day over 7d)

### Conditions That Would Change This Recommendation
- Error rate exceeds 0.06% (2x baseline) during canary → ROLLBACK
- Budget drops below 25% before promotion → DELAY
- New critical alert fires on market-api → INVESTIGATE

### Process Reminders
- SLO gate runs automatically (scripts/check-slo-gate.sh) — this advisory does not replace it
- Canary stages: 5% → 25% → 50% → 100% with verification at each stage
- Rollback: helm rollback cryptomarket <revision> -n cryptomarket-prod
```

---

## Integration Points

| Trigger | Context | Delivery |
|---------|---------|----------|
| PR merged to main | Pre-deployment assessment | PR comment (post-merge) |
| `deploy-production.yml` workflow_dispatch | Gate-stage assessment | Workflow check run (advisory, non-blocking) |
| Canary stage completion | Stage health assessment | Workflow annotation + channel message |
| Manual CLI query | On-demand assessment | Terminal output |

**Critical constraint:** The advisor runs as an advisory check — it reports status but does NOT gate the workflow. The existing `slo-gate` job remains the only automated blocker. This preserves human authority over deployment decisions.

---

## Boundaries and Safety

| The advisor CAN | The advisor CANNOT |
|-----------------|-------------------|
| Query metrics (read-only) | Approve or block deployments |
| Read PR diffs and workflow history | Override the SLO gate |
| Recommend PROCEED/DELAY/INVESTIGATE/ROLLBACK | Trigger rollbacks |
| Cite evidence for every factor | Skip canary stages |
| Note policy violations (budget zones) | Grant exceptions to policy |

**Override transparency:** If a human deploys against a DELAY/INVESTIGATE recommendation, the advisory is logged with the decision outcome for evaluation purposes (see [11-evaluation-framework.md](11-evaluation-framework.md)).

---

## Evaluation Criteria

| Metric | Target | Measurement |
|--------|--------|-------------|
| Recommendation accuracy | >90% of ROLLBACK/INVESTIGATE recs later confirmed correct | 30-day outcome review |
| False alarm rate | <10% of DELAY recs were unnecessary | Operator feedback survey |
| Missed incidents | 0 deployments that caused SEV-1/2 had PROCEED rec | Incident cross-reference |
| Explanation quality | 100% of factors have evidence citations | Automated format validation |
| Latency | Assessment completes <60s | Workflow timing |

**Backtesting:** Replay historical deployment windows (from release-please history) against current metrics to validate check thresholds before enabling in production workflow.
