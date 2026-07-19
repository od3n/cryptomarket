# ADR 005: Latest-State-Wins Delivery Policy

## Status

Accepted

## Context

When broadcasting market price updates to SSE clients, we need to handle slow consumers whose buffers fill up. Options include:

1. **Block the broadcaster** until the slow client catches up
2. **Drop the slow client** immediately
3. **Drop older messages** and deliver the latest state
4. **Queue all messages** unboundedly

## Decision

We implement a **latest-state-wins** policy per symbol:

- Each client has a bounded buffer (64 events)
- When the buffer is full, we compact it by keeping only the latest event per symbol
- The newest event for each symbol is always retained
- Intermediate updates for the same symbol are dropped

## Rationale

### Why latest-state-wins

For market price data:

- **Only the latest price matters**: Users care about current prices, not every intermediate tick
- **Bounded memory**: Prevents unbounded buffering that could exhaust server memory
- **No blocking**: One slow client cannot block broadcasts to other clients
- **Graceful degradation**: Slow clients still receive updates, just fewer intermediate states

### Why not other options

| Option | Problem |
|--------|---------|
| Block broadcaster | One slow client blocks all clients |
| Drop client immediately | Poor UX for temporarily slow clients |
| Unbounded queue | Memory exhaustion risk |

### Implementation

```go
// When buffer is full:
1. Drain buffer into map[symbol]event (keeps latest per symbol)
2. Add new event to map
3. Recreate buffer with compacted events
```

## Consequences

### Positive

- Bounded memory usage per client
- No head-of-line blocking
- Slow clients still receive current state
- Simple to implement and reason about

### Negative

- Slow clients miss intermediate price movements
- Chart history may have gaps for slow clients (acceptable — they can refetch via REST)
- Slightly more CPU during buffer compaction (rare event)

## Metrics

- `realtime_dropped_messages_total`: Counts messages dropped due to buffer compaction
- Alert if this metric spikes (indicates systemic slowness)
