package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/crypto-market-platform/internal/market"
)

// PostgresSyncLogRepository implements SyncLogRepository using PostgreSQL.
type PostgresSyncLogRepository struct {
	db *sql.DB
}

// NewPostgresSyncLogRepository creates a new PostgresSyncLogRepository.
func NewPostgresSyncLogRepository(db *sql.DB) *PostgresSyncLogRepository {
	return &PostgresSyncLogRepository{db: db}
}

func (r *PostgresSyncLogRepository) Insert(ctx context.Context, log *market.ProviderSyncLog) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO provider_sync_logs (provider, request_duration_ms, status, error_message, started_at, finished_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		log.Provider, log.RequestDurationMs, log.Status, log.ErrorMessage, log.StartedAt, log.FinishedAt)
	if err != nil {
		return fmt.Errorf("insert sync log: %w", err)
	}
	return nil
}
