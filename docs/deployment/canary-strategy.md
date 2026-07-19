# Canary Deployment Strategy

## Overview

The Crypto Market Platform uses progressive canary deployments for production releases. New versions are gradually rolled out to a subset of pods while monitoring key metrics, with automatic rollback on threshold breach.

## Deployment Flow

```
Push to main → CI Build → Push Image → Deploy Canary (5%) → Verify → 25% → Verify → 50% → Verify → 100% → Cleanup
```

## Progressive Rollout Steps

| Step | Traffic Weight | Duration | Action on Failure |
|------|---------------|----------|-------------------|
| 1    | 5%            | 5 min    | Immediate rollback |
| 2    | 25%           | 5 min    | Immediate rollback |
| 3    | 50%           | 5 min    | Immediate rollback |
| 4    | 100%          | —        | Full promotion     |

## Verification Gates

Each step must pass all gates before progressing:

| Metric              | Threshold     | Source              |
|---------------------|---------------|---------------------|
| Error rate          | < 1%          | Prometheus          |
| P99 latency         | < 500ms       | Prometheus          |
| Availability        | > 99.9%       | Kubernetes probes   |
| Data freshness      | < 120s        | Custom metric       |

## Implementation

### Helm Canary Release

The canary is deployed as a separate Helm release (`cryptomarket-canary`) with:
- Dedicated Deployment (`market-api-canary`) with `app.kubernetes.io/track: canary` label
- Dedicated Service for traffic routing
- Same ConfigMap/Secret as stable release

### CI/CD Integration

The `deploy-production.yml` workflow handles:
1. Deploy canary release with new image tag
2. Scale canary replicas progressively (1 → 2 → 4 → full)
3. Query Prometheus metrics between each step
4. On success: update stable release, delete canary
5. On failure: `kubectl rollout undo` + delete canary release

### Rollback Procedure

Automatic rollback triggers when any verification gate fails:

```bash
# Undo canary deployment
helm uninstall cryptomarket-canary -n cryptomarket-prod

# If stable was already updated
kubectl rollout undo deployment/market-api -n cryptomarket-prod
```

## Configuration

Canary settings in `values-prod.yaml`:

```yaml
canary:
  enabled: false  # Set true during deployment
  replicas: 1
  steps:
    - weight: 5
      pause: 300
    - weight: 25
      pause: 300
    - weight: 50
      pause: 300
    - weight: 100
  analysis:
    errorRateThreshold: 1.0
    latencyP99Threshold: 500
    availabilityThreshold: 99.9
    freshnessThreshold: 120
```

## Monitoring During Canary

Key dashboards to watch:
- **API Error Rate**: `rate(http_requests_total{status=~"5.."}[5m])`
- **P99 Latency**: `histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))`
- **Pod Restarts**: `kube_pod_container_status_restarts_total`
- **Data Freshness**: `ingestion_last_success_timestamp`

## Blue/Green Alternative

For major version upgrades or schema migrations, use blue/green:
1. Deploy full new version as "green"
2. Run integration tests against green
3. Switch ingress to green
4. Monitor for 15 minutes
5. Decommission blue

Blue/green is preferred when:
- Database schema changes are not backward-compatible
- Multiple services must be updated atomically
- Rollback must be instantaneous (< 5s)
