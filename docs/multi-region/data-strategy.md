# Phase 7 — Data Strategy: PostgreSQL, Redis, Events, and Ingestion Leadership

**Status:** Proposed — pending data gate approval
**Parent:** [Program Plan](./program-plan.md)
**Scope:** Deliverables 8-14

---

## D8. PostgreSQL Multi-Region Strategy

### D8.1 Current state (evidence)

- RDS PostgreSQL 16.2, `db.r6g.large`, Multi-AZ, 100GB gp3, 35-day backup retention, deletion protection, encrypted storage (`deploy/terraform/modules/rds/main.tf`).
- Schema: `coins` (BIGSERIAL PK), `price_snapshots` (migrated to UUID PK + monthly range partitions in `migrations/003_partition_snapshots.up.sql`), `provider_sync_logs` (BIGSERIAL PK).
- Access is via a single DSN (`POSTGRES_DSN`); no read/write split in application code; all queries hit the primary.
- No cross-region replica, no promotion procedure, no lag monitoring.

### D8.2 Option evaluation

| Option | Write topology | Lag | Failover/promotion | Failback | Conflict mgmt | Cost | Verdict |
|--------|---------------|-----|--------------------|----------|---------------|------|---------|
| **RDS cross-region read replica** | Single writer (primary region) | Async, typically <1s intra-continental; spikes under load | Managed promotion (replica becomes standalone primary); DNS/endpoint swap required | Re-establish old primary as replica of new primary (rebuild or reverse replication) | None needed (single writer) | ~+40% RDS cost + transfer | **SELECTED** |
| Aurora Global Database | Single writer; up to 5 secondary regions | Typically <1s; designed for <1min planned failover | Managed planned failover with Aurora-controlled promotion; faster than RDS replica | Reverse: demote and reattach | None needed | ~+55% (Aurora premium) | Strong alternative; rejected on migration cost from RDS PostgreSQL and limited benefit at this scale |
| Logical replication (pglogical/publication) | Single writer; selective tables | Configurable; per-table control | Manual; app-level endpoint switch | Flexible (can replicate back per-table) | Requires custom conflict rules if ever multi-writer | Moderate; ops burden | Rejected: operational burden without matching benefit; useful later for selective table replication |
| Backup-and-restore standby | None until restore | RPO = backup interval (hours) | Slow restore; 2-4h RTO (current documented state) | N/A | N/A | Cheapest | Current state; insufficient for hot-standby target |
| Application-level replication | App-controlled | App-controlled | App-controlled | App-controlled | Fully custom | High dev cost | Rejected: home-grown replication is out of scope |
| Multi-writer (BDR/CRDB-style) | Multiple writers | Sync or conflict-resolving | N/A | N/A | Requires rigorous conflict model | Very high | **Explicitly rejected** per charter; no conflict model exists or is justified |

### D8.3 Selected design: RDS cross-region read replica

**Primary writer location:** us-east-1 (current prod region) under normal topology.

**Read-replica usage:**
- The secondary-region replica serves read-only workloads after promotion only (not before — pre-failover reads stay in the primary region to avoid serving lagged data to users).
- Future option (deferred): regional history API reads from the replica for latency improvement. Not in Phase 7 scope.

**Replication lag management:**
- CloudWatch `ReplicaLag` metric exported with `region` label; alarm at >5s sustained 2 min (warning), >30s (critical, blocks failover readiness).
- Lag is displayed on the failover control dashboard; failover workflow step 4 evaluates lag before promotion and reports expected data loss to the approver.

**Promotion process (summary; full workflow in failover-strategy.md D16):**
1. Confirm primary region is genuinely unavailable (multi-signal validation, hold-down elapsed).
2. Freeze ingestion (leader lease released or expired; no writes in flight).
3. Read final `ReplicaLag`; if >0, approver accepts the RPO loss (expected ≤60s).
4. `aws rds promote-read-replica` on the secondary instance.
5. Wait for promotion completion (instance status `available`, accepting writes).
6. Update application DSN via Secrets Manager replica + rolling restart (or DNS CNAME swap in front of the endpoint — selected: CNAME `db.cryptomarket.internal` → regional endpoint, TTL 60s).
7. Verify writes: ingestor leader in secondary writes a canary snapshot; API reads it back.
8. Run pending migrations if the failed region had unapplied schema (migration job targets the new writer).

