# IAM Least Privilege

## Overview

All AWS IAM policies follow the principle of least privilege. Each service and component receives only the permissions required for its specific function.

## Architecture

```
EKS Pod → ServiceAccount → IAM Role (IRSA) → AWS API
```

IAM Roles for Service Accounts (IRSA) provides pod-level AWS access without node-level credentials.

## Role Inventory

| Role Name                          | Service    | Permissions                              |
|------------------------------------|------------|------------------------------------------|
| `cryptomarket-api-irsa`            | API        | Secrets Manager read (postgres, redis)   |
| `cryptomarket-ingestor-irsa`       | Ingestor   | Secrets Manager read (api-keys)          |
| `cryptomarket-external-secrets`    | ESO        | Secrets Manager read (all cryptomarket/) |
| `cryptomarket-backup`              | CronJob    | S3 write (backup bucket)                 |
| `cryptomarket-deploy`              | CI/CD      | ECR push, EKS describe, S3 read (state)  |

## Policy Details

### API Service Role

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue",
        "secretsmanager:DescribeSecret"
      ],
      "Resource": [
        "arn:aws:secretsmanager:us-east-1:*:secret:cryptomarket/prod/postgres-*",
        "arn:aws:secretsmanager:us-east-1:*:secret:cryptomarket/prod/redis-*"
      ]
    }
  ]
}
```

### Ingestor Service Role

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue"
      ],
      "Resource": [
        "arn:aws:secretsmanager:us-east-1:*:secret:cryptomarket/prod/api-keys-*"
      ]
    }
  ]
}
```

### External Secrets Operator Role

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue",
        "secretsmanager:DescribeSecret"
      ],
      "Resource": "arn:aws:secretsmanager:us-east-1:*:secret:cryptomarket/*"
    }
  ]
}
```

### Backup Role

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:GetObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::cryptomarket-backups",
        "arn:aws:s3:::cryptomarket-backups/*"
      ]
    }
  ]
}
```

## IRSA Configuration

### Terraform (modules/iam)

```hcl
module "iam" {
  source = "../modules/iam"
  
  cluster_name          = module.eks.cluster_name
  cluster_oidc_issuer   = module.eks.cluster_oidc_issuer_url
  namespace             = "cryptomarket-prod"
  secrets_manager_arns  = module.secrets.secret_arns
  backup_bucket_arn     = module.s3.backup_bucket_arn
}
```

### ServiceAccount Annotation

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: market-api
  namespace: cryptomarket-prod
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789:role/cryptomarket-api-irsa
```

## Principles Applied

| Principle                    | Implementation                                    |
|------------------------------|---------------------------------------------------|
| Least privilege              | Per-service roles with specific resource ARNs     |
| No shared credentials        | IRSA (no node-level IAM keys)                     |
| No wildcard actions          | Explicit action lists only                        |
| Resource-scoped              | ARNs include path prefix restrictions             |
| No admin access              | No `*:*` permissions anywhere                     |
| Temporary credentials        | IRSA tokens auto-rotate (1h STS sessions)         |
| Audit trail                  | CloudTrail logs all API calls                     |

## What We Explicitly Do NOT Grant

- `s3:DeleteObject` on backup bucket (prevents accidental deletion)
- `ec2:*` to any workload role
- `iam:*` to any workload role
- `eks:*` to any workload role
- Cross-account access
- Console access for service accounts

## Review Schedule

- IAM policies reviewed quarterly
- Access patterns audited via IAM Access Analyzer
- Unused permissions removed within 30 days of identification
- New permissions require security review
