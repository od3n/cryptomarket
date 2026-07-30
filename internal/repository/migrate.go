package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Migrate runs all pending up migrations from the given directory.
func Migrate(ctx context.Context, db *sql.DB, migrationsDir string) error {
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return err
	}

	applied, err := getAppliedMigrations(ctx, db)
	if err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
	sort.Strings(files)

	for _, f := range files {
		name := filepath.Base(f)
		if applied[name] {
			continue
		}

		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", name, err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", name, err)
		}

		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations (name) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}

		fmt.Printf("applied migration: %s\n", name)
	}

	return nil
}

// MigrateDown reverses the last applied migration.
func MigrateDown(ctx context.Context, db *sql.DB, migrationsDir string) error {
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return err
	}

	var lastMigration string
	err := db.QueryRowContext(ctx,
		`SELECT name FROM schema_migrations ORDER BY applied_at DESC LIMIT 1`).Scan(&lastMigration)
	if err == sql.ErrNoRows {
		fmt.Println("no migrations to reverse")
		return nil
	}
	if err != nil {
		return fmt.Errorf("query last migration: %w", err)
	}

	downFile := strings.Replace(lastMigration, ".up.sql", ".down.sql", 1)
	downPath := filepath.Join(migrationsDir, downFile)

	content, err := os.ReadFile(downPath)
	if err != nil {
		return fmt.Errorf("read down migration %s: %w", downFile, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("execute down migration %s: %w", downFile, err)
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM schema_migrations WHERE name = $1`, lastMigration); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("remove migration record %s: %w", lastMigration, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit down migration: %w", err)
	}

	fmt.Printf("reversed migration: %s\n", lastMigration)
	return nil
}

func ensureMigrationsTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id         BIGSERIAL PRIMARY KEY,
			name       VARCHAR(255) NOT NULL UNIQUE,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}
	return nil
}

func getAppliedMigrations(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT name FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan migration name: %w", err)
		}
		applied[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applied migrations: %w", err)
	}
	return applied, nil
}
