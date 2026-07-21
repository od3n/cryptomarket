# Phase 7 — Global Resilience and Multi-Region Program Plan

**Status:** Proposed — pending architecture gate approval
**Scope:** Program planning only; no infrastructure implementation
**Companion documents:** [Data Strategy](./data-strategy.md) · [Failover Strategy](./failover-strategy.md) · [Infrastructure Plan](./infrastructure-plan.md) · [Operations Plan](./operations-plan.md) · [Execution Plan](./execution-plan.md) · [ADR-020](../adr/020-multi-region-architecture.md)

---

## 1. Executive Summary

This document is the master plan for transforming the Crypto Market Data Platform from a highly available single-region system into a globally resilient platform capable of surviving a full AWS region failure.

**Current position (evidence-based):** The platform is a well-structured single-region system. It has multi-AZ production infrastructure (`deploy/terraform/environments/prod/main.tf`), defined SLOs with error budgets (`docs/sre/slos.md`), component-level resilience (circuit breakers, provider fallback in `internal/resilience/`), and a documented backup-restore DR strategy targeting RTO 4h / RPO 24h for full region loss (`docs/dr/strategy.md`). It has **no** multi-region capability: no region identity in application code, no cross-region data replication, no global traffic management, no ingestion coordination, and no CI/CD pipeline files in the repository (`.github/workflows/` is empty).

**Recommended target:** Active-Passive Hot Standby (Option B). One region serves all production traffic; a second region remains continuously deployed, data-replicated, and validated, able to take over within minutes rather than hours. This recommendation is driven by hard constraints in the current codebase: a single-writer PostgreSQL schema with `BIGSERIAL` identifiers, an ingestor with no leader election, and Redis used as ephemeral cache plus event stream. Active-active would require rebuilding all three; hot standby requires hardening them.

**Program outcome:** Regional failover validated in staging within RTO 15 minutes / RPO 60 seconds, with failback demonstrated, all 37 deliverables complete, and implemented behavior clearly distinguished from design-only work.

### Classification legend used throughout this program

| Label | Meaning |
|-------|---------|
| IMPLEMENTED | Code/config exists in the repository and is verified working |
| REUSABLE | Exists but requires changes for multi-region use |
| PROPOSED | Designed in this program; not yet implemented |
| VALIDATED | To be proven by experiment/exercise before being claimed |
| ASPIRATIONAL | Future option explicitly out of current scope |

---

## 2. Program Mission

Transform the platform from a highly available single-region system into a globally resilient platform that survives major regional failures while maintaining defined service levels.

The program demonstrates competency across:

- multi-region architecture and global traffic management;
- regional health evaluation and failover/failback orchestration;
- active-passive design with explicit active-active boundaries;
- cross-region data replication and consistency trade-offs;
- recovery orchestration and regional deployment automation;
- global observability, resilience testing, cost-aware architecture;
- operational governance suitable for senior/staff-level SRE, distributed systems, platform, cloud, DR, database, routing, and incident-management discussions.

**Explicit non-goals** (see Out of Scope, §9): multi-writer SQL, custom consensus, cross-cloud, global Kafka, autonomous production failover, automated production chaos.

---

## 3. Deliverable 1 — Current-State Regional Readiness Assessment

Scoring: 0 = none, 1 = ad hoc, 2 = partial, 3 = defined, 4 = production-proven, 5 = continuously verified.

**Overall score: 13 / 90 (14%).** The platform is not region-ready, but the gaps are structural and addressable; no architectural rewrite is required.

### 3.1 Stateless-service portability — Score 4

- **Evidence:** All four services (`market-api`, `market-ingestor`, `realtime-gateway`, `market-frontend`) take configuration entirely from environment variables (`internal/config/config.go`); no filesystem state, no host affinity. Docker images built from `deploy/docker/Dockerfile`.
- **Blocking gaps:** Frontend bakes `API_URL`/`REALTIME_URL` at build time (`docker-compose.yml` build args); no `REGION` identity variable exists.
- **Recommended work:** Add `REGION` and `REGION_ROLE` (primary/secondary) to config; make frontend URLs runtime-resolvable.
- **Dependencies:** None. **Risk:** Low. **Effort:** S (1-2 days).

