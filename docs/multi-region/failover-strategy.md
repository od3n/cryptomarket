# Phase 7 — Failover Strategy: Health, SLOs, Triggers, Orchestration, Failback, Partitions

**Status:** Proposed — pending operations gate approval
**Parent:** [Program Plan](./program-plan.md)
**Scope:** Deliverables 6, 7, 15, 16, 17, 18, 31

---

## D6. Regional Health Model

### D6.1 Principles

- A region is NOT healthy because one endpoint returns HTTP 200. Health is a composite of user-facing delivery, data pipeline, and dependency signals.
- Health is computed per region, in-region, and aggregated globally. Regional computation must survive loss of the other region (no cross-region dependency in the critical health path).
- Every signal has a weight, threshold, evaluation interval, and hold-down period. Transitions require sustained evidence.

### D6.2 Health indicators

| # | Indicator | Source | Healthy | Degraded | Unavailable | Weight |
|---|-----------|--------|---------|----------|-------------|--------|
| H1 | ALB/ingress target health | CloudWatch / kube probes | ≥99% targets healthy | 90-99% | <90% | 15% |
| H2 | API availability (5xx ratio) | Prometheus `api:success_ratio:5m` (existing recording rule) | ≥99.9% | 99-99.9% | <99% | 20% |
| H3 | API latency p99 | `http_request_duration_seconds` histogram | <300ms | 300-800ms | >800ms | 10% |
| H4 | Data freshness | `data_freshness_status` gauge (existing metric) | fresh (<120s) | delayed (<300s) | stale (>300s) | 20% |
| H5 | Ingestion success rate | `ingestion_success_total` / attempts (existing) | ≥99.5% | 95-99.5% | <95% | 10% |
| H6 | Provider availability | `provider_request_duration_seconds_count{status}` (existing) | primary up | fallback only | all down | 5% |
| H7 | PostgreSQL health | RDS status + app connection errors | writable (or replica lag <5s if standby) | lag 5-30s / elevated errors | unreachable | 10% |
| H8 | Redis health | ElastiCache status + app ping | reachable | elevated evictions/errors | unreachable | 5% |
| H9 | Realtime delivery success | `realtime_events_broadcast_total` / consumed (existing) | ≥99.5% | 95-99.5% | <95% or 0 connections served | 5% |
| H10 | Error-budget burn | Multi-window burn-rate rules (existing, `monitoring/prometheus/recording-rules.yml`) | <1x | 1-6x | >14.4x | — (alerting overlay, not score input) |
| H11 | Deployment status | Release manifest + rollout state | converged | rollout in progress | rollout failed/stuck | — (gating signal, not score input) |

### D6.3 Regional states

| State | Definition | Entry criteria | Allowed actions |
|-------|------------|----------------|-----------------|
| **healthy** | Composite score ≥ 99 | All weighted indicators in healthy band for ≥5 min | Normal operation; eligible as failover target |
| **degraded** | Score 90-99 | Any weighted indicator degraded for ≥5 min, or composite in band | Serve traffic; alert; not eligible as failover target; investigate |
| **failover-candidate** | Score < 90 | Composite <90 for ≥3 min (minimum failure duration) | Traffic shift pre-authorized; promotion requires human approval; standby verified |
| **unavailable** | Score < 50 or unmeasurable | Composite <50 for ≥2 min, or all health endpoints unreachable for ≥2 min | Traffic shift executed (automated per policy); promotion workflow initiated |
| **recovering** | Post-incident, restoring | Operator sets after remediation begins | Traffic may return only after observation window (D17); writes gated |
| **quarantined** | Excluded from routing by operator or automation after split-brain risk | Manual or automatic on dual-writer detection | No traffic; no leadership; forensics only |

### D6.4 Timing parameters

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| Evaluation interval | 30s | Balance of detection speed vs noise |
| Minimum failure duration (to candidate) | 3 min | Filters single-AZ blips and deploy transients |
| Hold-down (candidate → unavailable) | +2 min | Total 5 min sustained failure before automated traffic action |
| Recovery duration (unavailable → recovering) | All indicators healthy for 10 min continuous | Prevents flapping back |
| Recovering → healthy observation | 60 min minimum | Matches failback-lite observation |
| Manual override | Operator can force any state with audit entry | Overrides automation in both directions |

### D6.5 Alert behavior

