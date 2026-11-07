package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"chess_clone/internal/config"
	"chess_clone/internal/match"
	"chess_clone/internal/store/postgres"
	httpapi "chess_clone/internal/transport/http"
)

func TestRecoverSeatReturnsAuthoritativeBoardAndRunningClock(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for recovery integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	_, currentFile, _, _ := runtime.Caller(0)
	if err := postgres.ApplyMigrations(ctx, pool, filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	store := postgres.NewMatchStore(pool)
	service := match.NewService(store)
	created, err := service.Create(ctx, "5+0")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Join(ctx, created.InviteToken); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := service.ApplyMove(ctx, created.Match.ID, created.RecoveryCredential, 1, "e2", "e4", ""); err != nil {
		t.Fatalf("accept first move: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE matches SET clock_started_at = clock_timestamp() - interval '10 seconds' WHERE id = $1`, created.Match.ID); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(httpapi.NewApplicationHandler(config.Config{AllowedOrigin: "http://localhost:5173"}, pool))
	defer server.Close()

	requestRecovery := func(credential string) (*http.Response, error) {
		body, _ := json.Marshal(map[string]any{"recovery_credential": credential, "known_version": 1})
		return http.Post(server.URL+"/matches/"+created.Match.ID+"/recover", "application/json", bytes.NewReader(body))
	}
	response, err := requestRecovery(created.RecoveryCredential)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot struct {
		Status   string `json:"status"`
		Color    string `json:"color"`
		Position string `json:"position"`
		Version  int64  `json:"version"`
		Moves    []struct {
			Sequence int64  `json:"sequence"`
			Color    string `json:"color"`
			From     string `json:"from"`
			To       string `json:"to"`
			Notation string `json:"notation"`
		} `json:"moves"`
		Clock struct {
			WhiteRemainingMS int64  `json:"white_remaining_ms"`
			ActiveColor      string `json:"active_color"`
		} `json:"clock"`
	}
	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil {
		response.Body.Close()
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK || snapshot.Status != "active" || snapshot.Color != "white" || snapshot.Version != 2 || snapshot.Clock.ActiveColor != "black" {
		t.Fatalf("recovery response status=%d snapshot=%+v", response.StatusCode, snapshot)
	}
	if len(snapshot.Moves) != 1 || snapshot.Moves[0].Sequence != 1 || snapshot.Moves[0].Color != "white" || snapshot.Moves[0].From != "e2" || snapshot.Moves[0].To != "e4" || snapshot.Moves[0].Notation != "e4" {
		t.Fatalf("recovered moves = %+v, want accepted 1. e4", snapshot.Moves)
	}
	if snapshot.Position == "" || snapshot.Clock.WhiteRemainingMS >= 300_000 || snapshot.Clock.WhiteRemainingMS <= 280_000 {
		t.Fatalf("recovered position/clock = %q/%d", snapshot.Position, snapshot.Clock.WhiteRemainingMS)
	}
	invalid, err := requestRecovery("not-a-seat-credential")
	if err != nil {
		t.Fatal(err)
	}
	invalid.Body.Close()
	if invalid.StatusCode != http.StatusUnauthorized {
		t.Fatalf("invalid recovery status = %d, want %d", invalid.StatusCode, http.StatusUnauthorized)
	}
}