**Failback (summary; full in failover-strategy.md D17):**
- After the primary region recovers: rebuild it as a read replica of the new primary (from snapshot, then attach), or use reverse logical replication if rebuild cost is unacceptable.
- No automatic failback. Minimum 24h observation. Leadership and traffic transfer in a planned window.

**Sequence and ID behavior:**
- `coins.id` and `provider_sync_logs.id` are BIGSERIAL. Under single-writer operation, sequences are safe (only one DB accepts writes). Risk exists only if a stale writer briefly writes post-promotion — mitigated by epoch fencing (D13/D14), not by sequence design.
- Migration plan (D10): convert both to UUIDv7 to remove the class of risk entirely and enable future cross-region merges.

**Schema migrations:**
- Migrations run only against the writer region (single migration leader = same region as ingestion leader).
- Replicas receive schema via replication automatically.
- Deployment rule: schema migrations must be backward-compatible (expand/contract pattern) so the standby region running version N tolerates schema N+1 during sequential rollout (infrastructure-plan.md D24).

**Connection management:**
- Applications use an internal CNAME (`db.cryptomarket.internal`) resolved per-region; on promotion the CNAME is repointed. Go `database/sql` pool with `ConnMaxLifetime` 5 min (`cmd/ingestor/main.go`) bounds stale-connection duration; add retry-on-connection-error to the repository layer.
- Read-after-write: all reads in the active region hit the writer (no replica reads pre-failover), so read-after-write is trivially consistent.

**Backup integration:**
- Existing automated backups continue on the primary. Additionally: daily cross-region snapshot copy to the secondary (S3 + RDS snapshot copy), verified by quarterly restore drill (operations-plan.md D33).

**Data-loss expectations:**
- Planned failover with lag ≈ 0: negligible (≤1 ingestion cycle).
- Unplanned: RPO ≤ 60s target, measured per exercise. Worst documented case: last committed transaction batch at promotion time.

---

## D9. Database Consistency Model

### D9.1 Data classification

| Data category | Consistency class | Source of truth | Acceptable lag | Conflict policy | Reconciliation | Loss tolerance | Regional read policy |
|-----------------|-------------------|-----------------|----------------|-----------------|----------------|----------------|----------------------|
| Migration version (`schema_migrations`) | Strong | Writer-region PostgreSQL | 0 (replication delivers) | N/A — single writer | Migration job verifies version on promotion | 0 | Writer region only |
| Coin definitions (`coins`) | Strong | Writer-region PostgreSQL | ≤ replication lag | N/A — single writer; changes are operator-driven | Manual review; table is small and diffable | 0 | Writer region only for writes; replicated reads acceptable post-promotion |
| Provider configuration (env/feature flags) | Strong | Git + Secrets Manager | Deploy-time | Git wins; env overrides documented | Drift detection (infrastructure-plan.md D20) | 0 | Region-local from replicated secrets |
| Operational control state (lease/epoch table) | Strong | Writer-region PostgreSQL | 0 | Lease rules (D13): never steal live lease; epoch monotonic | Epoch comparison on any dispute | 0 (loss = leadership ambiguity → pause ingestion) | Writer region only |
| Price snapshots (`price_snapshots`) | Eventual | Writer-region PostgreSQL | ≤ 60s normal; ≤ 5 min degraded | Last-writer-wins by `captured_at`; single writer prevents conflicts | `sre-toolkit/reconcile_prices.py` extended for cross-region comparison | ≤ 60s (one cycle) | Replicated reads acceptable |
| Redis latest-value cache | Eventual (rebuildable) | Derived from PostgreSQL | ≤ ingestion interval | Overwritten each cycle | Full rebuild from DB | 100% (rebuilt) | Region-local only |
| Provider health history (`provider_sync_logs`) | Eventual | Writer-region PostgreSQL | ≤ replication lag | Append-only; no conflicts | None needed | ≤ 5 min acceptable | Replicated reads acceptable |
| Redis Stream events | Ephemeral | Region-local Redis | N/A | N/A — regenerated by leader | Replay from stream (bounded); DB backfill | ≤ 5 min (stream retention) | Region-local only |
| Derived aggregates (API-computed) | Ephemeral | Computed on read | N/A | N/A | Recomputed | 100% | Any region |

