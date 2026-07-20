# ADR-019: Release Strategy

## Status

Accepted

## Context

The platform requires a release process that minimizes risk to production users while enabling frequent deployments. The process must be auditable, reversible, and observable.

## Decision

### Release Pipeline

```
Feature → PR → CI → Security → Build → Sign → Publish → Canary → Observe → Promote → Rollback
```

### Stage Details

1. **Feature**: Developer creates feature branch, writes code + tests
2. **PR**: Pull request opened against `main`; requires 1 approval
3. **CI**: Lint, test, build, OpenAPI validation, monitoring config validation
4. **Security**: CodeQL, Trivy, Gitleaks, govulncheck, Checkov, kube-score
5. **Build**: Multi-stage Docker build with provenance
6. **Sign**: Cosign keyless signing + SBOM attestation
7. **Publish**: Push to ECR with SHA tag
8. **Canary**: Deploy to 5% → 25% → 50% with metric verification between steps
9. **Observe**: 60s observation window at each canary step; check error rate, latency, freshness
10. **Promote**: Full rollout via Helm upgrade
11. **Rollback**: Automatic `kubectl rollout undo` if promote fails; manual rollback available

### Canary Verification Criteria

| Metric | Threshold | Action |
|--------|-----------|--------|
| Error rate (5xx) | > 1% | Abort + rollback |
| p99 latency | > 500ms | Abort + rollback |
| Data freshness | > 120s | Warn (continue) |
| Pod restarts | > 2 | Abort + rollback |

### Versioning

- Images tagged by git SHA (immutable, traceable)
- Semantic version tags (`v1.2.3`) for releases
- Helm chart version tracks app version

### Rollback Procedures

- **Automated**: Canary failure triggers immediate rollback (< 2min)
- **Manual**: `helm rollback cryptomarket <revision>` or `kubectl rollout undo`
- **Database**: Migrations are forward-only; rollback uses previous image with compatible schema
- **Redis**: Schema-less; no rollback needed

### Frequency

- Target: 3+ deployments per week
- No deployment freeze windows (canary provides safety)
- Hotfixes: bypass canary with `skip_canary: true` (requires manual approval)

## Consequences

- Canary adds ~5 minutes to deployment time
- Requires Prometheus accessible from CI runner for metric verification
- Forward-only migrations require backward-compatible schema changes
- `skip_canary` is an escape hatch that increases risk — audit logged
