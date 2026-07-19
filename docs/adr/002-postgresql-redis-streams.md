# ADR-002: PostgreSQL for Persistence, Redis Streams for Events

## Status

Accepted

## Context

The platform needs:
1. Durable storage for historical price snapshots with relational queries
2. Low-latency access to latest market values
3. Event distribution for future realtime consumers

## Decision

Use **PostgreSQL** for persistent storage and **Redis** (with Redis Streams) for caching and event distribution.

## Rationale

### PostgreSQL:
- ACID guarantees for financial data integrity
- NUMERIC type avoids floating-point precision loss for prices
- Rich indexing (B-tree, partial) for time-series query patterns
- Mature ecosystem for migrations, backup, and replication
- Well-understood operational model

### Redis (cache):
- Sub-millisecond reads for latest market values
- TTL-based expiration prevents stale data accumulation
- Simple key-value model fits "latest price per symbol" access pattern

### Redis Streams (events):
- Built-in consumer group support for future realtime gateway
- Ordered, persistent log without requiring Kafka/NATS infrastructure
- Approximate max-length trimming keeps memory bounded
- Sufficient throughput for 6 coins at 60s intervals

### Why not Kafka/NATS yet:
- Current scale (6 coins, 1 provider) does not justify additional infrastructure
- Redis Streams provides adequate ordering and delivery guarantees
- Can migrate to Kafka/NATS when multi-consumer, replay, or higher throughput is needed

## Consequences

- Redis is a required dependency (not optional)
- Stream consumers must handle at-least-once delivery
- PostgreSQL stores authoritative history; Redis is a derived cache
- Future migration to Kafka requires a stream adapter layer
