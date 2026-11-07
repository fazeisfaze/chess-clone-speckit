package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ApplyMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name text PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT clock_timestamp()
	)`); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		if err := applyOne(ctx, pool, migrationsDir, name); err != nil {
			return err
		}
	}
	return nil
}

func applyOne(ctx context.Context, pool *pgxpool.Pool, dir, name string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", name, err)
	}
	defer tx.Rollback(ctx)

	var applied bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name = $1)`, name).Scan(&applied); err != nil {
		return fmt.Errorf("check migration %s: %w", name, err)
	}
	if applied {
		return tx.Commit(ctx)
	}
	contents, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return fmt.Errorf("read migration %s: %w", name, err)
	}
	if _, err := tx.Exec(ctx, string(contents)); err != nil {
		return fmt.Errorf("execute migration %s: %w", name, err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations(name) VALUES ($1)`, name); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}
	return nil
}
