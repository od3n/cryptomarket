# Realtime Delivery Architecture

This document describes the realtime market data delivery system implemented in Phase 2.

## Overview

```
┌─────────────┐     ┌──────────────┐     ┌──────────────────┐     ┌─────────────┐
│  Ingestor   │────▶│ Redis Stream │────▶│ Realtime Gateway │────▶│   Browser   │
│  (Go)       │     │ market:events│     │ (Go SSE)         │     │ (EventSource)│
└─────────────┘     └──────────────┘     └──────────────────┘     └─────────────┘
```

## SSE Endpoint

```
GET /events/markets
```

### Response Headers

```http
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
X-Accel-Buffering: no
```

### Event Format

```
event: market.price.updated
id: <event_id>
data: {"event_id":"...","event_type":"market.price.updated","symbol":"BTC",...}

```

### Heartbeat

A comment is sent every 15 seconds to keep connections alive:

```
:heartbeat

```

## Event Schema

```json
{
  "event_id": "uuid",
  "event_type": "market.price.updated",
  "symbol": "BTC",
  "price_usd": "118420.50",
  "market_cap": "2350000000000.00",
  "volume_24h": "42150000000.00",
  "change_24h": "1.87",
  "provider": "coingecko",
  "observed_at": "2026-07-20T08:15:00Z",
  "published_at": "2026-07-20T08:15:01Z"
}
```

All decimal values are strings to avoid floating-point precision loss. All timestamps are RFC3339 UTC.

## Reconnection Behavior

### Client-Side (Browser)

The frontend uses the `EventSource` API with automatic reconnection:

1. On connection loss, wait with exponential backoff (1s, 2s, 4s, ... max 30s)
2. Send `Last-Event-ID` header on reconnect
3. After 3+ failures, enter "degraded" mode and poll REST API every 30s

### Server-Side (Last-Event-ID)

When a client reconnects with `Last-Event-ID`:

1. Query Redis Stream for events after that ID using `XRANGE`
2. Replay up to 50 missed events
3. Continue with live stream

## Ordering Rules

- Events are ordered by Redis Stream ID (monotonically increasing)
- Client-side: reject events with `published_at` older than the latest seen for that symbol
- Duplicate events (same `event_id`) are ignored

## Duplicate Handling

### Server-Side

- Consumer maintains an in-memory seen-set (bounded at 10,000 entries)
- Duplicate `event_id` values are skipped and acknowledged

### Client-Side

- Track `published_at` per symbol
- Reject events older than or equal to the latest seen timestamp

## Data Freshness States

| State | Condition | Display |
|-------|-----------|---------|
| fresh | age ≤ 120s | Green badge |
| delayed | 120s < age ≤ 300s | Yellow badge |
| stale | age > 300s | Red badge |
| unknown | no timestamp | Gray badge |

Thresholds are configurable via environment variables.

## Failure Modes

| Failure | Behavior |
|---------|----------|
| Redis unavailable | Consumer retries with 2s backoff; clients see "reconnecting" |
| Client disconnect | Hub removes client; resources freed |
| Slow client | Latest-state-wins: drop intermediate updates, keep latest per symbol |
| Malformed event | Logged, acknowledged (skipped), validation failure metric incremented |
| Gateway restart | Consumer group recovers pending messages via XAUTOCLAIM |

## Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `realtime_active_connections` | Gauge | Current SSE connections |
| `realtime_connections_total` | Counter | Total connections since start |
| `realtime_connection_duration_seconds` | Histogram | Connection lifetimes |
| `realtime_events_consumed_total` | Counter | Events read from stream |
| `realtime_events_broadcast_total` | Counter | Events sent to clients |
| `realtime_event_validation_failures_total` | Counter | Malformed events |
| `realtime_redis_reconnects_total` | Counter | Redis reconnection attempts |
| `realtime_stream_consumer_lag` | Gauge | Pending messages in group |
| `realtime_dropped_messages_total` | Counter | Messages dropped (slow clients) |
| `realtime_heartbeat_failures_total` | Counter | Heartbeat write failures |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_PORT` | 8081 | Gateway listen port |
| `REDIS_ADDRESS` | localhost:6379 | Redis host:port |
| `LOG_LEVEL` | info | Log verbosity |

## Testing

```bash
# Smoke test
curl -N http://localhost:8081/events/markets

# Health check
curl http://localhost:8081/health

# Metrics
curl http://localhost:8081/metrics
```