### 3.2 Kubernetes portability — Score 4

- **Evidence:** Single Helm chart (`deploy/helm/cryptomarket/`) with per-environment values, HPA, PDBs, network policies, topology-spread-ready deployments; kind-based local cluster validated (`Makefile` kind targets).
- **Blocking gaps:** Chart assumes in-cluster Postgres/Redis for dev and external endpoints via secrets for prod; no regional values overlay; no cluster federation.
- **Recommended work:** Add `values-prod-secondary.yaml`; parameterize all regional endpoints.
- **Dependencies:** Terraform regionalization. **Risk:** Low. **Effort:** M (3-5 days).

### 3.3 Terraform regionalization — Score 2

- **Evidence:** Modular structure exists (13 modules under `deploy/terraform/modules/`); per-environment state (`deploy/terraform/environments/prod/backend.tf`, key `prod/terraform.tfstate`); region is a variable defaulting to `us-east-1`.
- **Blocking gaps:** Single provider block, no provider aliases, no `global/` layer, no secondary environment directory, no ECR module, no Route53 failover records, state bucket itself is single-region.
- **Recommended work:** Create `global/` and per-region environment roots; add ECR replication module; split state per region.
- **Dependencies:** None. **Risk:** Medium (state migration). **Effort:** L (2-3 weeks).

### 3.4 Image distribution — Score 1

- **Evidence:** Docs reference ECR push (`docs/architecture/overview.md` CI/CD diagram) but no ECR Terraform module and no workflow files exist in-repo; local builds via `make build`.
- **Blocking gaps:** No registry defined in code; no cross-region replication; standby region would depend on primary region's registry.
- **Recommended work:** Add ECR module with `replication_configuration` to secondary region; attestations + SBOM in pipeline.
- **Dependencies:** CI/CD pipeline build-out. **Risk:** Medium. **Effort:** M.

### 3.5 Secret replication — Score 1

- **Evidence:** `deploy/terraform/modules/secrets/main.tf` creates Secrets Manager secrets with KMS encryption; `secrets-rotation` module exists.
- **Blocking gaps:** No `replica` blocks; KMS keys are regional with no multi-region key; rotation Lambda is single-region.
- **Recommended work:** Multi-region KMS key; Secrets Manager replica in secondary; rotation runbook for failover.
- **Dependencies:** KMS decision. **Risk:** Medium. **Effort:** M.

### 3.6 PostgreSQL replication readiness — Score 2

- **Evidence:** RDS PostgreSQL 16.2, Multi-AZ, 35-day retention, deletion protection, PITR-capable (`deploy/terraform/modules/rds/main.tf`); partitioned snapshots with UUID PKs (`migrations/003_partition_snapshots.up.sql`).
- **Blocking gaps:** No cross-region read replica; `coins` and `provider_sync_logs` still `BIGSERIAL` (`migrations/001_initial_schema.up.sql`); no promotion runbook; no lag monitoring.
- **Recommended work:** RDS cross-region replica; UUIDv7 migration for remaining serial IDs; promotion + failback procedures; lag alarms.
- **Dependencies:** Identifier migration. **Risk:** Medium. **Effort:** L.

### 3.7 Redis recovery readiness — Score 3

- **Evidence:** Redis explicitly treated as cache with 5-minute TTL (`internal/cache/redis.go`); documented rebuild-from-PostgreSQL path (`docs/sre/recovery-objectives.md`); consumer group self-recreates (`internal/stream/consumer.go` `EnsureGroup`).
- **Blocking gaps:** Rebuild path documented but never exercised; stream replay limited to 50 events (`internal/realtime/server.go`); hardcoded consumer name `realtime-1` collides across replicas.
- **Recommended work:** Unique consumer names (pod/region-derived); cache-warm script; exercise cold-Redis failover.
- **Dependencies:** None. **Risk:** Low. **Effort:** S-M.

