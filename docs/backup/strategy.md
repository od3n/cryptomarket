# Backup Strategy

## Overview

The Crypto Market Platform implements automated backups for PostgreSQL and Redis with S3 storage, lifecycle management, and regular restore verification.

## Recovery Objectives

| Component   | RPO          | RTO          | Method                    |
|-------------|--------------|--------------|---------------------------|
| PostgreSQL  | 5 minutes    | 30 minutes   | RDS automated + pg_dump   |
| Redis       | 1 hour       | 15 minutes   | BGSAVE snapshots          |
| Config      | 24 hours     | 5 minutes    | Git (Helm/Terraform)      |

## PostgreSQL Backups

### RDS Automated Backups (Production)

- **Automated snapshots**: Daily, retained 7 days
- **Point-in-time recovery (PITR)**: 5-minute granularity, 7-day window
- **Multi-AZ**: Synchronous replication to standby
- **Encryption**: AES-256 via KMS

### Application-Level Backups

```bash
# Full compressed dump
python3 scripts/backup/backup.py --type postgres --upload-s3

# Output: postgres_YYYYMMDD_HHMMSS.sql.gz
```

Schedule:
- **Full backup**: Daily at 02:00 UTC
- **Retention**: 7 daily, 4 weekly, 3 monthly

### Backup Verification

Every backup is verified via `scripts/restore/verify_restore.py`:
- Schema integrity (tables, indexes, constraints)
- Row count validation
- Sample data checks
- Referential integrity

## Redis Backups

### Method

- `BGSAVE` triggers async RDB snapshot
- RDB file copied and uploaded to S3
- Schedule: Every 6 hours

### Limitations

- Redis is used as a cache/stream buffer — data is reconstructable
- RDB snapshots are point-in-time (up to 6h data loss acceptable)
- Redis Streams consumer state is ephemeral

## S3 Storage

### Bucket Structure

```
s3://cryptomarket-backups/
├── postgres/
│   ├── daily/
│   ├── weekly/
│   └── monthly/
├── redis/
│   └── snapshots/
└── manifests/
    └── manifest_YYYYMMDD_HHMMSS.json
```

### Lifecycle Policy

| Prefix       | Transition to IA | Transition to Glacier | Expiration |
|--------------|------------------|-----------------------|------------|
| `postgres/daily/`   | 7 days     | 30 days               | 90 days    |
| `postgres/weekly/`  | 30 days    | 90 days               | 365 days   |
| `postgres/monthly/` | 90 days    | 180 days              | 730 days   |
| `redis/`            | 7 days     | 30 days               | 60 days    |

### Encryption

- Server-side encryption: AES-256 (SSE-S3)
- KMS key for cross-account restore access
- Bucket policy: deny unencrypted uploads

## Backup Monitoring

Alerts fire when:
- Backup job fails (PagerDuty)
- Backup size deviates >50% from baseline
- No successful backup in 26 hours (daily SLA)
- Restore verification fails

## Restore Procedures

See [restore-verification.md](./restore-verification.md) for detailed restore steps.

### Quick Restore (PostgreSQL)

```bash
# From RDS PITR
aws rds restore-db-instance-to-point-in-time \
  --source-db-instance-identifier cryptomarket-prod \
  --target-db-instance-identifier cryptomarket-prod-restore \
  --restore-time "2024-01-15T10:30:00Z"

# From pg_dump
pg_restore -d cryptomarket postgres_20240115_020000.sql.gz
```

### Quick Restore (Redis)

```bash
# Copy RDB to Redis data dir and restart
aws s3 cp s3://cryptomarket-backups/redis/snapshots/redis_latest.rdb /var/lib/redis/dump.rdb
systemctl restart redis
```