### D9.2 Principles

1. **PostgreSQL is the only durable source of truth.** Redis never holds data that cannot be reconstructed from PostgreSQL within one ingestion cycle (existing design intent, `internal/cache/redis.go` 5-min TTL — now an explicit architectural contract).
2. **Strong consistency is achieved by single-writer topology**, not by distributed consensus. Anything requiring strong consistency lives in the writer region's PostgreSQL.
3. **Eventual consistency data must be idempotently regenerable.** Every consumer of snapshots or events must tolerate re-delivery (D12).
4. **No data category requires cross-region synchronous replication.** The cost (50-100ms write latency) is not justified for any identified workload.

---

## D10. Globally Unique Identifiers

### D10.1 Current identifier inventory

| Identifier | Current scheme | Location | Cross-region safety | Ordering | Notes |
|------------|---------------|----------|---------------------|----------|-------|
| `coins.id` | BIGSERIAL | `migrations/001` | UNSAFE if regions ever merge | Sequential | Small table; merge risk low but nonzero |
| `price_snapshots.id` | UUIDv4 (`gen_random_uuid()`) | `migrations/003` | SAFE (collision ~0) | None (random) | Post-003 rows only; pre-003 rows were BIGSERIAL |
| `provider_sync_logs.id` | BIGSERIAL | `migrations/001` | UNSAFE if merged | Sequential | Operational log; merge unlikely |
| Event `event_id` | UUIDv4 (`uuid.New()`) | `internal/cache/redis.go` | SAFE | None | Used for consumer dedup |
| Redis Stream entry IDs | Redis-generated `ms-seq` | Redis runtime | Region-local only | Time-ordered per stream | Used for SSE `Last-Event-ID` replay |
| Ingestion jobs | None (no job ID exists) | `internal/worker/ingestor.go` | MISSING | n/a | Cycles identified only by log timestamp |
| Traces | Request IDs (middleware-generated) | `internal/api/middleware.go` | Not verified format | n/a | Needs W3C trace context |
| Alerts | Prometheus alertname+labels | `monitoring/` | SAFE with region label | n/a | Needs `region` label added |

### D10.2 Target scheme: UUIDv7

