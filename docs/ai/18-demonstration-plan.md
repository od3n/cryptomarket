# Demonstration Plan

## Purpose

Define seven demonstrations that prove AI-assisted capabilities work on this platform — with real tooling, real scenarios, and verifiable outcomes. Each demo is repeatable, uses existing infrastructure, and documents its limitations honestly.

**Demo environment:** Local stack via `make demo` (docker-compose: API, ingestor, realtime, frontend, PostgreSQL, Redis, Prometheus, Grafana, Alertmanager) + failure injection via `sre-toolkit/inject_failures.py`.

---

## Demo 1: AI-Assisted Incident Response

### Scenario

CoinGecko begins rate-limiting (429 responses), circuit breaker opens, fallback also degrades, data goes stale. This replays postmortem 001 (`docs/postmortems/001-primary-provider-rate-limit.md`).

### Inputs

```bash
# Inject the failure (existing tooling, ADR-014 safeguards)
ALLOW_FAILURE_INJECTION=true python3 sre-toolkit/inject_failures.py --scenario provider_429

# Alerts fire: HighRateLimitFrequency → PrimaryProviderDown → DataStaleCritical
# (per monitoring/prometheus/alerts.yml thresholds)
```

### Expected Outputs

1. **Alert summary** (<30s of DataStaleCritical firing):
   - What: data stale >10 min, freshness value cited from `data_freshness_seconds`
   - SLO impact: data-freshness budget burn rate cited from recording rules
   - Runbook: `docs/runbooks/data-freshness-alert.md` linked
2. **Correlation**: 3 alerts grouped as single incident with causal chain (rate limit → breaker → stale)
3. **Hypotheses**: Rank 1 = provider rate limit exhaustion, citing `provider_rate_limited_total` and postmortem 001 pattern
4. **Runbook recommendation**: provider-rate-limiting runbook with pre-filled diagnostics (rate limit confirmed ✓, breaker state cited)

### Validation Steps

1. Verify correlation matches known causal chain (ground truth: postmortem 001)
2. Verify every numeric claim against live Prometheus query
3. Verify runbook recommendation matches alert annotation
4. Cleanup: `make incident-reset`
5. Verify alerts resolve and assistant reports recovery

### Limitations

- Local stack has lower metric volume than production; timing may differ
- Single historical incident in corpus; pattern matching limited to known chains
- Assistant cannot detect novel failure modes outside knowledge graph

---

## Demo 2: AI-Assisted Deployment Review

### Scenario

Deploy a change while error budget is partially consumed. Advisor assesses whether to proceed.

### Inputs

```bash
# 1. Check current budget state (existing gate)
make slo-gate

# 2. Simulate budget pressure (optional: inject provider_500 briefly to consume budget)
ALLOW_FAILURE_INJECTION=true python3 sre-toolkit/inject_failures.py --scenario provider_500 --duration 120

# 3. Trigger deployment advisory for a real PR
# Advisor evaluates: error rate vs baseline, p99 latency, budget state, rollback history,
# migrations in changeset, dependency changes, drift status
```

### Expected Outputs

1. **Advisory report** with all 8 checks from [07-deployment-advisor.md](07-deployment-advisor.md):
   - Error rate: current vs. 24h baseline (PromQL cited)
   - Latency: p99 vs. 300ms SLO threshold
   - Budget: % remaining + policy zone classification (per `docs/runbooks/error-budget-burn.md` zones)
   - Rollback history, migrations, dependencies, drift
2. **Recommendation**: PROCEED/DELAY/INVESTIGATE with ranked factors and evidence
3. **Change conditions**: explicit statement of what would flip the recommendation

### Validation Steps

1. Verify budget % matches `check-slo-gate.sh` output
2. Verify latency value matches Grafana platform-overview dashboard
3. Verify recommendation logic: budget <25% should NOT produce unconditional PROCEED
4. Verify advisory is non-blocking: deployment workflow unaffected by advisor output
5. Compare with human assessment: 3 engineers independently rate the same deployment

### Limitations

- Baseline comparison requires 24h of metric history (short in fresh environments)
- Cost of delay not quantified (advisor does not weigh business urgency)
- Canary comparison requires canary labels not present in local stack

