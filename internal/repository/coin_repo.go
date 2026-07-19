package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/crypto-market-platform/internal/market"
)

// PostgresCoinRepository implements CoinRepository using PostgreSQL.
type PostgresCoinRepository struct {
	db *sql.DB
}

// NewPostgresCoinRepository creates a new PostgresCoinRepository.
func NewPostgresCoinRepository(db *sql.DB) *PostgresCoinRepository {
	return &PostgresCoinRepository{db: db}
}

func (r *PostgresCoinRepository) GetActiveCoins(ctx context.Context) ([]market.Coin, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, symbol, name, provider_symbol, is_active, created_at, updated_at
		 FROM coins WHERE is_active = TRUE ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("query active coins: %w", err)
	}
	defer rows.Close()

	var coins []market.Coin
	for rows.Next() {
		var c market.Coin
		if err := rows.Scan(&c.ID, &c.Symbol, &c.Name, &c.ProviderSymbol, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan coin: %w", err)
		}
		coins = append(coins, c)
	}
	return coins, rows.Err()
}

func (r *PostgresCoinRepository) GetBySymbol(ctx context.Context, symbol string) (*market.Coin, error) {
	var c market.Coin
	err := r.db.QueryRowContext(ctx,
		`SELECT id, symbol, name, provider_symbol, is_active, created_at, updated_at
		 FROM coins WHERE symbol = $1`, symbol).
		Scan(&c.ID, &c.Symbol, &c.Name, &c.ProviderSymbol, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query coin by symbol %s: %w", symbol, err)
	}
	return &c, nil
}
