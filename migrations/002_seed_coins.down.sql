-- 002_seed_coins.down.sql
-- Removes seeded coins.

DELETE FROM coins WHERE symbol IN ('BTC', 'ETH', 'SOL', 'BNB', 'XRP', 'DOGE');
