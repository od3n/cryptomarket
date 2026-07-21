# Operational Knowledge Model

## Purpose

Define a structured knowledge graph connecting all operational artifacts in this repository. The model enables AI assistants to navigate relationships (e.g., "alert → runbook → service → dashboard → SLO") without inventing connections.

**Core rule:** Every edge in the graph must be derivable from a parseable artifact (annotation, import, file reference, label). No inferred edges are stored as facts.

---

## Entity Types

### Services

| Entity ID | Type | Source Artifact | Attributes |
|-----------|------|----------------|-----------|
| `svc:market-api` | Go service | `cmd/api/main.go`, `internal/api/` | port 8080, endpoints: /health, /ready, /coins, /markets, /providers/status, /metrics |
| `svc:market-ingestor` | Go worker | `cmd/ingestor/main.go`, `internal/worker/` | background-only, 60s interval, providers: coingecko, coincap |
| `svc:market-realtime` | Go SSE gateway | `cmd/realtime/main.go`, `internal/realtime/` | port 8081, Redis Streams consumer |
| `svc:market-frontend` | Next.js app | `frontend/` | port 3000, SSE client |
| `svc:mock-provider` | Test double | `cmd/mockprovider/main.go` | port 8082, modes: success/failure |

### Data Stores

| Entity ID | Type | Source Artifact |
|-----------|------|----------------|
| `store:postgresql` | RDS PostgreSQL Multi-AZ | `deploy/terraform/modules/rds/`, `migrations/` |
| `store:redis` | ElastiCache Redis Multi-AZ | `deploy/terraform/modules/elasticache/`, `deploy/redis/redis-prod.conf` |
| `store:redis-streams` | Redis Streams (stream: `market:prices`) | `internal/stream/`, ADR-002, ADR-004 |

### External Dependencies

| Entity ID | Type | Source Artifact |
|-----------|------|----------------|
| `ext:coingecko` | Primary provider | `internal/provider/coingecko.go`, ADR-007 |
| `ext:coincap` | Fallback provider | `internal/provider/coincap.go`, ADR-007 |

### Alerts (18 total, from `monitoring/prometheus/alerts.yml`)

| Entity ID | Severity | Service Label |
|-----------|----------|--------------|
| `alert:APIUnavailable` | critical | market-api |
| `alert:IngestionFailing` | critical | market-ingestor |
| `alert:AllProvidersUnavailable` | critical | market-ingestor |
| `alert:DataStaleCritical` | critical | market-ingestor |
| `alert:PostgreSQLUnavailable` | critical | postgres |
| `alert:RedisUnavailable` | critical | redis |
| `alert:PrimaryProviderDown` | warning | market-ingestor |
| `alert:ProviderHighLatency` | warning | market-ingestor |
| `alert:HighRateLimitFrequency` | warning | market-ingestor |
| `alert:APIHighLatency` | warning | market-api |
| `alert:RealtimeConsumerLag` | warning | realtime-gateway |
| `alert:SustainedStaleSymbols` | warning | market-ingestor |
| `alert:ProviderDataMismatch` | warning | market-ingestor |
| `alert:APIAvailabilityFastBurn` | critical | market-api |
| `alert:APIAvailabilitySlowBurn` | warning | market-api |
| `alert:IngestionErrorBudgetBurn` | warning | market-ingestor |
| `alert:APIAvailabilityMultiWindowBurn` | critical | market-api |

### Runbooks (12, from `docs/runbooks/`)

| Entity ID | File |
|-----------|------|
| `runbook:api-unavailable` | `docs/runbooks/api-unavailable.md` |
| `runbook:ingestion-failure` | `docs/runbooks/ingestion-failure.md` |
| `runbook:all-providers-unavailable` | `docs/runbooks/all-providers-unavailable.md` |
| `runbook:data-freshness-alert` | `docs/runbooks/data-freshness-alert.md` |
| `runbook:postgresql-unavailable` | `docs/runbooks/postgresql-unavailable.md` |
| `runbook:redis-unavailable` | `docs/runbooks/redis-unavailable.md` |
| `runbook:provider-unavailable` | `docs/runbooks/provider-unavailable.md` |
| `runbook:provider-rate-limiting` | `docs/runbooks/provider-rate-limiting.md` |
| `runbook:provider-ingestion-failures` | `docs/runbooks/provider-ingestion-failures.md` |
| `runbook:high-api-latency` | `docs/runbooks/high-api-latency.md` |
| `runbook:realtime-delivery-degraded` | `docs/runbooks/realtime-delivery-degraded.md` |
| `runbook:error-budget-burn` | `docs/runbooks/error-budget-burn.md` |

