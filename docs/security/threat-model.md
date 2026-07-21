# Threat Model — Crypto Market Data Platform

> **Methodology**: STRIDE (Spoofing, Tampering, Repudiation, Information Disclosure, Denial of Service, Elevation of Privilege)
> **Last reviewed**: 2026-07-21
> **Next review**: 2027-07-21 (annual) or upon significant architecture change

## System Overview

```
                    ┌─────────────────────────────────────────────────┐
                    │                  Internet                        │
                    └─────────────┬───────────────────┬───────────────┘
                                  │                   │
                    ┌─────────────▼──────┐  ┌────────▼──────────────┐
                    │   Ingress (nginx)  │  │   Provider APIs       │
                    │   TLS termination  │  │   (CoinGecko/CoinCap) │
                    └────────┬───────────┘  └────────▲──────────────┘
                             │                       │
              ┌──────────────▼───────────────────────┼──────────────┐
              │              Kubernetes Cluster       │              │
              │                                      │              │
              │  ┌──────────┐  ┌──────────┐  ┌──────┴─────┐       │
              │  │ Frontend │  │   API    │  │  Ingestor  │       │
              │  │ (Next.js)│  │  (Go)    │  │   (Go)     │       │
              │  └──────────┘  └────┬─────┘  └──────┬─────┘       │
              │                     │               │              │
              │  ┌──────────┐  ┌───▼───────────────▼──┐           │
              │  │ Realtime │  │      Redis           │           │
              │  │  (SSE)   │  │  (cache + streams)   │           │
              │  └──────────┘  └──────────────────────┘           │
              │                     │                              │
              │              ┌──────▼──────┐                      │
              │              │ PostgreSQL  │                      │
              │              │  (history)  │                      │
              │              └─────────────┘                      │
              └───────────────────────────────────────────────────┘
```

## Trust Boundaries

| # | Boundary | Crosses |
|---|----------|---------|
| TB-1 | Internet → Ingress | External users → platform |
| TB-2 | Ingress → Services | Ingress controller → application pods |
| TB-3 | Services → Data Stores | Application → PostgreSQL/Redis |
| TB-4 | Ingestor → Provider APIs | Platform → external third-party APIs |
| TB-5 | CI/CD → Production | Build pipeline → runtime environment |
| TB-6 | Operator → Cluster | Human → Kubernetes API |

## STRIDE Analysis

### 1. Spoofing

| ID | Threat | Component | Likelihood | Impact | Mitigation | Status |
|----|--------|-----------|-----------|--------|------------|--------|
| S-1 | Attacker impersonates legitimate user | API (TB-1) | High | Medium | Rate limiting per IP; **future**: JWT authentication (SEC-04) | Partial |
| S-2 | Attacker spoofs X-Forwarded-For header | API middleware | Medium | Low | Trust only first hop from known ingress; document limitation | Mitigated |
| S-3 | Compromised CI pipeline pushes malicious image | CI/CD (TB-5) | Low | Critical | Cosign keyless signing tied to GitHub OIDC; signature verification at deploy | Mitigated |
| S-4 | Attacker impersonates provider API | Ingestor (TB-4) | Low | High | HTTPS with certificate validation; pinned CA bundle in distroless image | Mitigated |
| S-5 | Unauthorized kubectl access to cluster | K8s API (TB-6) | Medium | Critical | OIDC-only auth, RBAC least-privilege, no static credentials | Mitigated |

### 2. Tampering

| ID | Threat | Component | Likelihood | Impact | Mitigation | Status |
|----|--------|-----------|-----------|--------|------------|--------|
| T-1 | Attacker modifies market data in transit | API ↔ Client (TB-1) | Low | High | TLS 1.3 enforced; HSTS headers; certificate pinning (future) | Mitigated |
| T-2 | Attacker modifies data in Redis | Redis (TB-3) | Low | High | NetworkPolicy restricts access to app pods only; no external exposure; AUTH enabled in prod | Mitigated |
| T-3 | Attacker modifies PostgreSQL data | PostgreSQL (TB-3) | Low | Critical | Parameterized queries (no SQL injection); NetworkPolicy; TLS in prod; least-privilege DB user | Mitigated |
| T-4 | Attacker tampers with Terraform state | S3 backend | Low | Critical | State locking (DynamoDB); encryption at rest (KMS); IAM restricted to CI role | Mitigated |
| T-5 | Malicious dependency introduced | go.mod / package.json | Medium | High | Dependabot; govulncheck; npm audit; lock files; license check | Mitigated |
| T-6 | Attacker modifies container image in registry | ECR | Low | Critical | Immutable tags (SHA); Cosign signature verification before deploy; ECR scan-on-push | Mitigated |

### 3. Repudiation

| ID | Threat | Component | Likelihood | Impact | Mitigation | Status |
|----|--------|-----------|-----------|--------|------------|--------|
| R-1 | Attacker denies making API requests | API | Medium | Low | Request ID logging; structured access logs with client IP; **future**: audit logging with auth | Partial |
| R-2 | Operator denies making cluster changes | K8s API | Low | Medium | Kubernetes audit logging enabled; CloudTrail for AWS API calls | Mitigated |
| R-3 | Developer denies pushing vulnerable code | CI/CD | Low | Low | Git history immutable; signed commits (future); PR review trail | Partial |

### 4. Information Disclosure

