# AWS Cost Estimates

## Overview

Monthly cost estimates for running the Crypto Market Platform on AWS across three environments. Estimates based on us-east-1 pricing (January 2024).

## Environment Summary

| Environment | Monthly Estimate | Purpose                    |
|-------------|-----------------|----------------------------|
| Dev         | ~$150-200       | Development & testing      |
| Staging     | ~$400-550       | Pre-production validation  |
| Production  | ~$1,200-1,800   | Live traffic               |

---

## Development Environment

| Resource                | Specification              | Monthly Cost |
|-------------------------|----------------------------|--------------|
| EKS Cluster             | 1 cluster                  | $73          |
| EC2 Nodes               | 2× t3.medium (2vCPU/4GB)  | $60          |
| RDS PostgreSQL          | db.t3.micro, single-AZ     | $15          |
| ElastiCache Redis       | cache.t3.micro             | $12          |
| S3 (backups)            | ~5 GB                      | $1           |
| ECR                     | ~2 GB images               | $1           |
| CloudWatch              | Basic monitoring           | $5           |
| Route53                 | 1 hosted zone              | $1           |
| Data transfer           | ~50 GB                     | $5           |
| **Total**               |                            | **~$173**    |

---

## Staging Environment

| Resource                | Specification              | Monthly Cost |
|-------------------------|----------------------------|--------------|
| EKS Cluster             | 1 cluster                  | $73          |
| EC2 Nodes               | 3× t3.large (2vCPU/8GB)   | $180         |
| RDS PostgreSQL          | db.t3.small, single-AZ     | $35          |
| ElastiCache Redis       | cache.t3.small             | $25          |
| S3 (backups)            | ~20 GB                     | $2           |
| ECR                     | ~5 GB images               | $1           |
| ACM Certificate         | Free (with Route53)        | $0           |
| CloudWatch              | Enhanced monitoring        | $15          |
| NAT Gateway             | 1× (shared)               | $32          |
| Data transfer           | ~100 GB                    | $9           |
| **Total**               |                            | **~$372**    |

---

## Production Environment

| Resource                | Specification              | Monthly Cost |
|-------------------------|----------------------------|--------------|
| EKS Cluster             | 1 cluster                  | $73          |
| EC2 Nodes               | 4× m5.xlarge (4vCPU/16GB) | $690         |
| RDS PostgreSQL          | db.r6g.large, Multi-AZ     | $350         |
| ElastiCache Redis       | cache.r6g.large, Multi-AZ  | $280         |
| S3 (backups)            | ~100 GB + lifecycle        | $5           |
| ECR                     | ~10 GB images              | $1           |
| ACM Certificate         | Free (with Route53)        | $0           |
| CloudWatch              | Logs + metrics + alarms    | $50          |
| NAT Gateway             | 2× (HA)                   | $64          |
| ALB                     | 1× application LB          | $25          |
| KMS                     | Key management             | $3           |
| Secrets Manager         | 4 secrets                  | $2           |
| Data transfer           | ~500 GB                    | $45          |
| **Total**               |                            | **~$1,588**  |

---

## Cost Optimization Strategies

### Implemented

| Strategy                        | Savings         | Status    |
|---------------------------------|-----------------|-----------|
| Right-sized instances           | 30-40%          | Active    |
| S3 lifecycle policies           | 50% on old data | Active    |
| Spot instances (dev/staging)    | 60-70%          | Planned   |
| Reserved instances (prod)       | 30-40%          | Planned   |
| Auto-scaling (off-peak)         | 20-30%          | Active    |

### Recommended

| Strategy                              | Potential Savings | Effort |
|---------------------------------------|-------------------|--------|
| Graviton (ARM) instances              | 20% compute       | Low    |
| S3 Intelligent-Tiering                | 10-20% storage    | Low    |
| Savings Plans (1yr commitment)        | 30-40% compute    | Low    |
| Dev environment auto-shutdown (nights)| 50% dev cost      | Medium |
| EKS on Fargate (burst workloads)      | Variable          | Medium |

### Dev Environment Scheduling

```hcl
# Auto-shutdown dev cluster nights/weekends
resource "aws_scheduler_schedule" "dev_shutdown" {
  name = "cryptomarket-dev-shutdown"
  schedule_expression = "cron(0 20 ? * MON-FRI *)"
  # Scale node group to 0
}

resource "aws_scheduler_schedule" "dev_startup" {
  name = "cryptomarket-dev-startup"
  schedule_expression = "cron(0 7 ? * MON-FRI *)"
  # Scale node group to 2
}
```

## Monitoring Costs

- AWS Cost Explorer: Track daily/monthly spend
- Budget alerts: Warn at 80% of monthly budget
- Per-environment tagging: `Environment: dev|staging|prod`
- Anomaly detection: Alert on unusual spend patterns

## Cost Allocation Tags

| Tag Key       | Values                    | Purpose              |
|---------------|---------------------------|----------------------|
| Environment   | dev, staging, prod        | Environment tracking |
| Service       | api, ingestor, realtime   | Service-level cost   |
| ManagedBy     | terraform                 | IaC identification   |
| Project       | cryptomarket              | Project grouping     |
