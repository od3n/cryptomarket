-- 003: Partition price_snapshots table by month for query performance
-- Large history tables benefit from range partitioning on captured_at.
-- This reduces index scan time for recent queries and enables efficient
-- data archival (DROP PARTITION instead of DELETE).
--
-- Strategy: Create partitioned table, migrate data, swap.
-- Note: This is a forward-only migration. Rollback requires restoring from backup.

-- Step 1: Create the partitioned table structure
CREATE TABLE IF NOT EXISTS price_snapshots_partitioned (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    coin_id UUID NOT NULL REFERENCES coins(id),
    price NUMERIC(20, 8) NOT NULL,
    market_cap NUMERIC(30, 2),
    volume_24h NUMERIC(30, 2),
    change_24h NUMERIC(10, 4),
    provider VARCHAR(50) NOT NULL DEFAULT 'coingecko',
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, captured_at)
) PARTITION BY RANGE (captured_at);

-- Step 2: Create partitions for recent months and future
-- Adjust date ranges as needed. Create partitions 3 months ahead.
CREATE TABLE IF NOT EXISTS price_snapshots_y2024m01 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2024m02 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2024m03 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2024m04 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2024m05 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2024m06 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2024-06-01') TO ('2024-07-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2024m07 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2024-07-01') TO ('2024-08-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2024m08 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2024-08-01') TO ('2024-09-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2024m09 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2024-09-01') TO ('2024-10-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2024m10 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2024-10-01') TO ('2024-11-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2024m11 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2024-11-01') TO ('2024-12-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2024m12 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

-- 2025-2026 partitions
CREATE TABLE IF NOT EXISTS price_snapshots_y2025m01 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2025m02 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2025m03 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2025m04 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2025-04-01') TO ('2025-05-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2025m05 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2025-05-01') TO ('2025-06-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2025m06 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2025-06-01') TO ('2025-07-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2025m07 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2025-07-01') TO ('2025-08-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2025m08 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2025-08-01') TO ('2025-09-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2025m09 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2025-09-01') TO ('2025-10-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2025m10 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2025m11 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2025m12 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2026m01 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2026m02 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2026m03 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2026m04 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2026m05 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2026m06 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2026m07 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2026m08 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');
CREATE TABLE IF NOT EXISTS price_snapshots_y2026m09 PARTITION OF price_snapshots_partitioned
    FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');

-- Default partition for any data outside defined ranges
CREATE TABLE IF NOT EXISTS price_snapshots_default PARTITION OF price_snapshots_partitioned DEFAULT;

-- Step 3: Create indexes on partitioned table
CREATE INDEX IF NOT EXISTS idx_snapshots_part_coin_captured
    ON price_snapshots_partitioned (coin_id, captured_at DESC);
CREATE INDEX IF NOT EXISTS idx_snapshots_part_captured
    ON price_snapshots_partitioned (captured_at DESC);

-- Step 4: Migrate existing data (if table has data)
INSERT INTO price_snapshots_partitioned (id, coin_id, price, market_cap, volume_24h, change_24h, provider, captured_at)
SELECT id, coin_id, price, market_cap, volume_24h, change_24h, provider, captured_at
FROM price_snapshots
WHERE EXISTS (SELECT 1 FROM price_snapshots LIMIT 1)
ON CONFLICT DO NOTHING;

-- Step 5: Swap tables
ALTER TABLE price_snapshots RENAME TO price_snapshots_old;
ALTER TABLE price_snapshots_partitioned RENAME TO price_snapshots;

-- Step 6: Create function for automatic partition management
CREATE OR REPLACE FUNCTION create_monthly_partition()
RETURNS void AS $$
DECLARE
    next_month DATE;
    partition_name TEXT;
    start_date TEXT;
    end_date TEXT;
BEGIN
    -- Create partition for 2 months ahead
    next_month := date_trunc('month', NOW() + interval '2 months');
    partition_name := 'price_snapshots_y' || to_char(next_month, 'YYYY') || 'm' || to_char(next_month, 'MM');
    start_date := to_char(next_month, 'YYYY-MM-DD');
    end_date := to_char(next_month + interval '1 month', 'YYYY-MM-DD');

    -- Check if partition already exists
    IF NOT EXISTS (
        SELECT 1 FROM pg_class WHERE relname = partition_name
    ) THEN
        EXECUTE format(
            'CREATE TABLE %I PARTITION OF price_snapshots FOR VALUES FROM (%L) TO (%L)',
            partition_name, start_date, end_date
        );
        RAISE NOTICE 'Created partition: %', partition_name;
    END IF;
END;
$$ LANGUAGE plpgsql;