**Selection rationale:**
- UUIDv7 = 48-bit unix-ms timestamp + version + 74 random bits. Globally unique without coordination; time-ordered (B-tree friendly, unlike UUIDv4's random scatter which causes index page churn on the high-insert `price_snapshots` table).
- The platform already depends on `github.com/google/uuid` v1.6.0 (`go.mod`), which supports `uuid.Must(uuid.NewV7())`.
- Region attribution: add an explicit `source_region` column/field rather than encoding region into the ID (keeps IDs standard-parseable).

**Rejected alternatives:**
- ULID: equivalent properties but non-standard library dependency; UUIDv7 preferred for PostgreSQL `gen_random_uuid()`-adjacent tooling and RFC standardization.
- Region-prefixed IDs (`use1_01H...`): breaks column types and index uniformity; rejected except for human-facing incident IDs where a region prefix aids operations (e.g., `INC-us-east-1-2026-0042`).
- Continued BIGSERIAL + region offset (e.g., odd/even): fragile, requires coordination on every region addition; rejected.

### D10.3 Migration approach

1. **`coins`**: Add `uuid UUID` column with UUIDv7 default; backfill; dual-write period; swap PK in an expand/contract migration. Table is tiny (<100 rows); low risk. FK references in `price_snapshots.coin_id` (post-003 schema already UUID — verify referential alignment during migration; the 003 migration's `coin_id UUID REFERENCES coins(id)` implies coins.id must become UUID for FK integrity — confirm current production state before planning).
2. **`provider_sync_logs`**: Same pattern; or accept BIGSERIAL (single-writer, never merged) with documented rationale. Recommendation: convert for uniformity.
3. **Event IDs**: Switch `PublishEvent` from `uuid.New()` to `uuid.NewV7()`; no schema change (string field). Ordering benefit for stream replay.
4. **Ingestion cycle IDs**: Add a cycle ID (UUIDv7) generated at cycle start; logged and written to sync logs; enables cross-region dedup of cycles during leadership transfer.
5. **Traces**: Adopt W3C `traceparent` propagation in middleware; trace IDs are inherently global.

**Indexing implications:** UUIDv7's time-prefix yields append-mostly B-tree inserts — better than UUIDv4 for the partitioned snapshots table. PK remains `(id, captured_at)` per partitioning requirement.

---

## D11. Redis Multi-Region Strategy

### D11.1 Redis usage inventory (evidence)

| Usage | Key pattern | Persistence | Cross-region need |
|-------|-------------|-------------|-------------------|
| Latest-value cache | `market:latest:{SYMBOL}`, 5-min TTL (`internal/cache/redis.go`) | Ephemeral | NONE — rebuilt each ingestion cycle |
| Event stream | `market:events`, MAXLEN ~10000 (`internal/cache/redis.go`) | Ephemeral (AOF locally) | NONE — regenerated by active-region leader; region-local consumers |
| Consumer group state | `realtime-gateway` group, pending entries (`internal/stream/consumer.go`) | Ephemeral | NONE — group recreates via `EnsureGroup`; pending claims via XAUTOCLAIM |
| Rate-limit state | In-memory only (`internal/resilience/ratelimit.go`) — not in Redis today | n/a | NONE per region; provider rate limits are consumed globally by the single leader (D13) |
| Distributed locks | NONE today | n/a | Leadership uses PostgreSQL advisory locks instead (D13) — deliberately NOT Redis |

### D11.2 Option evaluation

| Option | Verdict | Rationale |
|--------|---------|-----------|
| Independent regional caches | **SELECTED** | Matches actual data semantics; zero replication complexity; rebuild path already documented |
| ElastiCache Global Datastore | Rejected | Replicates cache nobody needs replicated; adds failover coupling; cost without benefit. (Global Datastore also does not support Streams semantics for our use.) |
| Cross-region stream replication | Rejected | Events are derivable from the DB; replicating a bounded ephemeral stream adds a failure mode (replication break) for no durable value |
| Rebuild Redis from PostgreSQL | **SELECTED as recovery path** | Formalized: on regional Redis loss or post-failover, a warm-up routine populates `market:latest:*` from the latest snapshots before traffic ramp |
| Application-level event replication | Rejected | Duplicate of rejected stream replication |
| Redis as region-local ephemeral infrastructure | **SELECTED framing** | This is the architectural contract: Redis may die at any time without data loss |

### D11.3 Failover behavior

1. Primary region fails → secondary region's Redis exists (running, empty or stale).
2. Ingestion leadership transfers → first cycle in secondary repopulates all `market:latest:*` keys and begins publishing to the secondary's `market:events` stream.
3. Realtime gateway in secondary: `EnsureGroup` creates the consumer group on the (new) stream; consumers read from `>` (new events only) — historical replay for reconnecting clients is bounded (50 events, `internal/realtime/server.go`) and acceptable.
4. API in secondary serves from cache once repopulated; until then it falls back to PostgreSQL latest-snapshot queries (existing fallback path).

### D11.4 Duplicate and replay handling

- **Duplicate delivery**: possible during leadership transfer (old leader publishes before fence, new leader publishes after). Consumer dedup by `event_id` (D12) makes this harmless for realtime broadcast (idempotent: broadcast is naturally at-least-once).
- **Replay**: SSE clients reconnecting with `Last-Event-ID` referencing the OLD region's stream IDs will not match the new stream. Mitigation: on regional failover, clients receive a stream-reset signal (HTTP 503 + `Retry-After`, or SSE `retry:` directive) and reconnect without `Last-Event-ID`, accepting a bounded gap filled by the latest-value cache.
- **Consumer-group behavior**: groups are per-Redis-instance. The secondary's group starts fresh; no pending entries carry over. This is correct — pending entries from the dead region are unprocessable anyway.

### D11.5 Hardening required (PROPOSED)

1. Consumer name must be unique per pod and region: replace hardcoded `realtime-1` (`internal/stream/consumer.go`) with `{region}-{pod-name}`.
2. Dedup store must survive restarts and scale: replace the in-memory map (which clears at 10k entries) with a Redis SET of seen event IDs with 10-min TTL, or a bounded LRU with persistent overflow — selected: Redis `SET market:seen {event_id} EX 600 NX` check (idempotent, shared across replicas).
3. Cache-warm routine: explicit `WarmCache(ctx)` invoked at ingestor startup and on leadership acquisition, populating all latest values from PostgreSQL before the first publish cycle completes.

---

## D12. Event Delivery and Replication Strategy

### D12.1 Event contract v2 (PROPOSED)

Current schema (`internal/stream/event.go`): `event_id, event_type, symbol, price_usd, market_cap, volume_24h, change_24h, provider, observed_at, published_at`.

Target contract (v2, additive and backward-compatible):

```json
{
  "event_id": "0190f3a7-...(UUIDv7)",
  "event_type": "market.price.updated",
  "schema_version": 2,
  "source_region": "us-east-1",
  "cycle_id": "0190f3a6-...(UUIDv7, ingestion cycle)",
  "symbol": "BTC",
  "price_usd": "67123.45",
  "market_cap": "1320000000000",
  "volume_24h": "28000000000",
  "change_24h": "2.31",
  "provider": "coingecko",
  "observed_at": "2026-07-21T12:00:00Z",
  "ingested_at": "2026-07-21T12:00:01Z",
  "published_at": "2026-07-21T12:00:01Z",
  "dedup_key": "coingecko:BTC:2026-07-21T12:00",
  "trace_context": "00-<trace-id>-<span-id>-01"
}
```

Field rationale:
- `schema_version`: consumers reject unknown major versions; enables evolution.
- `source_region`: provenance; global dashboards; post-failover forensics.
- `ingested_at`: distinguishes provider observation time from platform ingestion time (freshness SLO accuracy).
- `cycle_id`: groups all symbols from one ingestion cycle; enables gap detection and cycle-level reconciliation.
- `dedup_key`: `{provider}:{symbol}:{minute-bucket of observed_at}` — deterministic, so duplicates from retried cycles or dual-leadership windows are detectable by any consumer independently of event_id.
- `trace_context`: W3C traceparent for cross-service correlation.

**Compatibility:** v1 consumers ignore unknown fields (JSON). v2 consumers accept v1 events (missing fields defaulted; `dedup_key` synthesized from `provider+symbol+observed_at`). Dual-format support mirrors the existing legacy-format handling in `internal/stream/consumer.go` `parseLegacyEntry`.

### D12.2 Delivery semantics per consumer

| Consumer | Semantics | Idempotency mechanism |
|----------|-----------|----------------------|
| Realtime broadcast (`subscriber.Hub`) | At-least-once | Broadcast is idempotent (latest value wins per client render); dedup suppresses visible duplicates |
| Snapshot persistence (ingestor → PostgreSQL) | Effectively-once | Unique constraint on `(coin_id, provider, date_trunc('minute', captured_at))` + `ON CONFLICT DO UPDATE` (last value wins); makes re-ingestion of the same cycle safe |
| Cache write (`SetLatest`) | At-least-once | SET is naturally idempotent (last write wins) |
| SSE replay | At-most-once per connection attempt | Client renders latest; duplicates harmless |

### D12.3 Failure modes and handling

| Failure | Handling |
|---------|----------|
| Duplicate delivery (retry, dual-publish window) | `dedup_key` constraint (DB) + seen-set (stream consumers) |
| Out-of-order delivery | Consumers compare `observed_at`; stale events (older than current latest) are dropped for cache writes, stored for history (history is append-only time-series; order at rest is by `captured_at`) |
| Late events (>5 min) | Stored to history; not published to realtime stream (freshness gate) |
| Region replay (post-failover warm-up) | New stream in new region; old stream abandoned; clients reset per D11.4 |
| Conflicting provider data | Single leader ingests from one provider chain at a time (fallback orchestrator, `internal/provider/fallback.go`); cross-provider reconciliation is offline tooling (`sre-toolkit/reconcile_prices.py`), never a write-path conflict |
| Stream gaps (XADD failure mid-cycle) | Cycle-level metric `cycle_events_published` vs expected coin count; gap alarm; DB remains complete (stream is best-effort) |
| Event retention | Stream MAXLEN ~10000 (existing); DB is the durable record; no cross-region retention sync needed |
| Consumer recovery | XAUTOCLAIM loop (existing, `claimPendingLoop`) + durable seen-set prevents reprocessing side effects |

---

## D13. Ingestion Leadership Strategy

### D13.1 Problem statement (evidence)

The ingestor (`internal/worker/ingestor.go` + `internal/scheduler/scheduler.go`) runs an uncoordinated ticker. Deployed in two regions simultaneously it would:
- double every provider API call (rate limits are per-API-key, global — CoinGecko free tier ~10-30 req/min);
- write duplicate snapshots (no unique constraint today);
- publish duplicate events (different event_ids, same data);
- corrupt sync-log semantics (double-counted cycles).

There is no leadership mechanism anywhere in the codebase.

### D13.2 Option evaluation

| Option | Reliability | Rate-limit safety | Duplicate writes | Consistency | Failover speed | Split-brain risk | Ops complexity | Verdict |
|--------|-------------|-------------------|------------------|-------------|----------------|------------------|----------------|---------|
| Single active ingestor region (standby region runs nothing) | Low — failover requires deploy/config change | Safe | Safe | Safe | Slow (10-30 min) | None | Low | Rejected: fails hot-standby RTO |
| **Active ingestor with standby (lease-based leadership)** | High | Safe (one leader) | Safe (fenced) | Safe | Fast (lease TTL bounded, 15-30s) | Low with fencing | Moderate | **SELECTED** |
| Region-partitioned ingestion | Moderate | Doubles provider load by design | Safe per partition | Complex (two write streams into one DB) | Moderate | Moderate (partition reassignment) | High | Rejected: both regions still write to one DB; partition coordination = hidden leadership problem |
| Dual ingestion with reconciliation | High availability | UNSAFE (2x rate consumption) | Requires dedup infra | Weak (conflicting snapshots need resolution) | Instant | High | Very high | Rejected: rate-limit violation is disqualifying |

### D13.3 Selected design: PostgreSQL advisory-lock lease with epoch fencing

**Why PostgreSQL and not Redis for the lease:** the lease must survive Redis loss (Redis is ephemeral by contract, D11) and must be co-located with the data being protected (the DB itself). If the DB is unreachable, ingestion must stop anyway — the lease naturally becomes unacquirable, which is the correct failure mode.

**Mechanism:**

```
Table: ingestion_leases
  lease_key     TEXT PRIMARY KEY,        -- 'market-ingestor'
  holder_id     TEXT NOT NULL,           -- '{region}/{pod-name}'
  epoch         BIGINT NOT NULL,         -- monotonically increasing
  acquired_at   TIMESTAMPTZ NOT NULL,
  renewed_at    TIMESTAMPTZ NOT NULL,
  expires_at    TIMESTAMPTZ NOT NULL     -- acquired + TTL
```

1. **Acquire:** `INSERT ... ON CONFLICT (lease_key) DO UPDATE SET holder=me, epoch=epoch+1, ... WHERE excluded.expires_at < now()` — atomically takes the lease only if expired or self-held. Epoch increments on every acquisition (monotonic fencing token).
2. **Renew:** leader updates `renewed_at`/`expires_at` every TTL/3 (e.g., every 10s for a 30s TTL). Renewal failure × 2 → leader voluntarily stops ingesting (self-fence).
3. **Fence:** every snapshot batch insert includes `WHERE` guard via a session GUC or an explicit check: the ingestor reads its epoch at cycle start and the write path verifies `epoch = (SELECT epoch FROM ingestion_leases WHERE holder_id = me)` in the same transaction. A stale leader's writes fail with a fencing error and are dropped.
4. **Standby behavior:** polls acquisition every 5s. On success: logs leadership acquisition with new epoch, runs `WarmCache`, begins scheduling.
5. **Voluntary transfer (failover/failback):** operator sets `force_release = true` (or deletes the lease row); current leader observes on next renewal and stops; standby acquires. Deterministic, ordered, no expiry wait.

**Parameters:** TTL 30s; renew 10s; standby poll 5s; worst-case leadership gap on crash = TTL + poll ≈ 35s (within RPO 60s since the gap is a pause, not a loss — the next cycle after acquisition ingests current provider state).

**Split-brain prevention:** §D14. Key invariant: the acquisition query's `WHERE expires_at < now()` guarantees a lease is never stolen while live; the epoch check guarantees a stale holder's writes are rejected even if it believes itself leader.

**Observability:** gauges `ingestion_leader{region,holder}` (1/0), `ingestion_lease_epoch`, `ingestion_lease_renewal_age_seconds`; alarms: no leader >60s, two holders claiming (epoch divergence), renewal age > TTL/2.

---

## D14. Split-Brain Prevention

### D14.1 Split-brain vector inventory

| # | Vector | Scenario | Consequence if unguarded |
|---|--------|----------|--------------------------|
| 1 | Ingestion leadership | Network partition between regions; both ingestors believe themselves leader | Duplicate provider calls (rate exhaustion), duplicate snapshots, conflicting events |
| 2 | PostgreSQL writes | Old primary still accepting writes after replica promotion | Divergent histories; unrecoverable without reconciliation |
| 3 | Migrations | Migration job runs in both regions during failover | Schema divergence; failed migration leaves one region unbootable |
| 4 | Deployment promotion | CI/CD deploys different versions to each region during partition | Version skew; event contract mismatch |
| 5 | Failover orchestration | Two operators (or operator + automation) initiate conflicting failover actions | Partial failover: traffic shifted but DB not promoted (or vice versa) |
| 6 | Scheduled jobs | Partition management function (`create_monthly_partition`) runs in both regions | Duplicate DDL attempts (benign due to IF NOT EXISTS, but indicates uncoordinated execution) |
| 7 | Reconciliation jobs | Reconciliation runs against inconsistent region states | False-positive conflicts; operator confusion |

### D14.2 Safeguards

| Vector | Safeguard | Mechanism |
|--------|-----------|-----------|
| 1 | Lease-based leadership + fencing token | D13: atomic conditional acquisition; epoch verified in write transaction; self-fence on renewal failure |
| 2 | Single-writer topology + RDS promotion semantics | `promote-read-replica` creates an independent instance; old primary (if it returns) is NEVER reattached as writable — it is rebuilt as a replica from the new primary's snapshot. Application DSN points only at the current writer via CNAME. Stale-writer writes additionally fail the epoch fence (vector 1's guard covers the ingestor, the only writer) |
| 3 | Migration leadership = ingestion leadership region | Migration job acquires the same lease class (`migration` key) before applying; runs only where the writer DB is reachable and lease is held |
| 4 | Deployment lock | CI/CD pipeline uses a single promotion state record (Git tag + deployment manifest in repo); regional deploys read the same manifest; partition → deploys pause (fail-closed), never diverge |
| 5 | Orchestration mutex + operator approval | Failover workflow holds an `orchestration_lock` (lease table); every state transition is logged with actor + epoch; automation and humans share the same lock; conflicting actions queue rather than interleave |
| 6 | Job execution under lease | Partition maintenance invoked only by the ingestion leader (piggybacks on leadership), single execution per region per window |
| 7 | Reconciliation is read-only and region-aware | Tool takes explicit `--region-a/--region-b` and a consistency timestamp; never writes; results are advisory |

### D14.3 Residual risks (accepted, documented)

1. **Clock skew across regions** affecting `expires_at` evaluation: mitigated by NTP (EKS nodes) and conservative TTL (30s >> realistic skew); lease checks execute in PostgreSQL (single clock domain per DB), not application clocks — acquisition uses `now()` server-side. Residual: negligible.
2. **Stale leader writes between lease expiry and its next renewal check** (≤10s window): mitigated by in-transaction epoch verification (the DB is the arbiter, not the lease holder's belief). Residual: zero for DB writes (epoch check is transactional); possible duplicate stream publish in the window — harmless (D12.2 idempotency).
3. **Old primary "returns from the dead" after promotion with unreplicated transactions**: data beyond the promotion point is lost by design (accepted RPO); the instance is quarantined (security group isolation) until forensics complete, then rebuilt. No automatic rejoin.
4. **Operator error during manual promotion**: mitigated by checklist-driven runbook with two-person rule and pre-promotion lag display.

---

## D-Gate: Data Gate Criteria (exit conditions for this strategy)

- [ ] Cross-region replica provisioned and lag monitored with alarms.
- [ ] Promotion tested in staging end-to-end (exercise evidence).
- [ ] UUIDv7 migrations applied; no BIGSERIAL in active write paths.
- [ ] Snapshot idempotency constraint deployed; duplicate-cycle test passes.
- [ ] Lease table deployed; dual-ingestor test shows exactly one leader and zero duplicate cycles.
- [ ] Event contract v2 deployed with v1 compatibility verified.
- [ ] RPO measured in exercise ≤ 60s.
