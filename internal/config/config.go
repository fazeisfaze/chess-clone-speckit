package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

type Config struct {
	DatabaseURL   string
	HTTPAddr      string
	WebDir        string
	AllowedOrigin string
}

func Load() (Config, error) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	parsed, err := url.Parse(databaseURL)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Host == "" {
		return Config{}, fmt.Errorf("DATABASE_URL must be a valid postgres:// URL")
	}
	if _, err := pgx.ParseConfig(databaseURL); err != nil {
		return Config{}, fmt.Errorf("DATABASE_URL is invalid: %w", err)
	}

	return Config{
		DatabaseURL:   databaseURL,
		HTTPAddr:      envOr("HTTP_ADDR", ":8080"),
		WebDir:        envOr("WEB_DIR", "web/dist"),
		AllowedOrigin: envOr("ALLOWED_ORIGIN", "http://localhost:5173"),
	}, nil
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
