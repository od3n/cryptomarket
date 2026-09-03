# Container Security

## Overview

Container security for the Crypto Market Platform covers image building, scanning, signing, runtime hardening, and network policies.

## Image Building

### Base Images

| Service    | Build Stage       | Runtime Stage  | User  |
|------------|-------------------|----------------|-------|
| Go services| golang:1.21-alpine| alpine:3.19    | 10001 |
| Frontend   | node:26-alpine    | alpine:3.19    | 1001  |

### Hardening Practices

- Multi-stage builds (no build tools in runtime)
- Non-root user execution
- Read-only root filesystem where possible
- No shell in production (distroless considered)
- Minimal package installation (`ca-certificates` only)
- `.dockerignore` excludes source, tests, docs

### Dockerfile Security

```dockerfile
# Runtime stage
FROM alpine:3.19
RUN addgroup -g 10001 appgroup && \
    adduser -u 10001 -G appgroup -s /sbin/nologin -D appuser
USER 10001
```

## Vulnerability Scanning

### Trivy (CI Pipeline)

Container images scanned on every build:

```yaml
# .github/workflows/security.yml
- name: Run Trivy
  uses: aquasecurity/trivy-action@master
  with:
    image-ref: market-api:latest
    severity: CRITICAL,HIGH
    exit-code: 1  # Fail on critical/high
```

### Scan Policy

| Severity | Action                    | SLA     |
|----------|---------------------------|---------|
| CRITICAL | Block deployment          | 24h fix |
| HIGH     | Block deployment          | 72h fix |
| MEDIUM   | Warn, track in backlog    | 30 days |
| LOW      | Informational             | Best effort |

### IaC Scanning

Terraform configurations scanned with Trivy:
```bash
trivy config deploy/terraform/ --severity CRITICAL,HIGH,MEDIUM
```

## Image Signing (Cosign)

### Signing Process

Images are signed after successful CI build:

```bash
# Sign with keyless (OIDC)
cosign sign --yes $ECR_REGISTRY/market-api:$TAG

# Verify before deployment
cosign verify $ECR_REGISTRY/market-api:$TAG \
  --certificate-identity-regexp="github.com/crypto-market-platform" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
```

### Admission Control

In production, an admission webhook (Kyverno/OPA) rejects unsigned images:

```yaml
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: verify-image-signatures
spec:
  validationFailureAction: Enforce
  rules:
    - name: verify-signature
      match:
        resources:
          kinds: ["Pod"]
      verifyImages:
        - imageReferences:
            - "*.dkr.ecr.*.amazonaws.com/market-*"
          attestors:
            - entries:
                - keyless:
                    issuer: https://token.actions.githubusercontent.com
```

## SBOM (Software Bill of Materials)

Generated with Syft for every image:

```bash
syft market-api:latest -o spdx-json > sbom-api.spdx.json
```

SBOMs stored as GitHub Actions artifacts and attached to ECR images.

## Runtime Security

### Pod Security Standards

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 10001
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
```

### Network Policies

Default-deny with explicit allow rules:
- API → PostgreSQL, Redis
- Ingestor → Redis, external APIs (egress)
- Realtime → Redis
- Frontend → API (internal only)
- Monitoring → All pods (metrics scrape)

### Resource Limits

All containers have CPU/memory requests and limits to prevent:
- Resource exhaustion attacks
- Noisy neighbor problems
- OOM kills from memory leaks

## Dependency Scanning

| Tool         | Target          | Frequency  |
|--------------|-----------------|------------|
| govulncheck  | Go dependencies | Every CI   |
| npm audit    | Node packages   | Every CI   |
| Trivy        | OS packages     | Every build|
| Dependabot   | All             | Daily      |

## Compliance Checklist

- [ ] All images scanned before deployment
- [ ] No CRITICAL/HIGH vulnerabilities in production images
- [ ] Images signed with Cosign
- [ ] SBOM generated and stored
- [ ] Non-root execution enforced
- [ ] Network policies active
- [ ] Secrets not baked into images
- [ ] Base images updated monthly