| ID | Threat | Component | Likelihood | Impact | Mitigation | Status |
|----|--------|-----------|-----------|--------|------------|--------|
| I-1 | Database credentials exposed in logs/config | All services | Low | Critical | Secrets via AWS Secrets Manager + External Secrets Operator; Gitleaks in CI; no secrets in git | Mitigated |
| I-2 | pprof endpoints exposed to internet | API | Low | High | NetworkPolicy restricts /debug/pprof to internal; not exposed via ingress | Mitigated |
| I-3 | Error messages leak internal details | API | Medium | Low | Generic error responses to clients; detailed errors only in structured logs | Mitigated |
| I-4 | Prometheus metrics expose sensitive data | Metrics | Low | Medium | No PII in metric labels; metrics endpoint internal-only in production | Mitigated |
| I-5 | Container filesystem exposes secrets | Pods | Low | High | Distroless images (no shell); read-only root filesystem; secrets mounted as tmpfs | Mitigated |
| I-6 | Terraform state contains secrets | S3 | Low | Critical | State encrypted at rest (KMS); access restricted to CI role; no secrets in .tf files | Mitigated |

### 5. Denial of Service

| ID | Threat | Component | Likelihood | Impact | Mitigation | Status |
|----|--------|-----------|-----------|--------|------------|--------|
| D-1 | Volumetric DDoS overwhelms ingress | Ingress (TB-1) | Medium | High | nginx rate limiting (50 rps); connection limits; **future**: AWS WAF + Shield (SEC-05) | Partial |
| D-2 | Application-layer DDoS (slowloris, etc.) | API | Medium | Medium | Request timeout; max body size (1MiB); per-IP token bucket rate limiter | Mitigated |
| D-3 | Provider rate limiting starves ingestion | Ingestor (TB-4) | High | Medium | Circuit breaker; exponential backoff; multi-provider fallback; cache serves stale data | Mitigated |
| D-4 | Redis memory exhaustion | Redis (TB-3) | Low | High | maxmemory policy; TTL on all keys (5min); monitoring + alerts on memory usage | Mitigated |
| D-5 | PostgreSQL connection exhaustion | PostgreSQL (TB-3) | Low | High | Connection pool limits (25 max open, 5 idle); PgBouncer (future) | Mitigated |
| D-6 | SSE connection exhaustion (slow clients) | Realtime | Medium | Medium | Per-connection write deadline; backpressure with message dropping; max connections per pod | Mitigated |
| D-7 | Fork bomb / resource exhaustion in pod | K8s | Low | Medium | Resource limits (CPU/memory) on all pods; LimitRange; ResourceQuota per namespace | Mitigated |

### 6. Elevation of Privilege

| ID | Threat | Component | Likelihood | Impact | Mitigation | Status |
|----|--------|-----------|-----------|--------|------------|--------|
| E-1 | Container escape to host | Pods | Low | Critical | Distroless (no shell/tools); non-root; dropped capabilities; seccomp RuntimeDefault; Pod Security Admission restricted | Mitigated |
| E-2 | Lateral movement between pods | K8s network | Low | High | NetworkPolicies per service (default-deny); no shared service accounts | Mitigated |
| E-3 | Compromised pod accesses AWS APIs | IAM | Low | Critical | IRSA (IAM Roles for Service Accounts); least-privilege policies; no node-level IAM | Mitigated |
| E-4 | CI pipeline compromise leads to prod access | CI/CD (TB-5) | Low | Critical | OIDC-only (short-lived tokens); no long-lived AWS credentials; separate deploy role | Mitigated |
| E-5 | SQL injection leads to data access | PostgreSQL | Low | Critical | Parameterized queries only; no string concatenation; input validation | Mitigated |

## Risk Summary

| Risk Level | Count | Items |
|-----------|-------|-------|
| **Fully Mitigated** | 22 | T-1 through E-5 (majority) |
| **Partially Mitigated** | 4 | S-1 (no auth yet), R-1 (no audit log), R-3 (no signed commits), D-1 (no WAF yet) |
| **Accepted** | 0 | — |
| **Open** | 0 | — |

## Remediation Roadmap

| Priority | Item | Work Package | Target |
|----------|------|-------------|--------|
| P1 | API Authentication (S-1, R-1) | SEC-04 | M6 |
| P1 | WAF + DDoS Protection (D-1) | SEC-05 | M2 |
| P2 | Signed commits (R-3) | — | M6 |
| P2 | Runtime security monitoring | SEC-03 (Falco) | M2 |

## Assumptions

1. TLS certificates are managed by cert-manager/ACM and auto-renewed
2. Kubernetes cluster is managed by EKS (AWS handles control plane security)
3. Provider APIs are trusted but validated (response schema validation)
4. CI/CD pipeline (GitHub Actions) is trusted infrastructure
5. No insider threat modeling (standard access controls assumed sufficient)

## Review Triggers

This threat model must be reviewed when:
- A new service or data store is added
- Authentication/authorization model changes
- Network topology changes (new trust boundaries)
- A security incident occurs
- Annually (regardless of changes)

## References

- [ADR-015: Security Model](../adr/015-security-model.md)
- [ADR-017: Supply Chain Security](../adr/017-supply-chain-security.md)
- [Container Security](container-security.md)
- [Secrets Management](secrets-management.md)
- [IAM Least Privilege](iam-least-privilege.md)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