---

## Demo 3: AI-Assisted Code Review

### Scenario

Open a PR containing: a new API endpoint without metrics, a price handled as float (violating string-based price convention), and missing tests.

### Inputs

```bash
# Create demo branch with intentional issues:
# 1. internal/api/handlers.go: new endpoint without http_request_duration instrumentation
# 2. internal/market/: price calculation using float64 (violates design decision in
#    docs/architecture/overview.md: "String-based prices")
# 3. No test file for new code
git checkout -b demo/ai-review && # ... make changes ... && git push
```

### Expected Outputs

1. **Architecture comment**: float price flagged, citing `docs/architecture/overview.md` design decision #3
2. **Observability comment**: missing metrics instrumentation flagged, citing `internal/telemetry/` patterns
3. **Testing comment**: missing test coverage flagged, citing test patterns in `internal/api/handlers_test.go`
4. **Summary**: 3 findings, each with confidence + evidence + suggested fix
5. **No false positives** on compliant code in same PR

### Validation Steps

1. Verify each finding cites a real file/section (citation validation)
2. Verify architecture finding references actual documented convention
3. Verify human reviewer agrees with findings (ground truth: issues are intentional)
4. Verify AI comments do not duplicate golangci-lint output (complementary, not redundant)
5. Verify dismissal workflow: dismissing a comment does not block merge

### Limitations

- Convention detection limited to documented conventions (undocumented patterns not caught)
- Cannot assess business logic correctness (only structural/convention issues)
- Demo issues are obvious; subtle issues have lower detection rate

---

## Demo 4: AI-Assisted Infrastructure Review

### Scenario

Terraform PR that widens an IAM policy and increases RDS instance size in prod.

### Inputs

```bash
# Demo PR modifies deploy/terraform/modules/iam/ (policy widening)
# and deploy/terraform/environments/prod/ (instance size change)
# terraform-plan.yml generates plan output; advisor analyzes it
```

### Expected Outputs

1. **CRITICAL finding**: IAM policy widening (`Action: "*"` or `Resource: "*"`), citing `docs/security/iam-least-privilege.md`
2. **HIGH finding**: Cost impact of instance change with estimate (baseline: `docs/cost/estimates.md`)
3. **Blast radius**: Resources affected, environment scope, reversibility assessment
4. **Recommendation**: Do not merge until IAM finding resolved; cost finding requires utilization evidence

### Validation Steps

1. Verify IAM finding matches actual plan diff (no hallucinated resources)
2. Verify cost estimate within ±30% of AWS pricing for the instance change
3. Verify security constraint citation points to real policy text
4. Verify PR is NOT blocked by assistant (label added, human decides)
5. Verify finding severity matches review matrix in [08-infrastructure-review.md](08-infrastructure-review.md)

### Limitations

- Cost estimates from documented baseline, not live pricing API
- Cannot assess runtime behavior of infrastructure (only static plan analysis)
- Novel attack patterns outside threat model may be missed

---

## Demo 5: AI-Assisted Cost Optimization

### Scenario

Monthly cost review: identify resource waste from utilization data.

### Inputs

```bash
# Collect utilization from Prometheus (local stack) / CloudWatch (production)
# - CPU/memory per service vs. requests/limits
# - Redis memory usage vs. maxmemory (deploy/redis/redis-prod.conf)
# - PostgreSQL storage growth (migrations + snapshots)
# - Log volume from Loki
```

### Expected Outputs

1. **Utilization report**: Per-service usage vs. provisioned (with queries cited)
2. **Findings** (example format):
   - "market-ingestor CPU: 12% of requested 500m average over 30d → right-size to 200m (est. savings: $X/mo, risk: LOW — burst headroom maintained)"
   - "Loki log retention: debug-level logs from ingestor constitute 40% of volume → raise to info in production (est. savings: $Y/mo, risk: LOW)"
3. **Each finding**: estimated savings, operational impact, implementation steps, confidence
4. **Total**: aggregate potential savings with uncertainty range

### Validation Steps

