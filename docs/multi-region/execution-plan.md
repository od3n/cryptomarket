# Phase 7 — Execution Plan: Risks, Debt, Backlog, Milestones, Gates, Demos

**Status:** Proposed — program backlog pending approval to begin Wave 1
**Parent:** [Program Plan](./program-plan.md)
**Scope:** Deliverables 38, 39, 40, 41, 42, 43, 44, 45, 46 + recommended first work package

---

## D38. Multi-Region Risk Register

| # | Risk | Likelihood | Impact | Detection | Prevention | Mitigation | Contingency | Owner | Validation |
|---|------|-----------|--------|-----------|------------|------------|-------------|-------|------------|
| R1 | Replication lag spike at failover → data loss beyond RPO | M | H | ReplicaLag alarm; lag displayed at approval | Continuous lag monitoring; write-quiesce before planned promotion | Accept measured loss (≤1 cycle); document | Restore from PITR if loss unacceptable | DB Lead | Promotion drill measures actual lag |
| R2 | Split-brain: dual ingestion during partition | L | H | Epoch divergence alarm; dual-holder alarm | Lease never stolen while live; epoch fencing in write tx; self-fence on renewal failure | Fenced writes rejected; duplicate events deduped | Quarantine both leaders; manual single-leader restore | Platform | Dual-ingestor chaos test (T6, D30 L4) |
| R3 | Data loss during promotion (uncommitted/in-flight writes) | M | M | Cycle boundary check before promotion | Freeze ingestion before promote (workflow step ordering) | RPO acceptance at approval | PITR on new writer (rare) | DB Lead | Exercise measurement |
| R4 | DNS caching delays traffic shift beyond RTO | M | M | External resolution probes | TTL lowered to 60s pre-incident (automated on candidate state); health-check failover records | Client retry + reconnect absorbs gap | Console record force; communicate status | Ops Lead | T1 measures transition time |
| R5 | Partial failover (traffic shifted, DB not promoted) | M | H | Orchestration step-state dashboard; FailoverExceedingRTO alert | Single workflow with explicit sequencing; mutex lock | Workflow idempotent re-run | Manual completion per runbook | IC | Workflow interruption drill |
| R6 | Insufficient standby capacity under full load | L | H | Capacity panels; cold-cache load test T10 | Headroom targets (D28); reserved min nodes; quarterly load test | Autoscale burst (2-3 min) | Emergency DB resize pre-approved | Ops Lead | T10 + failover load test |
| R7 | Incompatible application versions across regions during incident | M | M | Version manifest skew alarm | Sequential deploy with gates; skew policy D24 | Deploy lock halts divergence | Emergency single-region deploy procedure | App Lead | Skew alarm test |
| R8 | Schema divergence (migration applied to one region only) | L | H | Migration version comparison across regions | Migration leadership = writer region; replicas receive via replication | Blocking gate in promotion workflow | Apply pending migration post-promotion (workflow step) | DB Lead | Promotion-with-pending-migration drill |
| R9 | Regional secret drift (stale replica secrets at failover) | L | M | Replica age alarm (D22) | Secrets Manager replication + freshness alarm | Break-glass credential path | Manual secret recreation runbook | Security | Secret-read-with-primary-down drill |
| R10 | Artifact replication failure (images unavailable in secondary) | L | M | Digest-presence check in CI; replication alarm | ECR replication + CI verification gate | Deploy blocked until replicated | Cross-region pull override (documented) | Ops Lead | Replication failure drill |
| R11 | Monitoring blind spot during regional loss | M | M | Aggregation health meta-alert | Hybrid observability (D25); aggregation outside primary | Regional stacks independent | External synthetic checks as fallback | Ops Lead | Aggregation-loss drill (D18 S6) |
| R12 | Cost escalation from standby + transfer | M | M | Budget alerts; transfer line-item tracking | Reserved capacity; budget gates; quarterly review | Trim standby to warm (documented degradation) | Executive decision on topology | Program | Monthly cost review |
| R13 | Operator error during manual promotion | M | H | Two-person rule; checklist-driven runbook | Lag display + blast-radius summary before approval; rehearsal | Workflow rollback where possible | Quarantine + forensics + rebuild | IC | Quarterly tabletop + drill |
| R14 | Premature failback to unstable region | M | H | 24h observation gate; health state history | Non-negotiable entry conditions (D17.1); IC approval | Abort-at-any-step reversibility | Re-failover (procedure identical) | IC | Failback drill (T11) |
| R15 | False-positive regional failover (automation over-reacts) | M | M | Multi-signal + external vantage requirement; cooldown | Hold-down periods; composite scoring; F1-only automation | Automated revert path; 30-min cooldown | Post-incident automation tuning | Platform | Fault-injection false-trigger tests |
| R16 | Provider terms violation from accidental dual ingestion | L | H | Provider request-rate monitoring per key | Single-leader design; lease before any provider call | Immediate leadership force-release | Provider communication; key rotation | Platform | Dual-ingestor test verifies single caller |

