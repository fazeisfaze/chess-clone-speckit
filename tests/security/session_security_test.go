package security_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"chess_clone/internal/config"
	"chess_clone/internal/store/postgres"
	httpapi "chess_clone/internal/transport/http"
)

func TestRateLimiterBlocksOverBudgetAndResetsWindow(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	limiter := httpapi.NewRateLimiter(2, time.Minute, func() time.Time { return now })
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := func() *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/matches", nil)
		req.RemoteAddr = "192.0.2.10:1234"
		handler.ServeHTTP(recorder, req)
		return recorder
	}
	if got := request().Code; got != http.StatusNoContent {
		t.Fatalf("first request status = %d", got)
	}
	if got := request().Code; got != http.StatusNoContent {
		t.Fatalf("second request status = %d", got)
	}
	if got := request().Code; got != http.StatusTooManyRequests {
		t.Fatalf("over-budget status = %d, want %d", got, http.StatusTooManyRequests)
	}
	now = now.Add(time.Minute)
	if got := request().Code; got != http.StatusNoContent {
		t.Fatalf("request after window reset status = %d", got)
	}
}

func TestUnauthenticatedStateRequestDoesNotEchoSeatSecrets(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for security integration tests")
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
	server := httptest.NewServer(httpapi.NewApplicationHandler(config.Config{AllowedOrigin: "http://localhost:5173"}, pool))
	defer server.Close()
	create, err := http.Post(server.URL+"/matches", "application/json", strings.NewReader(`{"time_control":"5+0"}`))
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		MatchID    string `json:"match_id"`
		InviteURL  string `json:"invite_url"`
		Credential string `json:"recovery_credential"`
	}
	if err := json.NewDecoder(create.Body).Decode(&body); err != nil {
		create.Body.Close()
		t.Fatal(err)
	}
	create.Body.Close()
	response, err := http.Get(server.URL + "/matches/" + body.MatchID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want %d", response.StatusCode, http.StatusUnauthorized)
	}
	if strings.Contains(string(data), body.Credential) || strings.Contains(string(data), body.InviteURL) {
		t.Fatalf("unauthenticated response disclosed a seat secret: %s", data)
	}
}