- State transitions emit `region_status` gauge (1 per state) + transition event alert.
- Degraded: ticket (warning). Failover-candidate: page (high). Unavailable: page (critical) + automated traffic policy engaged. Quarantined: page (critical, split-brain class).
- Alert dedup: one page per region per state transition; global incident alert groups regional pages (operations-plan.md D27).

---

## D7. Global SLO Model

Extends the existing single-region SLOs (`docs/sre/slos.md`), which remain authoritative per region.

### D7.1 SLO hierarchy

| Level | SLO | Target | Measurement |
|-------|-----|--------|-------------|
| **Global** | Availability (user-perceived) | 99.95% / 30d | Weighted union of regional success: a request succeeds if ANY region serves it; global failure = all regions failing for the user |
| **Global** | API latency p99 | <500ms (relaxed vs regional 300ms to absorb routing variance) | Per-region histograms aggregated with `region` label |
| **Global** | Data freshness | 99% of time all tracked symbols fresh | Freshness of the SERVING region |
| **Global** | Realtime delivery | 99.5% | Serving-region broadcast ratio |
| **Global** | Replication lag | 99% of time <5s | `ReplicaLag` |
| **Global** | Failover success rate | 100% of exercises meet RTO; production failovers best-effort with mandatory review | Exercise + incident records |
| **Global** | Regional recovery time | RTO ≤15 min (95th percentile of exercises) | Exercise measurements |
| **Regional** | All existing SLOs from `docs/sre/slos.md` | Unchanged | Now labeled by `region` |
| **Regional** | Ingestion health | Per existing | Labeled by `region`; only the leader region is expected to ingest (standby region ingestion SLO is suspended while standby — explicit rule) |

### D7.2 Global success semantics under partial failure

| Condition | Global SLO treatment |
|-----------|---------------------|
| One region degraded, other serving | Global availability counts serving region's ratio; degraded region's errors count toward its regional budget only (prevents double-punishment) |
| Traffic shifted (failover in progress) | The shift window (up to RTO) is excluded from availability SLO IF declared an incident (existing exclusion: "Requests during declared incidents"); shift completion time counts against the failover SLO instead |
| Cached/stale data served during gap | Freshness SLO: served-but-stale data counts as freshness failure (honest accounting); availability SLO: 200-with-stale counts as success (user received service) |
| Realtime unavailable, REST serving | Realtime SLO fails for the window; availability SLO unaffected; degraded banner shown (existing `StatusBanner`/`FreshnessBadge` components support this UX) |
| Only fallback providers active | Ingestion SLO unaffected (data flows); provider-availability SLO reflects fallback; freshness holds if cycle interval maintained |

### D7.3 Error budgets

- **Global budget**: 0.05% of 30d = 21.6 min/month for availability.
- **Regional budgets**: 0.1% = 43.2 min each (existing).
- **Failover budget**: separate tracking — detection + shift + promotion time summed per event; target ≤15 min; any exceedance consumes the global budget and triggers review.
- Burn-rate alerting extends existing multi-window rules with `region` label; a global burn rule ORs regional burns for paging dedup.

---

## D15. Failover Trigger Policy

### D15.1 Separation of failover classes (critical design rule)

These four actions are independent, have different triggers, different approvers, and need NOT occur simultaneously:

| Class | Action | Automation level |
|-------|--------|------------------|
| F1 | **Traffic failover** (DNS/routing shift for stateless tiers) | Automated when health policy criteria met (pre-approved by this policy) |
| F2 | **Database promotion** (replica → primary) | Human-approved (Incident Commander + Database Lead) |
| F3 | **Ingestion leadership transfer** | Semi-automated: automatic ONLY after F2 completes; manual transfer possible independently for maintenance |
| F4 | **Full regional disaster declaration** | Human-declared; enables emergency procedures and SLO exclusions |

Typical sequencing: F1 (stop user bleeding) → F2 (restore writes) → F3 (restore data pipeline) — with F4 declared upfront if the failure is clearly regional.

### D15.2 Trigger definitions