### 3.8 Event idempotency — Score 2

- **Evidence:** Events carry UUIDv4 `event_id` (`internal/cache/redis.go` `PublishEvent`); consumer deduplicates by event ID (`internal/stream/consumer.go`).
- **Blocking gaps:** Dedup map is in-memory, per-process, and **resets entirely at 10,000 entries**; snapshot inserts have no unique constraint, so re-ingestion duplicates rows; no `dedup_key` in schema.
- **Recommended work:** Durable dedup (Redis SET with TTL or DB unique constraint on `(provider, symbol, captured_at_bucket)`); idempotent insert path.
- **Dependencies:** None. **Risk:** Medium (data quality). **Effort:** M.

### 3.9 Global traffic management — Score 1

- **Evidence:** `deploy/terraform/modules/dns/` exists (Route53 zone + records); ingress per region (`deploy/helm/cryptomarket/templates/ingress.yaml`).
- **Blocking gaps:** No health-check-based failover routing, no weighted records, no latency-based routing, no DNS TTL strategy for failover.
- **Recommended work:** Route53 failover + weighted record sets; health checks against regional API `/ready`; TTL reduction policy.
- **Dependencies:** Secondary region ALB. **Risk:** Low. **Effort:** M.

### 3.10 Regional observability — Score 2

- **Evidence:** Full single-region stack: Prometheus with recording rules and alerts (`monitoring/prometheus/`), Grafana dashboards, Loki, Alertmanager, postgres/redis exporters.
- **Blocking gaps:** All collectors are regional; no remote-write aggregation; dashboards have no `region` label; alert routing is single-tenant.
- **Recommended work:** Add `region` external label; remote-write to a region-independent aggregation point; global dashboards.
- **Dependencies:** None. **Risk:** Low. **Effort:** M.

### 3.11 Regional deployment — Score 1

- **Evidence:** Canary deployment template exists (`deploy/helm/cryptomarket/templates/api-deployment-canary.yaml`); release-please configured (`.release-please-config.json`); canary pipeline described in docs.
- **Blocking gaps:** No workflow files in `.github/workflows/`; deployment tooling is documentation-only; no multi-region promotion sequence.
- **Recommended work:** Implement CI/CD workflows; define primary/secondary promotion order.
- **Dependencies:** Image registry. **Risk:** Medium. **Effort:** L.

### 3.12 Regional rollback — Score 2

- **Evidence:** Helm release semantics support `rollout undo`; runbook exists for bad deploys (`docs/runbooks/api-unavailable.md`).
- **Blocking gaps:** No cross-region version tracking; no version-skew policy; rollback during failover is undefined.
- **Recommended work:** Version manifest per region; skew alarm; rollback runbook per region.
- **Dependencies:** Regional deployment. **Risk:** Low. **Effort:** S-M.

### 3.13 Backup portability — Score 2

- **Evidence:** RDS automated backups (35 days) + S3 backup module with lifecycle policies (`deploy/terraform/modules/s3/`); backup strategy and restore verification documented (`docs/backup/`).
- **Blocking gaps:** No cross-region snapshot copy automation; restore verification is single-region.
- **Recommended work:** Automated cross-region snapshot copy; quarterly cross-region restore drill.
- **Dependencies:** None. **Risk:** Low. **Effort:** S-M.

### 3.14 Failover automation — Score 0

- **Evidence:** None. DR strategy is manual rebuild ("Provision infrastructure in DR region via Terraform", `docs/dr/strategy.md` scenario 7).
- **Blocking gaps:** No failover orchestration, no runbook for promotion, no traffic-shift automation.
- **Recommended work:** Full failover workflow design + staged automation (see failover-strategy.md).
- **Dependencies:** Most other workstreams. **Risk:** High. **Effort:** L.

