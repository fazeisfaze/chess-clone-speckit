package postgres

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestApplyMigrationsCreatesSchemaAndIsIdempotent(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for PostgreSQL migration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	admin, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	schema := "migration_test_" + time.Now().Format("150405000000")
	if _, err := admin.Exec(ctx, `CREATE SCHEMA `+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatalf("create isolated schema: %v", err)
	}
	defer func() {
		_, _ = admin.Exec(context.Background(), `DROP SCHEMA `+pgx.Identifier{schema}.Sanitize()+` CASCADE`)
		_ = admin.Close(context.Background())
	}()

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse database URL: %v", err)
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("create isolated pool: %v", err)
	}
	defer pool.Close()

	_, currentFile, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "migrations")
	if err := ApplyMigrations(ctx, pool, migrationsDir); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := ApplyMigrations(ctx, pool, migrationsDir); err != nil {
		t.Fatalf("apply migrations a second time: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("query migration ledger: %v", err)
	}
	if count != 2 {
		t.Fatalf("applied migration count = %d, want 2", count)
	}
	for _, table := range []string{"matches", "player_seats", "moves", "draw_offers"} {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, table).Scan(&exists); err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("migration did not create table %q", table)
		}
	}
	var hasCheckColumn bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'moves' AND column_name = 'gives_check')`).Scan(&hasCheckColumn); err != nil {
		t.Fatal(err)
	}
	if !hasCheckColumn {
		t.Fatal("migration did not add moves.gives_check")
	}
}
