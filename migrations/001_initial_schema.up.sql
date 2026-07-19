-- 001_initial_schema.up.sql
-- Creates the core tables for the crypto market data platform.

CREATE TABLE IF NOT EXISTS coins (
    id              BIGSERIAL PRIMARY KEY,
    symbol          VARCHAR(20) NOT NULL UNIQUE,
    name            VARCHAR(100) NOT NULL,
    provider_symbol VARCHAR(100) NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_coins_is_active ON coins (is_active) WHERE is_active = TRUE;

CREATE TABLE IF NOT EXISTS price_snapshots (
    id          BIGSERIAL PRIMARY KEY,
    coin_id     BIGINT NOT NULL REFERENCES coins(id) ON DELETE CASCADE,
    price_usd   NUMERIC(20, 8) NOT NULL,
    market_cap  NUMERIC(30, 2),
    volume_24h  NUMERIC(30, 2),
    change_24h  NUMERIC(10, 4),
    provider    VARCHAR(50) NOT NULL,
    captured_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_price_snapshots_coin_id ON price_snapshots (coin_id);
CREATE INDEX idx_price_snapshots_captured_at ON price_snapshots (captured_at DESC);
CREATE INDEX idx_price_snapshots_coin_captured ON price_snapshots (coin_id, captured_at DESC);

CREATE TABLE IF NOT EXISTS provider_sync_logs (
    id                  BIGSERIAL PRIMARY KEY,
    provider            VARCHAR(50) NOT NULL,
    request_duration_ms BIGINT NOT NULL DEFAULT 0,
    status              VARCHAR(20) NOT NULL,
    error_message       TEXT,
    started_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at         TIMESTAMPTZ
);

CREATE INDEX idx_provider_sync_logs_started_at ON provider_sync_logs (started_at DESC);
CREATE INDEX idx_provider_sync_logs_status ON provider_sync_logs (status);