### 3.15 Failback readiness — Score 0

- **Evidence:** None. No failback procedure exists anywhere in the repository.
- **Blocking gaps:** No reconciliation tooling for writes made in the secondary region; no role-reversal procedure.
- **Recommended work:** Failback design including reverse replication and observation window.
- **Dependencies:** Failover automation. **Risk:** High. **Effort:** L.

### 3.16 Network-partition tolerance — Score 1

- **Evidence:** Services degrade gracefully on dependency loss (API falls back to Postgres when Redis is down; circuit breakers isolate provider faults); chaos scripts exist (`sre-toolkit/inject_failures.py`, `scripts/chaos/`).
- **Blocking gaps:** No partition analysis; ingestor has no leadership, so a partitioned dual-ingestor scenario is unguarded; no quorum model.
- **Recommended work:** Partition scenario analysis (failover-strategy.md §D18); leadership lease.
- **Dependencies:** Ingestion leadership. **Risk:** High. **Effort:** M.

### 3.17 Incident readiness — Score 3

- **Evidence:** Severity matrix, on-call doc, 13 runbooks (`docs/runbooks/`), postmortem template and one completed postmortem (`docs/postmortems/001-primary-provider-rate-limit.md`), tabletop simulation records (`docs/dr/tabletop-simulations.md`), incident demo script (`scripts/incident-demo.sh`).
- **Blocking gaps:** No incident command model for multi-region; no regional-declaration criteria; runbooks are single-region.
- **Recommended work:** Global incident command model; regional DR runbooks.
- **Dependencies:** None. **Risk:** Low. **Effort:** M.

### 3.18 Cost sustainability — Score 3

- **Evidence:** Documented per-environment cost model with production baseline ~$1,588/month (`docs/cost/estimates.md`); cost allocation tags; optimization roadmap.
- **Blocking gaps:** No multi-region cost model; no standby-cost policy; no budget alerts for replication traffic.
- **Recommended work:** Incremental multi-region cost model (infrastructure-plan.md §D35).
- **Dependencies:** Architecture decision. **Risk:** Low. **Effort:** S.

### 3.19 Assessment summary

| Category | Score | Category | Score |
|----------|-------|----------|-------|
| Stateless-service portability | 4 | Backup portability | 2 |
| Kubernetes portability | 4 | Failover automation | 0 |
| Terraform regionalization | 2 | Failback readiness | 0 |
| Image distribution | 1 | Network-partition tolerance | 1 |
| Secret replication | 1 | Incident readiness | 3 |
| PostgreSQL replication readiness | 2 | Cost sustainability | 3 |
| Redis recovery readiness | 3 | | |
| Event idempotency | 2 | **Total** | **13/90 → 14%** |
| Global traffic management | 1 | | |
| Regional observability | 2 | | |
| Regional deployment | 1 | | |
| Regional rollback | 2 | | |

---

## 4. Deliverable 2 — Regional Capability Map

Classification: **I** = implemented, **R** = reusable with changes, **P** = partially ready, **M** = missing, **D** = intentionally deferred.

### 4.1 Global Edge

| Capability | Class | Evidence / Gap |
|------------|-------|----------------|
| DNS | P | Route53 module exists (`modules/dns/`); no failover/weighted/latency records |
| CDN | M | No CloudFront/CDN anywhere in repo |
| TLS | I | ACM module + ingress TLS (`modules/acm/`, Helm ingress template) — regional only |
| WAF | I | WAF module with rate rules (`modules/waf/`) — regional only |
| Rate limiting | I | Ingress RPS limits + WAF rules + app-level limiter (`internal/resilience/ratelimit.go`) |
| Origin health | P | App `/health`+`/ready` endpoints exist; no Route53 health checks wired |
| Traffic steering | M | No steering policy of any kind |

### 4.2 Regional Compute

