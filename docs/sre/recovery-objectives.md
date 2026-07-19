# Recovery Objectives

This document defines the Recovery Time Objectives (RTO) and Recovery Point Objectives (RPO) for the Crypto Market Data Platform.

## Overview

Recovery objectives define how quickly systems must recover from failures and how much data loss is acceptable.

## Recovery Time Objectives (RTO)

RTO defines the maximum acceptable time to restore service after a failure.

| Component | RTO | Rationale |
|-----------|-----|-----------|
| Provider Failure | < 30 seconds | Automatic fallback to secondary provider |
| Redis Failure | < 5 minutes | Container restart, cache rebuild from PostgreSQL |
| PostgreSQL Failure | < 10 minutes | Container restart, connection pooling recovery |
| API Service | < 1 minute | Container restart with health checks |
| Realtime Gateway | < 1 minute | Container restart, stream reconnection |
| Ingestor | < 2 minutes | Container restart, scheduler recovery |

### Provider Failure Recovery

**Detection:** Circuit breaker opens after 5 consecutive failures
**Recovery:**
1. Circuit breaker blocks requests to failed provider
2. Fallback orchestrator selects next eligible provider
3. Data ingestion continues via fallback
4. Circuit breaker probes primary after 30 seconds
5. Primary restored after 2 successful probes

**Expected RTO:** < 30 seconds (automatic)

### Redis Failure Recovery

**Detection:** Health check fails, connection errors
**Recovery:**
1. Docker health check detects failure
2. Container automatically restarts
3. Cache is rebuilt on next ingestion cycle
4. Redis Streams consumer group recreates if needed

**Expected RTO:** < 5 minutes

### PostgreSQL Failure Recovery

**Detection:** Connection pool errors, health check fails
**Recovery:**
1. Docker health check detects failure
2. Container automatically restarts
3. Connection pool re-establishes connections
4. Ingestion resumes on next cycle

**Expected RTO:** < 10 minutes

## Recovery Point Objectives (RPO)

RPO defines the maximum acceptable data loss measured in time.

| Data Type | RPO | Rationale |
|-----------|-----|-----------|
| Price Snapshots | < 60 seconds | One ingestion cycle |
| Cache (Latest Prices) | < 60 seconds | Rebuilt from snapshots |
| Redis Streams | < 5 minutes | Stream persistence with AOF |
| Configuration | 0 | Stored in version control |

### Price Snapshot RPO

- Ingestion interval: 60 seconds
- Maximum data loss: One cycle (60 seconds)
- Historical data: Persisted in PostgreSQL, durable

### Cache Recovery Strategy

When Redis cache is lost:
1. API falls back to PostgreSQL for latest data
2. Next ingestion cycle repopulates cache
3. No user-visible data loss (slightly higher latency)

### Stream Recovery Strategy

When Redis Streams are lost:
1. Consumer group recreates on startup
2. Realtime clients reconnect and receive new events
3. Historical replay not available (acceptable for realtime use case)

## Disaster Recovery

**Note:** Full disaster recovery is deferred to Phase 4.

Current limitations:
- No cross-region replication
- No automated backup verification
- No point-in-time recovery for PostgreSQL

### Current Backup Strategy

- PostgreSQL: Docker volume persistence
- Redis: AOF persistence enabled
- Configuration: Version controlled in Git

## Recovery Testing

Recovery procedures should be tested:
- Monthly: Provider failover simulation
- Quarterly: Redis failure simulation
- Quarterly: PostgreSQL failure simulation

### Testing Commands

```bash
# Simulate provider failure
make incident-demo

# Test Redis recovery
docker compose restart redis

# Test PostgreSQL recovery
docker compose restart postgres
```

## Escalation

If RTO cannot be met:
1. Page on-call engineer
2. Declare incident
3. Follow relevant runbook
4. Communicate status to stakeholders

## Version History

| Date | Version | Changes |
|------|---------|---------|
| 2024-01-15 | 1.0 | Initial recovery objectives |
