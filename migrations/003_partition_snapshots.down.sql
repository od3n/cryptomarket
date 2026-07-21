-- Rollback: Restore original non-partitioned table
-- WARNING: This drops the partitioned table and all its data.
-- Ensure you have a backup before running this.

DROP FUNCTION IF EXISTS create_monthly_partition();

-- Swap back
ALTER TABLE price_snapshots RENAME TO price_snapshots_partitioned;
ALTER TABLE price_snapshots_old RENAME TO price_snapshots;

-- Drop partitioned table and all partitions
DROP TABLE IF EXISTS price_snapshots_partitioned CASCADE;
