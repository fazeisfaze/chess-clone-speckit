package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chess_clone/internal/clock"
	"chess_clone/internal/config"
	"chess_clone/internal/store/postgres"
	httpapi "chess_clone/internal/transport/http"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("server exited", "error", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	migrateOnly, err := parseArgs(args)
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	if err := postgres.ApplyMigrations(context.Background(), pool, "migrations"); err != nil {
		return fmt.Errorf("apply database migrations: %w", err)
	}
	if migrateOnly {
		return nil
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	app := httpapi.NewApplication(cfg, pool)
	go func() {
		if err := clock.RunTimeoutWorker(ctx, time.Second, app.MatchStore, app.Hub.Publish); err != nil {
			slog.Error("timeout worker stopped", "error", err)
		}
	}()
	server := httpapi.NewServer(cfg, app.Handler)
	return httpapi.Serve(ctx, server)
}
