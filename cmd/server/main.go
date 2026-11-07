package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
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
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load configuration", "error", err)
		os.Exit(1)
	}
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		slog.Error("ping database", "error", err)
		os.Exit(1)
	}
	if err := postgres.ApplyMigrations(context.Background(), pool, "migrations"); err != nil {
		slog.Error("apply database migrations", "error", err)
		os.Exit(1)
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
	if err := httpapi.Serve(ctx, server); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("http server stopped", "error", err)
		os.Exit(1)
	}
}
