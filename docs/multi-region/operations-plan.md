# Phase 7 — Operations Plan: Observability, Testing, Chaos, Runbooks, DR Exercise, Incident Command

**Status:** Proposed — pending operations gate approval
**Parent:** [Program Plan](./program-plan.md)
**Scope:** Deliverables 25, 26, 27, 29, 30, 32, 33, 34

---

## D25. Global Observability Architecture

### D25.1 Current state (evidence)

- In-cluster: Prometheus (15d retention) with recording rules and alerts (`monitoring/prometheus/`), Grafana with provisioned dashboards (`monitoring/grafana/`), Loki, Alertmanager, postgres/redis exporters (docker-compose + Helm monitoring subchart).
- All collection is single-region. No `region` label anywhere. No remote-write. No cross-region log search.
- Tracing: Tempo referenced in architecture docs; OTel collector in the deployment view.

### D25.2 Target: hybrid (regionally independent + centrally aggregated)

**Decision: hybrid.** Rationale: pure centralization creates a single observability dependency whose failure blinds both regions (explicitly forbidden by the charter); pure regional isolation makes global incident correlation impossible.

```
REGION A (primary)                    REGION B (secondary)
┌─────────────────────────┐          ┌─────────────────────────┐
│ Prometheus (regional)   │          │ Prometheus (regional)   │
│  + external_labels:     │          │  + external_labels:     │
│    region: us-east-1    │          │    region: us-west-2    │
│ Loki (regional logs)    │          │ Loki (regional logs)    │
│ Tempo (regional traces) │          │ Tempo (regional traces) │
│ Alertmanager (regional) │          │ Alertmanager (regional) │
└──────────┬──────────────┘          └──────────┬──────────────┘
           │ remote-write (metrics only)        │
           ▼                                    ▼
      AGGREGATION POINT (us-west-2 + S3 mirror; NOT in primary)
      ┌──────────────────────────────────────────┐
      │ Global Prometheus (long retention 90d)   │
      │ Global Grafana (dashboards by region+global) │
      │ Global Alertmanager (dedup/group router) │
      └──────────────────────────────────────────┘
```

Design rules:
1. **Regional stacks are complete and independent.** A region losing the aggregation link loses nothing operationally: local dashboards, alerts, and logs keep working.
2. **Aggregation point lives OUTSIDE the primary region** (secondary region + object-storage mirror) so primary loss does not blind the failover decision-makers. Aggregation loss degrades to regional-only visibility — acceptable, alarmed, never fatal.
3. **Only metrics are aggregated.** Logs and traces stay regional (cost + sensitivity); global investigation uses region-switching in Grafana (datasource per region), not a global log store.
4. **`region` is a mandatory external label** on all metrics; recording rules and alerts are extended to preserve it (existing rules in `monitoring/prometheus/recording-rules.yml` gain `by (region)` variants).
5. **Incident evidence:** every failover workflow step snapshots the relevant panels (Prometheus query results to the incident record) — observability during the incident is supplemented by captured evidence, since dashboards themselves may be affected.
6. **Retention:** regional 15d (existing), global 90d for SLO/failover forensics; Loki regional 30d (existing policy).

---

## D26. Regional and Global Dashboards

Three dashboard families, provisioned in Grafana (extending `monitoring/grafana/dashboards/`).

### D26.1 Global Overview (audience: on-call, incident command)

| Panel | Query basis | Purpose |
|-------|-------------|---------|
| Traffic by region | `sum(rate(http_responses_total[5m])) by (region)` | Load distribution; shift verification |
| Global availability | Weighted `api:success_ratio:5m` across regions | Top-line SLO |
| Global latency p99 | `histogram_quantile(0.99, ...)` by region + merged | SLO watch |
| Error rate by region | 5xx rate by region | Spot the failing region |
| Data freshness (serving region) | `data_freshness_status` | Core product health |
| Active DB writer | `ingestion_lease_epoch` + writer-region gauge (D13) | Single-writer invariant visible |
| Active ingestor | `ingestion_leader{region}` | Leadership state |
| Replication lag | `ReplicaLag` | RPO readiness |
| Regional status | `region_status` state gauge (D6) | At-a-glance health states |
| Failover state | Orchestration audit events as annotations + current step | What is automation doing right now |