| Capability | Class | Evidence / Gap |
|------------|-------|----------------|
| EKS | I | EKS module v1.29, managed node groups (`modules/eks/`) — single cluster |
| Workload scheduling | I | HPA, PDB, resource requests/limits in Helm chart |
| Autoscaling | I | HPA on CPU/memory for api/realtime/frontend; node group 3-12 |
| Ingress | I | NGINX ingress template with TLS and per-service hosts |
| Service discovery | I | In-cluster DNS; frontend uses in-cluster service URLs |
| Configuration | R | ConfigMap + env (`templates/configmap.yaml`); needs regional overlay |
| Secrets | R | Helm secret template + Secrets Manager; no replication |

### 4.3 Data

| Capability | Class | Evidence / Gap |
|------------|-------|----------------|
| PostgreSQL | I | RDS 16.2 Multi-AZ, PITR, partitioned snapshots |
| Redis | I | ElastiCache 3-node; app treats it as ephemeral cache |
| Event streams | R | Redis Streams with consumer groups; no region awareness, no schema version |
| Object storage | I | S3 module with lifecycle + versioning |
| Backups | R | RDS snapshots + S3; single-region only |
| Replication | M | No cross-region replication for any data store |
| Reconciliation | P | Price reconciliation tool exists (`sre-toolkit/reconcile_prices.py`) — provider-level, not region-level |

### 4.4 Delivery

| Capability | Class | Evidence / Gap |
|------------|-------|----------------|
| Multi-region Terraform | M | Single-region roots only |
| Regional deployment workflows | M | `.github/workflows/` empty |
| Promotion | P | Canary template + release-please config; no pipeline |
| Rollback | R | Helm rollback semantics; no cross-region awareness |
| Artifact distribution | M | No ECR module, no replication |

### 4.5 Operations

| Capability | Class | Evidence / Gap |
|------------|-------|----------------|
| Global dashboards | M | Dashboards are single-region (`monitoring/grafana/`) |
| Regional alerts | I | Prometheus alerts + burn-rate rules (`monitoring/alerts/`, recording rules) |
| Failover controls | M | None |
| Incident coordination | R | Severity matrix + on-call + postmortems; single-region scope |
| DR exercises | P | Tabletop records + game-day template exist; no regional exercise |

---

## 5. Deliverable 3 — Target Multi-Region Architecture Options

### 5.1 Option A — Warm Standby

One active region; a provisioned but scaled-down standby with asynchronous replication; traffic shifts only on failure.

| Criterion | Assessment |
|-----------|------------|
| Availability | ~99.95% (single-region + recovery) |
| RTO | 30-60 min (scale-up + warm caches + validation) |
| RPO | ≤ 5 min (async replication lag) |
| Cost | +35-45% of prod baseline (~$550-700/mo) |
| Operational complexity | Low-moderate |
| Data-consistency complexity | Low (single writer) |
| Deployment complexity | Moderate (standby drift risk) |
| Observability | Moderate (standby partially dark) |
| Failure modes | Standby rot: capacity/config drift discovered during failover |
| Testing burden | High per-test cost (must scale up to test) |
| Suitability | Acceptable but fails the "continuously ready" requirement |

### 5.2 Option B — Active-Passive Hot Standby (RECOMMENDED)

Both regions fully deployed; one serves production traffic; standby is continuously ready, data-replicated, and regularly validated; semi-automated failover with human approval.

| Criterion | Assessment |
|-----------|------------|
| Availability | ~99.99% achievable for stateless tier |
| RTO | 5-15 min (promotion + traffic shift + validation) |
| RPO | ≤ 60 s (continuous async replication, measured lag) |
| Cost | +70-90% of prod baseline (~$1,100-1,400/mo) |
| Operational complexity | Moderate |
| Data-consistency complexity | Low-moderate (single writer; lag management) |
| Deployment complexity | Moderate (both regions always deployed) |
| Observability | High (both regions fully instrumented) |
| Failure modes | Failover procedure failure; replication lag spike at failover time |
| Testing burden | Low per-test (standby is live; regular drills cheap) |
| Suitability | **Best fit** — matches single-writer DB and leader-based ingestion |