| Trigger | Signal | Threshold | Duration | Validation | Automated action | Human approval | Rollback condition | False-positive risk |
|---------|--------|-----------|----------|------------|------------------|----------------|--------------------|--------------------|
| Regional compute failure | EKS API unreachable + all nodes NotReady | 100% node loss | 5 min | Cross-checked from external probe (Route53 health check from 3 regions) | F1 traffic shift | F2/F3 after IC approval | Nodes recover during shift → abort F2, hold traffic | Low (multi-signal) |
| Load balancer failure | ALB 5xx + target group unhealthy | >50% targets unhealthy | 3 min | Ingress controller logs + CloudWatch | F1 | F2 if ALB unrecoverable | Targets recover | Medium — distinguish from bad deploy |
| API error rate | `api:success_ratio:5m` | <99% | 5 min | Exclude 4xx; check deploy status first | F1 if <95% | F2 only if DB-linked errors | Rollback deploy if deploy-linked | Medium |
| Severe latency | p99 | >2s | 10 min | Check DB latency + provider latency | F1 (shed to other region) | None typically | Latency normalizes | Medium |
| Database unavailability | RDS status + app connection failures | Writer unreachable | 5 min | Confirm not a network-only partition (D18 scenario 2) | None (traffic stays; reads degrade to cache) | **F2 required** | DB returns before promotion → abort | Low |
| Data staleness | Freshness gauge | Stale >300s | 10 min | Check provider status + ingestor logs | Alert only | F3 if leader dead; F2 if DB dead | Provider recovers | Medium (provider-caused staleness ≠ region failure) |
| Network partition | Inter-region health check failure with both regions serving locally | Asymmetric reachability | 5 min | External vantage points (Route53 multi-region checks) | F1 toward region with DB writer | Quarantine the writer-less side's leadership claims | Partition heals | **High** — most dangerous trigger; requires external validation |
| AWS regional impairment | Multiple AWS service health dashboard failures + local signals | ≥2 services impaired | 15 min | AWS Health Dashboard + personal health dashboard | F1 | F2/F3 per impact | AWS recovery | Medium |
| Operator-declared incident | Human judgment via incident channel | n/a | n/a | IC assessment | Any class per IC direction | Inherent | IC decision | n/a |

### D15.3 Anti-flap and safety rules

1. No automated action on a single signal. F1 requires composite health state (D6) at failover-candidate or below, sustained through the hold-down.
2. Automated F1 executes at most once per 30 min per region (cooldown), then requires human re-authorization.
3. F2 is NEVER automated. The orchestration prepares and presents (lag, blast radius, checklist) but a human approves.
4. If the standby region is not `healthy` (D6), all failover is blocked and the incident escalates instead — failing over to a broken standby compounds the outage.
5. Every trigger firing is recorded to the failover audit log with signal values, regardless of action taken.

---

## D16. Failover Orchestration Workflow

### D16.1 Workflow (12 steps)

```
 1. DETECT          Regional health → failover-candidate/unavailable (D6)
 2. VALIDATE        Multi-signal + external vantage; rule out deploy/provider-only causes
 3. FREEZE          Suspend deployments to BOTH regions; pause non-critical writes
                    (feature flag: maintenance_mode on affected paths if partial)
 4. ASSESS LAG      Read ReplicaLag; estimate RPO loss; present to approver
 5. PROMOTE         [APPROVAL GATE: IC + DB Lead] → promote-read-replica; wait available
 6. TRANSFER        Ingestion leadership: force-release lease → standby acquires;
                    WarmCache; first cycle verified
 7. REROUTE         Route53 weights shift (staged 100% once verified, or immediate if F1 pre-executed)
 8. VERIFY          Synthetic checks: API reads, write canary, freshness, realtime event flow
 9. COMMUNICATE     Status: degraded mode declared; banner up; stakeholders notified
10. MONITOR         Error-budget tracking in new region; heightened alerting for 4h
11. STABILIZE       Capacity check (autoscaling headroom); cache warm confirmed; backlog drained
12. PREPARE FAILBACK  Old region quarantined; rebuild-as-replica initiated when feasible
```

### D16.2 Automation vs approval

| Step | Mode | Actor | Timeout | On timeout |
|------|------|-------|---------|------------|
| 1-2 | Automated | Health system | 5 min | Escalate to on-call |
| 3 | Automated | Orchestrator (pipeline freeze API) | 60s | Manual freeze + escalate |
| 4 | Automated (informational) | Orchestrator | 60s | Assume worst-case lag; present |
| 5 | **Human-approved** | IC + DB Lead | 10 min | Escalate to engineering leadership; re-evaluate F1-only posture |
| 6 | Semi-automated | Orchestrator executes after step 5 | 5 min | Retry once; then manual leadership transfer runbook |
| 7 | Automated (pre-authorized by policy) | Orchestrator via Route53 API | 5 min (TTL-bounded) | Verify record state; manual console action |
| 8 | Automated | Synthetic suite | 5 min | **Rollback: revert routing if promotion unhealthy** |
| 9-12 | Human-guided | Ops lead + automation assist | n/a | n/a |

