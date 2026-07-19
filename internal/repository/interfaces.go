package repository

import (
	"context"
	"time"

	"github.com/crypto-market-platform/internal/market"
)

// CoinRepository defines operations on the coins table.
type CoinRepository interface {
	GetActiveCoins(ctx context.Context) ([]market.Coin, error)
	GetBySymbol(ctx context.Context, symbol string) (*market.Coin, error)
}

// SnapshotRepository defines operations on the price_snapshots table.
type SnapshotRepository interface {
	InsertBatch(ctx context.Context, snapshots []market.PriceSnapshot) error
	GetLatestByCoin(ctx context.Context, coinID int64) (*market.PriceSnapshot, error)
	GetHistory(ctx context.Context, coinID int64, limit int, before *time.Time, from *time.Time, to *time.Time) ([]market.PriceSnapshot, error)
}

// SyncLogRepository defines operations on the provider_sync_logs table.
type SyncLogRepository interface {
	Insert(ctx context.Context, log *market.ProviderSyncLog) error
	GetProviderStatus(ctx context.Context) ([]market.ProviderStatus, error)
}
