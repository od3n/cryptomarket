# ADR-017: Supply Chain Security

## Status

Accepted

## Context

Software supply chain attacks (SolarWinds, Log4Shell, codecov) demonstrate that build pipelines and dependencies are high-value targets. The platform must ensure that every artifact deployed to production is verifiable, traceable, and free of known vulnerabilities.

## Decision

### Image Build Pipeline

1. **Build**: Multi-stage Dockerfile with pinned base images (`golang:1.21-alpine` builder, `gcr.io/distroless/static-debian12:nonroot` runtime)
2. **Provenance**: Docker Buildx generates SLSA provenance attestation (`provenance: mode=max`)
3. **SBOM**: Syft generates SPDX SBOM attached as Cosign attestation
4. **Signing**: Cosign keyless signing via GitHub OIDC (no long-lived keys)
5. **Verification**: Deployment pipeline verifies signatures before rollout

### Dependency Management

- **Go**: `govulncheck` in CI; Dependabot weekly updates; `go.sum` lock file
- **Node**: `npm audit` in CI; Dependabot; `package-lock.json` lock file
- **Python**: `pip-audit` (future); Dependabot; `requirements.txt` pinned
- **Docker**: Dependabot for base image updates
- **Terraform**: Dependabot for provider updates; Checkov + Trivy IaC scanning
- **GitHub Actions**: Dependabot; pinned to major versions

### Verification at Deploy Time

```
cosign verify \
  --certificate-identity-regexp ".*" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  <image>:<tag>
```

Deployment is blocked if signature verification fails.

### Immutable Builds

- Images tagged by git SHA (immutable)
- `latest` tag updated for convenience but never used in production Helm values
- No `:latest` in production deployments

## Consequences

- Keyless signing ties trust to GitHub Actions OIDC — compromise of CI = compromise of signatures
- SBOM increases build time by ~10s per image
- Provenance attestation requires Buildx (not available in all CI environments)
- Verification step adds ~5s to deployment pipeline