### D16.3 Safety properties

- **Idempotent re-run:** every step checks current state before acting (promote is no-op if already primary; route change is no-op if already shifted; lease release is no-op if already standby-held). The workflow can be safely re-invoked after any interruption.
- **Audit trail:** append-only log entries: `{timestamp, step, actor, input-state, action, output-state, approval-ref}`. Stored in the surviving region's S3 + incident record.
- **Emergency override:** IC may skip steps 3-4 (freeze/lag assessment) with recorded justification when user impact is total and unambiguous. Steps 5 (approval) and 8 (verification) are NEVER skippable.
- **Rollback behavior:** if step 8 fails and the old primary is still alive (partial failure case), routing reverts and the incident continues in degraded mode. If the old primary is gone, rollback is impossible — the workflow escalates to stabilization of the new region (this asymmetry is deliberate and documented).

---

## D17. Failback Strategy

Failback is a planned operation, not a recovery reflex. **No automatic failback exists.**

### D17.1 Entry conditions (all required)

1. Original region healthy (D6 state `healthy`) for ≥24 continuous hours.
2. No active incidents anywhere.
3. Data reconciliation complete: all writes made in the secondary region during the failover window are present in the rebuilt replica (verified by cycle_id range comparison, D12).
4. Rebuilt replica lag <5s sustained for ≥4h.
5. IC approval with DB Lead and Ops Lead concurrence.
6. Scheduled in a low-traffic window (documented; typically weekend 02:00-05:00 UTC).

### D17.2 Sequence

```
1. VERIFY replica health and reconciliation (entry condition audit)
2. FREEZE deployments; announce maintenance window
3. QUIESCE ingestion: leader completes current cycle, then voluntary lease release
   (bounded pause ≤2 cycles ≈ 2 min; acceptable planned staleness)
4. FINAL SYNC: confirm replica caught up to zero lag (drain window ~60s)
5. ROLE REVERSAL: promote the rebuilt original-region replica? NO —
   preferred path: swap leadership and traffic WITHOUT DB promotion:
   the current writer (secondary) remains writer; instead execute a
   CONTROLLED WRITER MIGRATION:
     a. Demote secondary writer to replica of original (requires brief write pause)
     b. Original becomes writer
   Alternative (simpler, chosen for first exercises): keep secondary as writer,
   fail back ONLY traffic + ingestion leadership, defer DB role reversal to a
   separate planned migration. Rationale: DB role reversal is the highest-risk
   step and its benefit (restoring "primary in primary region") is cosmetic
   until the next failover.
6. TRANSFER ingestion leadership back (voluntary release → original acquires)
7. REBUILD caches in original region (WarmCache)
8. SHIFT traffic gradually: 10% → 25% → 50% → 100% over 60 min with health gates
9. OBSERVE 4h at 100% before closing the operation
10. RESTORE normal topology: re-establish secondary as hot standby; resume DR readiness
```

### D17.3 Safeguards

- Abort at any step returns to pre-failback state (writer unchanged, traffic reverted) — every step is reversible until step 5b, which has a 10-min decision window with the write pause held.
- Redis/cache rebuild precedes traffic (never route traffic to cold caches).
- Realtime clients: gradual shift means per-pod connection draining; clients reconnect naturally (existing SSE retry).
- Communication: status updates at each phase; post-operation review mandatory.
- **Premature failback is a registered risk** (execution-plan.md D38): the 24h observation is non-negotiable for unplanned failovers; planned maintenance returns may shorten to 4h with IC approval.

---

## D18. Network Partition Analysis

Seven scenarios, each with: behavior, writes, reads, ingestion, user impact, alerts, operator action, recovery.

### S1. Region A cannot reach Region B (inter-region link loss; both otherwise healthy)

- **Behavior:** Each region serves locally. Health checks from A to B fail, but external Route53 checks see both healthy → no automated failover (external vantage is authoritative, D15.3).
- **Writes:** Continue on the writer region only. Replication queues build (async); lag alarm fires.
- **Reads:** Normal in both regions (reads are writer-region-local pre-failover by design).
- **Ingestion:** Unaffected (leader in writer region; lease renewal is local).
- **User impact:** None.
- **Alerts:** Replication lag critical; inter-region probe failure (informational).
- **Operator:** Do nothing to traffic. Monitor lag. If lag approaches storage limits, throttle non-critical writes.
- **Recovery:** Link restores → replication drains → lag normalizes → auto-clear.

### S2. Region A serves users but cannot reach the database (DB is in A's region but network path is broken)