1. Verify utilization numbers against Grafana dashboards
2. Verify savings math (unit price × quantity)
3. Verify risk assessments account for burst behavior (not just averages)
4. Human review: are recommendations actionable and safe?
5. Verify no recommendation would violate SLO headroom requirements

### Limitations

- Local stack utilization not representative of production patterns
- Savings estimates ±30% (documented uncertainty)
- Does not account for contractual commitments (reserved instances)

---

## Demo 6: AI-Assisted Onboarding

### Scenario

New engineer asks architecture and operational questions; assistant answers with citations.

### Inputs

```
Q1: "How does market data get from providers to the user's browser?"
Q2: "Why did you choose SSE instead of WebSockets?"
Q3: "What happens if CoinGecko goes down?"
Q4: "How do I run the platform locally?"
Q5: "What are the SLO targets and what happens when budget runs out?"
```

### Expected Outputs

1. **Q1**: Data flow answer citing `docs/architecture/overview.md` (provider → ingestor → PostgreSQL/Redis/Stream → API/realtime → frontend)
2. **Q2**: Cites ADR-003 with its context, decision, and consequences
3. **Q3**: Cites ADR-007 (provider fallback), circuit breaker behavior (ADR-009), relevant runbook, and postmortem 001 as real example
4. **Q4**: Cites `docs/onboarding.md` + Makefile targets (`make setup`, `make up`, `make demo`)
5. **Q5**: Cites `docs/sre/slos.md` targets table + error budget policy + `check-slo-gate.sh` behavior

### Validation Steps

1. Every answer carries file citations; verify each file exists and contains claimed content
2. Verify answers match source documents (no contradictions)
3. Verify "I don't know" behavior: ask about something not documented (e.g., "What's the mobile strategy?") — must NOT fabricate
4. Time comparison: engineer finds same answers manually (measure time saved)
5. New engineer (if available) rates answer helpfulness

### Limitations

- Answers only as good as documentation (gaps in docs = gaps in answers)
- Cannot answer questions about undocumented tribal knowledge
- Citation accuracy depends on index freshness

---

## Demo 7: AI-Assisted Documentation Updates

### Scenario

Configuration changes create documentation drift; assistant detects and proposes fixes.

### Inputs

```bash
# 1. Introduce drift: change Grafana port in docker-compose.yml (3001 → 3002)
# 2. Add a new API endpoint without updating OpenAPI spec
# 3. Change HPA max replicas in Helm values without updating autoscaling-strategy.md
# 4. Run drift detection
```

### Expected Outputs

1. **Drift report** detecting all 3 planted drifts:
   - Port mismatch: `docs/onboarding.md` says 3001, docker-compose says 3002
   - Endpoint gap: route exists in `internal/api/` but not in `api/openapi.yaml`
   - HPA mismatch: `docs/deployment/autoscaling-strategy.md` says 2-8, values say 2-4
2. **Each finding**: doc location (file + section), actual value (file + line), suggested fix, confidence
3. **PR creation**: Fix proposed as PR (not direct edit)
4. **Zero false positives** on unchanged docs

### Validation Steps

1. All 3 planted drifts detected (recall = 100% for this exercise)
2. Zero false positives on control docs (precision check)
3. Suggested fixes are correct (human review of PR)
4. Verify no direct pushes — PR workflow only
5. Revert changes; verify report clears on next run

### Limitations

- Detection limited to defined check categories (novel drift types not caught)
- Semantic drift (meaning changed, text technically accurate) not detected
- Planted drifts are obvious; real drift is often subtler

---

## Demo Execution Checklist

| Step | Command/Action |
|------|---------------|
| 1. Start stack | `make demo` |
| 2. Verify baseline | `make smoke && make smoke-realtime` |
| 3. Verify monitoring | `make prometheus-check`, Grafana at :3001 |
| 4. Run selected demo | Per demo inputs above |
| 5. Validate outputs | Per demo validation steps |
| 6. Cleanup | `make incident-reset` (if injection used) |
| 7. Record results | Demo log: date, version, pass/fail per validation step, observations |

**Results tracking:** Demo outcomes recorded in eval results log (same infrastructure as scenario evaluation). Failed validations become backlog items.
