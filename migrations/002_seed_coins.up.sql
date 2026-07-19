-- 002_seed_coins.up.sql
-- Seeds the initial set of supported coins.

INSERT INTO coins (symbol, name, provider_symbol, is_active)
VALUES
    ('BTC', 'Bitcoin', 'bitcoin', TRUE),
    ('ETH', 'Ethereum', 'ethereum', TRUE),
    ('SOL', 'Solana', 'solana', TRUE),
    ('BNB', 'BNB', 'binancecoin', TRUE),
    ('XRP', 'XRP', 'ripple', TRUE),
    ('DOGE', 'Dogecoin', 'dogecoin', TRUE)
ON CONFLICT (symbol) DO NOTHING;
