# Distributed Tracing

## Overview

The Crypto Market Platform uses OpenTelemetry for distributed tracing, with traces collected by the OTel Collector and stored in Grafana Tempo. Traces provide end-to-end visibility across all services.

## Architecture

```
Application → OTel SDK → OTel Collector (DaemonSet) → Tempo → Grafana
```

## Trace Flow

A typical API request generates spans across multiple services:

```
[Frontend] → [API Gateway/Ingress] → [API Service] → [PostgreSQL]
                                                    → [Redis Cache]
                                                    → [Provider API] (if cache miss)
```

### Span Hierarchy Example

```
GET /api/v1/ticker/BTC-USD (ingress: 45ms)
└── market-api: handleTickerRequest (42ms)
    ├── redis: GET ticker:BTC-USD (2ms) [cache hit]
    └── response serialization (1ms)

GET /api/v1/ticker/ETH-USD (ingress: 850ms) [cache miss]
└── market-api: handleTickerRequest (845ms)
    ├── redis: GET ticker:ETH-USD (2ms) [cache miss]
    ├── postgres: SELECT ticker_data (15ms)
    └── provider: coingecko API call (820ms)
```

## Trace ID Propagation

### W3C Trace Context

All services propagate `traceparent` header:

```
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
```

| Service    | Propagation Method                              |
|------------|-------------------------------------------------|
| Frontend   | `traceparent` header on API requests            |
| API        | W3C Trace Context (incoming + outgoing)         |
| Ingestor   | Generates root span per ingestion cycle         |
| Realtime   | Propagates trace_id from Redis Stream metadata  |

### Redis Stream Trace Context

Ingestor embeds trace context in Redis Stream messages:

```go
redis.XAdd(ctx, &redis.XAddArgs{
    Stream: "market:ticker",
    Values: map[string]interface{}{
        "data":     jsonPayload,
        "trace_id": span.SpanContext().TraceID().String(),
        "span_id":  span.SpanContext().SpanID().String(),
    },
})
```

Realtime service extracts and continues the trace on delivery.

## OTel Collector Configuration

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 5s
    send_batch_size: 1024
  memory_limiter:
    check_interval: 1s
    limit_mib: 512
  attributes:
    actions:
      - key: environment
        value: production
        action: upsert

exporters:
  otlp/tempo:
    endpoint: tempo:4317
    tls:
      insecure: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch, attributes]
      exporters: [otlp/tempo]
```

## Instrumentation

### Go Services

Using `go.opentelemetry.io/otel`:

```go
// Initialize tracer
tp := sdktrace.NewTracerProvider(
    sdktrace.WithBatcher(exporter),
    sdktrace.WithResource(resource.NewWithAttributes(
        semconv.SchemaURL,
        semconv.ServiceName("market-api"),
        semconv.ServiceVersion(version),
    )),
)
otel.SetTracerProvider(tp)
otel.SetTextMapPropagator(propagation.TraceContext{})
```

### Key Spans

| Service    | Span Name                    | Attributes                          |
|------------|------------------------------|-------------------------------------|
| API        | `http.request`               | method, path, status, duration      |
| API        | `db.query`                   | db.system, db.statement, rows       |
| API        | `cache.get`                  | cache.hit, key                      |
| Ingestor   | `ingestion.cycle`            | provider, records, duration         |
| Ingestor   | `provider.fetch`             | provider.name, status, retry_count  |
| Ingestor   | `redis.publish`              | stream, message_count               |
| Realtime   | `sse.connection`             | client_id, duration                 |
| Realtime   | `stream.consume`             | stream, lag, batch_size             |
| Realtime   | `sse.deliver`                | client_count, message_type          |

## Tempo Configuration

### Retention

| Environment | Retention | Storage Backend |
|-------------|-----------|-----------------|
| Dev         | 2 days    | Local           |
| Staging     | 7 days    | S3              |
| Production  | 14 days   | S3              |

### Sampling

| Environment | Strategy          | Rate  |
|-------------|-------------------|-------|
| Dev         | Always sample     | 100%  |
| Staging     | Probabilistic     | 50%   |
| Production  | Probabilistic     | 10%   |

Error traces are always sampled (100%) regardless of rate.

## Querying Traces (Grafana)

### TraceQL Queries

```
# Slow API requests
{ span.http.method = "GET" && duration > 500ms && resource.service.name = "market-api" }

# Provider failures
{ span.provider.name != "" && span.http.status_code >= 500 }

# Database slow queries
{ span.db.system = "postgresql" && duration > 100ms }

# Full trace by ID
{ trace:id = "4bf92f3577b34da6a3ce929d0e0e4736" }
```

### Grafana Dashboards

- **Service Map**: Visual dependency graph with error rates
- **Trace Explorer**: Search and filter traces
- **Latency Breakdown**: P50/P95/P99 by service and operation
- **Error Traces**: Failed requests with full span context

## Correlation: Logs ↔ Traces ↔ Metrics

| Signal  | Correlation Field | Use Case                          |
|---------|-------------------|-----------------------------------|
| Logs    | `trace_id`        | Jump from log entry to full trace |
| Traces  | `trace_id`        | Navigate span hierarchy           |
| Metrics | Exemplars         | Jump from metric spike to trace   |

In Grafana, clicking a log line with `trace_id` opens the corresponding trace in Tempo.
