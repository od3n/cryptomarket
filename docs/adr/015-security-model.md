# ADR-015: Security Model

## Status

Accepted

## Context

The platform handles real-time financial market data and must meet security expectations comparable to production fintech systems. As the system matured through Phases 1–4, security controls were added incrementally. This ADR documents the consolidated security model.

## Decision

### Defense in Depth

Security is applied at every layer:

1. **Network**: VPC isolation, private subnets, NAT gateway, security groups with least-privilege ingress.
2. **Kubernetes**: Pod Security Admission (restricted), NetworkPolicies per service, RBAC via IRSA, ResourceQuotas, LimitRanges.
3. **Container**: Distroless images (no shell, no package manager), read-only root filesystem, dropped capabilities, non-root execution, seccomp RuntimeDefault.
4. **Application**: Security headers (CSP, HSTS, X-Frame-Options, nosniff), rate limiting (token bucket per IP), request body size limits, input validation.
5. **Data**: Encryption at rest (KMS for RDS/S3), TLS in transit, secrets via AWS Secrets Manager + External Secrets Operator.
6. **Supply Chain**: Cosign keyless signing, SBOM (Syft/SPDX), provenance attestation, dependency scanning (Trivy, Grype, govulncheck).
7. **CI/CD**: OIDC-only AWS auth (no long-lived credentials), CodeQL static analysis, Gitleaks secret scanning, Dependabot.

### Authentication (Future)

When authentication is added:
- JWT with short-lived tokens (15min access, 7d refresh)
- OAuth2/OIDC via external identity provider
- Rate limiting on auth endpoints (stricter: 10 req/min)
- Audit logging of all auth events
- CSRF protection via SameSite cookies + token binding

### Secrets Management

- No secrets in git (enforced by Gitleaks + branch protection)
- Kubernetes base manifests use placeholder values; real secrets injected via External Secrets Operator from AWS Secrets Manager
- Helm values reference secret names, never values
- Rotation policy: 90 days for database credentials

## Consequences

- Distroless images prevent debugging via `kubectl exec` — use ephemeral debug containers
- Rate limiting is per-pod (in-memory) — for multi-replica consistency, consider Redis-backed limiter
- Security headers block all framing — acceptable for API-only service
