# ADR-020: Multi-Region Architecture — Active-Passive Hot Standby

## Status

Proposed (pending Architecture Gate — see `docs/multi-region/program-plan.md` D43)

## Context

The platform is a single-region system (us-east-1). The current disaster-recovery posture for full region loss is backup-and-restore with RTO 4 hours / RPO 24 hours (`docs/dr/strategy.md`). Phase 7 requires surviving a major regional failure with dramatically better recovery while controlling cost and complexity.

Constraints discovered during the repository assessment (`docs/multi-region/program-plan.md` §3):

1. PostgreSQL schema uses a single-writer model with `BIGSERIAL` identifiers on `coins` and `provider_sync_logs` (`migrations/001_initial_schema.up.sql`).
2. The ingestor has no leader election (`internal/scheduler/scheduler.go`); two instances would duplicate writes and exhaust provider rate limits.
3. Redis is architecturally ephemeral — cache with 5-minute TTL plus a bounded event stream (`internal/cache/redis.go`); the documented recovery path is rebuild-from-PostgreSQL.
4. The workload is read-heavy with a single natural write path (one ingestion cycle per minute).

## Decision

Adopt an **Active-Passive Hot Standby** multi-region topology (us-east-1 primary, us-west-2 secondary), with:

1. **Single database writer at all times.** RDS cross-region read replica in the secondary; explicit, human-approved promotion; the old primary is never reattached as writable.
2. **Single ingestion leader at all times.** PostgreSQL advisory-lock lease with monotonic epoch fencing; standby ingestor acquires only after transfer or expiry.
3. **Redis as region-local ephemeral infrastructure.** No cross-region Redis replication; caches rebuilt from PostgreSQL; consumer groups recreated per region.
4. **Route 53 for global traffic management.** Health-checked failover records for automated traffic shift (pre-approved class), weighted records for controlled shifts; database promotion and traffic shifting are independent, separately-approved actions.
5. **UUIDv7 identifiers and event contract v2** (schema_version, source_region, cycle_id, dedup_key) to make all records and events region-safe and idempotently consumable.

## Rationale

- **Why not Warm Standby:** recovery requires scaling up and warming a cold standby (30-60 min RTO) and carries standby-rot risk; the cost saving (~$600/month) does not justify a 4x slower recovery against the program mission.
- **Why not Active-Active:** the write path is inherently centralized. Active-active would require multi-writer conflict resolution for snapshots, global ingestion coordination, and cross-region consumer groups — rebuilding the three components that currently assume single-region — for benefits (local writes) the workload cannot use. Rejected on cost, complexity, and split-brain surface.
- **Why RDS cross-region replica over Aurora Global Database:** comparable recovery properties for this scale at lower migration cost from the existing RDS PostgreSQL deployment; Aurora remains a documented upgrade path.
- **Why PostgreSQL leases over Redis/ZooKeeper for leadership:** the lease must survive Redis loss (Redis is ephemeral by contract) and must be co-located with the data it protects; if the database is unreachable, ingestion must stop regardless — an unacquirable lease is the correct failure mode.
- **Why human-approved database promotion:** promotion is irreversible in the failover direction and carries RPO consequences; the automation prepares and presents (lag, blast radius, checklist) but a human decides. Traffic shifting alone is automated because it is reversible and does not risk data.

## Consequences

Positive:
- Region-loss RTO improves from 4 hours to a 15-minute target; RPO from 24 hours to 60 seconds — both to be validated by staging exercise before being claimed as production capability.
- The standby region serves continuously as a validation environment (secondary-first canary deployments), paying rent for its cost.
- Single-writer and single-leader invariants keep the consistency model simple and auditable.

Negative:
- Approximately +$1,300/month (+82%) production cost (`docs/multi-region/infrastructure-plan.md` D35).
- Failover requires a human approval step for database promotion; fully autonomous recovery is explicitly out of scope.
- The secondary region runs continuously; operational parity must be actively maintained (drift detection required).

Explicitly not claimed:
- Zero-downtime regional failover.
- Cross-region Redis Streams replication.
- Production-validated recovery (status is PROPOSED until the staging DR exercise in `docs/multi-region/operations-plan.md` D33 passes).

## Related

- Program plan and full deliverables: `docs/multi-region/` (program-plan, data-strategy, failover-strategy, infrastructure-plan, operations-plan, execution-plan).
- Supersedes the full-region-failure section of `docs/dr/strategy.md` (backup-restore remains the fallback if promotion fails).
- Extends ADR-002 (PostgreSQL/Redis roles), ADR-004 (consumer groups), ADR-013 (degraded-mode semantics), ADR-015 (security model).
