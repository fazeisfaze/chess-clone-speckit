package config

import (
	"strings"
	"testing"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("HTTP_ADDR", ":9090")
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("Load() error = %v, want DATABASE_URL required", err)
	}
}

func TestLoadRejectsInvalidDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "not-a-postgres-url")
	_, err := Load()
	if err == nil {
		t.Fatal("Load() accepted an invalid database URL")
	}
}

func TestLoadUsesLocalDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://chess:secret@localhost:5432/chess?sslmode=disable")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("WEB_DIR", "")
	t.Setenv("ALLOWED_ORIGIN", "")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.HTTPAddr != ":8080" || got.WebDir != "web/dist" || got.AllowedOrigin != "http://localhost:5173" {
		t.Fatalf("Load() defaults = %+v", got)
	}
}