### 5.3 Option C — Active-Active

Both regions serve production traffic simultaneously with coordinated writes.

| Criterion | Assessment |
|-----------|------------|
| Availability | Highest theoretical (~99.995%+) |
| RTO | Near-zero for stateless tier |
| RPO | 0 for replicated data only if synchronous — cross-region sync adds 50-100ms write latency |
| Cost | +100-130% of prod baseline |
| Operational complexity | Very high |
| Data-consistency complexity | **Very high** — requires conflict resolution for snapshot writes, global ingestion coordination, shared consumer groups |
| Deployment complexity | High (simultaneous compatible deploys) |
| Observability | Very high burden (global dedup of metrics/events) |
| Failure modes | Split-brain, write conflicts, cascading cross-region failures |
| Testing burden | Very high (combinatorial failure space) |
| Suitability | **Not justified.** Current schema (`BIGSERIAL` coins), ingestor (no leadership), and Redis Streams (region-local consumer groups) all assume single-region. The platform is read-heavy with a single natural write path — the core benefit of active-active (local writes) does not apply. |

### 5.4 Decision

**Option B — Active-Passive Hot Standby** is recommended, with a documented evolution path to selective active-active (read replicas serving regional history queries) only if latency or growth demands it. Rationale:

1. The write path is inherently centralized (one ingestor writing snapshots); active-active adds conflict complexity for zero functional gain.
2. Hot standby cuts region-loss RTO from 4 hours (current) to under 15 minutes at moderate cost.
3. The standby doubles as a staging-adjacent validation environment, paying continuous rent.
4. Every gap identified in §3 is fixable within Option B without schema re-architecture.

---

## 6. Deliverable 4 — Recommended Target Architecture

### 6.1 Topology

```
                    Route 53 (failover + weighted records, health-checked)
                         │
            ┌────────────┴────────────┐
            ▼                         ▼
   PRIMARY (us-east-1)       SECONDARY (us-west-2)
   ┌──────────────────┐     ┌──────────────────┐
   │ EKS              │     │ EKS              │
   │  api (active)    │     │  api (hot, idle) │
   │  realtime (act.) │     │  realtime (hot)  │
   │  frontend (act.) │     │  frontend (hot)  │
   │  ingestor LEADER │     │  ingestor STANDBY│
   └───────┬──────────┘     └───────┬──────────┘
           │                        │
   RDS PRIMARY ──async repl──► RDS READ REPLICA (promotable)
   ElastiCache (local)         ElastiCache (local, rebuilt on demand)
   ECR (source) ──replication──► ECR (replica)
   Secrets Manager ──replica──► Secrets Manager (replica)
```

### 6.2 Core principles

1. **Single database writer at all times.** Writes go to the RDS primary in the active region. The replica is read-only until explicitly promoted. Promotion is fenced: the old primary is demoted or isolated before the replica is promoted.
2. **Single ingestion leader at all times.** A lease (PostgreSQL advisory lock + monotonic epoch) guarantees exactly one region ingests from providers. The standby ingestor holds a hot loop that acquires the lease only after leadership transfer.
3. **Redis is region-local and ephemeral.** No cross-region Redis replication. On failover the secondary's cache is rebuilt from the (promoted) database within one ingestion cycle; realtime clients reconnect and receive fresh events.
4. **Traffic steering is decoupled from data promotion.** Traffic can shift before, after, or without DB promotion depending on the failure class (§ failover-strategy.md D15).
5. **Every component in the secondary region is independently deployable** — artifacts, secrets, and configuration are replicated such that the secondary never depends on the primary being alive.

### 6.3 What this architecture does NOT claim

- It does not provide zero-downtime regional failover (target: <15 min, measured).
- It does not replicate Redis Streams across regions.
- It does not automate production failover without human approval (automation executes only pre-approved steps; promotion requires operator confirmation).
- It is not validated until the staging DR exercise passes (§ operations-plan.md D33).

