package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/crypto-market-platform/internal/market"
)

// PostgresSnapshotRepository implements SnapshotRepository using PostgreSQL.
type PostgresSnapshotRepository struct {
	db *sql.DB
}

// NewPostgresSnapshotRepository creates a new PostgresSnapshotRepository.
func NewPostgresSnapshotRepository(db *sql.DB) *PostgresSnapshotRepository {
	return &PostgresSnapshotRepository{db: db}
}

func (r *PostgresSnapshotRepository) InsertBatch(ctx context.Context, snapshots []market.PriceSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO price_snapshots (coin_id, price_usd, market_cap, volume_24h, change_24h, provider, captured_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, s := range snapshots {
		_, err := stmt.ExecContext(ctx, s.CoinID, s.PriceUSD, s.MarketCap, s.Volume24h, s.Change24h, s.Provider, s.CapturedAt)
		if err != nil {
			return fmt.Errorf("insert snapshot for coin %d: %w", s.CoinID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit snapshot batch: %w", err)
	}
	return nil
}

func (r *PostgresSnapshotRepository) GetLatestByCoin(ctx context.Context, coinID int64) (*market.PriceSnapshot, error) {
	var s market.PriceSnapshot
	err := r.db.QueryRowContext(ctx,
		`SELECT id, coin_id, price_usd, market_cap, volume_24h, change_24h, provider, captured_at
		 FROM price_snapshots WHERE coin_id = $1 ORDER BY captured_at DESC LIMIT 1`, coinID).
		Scan(&s.ID, &s.CoinID, &s.PriceUSD, &s.MarketCap, &s.Volume24h, &s.Change24h, &s.Provider, &s.CapturedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query latest snapshot for coin %d: %w", coinID, err)
	}
	return &s, nil
}

func (r *PostgresSnapshotRepository) GetHistory(ctx context.Context, coinID int64, limit int, before *time.Time, from *time.Time, to *time.Time) ([]market.PriceSnapshot, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("coin_id = $%d", argIdx))
	args = append(args, coinID)
	argIdx++

	if before != nil {
		conditions = append(conditions, fmt.Sprintf("captured_at < $%d", argIdx))
		args = append(args, *before)
		argIdx++
	}
	if from != nil {
		conditions = append(conditions, fmt.Sprintf("captured_at >= $%d", argIdx))
		args = append(args, *from)
		argIdx++
	}
	if to != nil {
		conditions = append(conditions, fmt.Sprintf("captured_at <= $%d", argIdx))
		args = append(args, *to)
		argIdx++
	}

	query := fmt.Sprintf(
		`SELECT id, coin_id, price_usd, market_cap, volume_24h, change_24h, provider, captured_at
		 FROM price_snapshots WHERE %s ORDER BY captured_at DESC LIMIT $%d`,
		strings.Join(conditions, " AND "), argIdx)
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query history for coin %d: %w", coinID, err)
	}
	defer rows.Close()

	var snapshots []market.PriceSnapshot
	for rows.Next() {
		var s market.PriceSnapshot
		if err := rows.Scan(&s.ID, &s.CoinID, &s.PriceUSD, &s.MarketCap, &s.Volume24h, &s.Change24h, &s.Provider, &s.CapturedAt); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		snapshots = append(snapshots, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshots: %w", err)
	}
	return snapshots, nil
}