---

## D39. Multi-Region Technical Debt Register

Prioritized P1 (blocks program) → P3 (hygiene).

| # | Debt item | Evidence | Why it blocks multi-region | Priority | Remediation | Epic |
|---|-----------|----------|---------------------------|----------|-------------|------|
| TD1 | No region identity in config or services | `internal/config/config.go` has no REGION field | Cannot label, route, or fence by region | P1 | Add REGION/REGION_ROLE to config; propagate to metrics/logs/events | MR-1 |
| TD2 | Numeric PKs on `coins`, `provider_sync_logs` | `migrations/001_initial_schema.up.sql` BIGSERIAL | Unsafe under any cross-region merge; FK alignment with 003's UUID coin_id | P1 | UUIDv7 expand/contract migration (D10.3) | MR-1 |
| TD3 | Region-unaware, unversioned events | `internal/stream/event.go` — no schema_version/source_region | No provenance; no evolution path; dedup by random ID only | P1 | Event contract v2 (D12.1) | MR-1 |
| TD4 | In-memory consumer dedup with full reset at 10k | `internal/stream/consumer.go` `markSeen` | Dedup silently lost under load/restart | P1 | Redis-backed seen-set with TTL (D11.5) | MR-5 |
| TD5 | Hardcoded consumer name `realtime-1` | `internal/stream/consumer.go` const | Collisions across replicas/regions | P1 | `{region}-{pod}` consumer names | MR-5 |
| TD6 | No ingestion leadership | `internal/scheduler/scheduler.go` bare ticker | Duplicate writes + provider rate exhaustion across regions | P1 | Lease + epoch (D13) | MR-6 |
| TD7 | No snapshot idempotency constraint | `InsertBatch` plain INSERT | Re-ingestion duplicates history rows | P1 | Unique constraint + upsert (D12.2) | MR-1 |
| TD8 | Hardcoded single-region endpoints | Frontend build-args `API_URL`/`REALTIME_URL`; single DSN pattern | Regional endpoint injection impossible | P2 | Runtime endpoint resolution; per-region config overlays | MR-2/MR-7 |
| TD9 | Single Terraform state per environment | `prod/backend.tf` one key | One bad plan can destroy the region pair | P2 | State split per region + global layer (D19) | MR-2 |
| TD10 | Single-region dashboards/alerts (no region label) | `monitoring/` configs | Cannot attribute or compare regions | P2 | External labels + rule variants (D25) | MR-8 |
| TD11 | Non-idempotent operational jobs (partition maintenance uncoordinated) | `create_monthly_partition()` callable anywhere | Duplicate DDL attempts; uncoordinated execution | P3 | Execute under ingestion leadership (D14.2 #6) | MR-6 |
| TD12 | CI/CD pipeline exists only as documentation | `.github/workflows/` empty | No repeatable regional delivery | P1 | Build workflows single-region first, then extend (D23) | MR-7 |
| TD13 | Migrations requiring write-pause for PK swaps | Forward-only 003 pattern | Downtime-sensitive steps during regional rollout | P3 | Expand/contract discipline; rehearsal in staging | MR-4 |

---

## D40. Program Backlog

Epics MR-1 … MR-12. Each story format: ID, title, objective, scope, dependencies, risk, effort, acceptance criteria, validation, rollback, documentation.

### MR-1 — Regional Readiness (Wave 1)

**MR-1.1 Region identity and configuration**
- Objective: Every service knows its region and role.
- Scope: `REGION`, `REGION_ROLE` in `internal/config/config.go`; propagated to logger fields, metric external labels, event `source_region`.
- Dependencies: none. Risk: low. Effort: S.
- Acceptance: all services log/metric-tag region; config validation requires REGION in prod.
- Validation: unit tests + smoke with REGION set. Rollback: env var optional in dev. Docs: config README update.

**MR-1.2 UUIDv7 identifier migration**
- Objective: Remove BIGSERIAL from active write paths (TD2).
- Scope: migrations for `coins`, `provider_sync_logs`; `PublishEvent` → `uuid.NewV7()`; cycle_id generation in ingestor.
- Dependencies: MR-1.1. Risk: medium (FK alignment with 003). Effort: M.
- Acceptance: no BIGSERIAL in write paths; FK integrity verified; event IDs parse as v7.
- Validation: migration up/down tested on copy of prod-shaped data; dual-format read test. Rollback: down migration (expand/contract window). Docs: D10 status → implemented.

**MR-1.3 Event contract v2**
- Objective: schema_version, source_region, ingested_at, cycle_id, dedup_key, trace_context (TD3).
- Scope: `internal/stream/event.go` + `internal/cache/redis.go` publish path; consumer v1/v2 dual-parse.
- Dependencies: MR-1.1, MR-1.2. Risk: medium. Effort: M.
- Acceptance: v2 events published; v1 consumers unaffected (compat test); contract test in CI.
- Validation: `internal/provider/contract_test.go`-style event contract suite. Rollback: publish v1 (flag-gated). Docs: event schema doc.

**MR-1.4 Snapshot idempotency**
- Objective: Re-ingestion never duplicates history (TD7).
- Scope: unique constraint `(coin_id, provider, date_trunc('minute', captured_at))`; `InsertBatch` → upsert.
- Dependencies: MR-1.2. Risk: medium (constraint build on large table — staged with `NOT VALID`). Effort: S-M.
- Acceptance: duplicate-cycle injection test produces zero duplicate rows.
- Validation: integration test replaying same cycle twice. Rollback: drop constraint. Docs: D12 status update.

### MR-2 — Terraform Regionalization (Wave 2)

**MR-2.1 State split + global layer**: split env states into per-region roots; create `global/dns`, `global/iam`, `global/artifacts`, `global/observability` (D19). Deps: none. Risk: medium (state surgery). Effort: L. Acceptance: `terraform plan` clean in all roots; destroy-protection verified. Validation: staged state migration rehearsal. Rollback: state backup restore.

**MR-2.2 Regional modules extension**: ECR replication module; RDS replica support; secrets replica + multi-region KMS; observability remote-write. Deps: MR-2.1. Risk: medium. Effort: L. Acceptance: modules validate; plan-only in secondary (no apply until MR-3). Rollback: n/a (plan stage).

**MR-2.3 Secondary region infrastructure (staging first)**: apply regional roots in us-west-2 staging. Deps: MR-2.2. Risk: medium. Effort: M. Acceptance: staging secondary EKS/RDS-replica/ElastiCache/ECR/secrets exist and pass parity checks (D20). Rollback: targeted destroy with protection flags off only in staging.

### MR-3 — Global Traffic Management (Wave 4)

**MR-3.1 Route53 health checks + failover records** (global/dns): health-checked failover records for api/realtime/frontend; weighted record support; TTL policy. Deps: MR-2.3. Risk: low. Effort: M. Acceptance: external probes resolve per weights; simulated unhealthy region shifts resolution. Validation: T1/T2 in staging. Rollback: static records restore.

**MR-3.2 Regional health model implementation**: composite scorer (Prometheus rules + exporter or dedicated evaluator); `region_status` gauge; transition alerts. Deps: MR-8.1 (labels). Risk: medium. Effort: M. Acceptance: states transition correctly under injected faults. Rollback: scorer disabled → static healthy.

### MR-4 — PostgreSQL Replication and Promotion (Wave 3)

**MR-4.1 Cross-region replica (prod)**: replica in us-west-2; lag monitoring + alarms. Deps: MR-2.2. Risk: medium. Effort: M. Acceptance: lag <5s sustained; alarms fire on induced lag. Rollback: replica removal.

**MR-4.2 Promotion tooling + runbook**: orchestration step implementation (promote, CNAME swap, canary write); runbooks 3/8/9. Deps: MR-4.1, MR-6.1 (freeze). Risk: high. Effort: M. Acceptance: staging promotion drill passes with zero old-writer writes. Rollback: n/a post-promotion (forward-only + rebuild path).

**MR-4.3 Failback tooling**: rebuild-as-replica procedure; reconciliation tooling extension; runbooks 11/12/13. Deps: MR-4.2. Risk: high. Effort: L. Acceptance: T11 passes. Rollback: abort-at-step reversibility.

### MR-5 — Redis and Realtime Recovery (Wave 3)

**MR-5.1 Consumer hardening**: unique consumer names (TD5); Redis-backed dedup (TD4). Deps: MR-1.3. Risk: low. Effort: S-M. Acceptance: multi-replica consumption with zero duplicate broadcasts; dedup survives restart. Rollback: config revert.

**MR-5.2 Cache warm + stream reset**: `WarmCache` routine; SSE stream-reset signaling; replay semantics post-failover. Deps: MR-1.3. Risk: low. Effort: M. Acceptance: cold-cache failover serves within 1 cycle (T10). Rollback: flag-gated.

### MR-6 — Ingestion Leadership (Wave 3)

**MR-6.1 Lease infrastructure**: `ingestion_leases` table; acquire/renew/fence in ingestor; standby loop; metrics/alarms. Deps: MR-1.2. Risk: medium. Effort: M. Acceptance: exactly one leader across two running ingestors; stale-leader writes rejected. Validation: dual-ingestor test. Rollback: leadership disabled → single-replica deploy (current behavior).

**MR-6.2 Leadership transfer workflows**: voluntary release API (operational endpoint); forced release; integration with failover workflow. Deps: MR-6.1. Risk: medium. Effort: S-M. Acceptance: T6 passes. Rollback: manual single-leader config.

### MR-7 — Multi-Region Delivery (Wave 5)

**MR-7.1 CI/CD pipeline build-out (single-region first)**: ci.yml, build-sign.yml, deploy workflows per D23.3. Deps: MR-2.2 (ECR). Risk: medium. Effort: L. Acceptance: full pipeline runs single-region prod deploy with gates. Rollback: manual deploy path retained.

**MR-7.2 Regional promotion sequence**: secondary-first canary sequence; version manifest; skew alarms; regional rollback workflow. Deps: MR-7.1, MR-2.3. Risk: medium. Effort: M. Acceptance: release promoted through both regions with staged gates; skew alarm verified. Rollback: per-region helm rollback.

### MR-8 — Global Observability (Wave 5, partially parallel with Wave 3-4)

**MR-8.1 Region labels + aggregation**: external labels on all Prometheus; remote-write to aggregation point; global Grafana. Deps: MR-2.2. Risk: low. Effort: M. Acceptance: global dashboards show both regions; aggregation-loss drill passes (regional stacks unaffected).

**MR-8.2 Dashboards + alert suite**: D26 dashboard families; D27 alerts with runbook lint. Deps: MR-8.1, MR-3.2, MR-6.1. Risk: low. Effort: M. Acceptance: all D27 alerts have runbook links; grouping verified in Alertmanager staging.

### MR-9 — Failover Automation (Wave 5-6)

**MR-9.1 Orchestration workflow engine**: D16 workflow with audit log, mutex, idempotent steps, approval gates. Deps: MR-3.2, MR-4.2, MR-6.2. Risk: high. Effort: L. Acceptance: full workflow executes in staging with interruption/re-run test. Rollback: manual runbook path always available.

**MR-9.2 Automated F1 traffic policy**: pre-authorized traffic shift on health criteria; cooldown; inhibit switch. Deps: MR-9.1. Risk: high. Effort: M. Acceptance: injected regional failure shifts traffic without human action, within hold-down; false-trigger tests pass. Rollback: policy disabled → manual F1.

### MR-10 — Resilience Testing (Wave 6)

**MR-10.1 Load/failover test suite**: T1-T11 scripted (k6 + chaos scripts extension). Deps: MR-9.1. Risk: low. Effort: M. Acceptance: all scenarios executable via `make` targets; measurements recorded.

**MR-10.2 Chaos levels 1-3 in staging**: experiment definitions with guards per D30. Deps: MR-10.1. Risk: medium. Effort: M. Acceptance: each experiment run with cleanup verification.

**MR-10.3 Full DR exercise (D33)**: execute, measure, retro. Deps: all of MR-1..MR-9. Risk: medium. Effort: M (execution). Acceptance: RTO ≤15m, RPO ≤60s measured; retro actions filed.

### MR-11 — Failback and Reconciliation (Wave 6)

**MR-11.1 Failback execution**: run T11 in staging; then production-ready failback procedure sign-off. Deps: MR-4.3, MR-10.3. Risk: high. Effort: M. Acceptance: topology restored; reconciliation report clean.

### MR-12 — Cost and Governance (continuous)

**MR-12.1 Cost controls**: budget alerts; transfer line items; monthly review cadence. Deps: MR-2.3. Risk: low. Effort: S. Acceptance: first monthly review completed with trend data.

**MR-12.2 Governance cadence**: quarterly DR exercise calendar; parity drift reviews; risk register review; SLO review integration. Deps: MR-10.3. Risk: low. Effort: S. Acceptance: calendar + owners assigned; first cycle executed.

---

## D41. Program Milestones

| Milestone | Deliverables | Entry criteria | Exit criteria | Blockers | Risks | Evidence |
|-----------|--------------|----------------|---------------|----------|-------|----------|
| **M1 — Region-Aware Application** | MR-1 complete | Program plan approved | REGION in all services; UUIDv7 live; event v2 live; idempotent inserts; all tests green | None (first milestone) | Migration surprises (003 FK state) | CI green; migration test report |
| **M2 — Independent Secondary Region** | MR-2 complete | M1 | Staging secondary fully deployed from replicated artifacts+secrets with primary simulated down; parity check green | State split rehearsal | State surgery error | Independence drill log |
| **M3 — Data Replication Ready** | MR-4.1, MR-5, MR-6 complete | M2 | Replica lag monitored <5s; leadership proven (dual-ingestor test); cache warm proven | Replica provisioning | Lag under load | Lag dashboard; test reports |
| **M4 — Controlled Traffic Shift** | MR-3, MR-7, MR-8 complete | M3 | T1/T2 pass in staging; pipeline promotes through both regions; global dashboards live | Health model accuracy | False triggers | Exercise measurements |
| **M5 — Regional Failover Validated** | MR-9, MR-10.1/10.2 complete | M4 | Full D33 exercise passes: RTO ≤15m, RPO ≤60s, zero dual-writer | Orchestration reliability | Workflow edge cases | Exercise report + retro |
| **M6 — Failback Validated** | MR-4.3, MR-11 complete | M5 | T11 passes; normal topology restored; reconciliation clean | Reconciliation completeness | Premature failback pressure | Failback report |
| **M7 — Operationally Governed** | MR-10.3, MR-12 complete; runbooks/alerts/roles closed | M6 | All D32 runbooks exercised; D27 alerts live with runbook lint; incident command drilled; cost review ×1; calendar set | Documentation discipline | Drift from process | Governance audit checklist |

---

## D42. Critical Path

```
M1 regional readiness (identity, UUIDv7, event v2, idempotency)
  → M2 Terraform regionalization + independent secondary
    → M3a PostgreSQL replication ─────────────┐
    → M3b ingestion leadership (parallel) ────┼→ M4 traffic + delivery + observability
    → M3c Redis/realtime hardening (parallel) ┘        → M5 failover validation (D33 exercise)
                                                         → M6 failback validation
                                                           → M7 governance
```

Critical path (longest chain): **MR-1 → MR-2 → MR-4.1 → MR-4.2 → MR-9.1 → MR-10.3 → MR-11.1.**

Parallelizable work:
- MR-6 (leadership) is independent of Terraform after MR-1 — start as soon as M1 closes.
- MR-5 (Redis hardening) independent after MR-1.3.
- MR-7.1 (single-region CI/CD) can start IMMEDIATELY (no multi-region dependency) — recommended to run alongside MR-1 to de-risk the longest external dependency.
- MR-8.1 (labels/aggregation) parallel with MR-2.
- Operations docs (runbooks, incident model, exercise design) can be written during any wave.

---

## D43. Quality Gates

Work packages cannot close until the applicable gate passes. Gates are checked at milestone boundaries.

| Gate | Criteria | Checked at |
|------|----------|------------|
| **Architecture Gate** | Target topology approved (ADR-020 merged); consistency model documented and reviewed; source-of-truth table signed off; split-brain prevention reviewed by a second engineer | Before M1 starts |
| **Infrastructure Gate** | All Terraform roots validate; states isolated (no cross-region resource in one state); destroy protections verified; region parity check green; drift detection running | M2 |
| **Data Gate** | Replication monitored with alarms; promotion tested end-to-end in staging; reconciliation tested; RPO measured ≤60s; idempotency tests green | M3 |
| **Application Gate** | Globally unique IDs verified (no BIGSERIAL writes); consumers idempotent (replay test); regional config complete; retries safe (no retry-amplified duplicates) | M1 (partial) + M3 |
| **Operations Gate** | All D32 runbooks complete and table-top-exercised; all D27 alerts actionable with runbook links; failover exercise passed (M5); failback tested (M6) | M5/M6/M7 |
| **Security Gate** | Least-privilege review of new roles; secrets replicated safely (replica-age alarm proven); break-glass process tested in staging; no cross-region secret leakage (audit) | M4 |

---

## D44. Work Package Template

Every implementation package MUST use this template (stored at `docs/multi-region/work-package-template.md` on first use):

```
ID:
Title:
Program: Phase 7 — Global Resilience
Objective:
Background:
Scope:
Out of Scope:
Dependencies:
Architecture Impact:
Data Impact:
Regional Impact:
Security Impact:
Cost Impact:
Failure Modes:
Implementation Steps:
Expected Files:
Acceptance Criteria:
Tests:
Observability:
Rollout:
Rollback:
Runbook Updates:
ADR Updates:
Risks:
Completion Evidence:
```

---

## D45. Demonstration Plan

### D45.1 Ten-minute demo — "Regional failover, end to end"

1. Architecture slide (program-plan.md §6 topology diagram) — 2 min.
2. Live: global dashboard showing both staging regions healthy, leader in primary — 1 min.
3. Inject failure (scripted: scale primary to zero + block DB) — 1 min.
4. Watch: health states transition, alarms page, traffic shift executes — 3 min.
5. Show: promoted writer, leadership epoch, freshness recovering, SSE clients reconnected — 2 min.
6. Close: RTO measurement panel vs target — 1 min.
- Commands: `make dr-exercise-inject`, dashboards at aggregation Grafana. Expected outputs scripted in demo runbook. Cleanup: `make dr-exercise-reset`.

### D45.2 Thirty-minute demo — architecture + data + recovery

Adds: replication lag live view; dual-ingestor split-brain prevention demo (start second ingestor → show lease denial + fenced write rejection); event dedup demo (replay cycle → zero duplicates); observability tour (regional vs global); RTO/RPO evidence from last three exercises.

### D45.3 Deep technical review

Consistency model walkthrough (data-strategy.md D9); split-brain vector table and fencing code path (lease acquisition SQL + epoch check); database promotion sequence and why failback ≠ reverse-failover; event idempotency design (dedup_key derivation); cost trade-off table vs active-active. Audience: staff-level interviewers. Materials: this doc set + code links.

### D45.4 Incident simulation

Full D33 exercise run with observers; roles assigned per D34; timeline captured; retro observed. Evidence: incident record + audit log + measurements.

---

## D46. Interview Readiness Matrix

| Subject | Topics | Implemented evidence | Documentation | Source files | Demo | Trade-offs to articulate | Likely questions |
|---------|--------|---------------------|---------------|--------------|------|--------------------------|------------------|
| Multi-Region PostgreSQL | Replication, consistency, RPO, promotion, split-brain | Replica + lag alarms (post-MR-4); promotion drill evidence | data-strategy.md D8/D9; failover-strategy.md D16 | `deploy/terraform/modules/rds/`, promotion tooling | D45.2 replication segment | Why not Aurora Global / multi-writer; async lag acceptance; BIGSERIAL→UUIDv7 | "Walk me through promotion"; "What's your RPO and how do you measure it?"; "Why not multi-master?" |
| Global Traffic Routing | DNS, health checks, failover, TTL, connection draining | Route53 failover records (post-MR-3); T1/T2 measurements | infrastructure-plan.md D23; failover-strategy.md D15 | `global/dns` Terraform | D45.1 traffic shift | Route53 vs Global Accelerator vs Cloudflare; SSE across regions; stale DNS | "How do long-lived SSE connections fail over?"; "DNS TTL strategy during incidents?" |
| Ingestion Leadership | Leases, fencing, idempotency, rate limits | Lease table + epoch fencing (post-MR-6); dual-ingestor test | data-strategy.md D13/D14 | `internal/worker/ingestor.go`, lease package | D45.2 split-brain segment | Why PG advisory lease over Redis/ZK; fencing token necessity; TTL choice | "Why isn't a distributed lock enough?"; "What happens in a partition?" |
| Regional DR | RTO/RPO, runbooks, incident command, testing | D33 exercise results; runbook set; retro actions | failover-strategy.md D31; operations-plan.md D33/D34 | `docs/runbooks/`, exercise scripts | D45.4 simulation | Hot standby vs warm vs active-active cost/recovery; failback ≠ failover | "How did you choose RTO 15m?"; "What did your last DR exercise find?" |
| Consistency & Events | Eventual vs strong, dedup, ordering, schema evolution | Event v2 + idempotent inserts (post-MR-1) | data-strategy.md D9/D12 | `internal/stream/`, `internal/cache/redis.go` | D45.2 dedup segment | At-least-once + idempotency vs exactly-once; dedup_key design; v1/v2 compat | "How do you handle duplicate events?"; "Out-of-order delivery?" |
| Platform Engineering | Terraform state boundaries, CI/CD, parity, drift | State split + parity checks (post-MR-2) | infrastructure-plan.md D19/D20 | `deploy/terraform/` | Architecture review | State blast radius; secondary-first canary; drift policy | "How do you prevent one bad plan destroying everything?" |

---

## Recommended First Implementation Work Package

**MR-1.1 + MR-1.2 + MR-1.3 + MR-1.4 as a single coordinated package "Region-Aware Foundation" (M1)**, with MR-7.1 (single-region CI/CD) started in parallel.

Rationale:
1. M1 items are pure application work — no AWS provisioning, no cost, fully reversible, testable locally (charter: "Do not begin expensive AWS provisioning before local design, static validation, and staging plans are complete").
2. Every subsequent epic depends on region identity, global IDs, event v2, and idempotency — doing them first removes the most cross-cutting risk.
3. Parallel CI/CD build-out de-risks the largest external dependency without touching multi-region scope.
4. The package is independently reviewable (four focused PRs), independently testable (unit + integration + migration tests), safely revertible (flag-gated event v2; down migrations), and produces immediately demonstrable evidence (event provenance, dedup test) for the portfolio.

First concrete PR: **MR-1.1 region identity** — ~1 day, zero risk, unlocks everything.
