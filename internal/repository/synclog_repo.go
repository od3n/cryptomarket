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

// GetProviderStatus returns the latest sync status for each provider.
func (r *PostgresSyncLogRepository) GetProviderStatus(ctx context.Context) ([]market.ProviderStatus, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT ON (provider)
			provider,
			finished_at,
			request_duration_ms,
			status
		 FROM provider_sync_logs
		 ORDER BY provider, started_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("query provider status: %w", err)
	}
	defer rows.Close()

	var statuses []market.ProviderStatus
	for rows.Next() {
		var s market.ProviderStatus
		if err := rows.Scan(&s.Provider, &s.LastSyncAt, &s.LastDurationMs, &s.LastStatus); err != nil {
			return nil, fmt.Errorf("scan provider status: %w", err)
		}
		statuses = append(statuses, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider status: %w", err)
	}

	// Get recent failure counts (last 24h).
	for i := range statuses {
		var count int
		err := r.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM provider_sync_logs
			 WHERE provider = $1 AND status = 'failure' AND started_at > NOW() - INTERVAL '24 hours'`,
			statuses[i].Provider).Scan(&count)
		if err == nil {
			statuses[i].RecentFailures = count
		}
	}

	return statuses, nil
}
