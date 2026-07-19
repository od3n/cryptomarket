# Secrets Management

## Overview

The Crypto Market Platform uses AWS Secrets Manager for secret storage with External Secrets Operator (ESO) for Kubernetes integration. No secrets are stored in Git, Helm values, or environment files.

## Architecture

```
AWS Secrets Manager → External Secrets Operator → Kubernetes Secrets → Pods
```

## Secret Inventory

| Secret Name                    | Contents                          | Rotation    |
|--------------------------------|-----------------------------------|-------------|
| `cryptomarket/prod/postgres`   | host, port, username, password, dbname | 90 days |
| `cryptomarket/prod/redis`      | host, port, auth_token           | 90 days     |
| `cryptomarket/prod/api-keys`   | Provider API keys (CoinGecko, etc.) | Manual   |
| `cryptomarket/prod/encryption` | JWT signing key, session secret  | 180 days    |

## Policies

### What Is a Secret

- Database credentials
- API keys and tokens
- TLS private keys
- JWT signing secrets
- Encryption keys
- Any value that grants access or could cause harm if exposed

### Rules

1. **Never commit secrets to Git** — Use `.gitleaks.toml` for enforcement
2. **Never put secrets in Helm values files** — Reference ExternalSecret resources
3. **Never log secrets** — Mask in application logs
4. **Rotate regularly** — Automated where possible (RDS managed passwords)
5. **Least privilege** — Each service only accesses its own secrets
6. **Audit access** — CloudTrail logs all Secrets Manager API calls

## External Secrets Operator

### ExternalSecret Resource

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: market-api-secrets
  namespace: cryptomarket-prod
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager
    kind: ClusterSecretStore
  target:
    name: market-api-secrets
    creationPolicy: Owner
  data:
    - secretKey: DATABASE_URL
      remoteRef:
        key: cryptomarket/prod/postgres
        property: connection_string
    - secretKey: REDIS_URL
      remoteRef:
        key: cryptomarket/prod/redis
        property: connection_string
```

### ClusterSecretStore

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ClusterSecretStore
metadata:
  name: aws-secrets-manager
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-1
      auth:
        jwt:
          serviceAccountRef:
            name: external-secrets
            namespace: external-secrets
```

## Rotation Policy

| Secret Type        | Method                    | Frequency | Automated |
|--------------------|---------------------------|-----------|-----------|
| RDS credentials    | AWS managed password      | 90 days   | Yes       |
| Redis auth token   | Manual + app restart      | 90 days   | No        |
| API keys           | Provider dashboard        | On compromise | No    |
| JWT signing key    | Manual rotation + deploy  | 180 days  | No        |

### Rotation Procedure (Manual)

1. Generate new secret value
2. Update in AWS Secrets Manager
3. ESO syncs within `refreshInterval` (1h)
4. Restart affected deployments if needed
5. Verify service health
6. Revoke old secret after grace period

## Gitleaks Enforcement

Repository scanning via `.gitleaks.toml`:
- Pre-commit hook (local development)
- CI pipeline (all PRs and pushes)
- Full history scan (weekly schedule)

```bash
# Local scan
gitleaks detect --source . --config .gitleaks.toml

# CI scan (in security.yml workflow)
gitleaks detect --source . --verbose --redact
```

## Terraform Integration

Secrets Manager resources defined in `deploy/terraform/modules/secrets/`:
- Secret creation with KMS encryption
- IAM policies for ESO access (IRSA)
- CloudWatch alarm for unauthorized access attempts

## Incident Response (Secret Exposure)

If a secret is accidentally committed:

1. **Revoke immediately** — Rotate the exposed credential
2. **Remove from history** — `git filter-branch` or BFG Repo Cleaner
3. **Scan** — Run Gitleaks to confirm removal
4. **Audit** — Check CloudTrail for unauthorized use
5. **Post-mortem** — Document and prevent recurrence
