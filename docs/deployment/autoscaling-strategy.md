# Autoscaling Strategy

## Overview

The Crypto Market Platform uses Horizontal Pod Autoscalers (HPA) with both resource-based and custom Prometheus metrics to scale workloads based on actual demand.

## Scaling Targets

| Service    | Min | Max | CPU Target | Memory Target | Custom Metric              |
|------------|-----|-----|------------|---------------|----------------------------|
| API        | 2   | 10  | 70%        | 80%           | http_requests_per_second   |
| Ingestor   | 1   | 5   | 75%        | 85%           | ingestion_queue_length     |
| Realtime   | 2   | 8   | 70%        | 80%           | realtime_active_connections|
| Frontend   | 2   | 6   | 70%        | 80%           | —                          |

## HPA Configuration

### Resource-Based Scaling

Standard CPU/memory utilization targets via `metrics-server`:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: market-api
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: market-api
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

### Custom Metrics (Prometheus Adapter)

Custom metrics are exposed via Prometheus Adapter for workload-specific scaling:

| Metric                        | Service   | Target | Description                          |
|-------------------------------|-----------|--------|--------------------------------------|
| `http_requests_per_second`    | API       | 100    | Scale when RPS exceeds 100/pod       |
| `realtime_active_connections` | Realtime  | 500    | Scale when connections exceed 500/pod|
| `ingestion_queue_length`      | Ingestor  | 1000   | Scale when queue depth exceeds 1000  |
| `provider_request_duration`   | Ingestor  | 2000ms | Scale when provider latency is high  |

### Prometheus Adapter Configuration

```yaml
rules:
  - seriesQuery: 'http_requests_total{namespace="cryptomarket-prod"}'
    resources:
      overrides:
        namespace: {resource: "namespace"}
        pod: {resource: "pod"}
    name:
      matches: "^(.*)$"
      as: "http_requests_per_second"
    metricsQuery: 'sum(rate(<<.Series>>{<<.LabelMatchers>>}[2m])) by (<<.GroupBy>>)'
```

## Scaling Behavior

### Scale-Up Policy

- Fast scale-up to handle traffic spikes
- `stabilizationWindowSeconds: 60` for scale-up
- `pods: 4` max scale-up per period (prevents thundering herd)

### Scale-Down Policy

- Conservative scale-down to prevent flapping
- `stabilizationWindowSeconds: 300` (5 min cooldown)
- `pods: 1` max scale-down per period

```yaml
behavior:
  scaleUp:
    stabilizationWindowSeconds: 60
    policies:
      - type: Pods
        value: 4
        periodSeconds: 60
  scaleDown:
    stabilizationWindowSeconds: 300
    policies:
      - type: Pods
        value: 1
        periodSeconds: 120
```

## Pod Disruption Budgets

PDBs ensure minimum availability during voluntary disruptions:

| Service    | minAvailable | Rationale                    |
|------------|--------------|------------------------------|
| API        | 1            | At least 1 pod always serving|
| Realtime   | 1            | Maintain SSE connections     |
| Ingestor   | —            | Can tolerate brief downtime  |
| Frontend   | 1            | At least 1 pod always serving|

## Cluster Autoscaler

Node-level scaling via EKS Cluster Autoscaler:
- Scales node groups when pods are pending due to resource constraints
- Scale-down delay: 10 minutes
- Minimum nodes: 2 (per AZ for HA)
- Maximum nodes: 12

## Monitoring

Key autoscaling metrics in Grafana:
- `kube_horizontalpodautoscaler_status_current_replicas`
- `kube_horizontalpodautoscaler_status_desired_replicas`
- `kube_horizontalpodautoscaler_spec_max_replicas`
- HPA events: `kubectl get events --field-selector reason=SuccessfulRescale`