### SLOs (5, from `docs/sre/slos.md`)

| Entity ID | Target | Window |
|-----------|--------|--------|
| `slo:api-availability` | 99.9% | 30d |
| `slo:api-latency` | 99% (p99 < 300ms) | 30d |
| `slo:ingestion-success` | 99.5% | 30d |
| `slo:data-freshness` | 99% | 30d |
| `slo:realtime-delivery` | 99.5% | 30d |

### Dashboards (5, from `monitoring/grafana/dashboards/`)

| Entity ID | File |
|-----------|------|
| `dash:platform-overview` | `platform-overview.json` |
| `dash:provider-reliability` | `provider-reliability.json` |
| `dash:slo-error-budget` | `slo-error-budget.json` |
| `dash:performance-engineering` | `performance-engineering.json` |
| `dash:security-posture` | `security-posture.json` |

### ADRs (20, from `docs/adr/`)

| Entity ID | Title | Status |
|-----------|-------|--------|
| `adr:001` | Go + Python tooling | Accepted |
| `adr:002` | PostgreSQL + Redis Streams | Accepted |
| `adr:003` | SSE over WebSockets | Accepted |
| `adr:004` | Redis Streams consumer groups | Accepted |
| `adr:005` | Latest-state delivery policy | Accepted |
| `adr:006` | Frontend framework and deployment | Accepted |
| `adr:007` | Provider fallback | Accepted |
| `adr:008` | Retry ownership | Accepted |
| `adr:009` | Circuit breaker | Accepted |
| `adr:010` | Freshness source | Accepted |
| `adr:011` | SLO definitions | Accepted |
| `adr:012` | Burn-rate alerting | Accepted |
| `adr:013` | Degraded mode semantics | Accepted |
| `adr:014` | Failure injection safeguards | Accepted |
| `adr:015` | Security model | Accepted |
| `adr:016` | Performance strategy | Accepted |
| `adr:017` | Supply chain security | Accepted |
| `adr:018` | Chaos testing | Accepted |
| `adr:019` | Release strategy | Accepted |
| `adr:020` | Multi-region architecture | Accepted |

### Incidents

| Entity ID | Date | Severity | Duration |
|-----------|------|----------|----------|
| `inc:001` | 2024-01-10 | SEV-2 | 47 min |

### Deployments

| Entity ID | Source |
|-----------|--------|
| `deploy:staging` | `.github/workflows/deploy-staging.yml` |
| `deploy:production` | `.github/workflows/deploy-production.yml` (canary 5%→25%→50%→100%) |

### Infrastructure

| Entity ID | Source |
|-----------|--------|
| `infra:eks` | `deploy/terraform/modules/eks/` |
| `infra:rds` | `deploy/terraform/modules/rds/` |
| `infra:elasticache` | `deploy/terraform/modules/elasticache/` |
| `infra:networking` | `deploy/terraform/modules/networking/` |
| `infra:iam` | `deploy/terraform/modules/iam/` |
| `infra:waf` | `deploy/terraform/modules/waf/` |
| `infra:s3` | `deploy/terraform/modules/s3/` |
| `infra:kms` | `deploy/terraform/modules/kms/` |
| `infra:dns` | `deploy/terraform/modules/dns/` |
| `infra:acm` | `deploy/terraform/modules/acm/` |
| `infra:secrets` | `deploy/terraform/modules/secrets/` |
| `infra:monitoring` | `deploy/terraform/modules/monitoring/` |
| `infra:secrets-rotation` | `deploy/terraform/modules/secrets-rotation/` |

---

## Relationship Types

### Derivation Rules

