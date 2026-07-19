# ADR 003: Server-Sent Events over WebSockets

## Status

Accepted

## Context

Phase 2 requires realtime delivery of market price updates to browser clients. We evaluated two primary options:

1. **Server-Sent Events (SSE)**: HTTP-based, unidirectional server-to-client streaming
2. **WebSockets**: Full-duplex bidirectional communication

## Decision

We chose **Server-Sent Events (SSE)** for realtime market data delivery.

## Rationale

### Why SSE is sufficient for this use case

- **Unidirectional data flow**: Market price updates flow only from server to client. Clients do not need to send data back over the same connection.
- **Built-in reconnection**: The browser's `EventSource` API handles automatic reconnection with `Last-Event-ID` support out of the box.
- **HTTP infrastructure compatibility**: SSE works through standard HTTP proxies, load balancers, and CDNs without special configuration.
- **Lower operational complexity**: No WebSocket upgrade handshake, no ping/pong frame management, simpler connection lifecycle.
- **Simpler implementation**: Both server-side (Go `net/http`) and client-side (`EventSource` API) implementations are straightforward.

### Why WebSockets were not chosen

- **Bidirectional not needed**: We have no requirement for client-to-server messaging over the realtime channel.
- **Higher complexity**: WebSocket connections require more careful handling of reconnection, heartbeats, and proxy configuration.
- **Infrastructure considerations**: Some corporate proxies and older load balancers have issues with WebSocket upgrades.

## Consequences

### Positive

- Simpler implementation and debugging
- Automatic browser reconnection
- Works with standard HTTP/2 multiplexing
- Easy to test with `curl`

### Negative

- Limited to text-based protocols (JSON is fine for our use case)
- Browser connection limits per domain (6 for HTTP/1.1, higher for HTTP/2)
- Cannot send binary data efficiently (not needed here)

## References

- [MDN: Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events)
- [HTML Living Standard: EventSource](https://html.spec.whatwg.org/multipage/server-sent-events.html)
