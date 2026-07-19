-- 001_initial_schema.down.sql
-- Reverses the initial schema creation.

DROP TABLE IF EXISTS provider_sync_logs;
DROP TABLE IF EXISTS price_snapshots;
DROP TABLE IF EXISTS coins;