---

## 7. Deliverable 5 — Service Placement Strategy

| Service | Regional role | Scaling | Failover behavior | Consistency needs | State ownership | Deployment |
|---------|--------------|---------|-------------------|-------------------|-----------------|------------|
| **market-api** | Active in every region | HPA 2-10 per region | Stateless: traffic shift only; reads replica until promotion completes | Read-your-writes not required (cache-first reads) | None (reads Redis/PG) | Simultaneous, version-locked |
| **market-frontend** | Active in every region | HPA 2-6 per region | Stateless static+SSR: traffic shift only | None | None | Simultaneous |
| **realtime-gateway** | Active in every region | HPA 2-8 per region | Traffic shift; clients reconnect via SSE `Last-Event-ID`; replay from local stream | At-least-once delivery, dedup by event_id | Consumer group offset (region-local, disposable) | Simultaneous |
| **market-ingestor** | Regionally active with coordination | 1 replica per region (leader + standby) | Lease transfer; standby acquires leadership after promotion | Must be sole writer of snapshots/sync logs | Lease + epoch in PostgreSQL | Simultaneous; role from lease |
| **Migration job** | Active in one region only (writer region) | 1, run-to-completion | Runs against promoted DB post-failover if pending | Strong (schema must converge before app promotion) | schema_migrations table | Sequential: writer region first |
| **Reconciliation tooling** | Globally shared (operator-run) | Ad hoc | Compares regions post-failover | Eventual | None | Artifact, not service |
| **Observability (Prometheus/Grafana/Loki)** | Active in every region + global aggregation | Regional | Regional stacks survive independently; aggregation point is in a third location or the secondary | Eventual (metrics) | Local TSDB | Simultaneous |
| **Alerting (Alertmanager)** | Active in every region; global dedup | Regional | Regional alerts fire locally; global router dedups cross-region pages | Eventual | Alert state local | Simultaneous |
| **Mock provider** | Standby only (test tool) | 1 | Not part of failover | None | None | On demand |
| **Operational endpoints** (`/health`, `/ready`, `/metrics`, feature flags) | Active in every region | n/a | Used BY failover (health signals) | None | Feature flags env-driven (`internal/featureflags/flags.go`) | Simultaneous |

---

## 8. Core Program Questions — Answers