- **Behavior:** A's API degrades to cache-only reads (existing fallback). A's ingestor fails lease renewal → self-fences → ingestion stops in A.
- **Writes:** None from A. If B holds leadership eligibility and can reach the DB (cross-region DB access intact), B's standby acquires the expired lease → ingestion continues via B writing to A-region DB (cross-region write latency acceptable temporarily).
- **Reads:** A serves stale cache (≤5 min TTL) then 503-with-stale-banner; freshness SLO degrades honestly.
- **User impact:** Stale data in A; realtime paused in A.
- **Alerts:** DB connectivity (A), freshness stale (A), leadership transfer event.
- **Operator:** Investigate A→DB path. Do NOT promote (DB is alive). This is a network incident, not a DB incident.
- **Recovery:** Path restores → A re-acquires leadership (B voluntarily releases on next cycle boundary) → caches rebuild.

### S3. Both regions believe the other is unavailable (symmetric partition)

- **Behavior:** The dangerous case. External vantage (Route53 checks from 3+ other regions) breaks symmetry: whichever region the external checks cannot reach is marked unavailable; if both unreachable externally, global incident declared, no automated action.
- **Writes:** Writer region continues locally. Non-writer region CANNOT promote (promotion requires AWS control plane access to the replica — if it has that, it also has external connectivity, contradicting the scenario for that side).
- **Ingestion:** Lease holder continues; the other side's claims expire harmlessly (acquisition requires DB access = writer-side resource).
- **User impact:** Depends on which side users route to; DNS stays with historically healthy records (TTL-bounded).
- **Alerts:** Quarantine-class page (dual-unavailable); split-brain watch alarm.
- **Operator:** Trust external vantage ONLY. Quarantine any region claiming leadership without lease. Never promote on partition evidence alone.
- **Recovery:** Partition heals → reconcile lease epoch → single leader converges → lag drains.

### S4. Global traffic manager reaches only one region

- **Behavior:** Route53 marks the unreachable region unhealthy → failover records shift traffic to the reachable region (this is the designed response, not a failure of the design).
- **Writes:** Unchanged (writer region unaffected by routing).
- **Reads:** All users converge on reachable region; capacity must absorb 100% load (D28 headroom).
- **Ingestion:** Unaffected.
- **User impact:** Possible latency increase for users far from the surviving region; no data impact.
- **Alerts:** Traffic shift event; capacity watch on surviving region.
- **Operator:** Verify surviving region capacity; scale preemptively; investigate unreachable region.
- **Recovery:** Region reachable again → health checks pass hold-down → weighted return (not instant).

### S5. Provider connectivity differs by region (leader's region loses provider egress; standby has it)

- **Behavior:** Leader's cycles fail → provider circuit breaker opens → fallback provider attempted → if all fail, freshness degrades. Standby region has connectivity but no leadership.
- **Writes:** Leader continues writing nothing (no data); sync logs record failures.
- **Ingestion:** Stalled in leader region.
- **User impact:** Growing staleness globally (single ingestion path — the cost of single-leader design; accepted, mitigated below).
- **Alerts:** Provider failure (existing runbook: `docs/runbooks/provider-unavailable.md`) + freshness.
- **Operator:** **Planned leadership transfer** to standby (F3 manual): voluntary lease release; standby ingests via its healthy provider path. This is the primary resilience benefit of the standby ingestor.
- **Recovery:** Leader connectivity restores → transfer back at convenience (or leave leadership where it is; topology allows either).

### S6. Observability control plane partitioned (metrics/aggregation unreachable from one region)

- **Behavior:** Regional Prometheus keeps collecting locally (in-region scrape path is independent). Global aggregation shows one region dark.
- **Writes/reads/ingestion:** Unaffected (data plane does not depend on observability plane).
- **User impact:** None.
- **Alerts:** Dark-region alert fires FROM the other region's aggregation; regional Alertmanager still pages locally if its own rules trip.
- **Operator:** Treat dark region as UNKNOWN, not healthy or failed. Correlate with external synthetic checks before any failover decision. Failover on observability loss alone is FORBIDDEN.
- **Recovery:** Aggregation link restores → backfill via remote-write (Prometheus handles gaps).

### S7. CI/CD can deploy to only one region