### D26.2 Regional Overview (one per region; audience: service owners)

Kubernetes health (node/pod/deployment state, HPA current/desired); API health (existing dashboard extended with region pin); realtime health (connections, broadcast ratio, consumer lag — existing `realtime_*` metrics); provider health (circuit breaker state, `provider_active` gauge, fallback counts — existing `provider_fallback_total`); PostgreSQL (exporter metrics + ReplicaLag); Redis (exporter: memory, evictions, stream length, consumer pending); deployment version (build-info metric with `version` label); regional error budget burn (multi-window burn-rate panels from existing recording rules).

### D26.3 Failover Control (audience: IC + DB lead during incidents)

Trigger signals (all D15 trigger signals live, with thresholds drawn as annotations); replication readiness (ReplicaLag + replica status + last-verified timestamp); routing weights (current Route53 weights, mirrored from the `global/dns` state output); region status (D6 states for both regions); active leadership epoch (epoch timeline — divergence = split-brain alarm); failover history (event log: every orchestration step with actor and duration, rendered from the audit trail).

---

## D27. Global Alerting Strategy

### D27.1 New multi-region alerts

| Alert | Condition | Severity | Runbook |
|-------|-----------|----------|---------|
| RegionUnavailable | `region_status{state="unavailable"} == 1` for 2m | critical/page | regional-failover runbook (D32 #1) |
| RegionDegraded | `region_status{state="degraded"}` for 10m | warning/ticket | degraded-region investigation |
| ReplicationLagHigh | ReplicaLag > 5s for 5m | warning | replication-lag runbook |
| ReplicationStopped | ReplicaLag missing/replica status != replicating for 5m | critical/page | replication-stopped runbook |
| DualWriterRisk | Two `ingestion_leader` holders OR epoch divergence | critical/page (split-brain class) | split-brain containment |
| IngestionLeadershipConflict | Acquisition failures + two cycle writers in sync logs | critical/page | leadership runbook |
| TrafficRoutingFailure | Route53 health check failing while region healthy (or vice versa) | high/page | routing runbook |
| StaleSecondary | Secondary region version != primary for >24h OR replica lag unbounded | warning | parity runbook |
| SecretReplicationFailure | Replica secret age >15m | high | secrets runbook |
| ArtifactReplicationFailure | Image digest missing in secondary registry >30m after push | high | artifact runbook |
| RegionalDeploymentFailed | Rollout stuck/failed in any region | high/page | regional rollback runbook |
| FailoverExceedingRTO | Orchestration active >15m without verification success | critical/page | failover escalation |
| FailbackInconsistency | Reconciliation mismatch during failback window | critical/page | failback abort + reconcile |

### D27.2 Grouping and deduplication

- **One page per incident, not per region:** global Alertmanager route groups by `incident_class` (e.g., `region_loss:us-east-1`); regional child alerts (API down + DB down + realtime down in the same region) collapse into the parent page via `group_by: [region, incident_class]` with 5-min group_wait.
- **Inhibition:** RegionUnavailable inhibits all component alerts for that region (they are symptoms, not new pages). DualWriterRisk inhibits ALL other alert routing until resolved (nothing else matters).
- **Cross-region dedup:** shared-infrastructure alerts (Route53, ECR replication) fire once globally, not per region.
- **Every alert maps to a runbook** (table above enforced by CI lint on alert definitions — `annotation.runbook_url` required, extending the existing alerts.yml pattern).

---

## D29. Global Load and Failover Testing

All scenarios execute in STAGING. Production testing remains component-level only (existing policy; `ALLOW_FAILURE_INJECTION` guard pattern from `scripts/incident-demo.sh`).

### D29.1 Scenarios

| # | Scenario | Method | Pass criteria |
|---|----------|--------|---------------|
| T1 | 100% traffic shift (instant) | Route53 weight 100→0 in one step | Errors <1% during shift; p99 <800ms; all clients reconnected within 5 min |
| T2 | Gradual shift 10/25/50/100 | Staged weights, 10 min soak each | No SLO breach at any stage; rollback at any stage restores baseline |
| T3 | Regional API failure | Kill all api pods in staging primary | Traffic shift fires per D6 timing; RTO ≤15 min |
| T4 | Database promotion | `promote-read-replica` on staging replica | Writes resume ≤5 min post-approval; zero writes accepted by old writer post-promotion; canary snapshot verified |
| T5 | Redis loss (secondary, post-failover) | Flush secondary Redis mid-exercise | Cache rebuild ≤1 cycle; no event loss beyond replay window |
| T6 | Ingestion leadership transfer | Force lease release | Standby leader within 60s; zero duplicate cycles (cycle_id audit); freshness gap ≤2 cycles |
| T7 | Long-lived realtime reconnection | 500 SSE clients across the shift | All clients reconnected; ≤50-event replay honored; no client stuck on dead region >10 min |
| T8 | Regional network latency | tc/netem 200ms on inter-region path (staging) | Replication survives; no false failover from lag alone |
| T9 | Provider availability asymmetry | Block provider egress from primary only | Leadership transfer to secondary restores ingestion (D18 S5) |
| T10 | Secondary cold cache | Flush secondary cache, shift 100% traffic | DB absorbs cache-miss burst; p99 <2s for ≤2 min; no 5xx >0.5% |
| T11 | Failback under load | Execute D17 at 50% synthetic load | No errors during writer migration pause (≤2 min); traffic gradual return verified |

### D29.2 Measurements (recorded per exercise)

Failover detection time (failure inject → candidate state); traffic transition time (decision → 100% served by target); error rate during transition; connection disruption count and duration (SSE clients); data freshness gap; replication lag at promotion; total recovery time (inject → verification pass); data loss (cycle_id gap audit across regions); duplicate-event rate (dedup counter deltas).

Results feed the quarterly SLO review and the failover SLO trend panel (D26.3).

---

## D30. Regional Chaos Engineering Program

Staged, manual-authorization-only, staging-first. Extends the existing component-level chaos (`sre-toolkit/inject_failures.py`, `scripts/chaos/`).

### D30.1 Levels

**Level 1 — Component (IMPLEMENTED foundation, extend regionally)**
Terminate application pods (existing kubectl patterns); restart nodes (drain + delete); pause stream consumers (scale realtime to 0); introduce provider latency (existing mock provider `MOCK_DELAY`, `cmd/mockprovider`).

**Level 2 — Dependency**
Block Redis (network policy injection — existing `networkpolicy.yaml` patterns); block PostgreSQL from app pods; block provider egress (egress policy); delay replication (parameter group tweak on staging replica).

**Level 3 — Regional**
Remove region from routing (Route53 weight 0 — the T1/T2 mechanism); simulate regional ingress failure (disable ingress controller); deny inter-region connectivity (VPC peering/TGW route withdrawal in staging); stop regional ingestion (lease force-release); simulate control-plane unavailability (revoke CI deploy role for a window).

**Level 4 — Failover**
Promote secondary database (T4); transfer traffic (T1/T2); transfer ingestion leadership (T6); verify global recovery (full D33 exercise).

### D30.2 Mandatory experiment controls (every experiment)

| Control | Requirement |
|---------|-------------|
| Safety guard | `ALLOW_FAILURE_INJECTION=true` explicit (existing pattern) + experiment ID tag on all injected faults |
| Blast-radius limit | Staging only for L3/L4; L1/L2 production allowed only for pod-termination class with PDB protection |
| Abort condition | Pre-defined metric threshold (e.g., error rate >5% outside target scope) triggers automatic cleanup script |
| Expected behavior | Documented hypothesis BEFORE execution; deviation = experiment failure, not system failure |
| Metrics | Dedicated experiment dashboard (D26 pattern) with injection/abort annotations |
| Cleanup | Automated reset script per experiment (`make incident-reset` pattern extended); verified clean state after |
| Post-experiment review | Written findings within 48h; action items tracked in backlog |

**Absolute rule:** regional chaos (L3/L4) is NEVER automated against production. Production L3/L4 requires written executive approval and is out of Phase 7 scope entirely.

---

## D32. Regional DR Runbooks

Thirteen runbooks to create under `docs/runbooks/` (extending the existing 13 single-region runbooks). Each MUST contain: exact evidence commands with expected output, decision points with explicit criteria, actor identification, rollback steps, and post-action verification.

| # | Runbook | Key decision points |
|---|---------|---------------------|
| 1 | Declare regional incident | Multi-signal validation checklist; external vantage confirmation; severity assignment; IC activation |
| 2 | Shift stateless traffic | Health state ≥ candidate; standby healthy confirmation; weight commands (Terraform + console fallback); post-shift verification |
| 3 | Promote PostgreSQL standby | Lag reading + RPO acceptance (IC + DB lead); freeze confirmation; promote command; post-promotion write canary |
| 4 | Activate secondary Redis | Verify instance health; WarmCache execution; stream group creation; cache-hit verification |
| 5 | Transfer ingestion leadership | Voluntary vs forced release decision; epoch verification; first-cycle verification; freshness confirmation |
| 6 | Validate data freshness | Freshness gauge check; cycle_id continuity audit; provider data spot-check against public source |
| 7 | Reconnect realtime clients | Connection count verification; replay behavior check; stuck-client detection query |
| 8 | Verify global routing | DNS resolution from 3 external vantages; weight state vs intended state; health check status |
| 9 | Halt unsafe writes | Freeze verification (deployment lock + write paths); quarantine check on old writer; fencing alarm silence confirmation |
| 10 | Perform emergency deployment | IC approval record; single-region deploy procedure; skew acceptance; catch-up plan |
| 11 | Begin failback | Entry-condition audit (D17.1, all six); window scheduling; role decision (writer migration vs traffic-only) |
| 12 | Reconcile data | cycle_id range comparison commands; snapshot diff tool usage; conflict resolution authority |
| 13 | Restore normal topology | Standby re-establishment verification; DR readiness checklist; exercise-level validation pass |

---

## D33. Regional Disaster-Recovery Exercise

### D33.1 Scenario (staging, portfolio-safe)

> The primary staging region (us-east-1) experiences a simulated multi-service failure: API traffic becomes unavailable, database connectivity fails, and realtime delivery stops. The team validates the secondary region, promotes the standby database, transfers ingestion leadership, shifts global traffic, and recovers within RTO 15 minutes / RPO 60 seconds.

### D33.2 Exercise design (17 phases)

| # | Phase | Actions | Evidence |
|---|-------|---------|----------|
| 1 | Preparation | Freeze deploys; confirm standby healthy (D6); brief participants; pre-stage dashboards; snapshot pre-state | Readiness checklist signed |
| 2 | Success criteria | RTO ≤15m; RPO ≤60s; zero dual-writer events; all SSE clients recovered; no data loss beyond accepted lag | Printed in exercise brief |
| 3 | Safety boundaries | Staging only; abort thresholds (error rate in PRIMARY unaffected scope, cost ceiling); kill-switch script tested | Abort script dry-run log |
| 4 | Failure injection | (a) Scale primary api/realtime/frontend to 0; (b) block app→RDS via SG rule; (c) block provider egress from primary | Injection timestamps recorded |
| 5 | Detection | Health model transitions: degraded → candidate → unavailable; alarms fire | Alert timeline screenshot |
| 6 | Incident declaration | IC declares SEV1 regional; channel opened; roles assigned (D34) | Incident record |
| 7 | Database readiness | ReplicaLag read; replica status verified; RPO estimate presented | Lag value at decision time |
| 8 | Promotion | IC + DB lead approve; promote executed; writer canary written and read back | Promotion timestamp; canary row |
| 9 | Traffic shift | Weights to secondary 100%; DNS verified from external vantages | Resolution evidence |
| 10 | Realtime reconnection | Client count in secondary matches pre-failure ±5%; replay verified | Connection panel |
| 11 | Ingestion recovery | Leadership acquired in secondary; WarmCache done; first cycle verified | cycle_id continuity |
| 12 | Data validation | Snapshot counts reconciled; freshness green; dedup counters checked | Reconciliation report |
| 13 | User-impact analysis | Synthetic user journey results during window; error budget consumed | Journey report |
| 14 | RTO/RPO measurement | Detection→verification time; accepted lag vs actual gap | Measurement table |
| 15 | Stabilization | 1h soak; capacity check; heightened alerting armed | Soak dashboard |
| 16 | Failback plan | Rebuild primary as replica; schedule failback window (or execute abbreviated failback in staging) | Failback ticket |
| 17 | Retrospective | Findings, timeline gaps, action items → backlog; update runbooks | Retro document |

### D33.3 Portfolio-safe properties

- Executable against staging with zero production impact (staging has its own region pair at reduced size).
- Total exercise duration ≤4h including retro; failure injection is reversible within minutes (scale back up, remove SG rule).
- Scripted variant (`scripts/chaos/` extension) allows solo rehearsal; full variant requires 3+ participants for role coverage.
- Evidence artifacts (dashboards, timelines, measurements) are the portfolio demonstration material (execution-plan.md D45).

---

## D34. Global Incident Command Model

### D34.1 Roles (regional incident)

| Role | Responsibility | Decision authority |
|------|----------------|--------------------|
| Incident Commander (IC) | Owns the incident; runs the workflow; approves/rejects failover steps | Final call on F1 timing, F4 declaration; veto on any step |
| Operations Lead | Executes traffic and orchestration steps; owns dashboards during incident | Executes approved steps; may halt on safety threshold |
| Database Lead | Owns promotion decision input: lag reading, RPO estimate, post-promotion verification | Co-approves F2 with IC; sole authority on DB-level commands |
| Application Lead | Service-level diagnosis; deploy rollback decisions; feature-flag actions | Rollback of code (not infra) within their scope |
| Communications Lead | Status page, stakeholder updates, customer-facing language | Owns external messaging |
| Scribe | Timeline capture (augmented by orchestration audit log) | None (records decisions) |
| Executive Liaison | Escalation bridge; approves RTO-exceeding continuations | SEV1 duration extensions |

Small-team adaptation: IC absorbs Communications; App Lead absorbs Scribe; minimum viable incident = 3 people (IC, Ops+DB combined, App).

### D34.2 Process definitions

- **Declaration criteria:** RegionUnavailable state, OR DualWriterRisk (immediate), OR operator judgment on multi-signal evidence. Declaration is cheap; undeclared simmering is the anti-pattern.
- **Severity levels:** Extend existing matrix (`docs/operations/severity-matrix.md`): SEV1 = regional loss/dual-writer (page + full command); SEV2 = single-region degradation with SLO burn >6x; SEV3 = replication/parity issues; SEV4 = latent.
- **Communication cadence:** SEV1: every 15 min until stable, then 30 min; SEV2: every 30 min. Updates go to incident channel + status page (SEV1/2).
- **Failover approval:** F1 automated per policy (no human in loop, but IC may inhibit). F2/F3: IC + DB Lead verbal + recorded confirmation in channel; orchestration requires the approval reference before proceeding. Emergency override per D16.3.
- **Emergency-change approval:** IC approves; change executed with audit tag; retroactive review within 48h mandatory.
- **Closure criteria:** All health states green for 60 min; data reconciled; failback plan filed (if topology changed); communications sent; retro scheduled (within 48h).
