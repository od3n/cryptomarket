# Production Logging

## Overview

The Crypto Market Platform uses structured JSON logging shipped to Grafana Loki via Promtail/Alloy. All services emit structured logs with correlation IDs for distributed traceability.

## Architecture

```
Application Pods → stdout/stderr → Promtail (DaemonSet) → Loki → Grafana
```

## Log Format

All Go services emit structured JSON logs:

```json
{
  "level": "info",
  "timestamp": "2024-01-15T10:30:00.123Z",
  "service": "market-api",
  "trace_id": "abc123def456",
  "span_id": "789ghi",
  "message": "request completed",
  "method": "GET",
  "path": "/api/v1/ticker/BTC-USD",
  "status": 200,
  "duration_ms": 12.5,
  "client_ip": "10.0.1.50"
}
```

### Required Fields

| Field       | Description                          | Example              |
|-------------|--------------------------------------|----------------------|
| `level`     | Log level (debug/info/warn/error)    | `info`               |
| `timestamp` | RFC3339 with milliseconds            | `2024-01-15T10:30:00.123Z` |
| `service`   | Service identifier                   | `market-api`         |
| `trace_id`  | Distributed trace correlation ID     | `abc123def456`       |
| `message`   | Human-readable log message           | `request completed`  |

## Log Levels

| Level   | Usage                                          | Production |
|---------|------------------------------------------------|------------|
| DEBUG   | Detailed flow, variable values                 | Disabled   |
| INFO    | Normal operations, request completion          | Enabled    |
| WARN    | Degraded state, retries, fallbacks             | Enabled    |
| ERROR   | Failures requiring attention                   | Enabled    |

## Service-Specific Logging

### API Service

- Request/response logging with method, path, status, duration
- Database query duration (slow query > 100ms logged as WARN)
- Provider API call results

### Ingestor Service

- Ingestion cycle start/complete with record counts
- Provider failover events (INFO)
- Rate limit hits (WARN)
- Provider errors (ERROR)

### Realtime Service

- SSE connection open/close with client count
- Redis Stream consumer lag
- Message delivery latency

### Frontend (Next.js)

- Server-side request logs
- API proxy errors
- Static asset cache hits/misses

## Loki Configuration

### Retention

| Environment | Retention | Storage     |
|-------------|-----------|-------------|
| Dev         | 3 days    | Local PVC   |
| Staging     | 7 days    | S3          |
| Production  | 30 days   | S3          |

### Labels

```yaml
scrape_configs:
  - job_name: kubernetes-pods
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_namespace]
        target_label: namespace
      - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_name]
        target_label: app
      - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_component]
        target_label: component
```

## Querying Logs (Grafana)

### Common Queries

```logql
# All errors from API in last hour
{namespace="cryptomarket-prod", app="market-api"} |= "error" | json | level="error"

# Trace a specific request
{namespace="cryptomarket-prod"} | json | trace_id="abc123def456"

# Slow requests (>500ms)
{app="market-api"} | json | duration_ms > 500

# Provider failover events
{app="market-ingestor"} |= "failover" | json

# SSE connection events
{app="market-realtime"} |= "connection" | json
```

### Dashboards

- **Service Overview**: Error rate, log volume by level, top error messages
- **Request Tracing**: Single trace view across all services
- **Ingestion Health**: Provider status, cycle duration, record counts
- **Realtime**: Connection count, delivery latency, consumer lag

## Alerting on Logs

| Alert                    | Condition                              | Severity |
|--------------------------|----------------------------------------|----------|
| High error rate          | `rate({app="market-api"} \|= "error" [5m]) > 10` | Critical |
| Provider all failing     | `{app="market-ingestor"} \|= "all providers failed"` | Critical |
| OOM kills                | `kube_pod_container_status_last_terminated_reason{reason="OOMKilled"}` | Warning |
| Slow queries             | `rate({app="market-api"} \| json \| duration_ms > 1000 [5m]) > 5` | Warning |

## Best Practices

1. **Never log secrets** — Mask API keys, passwords, tokens
2. **Include trace_id** — Enable cross-service correlation
3. **Structured over unstructured** — JSON enables filtering
4. **Appropriate levels** — Don't cry wolf with ERROR for expected retries
5. **Bounded cardinality** — Don't log user-specific IDs in high-volume paths