- **Behavior:** Deployment lock (D14.2 #4) prevents partial promotion: pipeline detects unreachable region and HALTS the promotion, holding both regions at the current version.
- **Writes/reads/ingestion:** Unaffected.
- **User impact:** None.
- **Alerts:** Deployment failure alert (regional); version-skew watch remains green (by design: skew never occurs because deploys halt).
- **Operator:** Wait for connectivity, or invoke emergency single-region deploy procedure (requires IC approval + explicit skew acceptance + feature-flag freeze on new-version features until the other region catches up).
- **Recovery:** Connectivity restores → pipeline resumes → versions converge → skew alarm clears.

### S8. How two active database writers are impossible (summary)

1. RDS read replicas are physically incapable of accepting writes until promoted.
2. Promotion is an explicit, approved, audited AWS API action — never automatic.
3. The only application write path (ingestor) is epoch-fenced against the lease table inside the writer DB; a stale writer in a partitioned region cannot reach the promoted DB and cannot pass the epoch check against its own (now isolated) old primary.
4. If an old primary surfaces post-promotion, it is quarantined at the network level before any reconnection decision.

---

## D31. Multi-Region Recovery Objectives

Targets are PROPOSED and become production commitments only after exercise validation (operations-plan.md D33). "Automated" = executes without human action; "manual" = runbook-driven.

| # | Scenario | RTO | RPO | Customer impact | Automated recovery | Manual recovery | Dependencies | Validation |
|---|----------|-----|-----|-----------------|--------------------|--------------------|--------------|------------|
| 1 | API regional loss | 5 min | 0 | Brief errors during shift; then normal from standby | Traffic shift (F1) after hold-down | IC-directed shift if automation inhibited | Standby healthy; DNS TTL | Quarterly traffic-shift drill |
| 2 | Realtime regional loss | 5 min | ≤60s events | SSE gap; auto-reconnect; ≤50 event replay | Traffic shift; client retry logic | n/a | Standby stream active (post-leadership) | Reconnection load test |
| 3 | Ingestion regional loss | 2 min (transfer) | ≤2 cycles | Staleness ≤2-3 min then fresh | Lease expiry → standby acquires (35s) + warm cache | Voluntary transfer runbook | Lease infra; standby deploy | Dual-ingestor drill |
| 4 | Redis regional loss | 5 min | ≤60s (cache rebuild) | Higher latency (DB fallback); realtime gap ≤1 cycle | Cache rebuild on next cycle; consumer group recreate | WarmCache script | PostgreSQL available | Redis kill drill (existing pattern: `docs/sre/recovery-objectives.md`) |
| 5 | PostgreSQL regional loss (writer) | 15 min | ≤60s | 5-15 min stale reads; write pause ≤2 min | F1 traffic shift; promotion PREPARED automatically | **Promotion approved by IC+DB lead**; DSN/CNAME swap; canary write | Replica lag known; standby healthy | Promotion drill (staging) |
| 6 | Availability Zone failure | 5 min | 0 | Seconds of disruption | Multi-AZ: pod reschedule, RDS failover, ElastiCache promote (all existing AWS-managed) | n/a | Multi-AZ config (implemented) | AZ failover sim (existing quarterly plan) |
| 7 | Entire AWS region failure | 15 min | ≤60s | 5-15 min degraded; stale data served | F1 automated; F2/F3 per approval | Full orchestration workflow D16 | All MR workstreams | Full DR exercise (D33) |
| 8 | Traffic-management (Route53) failure | 30 min | 0 | Users stuck on possibly-dead region until TTL expiry | None (manager IS the automation) | Direct API/console record edit; emergency TTL pre-lowering | Console access; documented records | Tabletop + manual drill |
| 9 | Secret replication failure | 4 h (standby secret staleness) | n/a | None until failover attempted with stale secrets | Replication alarm | Manual secret sync runbook; break-glass access | Secrets Manager replica health | Replication alarm test |
| 10 | Artifact registry (ECR) regional failure | 2 h (image pull blocked for NEW pods only) | 0 | None (running pods unaffected; scaling blocked) | Cross-region replication (prevents most cases) | Pull from replica region registry (documented tag override) | ECR replication config | Replication lag check |

### D31.1 Consistency of objectives

- Scenario 7 RTO 15 min replaces the current documented 4h (`docs/dr/strategy.md`) — the improvement is entirely attributable to hot standby + pre-replicated data. The 4h remains the fallback if promotion itself fails (restore-from-backup path).
- RPO ≤60s = one ingestion cycle + typical replication lag; bounded by the lag displayed at approval time (D16 step 4).
- All objectives assume the standby region is healthy; standby-unavailable cases escalate to the existing backup-restore objectives.
