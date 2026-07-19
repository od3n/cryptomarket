# Restore Verification

## Overview

Automated restore verification ensures backups are recoverable before they're needed in an emergency. The verification tool (`scripts/restore/verify_restore.py`) validates schema, data integrity, and consistency.

## Verification Checks

| Check              | What It Validates                              | Failure Impact |
|--------------------|------------------------------------------------|----------------|
| Connectivity       | Can connect to restored database               | Critical       |
| Schema Tables      | All expected tables exist                      | Critical       |
| Indexes            | Minimum index count present                    | Warning        |
| Row Counts         | Tables have data (non-zero counts)             | Warning        |
| Sample Data        | Asset records have valid symbol/name fields    | Critical       |
| Integrity          | Foreign key constraints are valid              | Critical       |
| Manifest           | Backup manifest JSON is parseable              | Warning        |

## Running Verification

### Against a Restored Instance

```bash
# Verify a freshly restored database
python3 scripts/restore/verify_restore.py \
  --dsn "postgres://cryptouser:cryptopass@restored-host:5432/cryptomarket" \
  --report /tmp/restore-report.json

# With manifest validation
python3 scripts/restore/verify_restore.py \
  --dsn "$POSTGRES_DSN" \
  --manifest /tmp/cryptomarket-backups/manifest_20240115_020000.json
```

### Output

```
============================================================
  Crypto Market Platform - Restore Verification
============================================================

Results:
  ✓ [PASS] connectivity: Connected successfully
  ✓ [PASS] schema_tables: All 6 expected tables present
  ✓ [PASS] indexes: 12 indexes present
  ✓ [PASS] row_counts: Row counts: {'assets': 150, 'markets': 45, ...}
  ✓ [PASS] sample_data: Sample data valid (5 assets checked)
  ✓ [PASS] integrity: 4 foreign key constraints present
  ✓ [PASS] manifest: Manifest valid: 3 files, created 2024-01-15T02:00:00Z

Summary: 7 passed, 0 failed, 0 warnings
Overall: PASS
```

## Scheduled Verification

### Weekly Automated Restore Test

1. Provision temporary RDS instance from latest snapshot
2. Run `verify_restore.py` against it
3. Report results to Slack/PagerDuty
4. Tear down temporary instance

```bash
# CI/CD scheduled job (weekly)
aws rds restore-db-instance-to-point-in-time \
  --source-db-instance-identifier cryptomarket-prod \
  --target-db-instance-identifier cryptomarket-verify-$(date +%Y%m%d) \
  --use-latest-restorable-time

python3 scripts/restore/verify_restore.py \
  --dsn "postgres://$VERIFY_USER:$VERIFY_PASS@$VERIFY_HOST:5432/cryptomarket" \
  --report verify-report.json

aws rds delete-db-instance \
  --db-instance-identifier cryptomarket-verify-$(date +%Y%m%d) \
  --skip-final-snapshot
```

## Expected Tables

| Table              | Purpose                          | Min Rows |
|--------------------|----------------------------------|----------|
| assets             | Cryptocurrency asset registry    | 1        |
| markets            | Trading pair definitions         | 1        |
| ticker_data        | Price/volume ticker snapshots    | 0        |
| order_books        | Order book depth snapshots       | 0        |
| ingestion_state    | Provider ingestion tracking      | 0        |
| schema_migrations  | Migration version tracking       | 1        |

## Failure Response

| Result  | Action                                              |
|---------|-----------------------------------------------------|
| PASS    | Log success, no action needed                       |
| WARN    | Investigate within 24h, may indicate partial backup |
| FAIL    | Page on-call, initiate fresh backup immediately     |

## Integration with Backup Pipeline

The backup script (`scripts/backup/backup.py`) writes a manifest file with each backup. The restore verifier can validate this manifest to ensure:
- Backup completed without truncation
- File sizes are non-zero
- Timestamp is recent
