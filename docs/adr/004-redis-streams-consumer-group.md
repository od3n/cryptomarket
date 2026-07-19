# ADR 004: Redis Streams Consumer Group Strategy

## Status

Accepted

## Context

The realtime gateway needs to consume price events from Redis Streams reliably. We need to handle:

- Multiple gateway instances (horizontal scaling)
- Message acknowledgment and retry
- Recovery after crashes or restarts
- Avoiding message loss or duplication

## Decision

We use **Redis Streams Consumer Groups** with the following strategy:

- **Consumer group name**: `realtime-gateway`
- **Consumer name**: `realtime-1` (per instance)
- **Acknowledgment**: Messages are ACKed only after successful broadcast to clients
- **Pending recovery**: `XAUTOCLAIM` recovers messages idle for >60 seconds
- **Deduplication**: In-memory seen-set with bounded size (10,000 entries)

## Rationale

### Why Consumer Groups

- **At-least-once delivery**: Messages are redelivered if not acknowledged
- **Load balancing**: Multiple consumers in a group share the message stream
- **Pending entries list**: Tracks unacknowledged messages for recovery
- **Built-in to Redis**: No additional infrastructure required

### Message Flow

```
Ingestor → XADD → Redis Stream → XREADGROUP → Consumer → Broadcast → XACK
                                              ↓ (failure)
                                         Pending List → XAUTOCLAIM → Retry
```

### Why XAUTOCLAIM over XPENDING + XCLAIM

- Single command instead of two
- Atomic claim operation
- Simpler implementation
- Available since Redis 6.2

## Consequences

### Positive

- Reliable message delivery
- Safe recovery after crashes
- Horizontal scaling support
- No message loss on transient failures

### Negative

- At-least-once (not exactly-once) — requires client-side deduplication
- Pending list can grow if consumer is permanently stuck
- In-memory deduplication lost on restart (acceptable for price updates)

## Configuration

| Parameter | Value | Description |
|-----------|-------|-------------|
| BatchSize | 100 | Messages per XREADGROUP call |
| BlockTime | 5s | Blocking wait for new messages |
| ClaimInterval | 30s | How often to check for pending messages |
| ClaimMinIdle | 60s | Minimum idle time before claiming |