1. **Which services can safely run active-active?** market-api, market-frontend, realtime-gateway — all stateless or region-local-state. They will run hot in both regions but serve traffic only in the active region under Option B.
2. **Which services should remain active-passive?** market-ingestor (write leadership) and the migration job (schema writer).
3. **What is the source of truth for each data type?** PostgreSQL for all durable data (coins, snapshots, sync logs, config-like state); Redis for ephemeral latest-value cache and realtime delivery; Git for configuration; providers for raw market truth.
4. **Which data requires synchronous consistency?** Schema migrations and lease/epoch records (both in PostgreSQL, single writer). No cross-region synchronous replication is required or proposed.
5. **Which data can tolerate eventual consistency?** Price snapshots (async replication lag ≤60s), cache values (rebuildable), provider health history, sync logs, aggregates.
6. **How are duplicate events prevented or reconciled?** Event-level: consumer dedup by `event_id` (to be hardened with durable store). Write-level: single ingestor leadership prevents duplicate snapshot writes; a unique constraint on `(coin_id, provider, captured_at)` bucket will make inserts idempotent.
7. **How are conflicting writes resolved?** By construction: only the lease-holding ingestor in the writer region writes. Post-failover, writes made in the new region are authoritative; the old region is fenced before re-admission.
8. **How is provider ingestion coordinated across regions?** PostgreSQL advisory-lock lease with TTL and monotonic epoch; standby polls but cannot acquire while the leader renews. Fencing token (epoch) is checked on every write path.
9. **How is split-brain prevented?** Single-writer topology; lease expiry before acquisition (never steal a live lease); epoch fencing rejects stale writes; DB promotion demotes/isolates the old primary; traffic failover is independent of and sequenced after data promotion.
10. **What triggers regional failover?** Composite regional health score below threshold for a sustained window, or operator declaration. Classes separated: traffic-only, DB promotion, ingestion transfer, full disaster (§ failover-strategy.md D15).
11. **Who or what approves failover?** Automated: traffic shift (pre-approved by policy when health criteria met). Human: DB promotion and full disaster declaration (Incident Commander, advised by Database Lead).
12. **How is failback performed?** Deliberately, never urgently: verify primary health ≥24h, establish reverse replication, reconcile writes, transfer leadership during low-traffic window, shift traffic gradually (§ failover-strategy.md D17).
13. **What happens during partial network partitions?** Modeled in failover-strategy.md §D18. Summary: the side without the DB writer serves stale cache read-only; ingestion pauses rather than forking; the lease expires to exactly one holder.
14. **What customer-visible degradation is acceptable?** During failover: ≤15 min of stale-but-served data (cache TTL 5 min + rebuild), brief realtime gap with automatic reconnection, no data loss beyond RPO 60s. Defined in degraded-mode semantics (existing ADR-013).
15. **How are SLOs measured globally and regionally?** Per-region SLIs with a `region` label; global SLO = user-weighted composite; failover SLOs (detection time, shift time, promotion time) tracked separately (§ failover-strategy.md D7).
16. **How is cost controlled?** Hot standby at reduced node counts (burst on failover via autoscaling headroom); budget alerts on cross-region transfer; quarterly cost review; standby doubles as validation environment (§ infrastructure-plan.md D35).
17. **How is multi-region behavior tested without risking production?** Staging-only failover exercises; production chaos limited to component-level (existing policy); traffic-weight experiments in staging; all regional chaos requires manual authorization (existing `ALLOW_FAILURE_INJECTION` guard pattern).

---

## 9. Out of Scope

Per program charter, the following are explicitly excluded unless separately approved:

- Globally distributed multi-writer SQL; custom consensus algorithms; home-grown database replication.
- Cross-region service mesh; multi-cloud failover; cross-cloud replication; global Kafka.
- Globally distributed user authentication.
- Automated production chaos; fully autonomous regional failover without safety controls.
- Compliance certification; financial trading functionality.

Complexity is not added for architectural novelty.

---

## 10. Document Map

| Deliverables | Document |
|--------------|----------|
| D1 Readiness, D2 Capability Map, D3 Options, D4 Target Architecture, D5 Service Placement | **This document** |
| D6 Health Model, D7 Global SLOs, D15 Triggers, D16 Orchestration, D17 Failback, D18 Partitions, D31 Recovery Objectives | [failover-strategy.md](./failover-strategy.md) |
| D8 PostgreSQL, D9 Consistency, D10 Identifiers, D11 Redis, D12 Events, D13 Ingestion Leadership, D14 Split-Brain | [data-strategy.md](./data-strategy.md) |
| D19 Terraform, D5 Global Traffic Strategy, D20 Parity, D21 Artifacts, D22 Secrets, D23 CI/CD, D24 Progressive Delivery, D28 Capacity, D35 Cost, D36 Security, D37 Compliance | [infrastructure-plan.md](./infrastructure-plan.md) |
| D25 Observability, D26 Dashboards, D27 Alerting, D29 Load/Failover Testing, D30 Chaos, D32 Runbooks, D33 DR Exercise, D34 Incident Command | [operations-plan.md](./operations-plan.md) |
| D38 Risk Register, D39 Debt Register, D40 Backlog, D41 Milestones, D42 Critical Path, D43 Quality Gates, D44 Work Package Template, D45 Demos, D46 Interview Matrix | [execution-plan.md](./execution-plan.md) |
| Architecture decision record | [ADR-020](../adr/020-multi-region-architecture.md) |