| Relationship | Derivation Source | Example |
|-------------|------------------|---------|
| `ALERT_HAS_RUNBOOK` | `runbook:` annotation in alerts.yml | `alert:APIUnavailable` → `runbook:api-unavailable` |
| `ALERT_AFFECTS_SERVICE` | `service:` label in alerts.yml | `alert:IngestionFailing` → `svc:market-ingestor` |
| `ALERT_MEASURES_SLO` | `slo:` label in alerts.yml | `alert:APIAvailabilityFastBurn` → `slo:api-availability` |
| `SERVICE_DEPENDS_ON` | Import statements, config references | `svc:market-api` → `store:redis`, `store:postgresql` |
| `SERVICE_CONSUMES` | Provider adapter registrations | `svc:market-ingestor` → `ext:coingecko`, `ext:coincap` |
| `SERVICE_PUBLISHES_TO` | Stream producer code | `svc:market-ingestor` → `store:redis-streams` |
| `SERVICE_SUBSCRIBES_TO` | Consumer group code | `svc:market-realtime` → `store:redis-streams` |
| `DASHBOARD_COVERS_SLO` | Panel expressions matching SLO metrics | `dash:slo-error-budget` → `slo:api-availability` |
| `RUNBOOK_ADDRESSES_ALERT` | Inverse of ALERT_HAS_RUNBOOK | `runbook:error-budget-burn` → 4 burn alerts |
| `ADR_GOVERNS` | ADR content references | `adr:009` → circuit breaker in `svc:market-ingestor` |
| `INCIDENT_TRIGGERED_ALERT` | Postmortem timeline | `inc:001` → `alert:DataStaleCritical` |
| `INCIDENT_AFFECTED_SERVICE` | Postmortem impact section | `inc:001` → `svc:market-ingestor` |
| `DEPLOY_TARGETS` | Workflow namespace config | `deploy:production` → `svc:*` (namespace cryptomarket-prod) |
| `INFRA_HOSTS` | Terraform module purpose | `infra:eks` → all services |
| `INFRA_PROVIDES` | Module outputs | `infra:rds` → `store:postgresql` |

### Known Causal Chains (from postmortem 001 and runbooks)

```
ext:coingecko rate-limits
  → alert:HighRateLimitFrequency
  → alert:PrimaryProviderDown (circuit breaker opens)
  → alert:IngestionFailing (if no fallback healthy)
  → alert:DataStaleCritical (>10 min)
  → slo:data-freshness budget burns
  → runbook:provider-rate-limiting / runbook:data-freshness-alert
```

---

## Graph Schema

```yaml
# knowledge-graph-schema.yaml
entities:
  - id: string          # "type:name" format
    type: enum          # service|store|external|alert|runbook|slo|dashboard|adr|incident|deployment|infra
    name: string
    source_file: string # artifact that defines this entity
    attributes: map     # type-specific metadata

relationships:
  - from: entity_id
    to: entity_id
    type: enum          # see Relationship Types above
    derivation: string  # exact artifact + location proving this edge
    confidence: enum    # derived (100%, parsed) | documented (explicit in docs)
```

---

## Query Patterns for Assistants

| Assistant | Query | Traversal |
|-----------|-------|-----------|
| Incident Assistant | "What runbook applies to this alert group?" | alert → ALERT_HAS_RUNBOOK → runbook |
| Incident Assistant | "What services are affected downstream?" | alert → service → SERVICE_DEPENDS_ON → stores → other services |
| Deployment Advisor | "Which SLOs does this service affect?" | service ← ALERT_AFFECTS_SERVICE ← alert → ALERT_MEASURES_SLO → slo |
| Infrastructure Review | "What services run on this infra?" | infra → INFRA_HOSTS → services |
| Documentation Assistant | "Which ADRs govern this service?" | service ← ADR_GOVERNS ← adr |
| Observability Assistant | "Is this SLO covered by dashboards?" | slo ← DASHBOARD_COVERS_SLO ← dashboard (gap = no edge) |

---

## Maintenance

| Trigger | Action |
|---------|--------|
| Alert rule added/modified in alerts.yml | Re-parse annotations; update alert entities and edges |
| New runbook added | Parse alert references in runbook; add edges |
| New ADR merged | Extract governed components from content |
| Postmortem published | Extract timeline events → incident-alert edges |
| Terraform module added | Add infra entity; link to environment |
| Service added | Parse imports/config for dependency edges |

**Validation:** A CI job verifies that every alert has at least one runbook edge and every SLO has at least one dashboard edge. Missing edges generate warnings (not failures) for human review.
