package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestMatchLifecycleCreatesJoinsAndRejectsThirdPlayer(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for match lifecycle integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer pool.Close()
	_, currentFile, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations")
	if err := postgres.ApplyMigrations(ctx, pool, migrationsDir); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	server := httptest.NewServer(httpapi.NewApplicationHandler(config.Config{AllowedOrigin: "http://localhost:5173"}, pool))
	defer server.Close()

	createBody, _ := json.Marshal(map[string]string{"time_control": "5+0"})
	createRequest, _ := http.NewRequest(http.MethodPost, server.URL+"/matches", bytes.NewReader(createBody))
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.Header.Set("Origin", server.URL)
	createResponse, err := http.DefaultClient.Do(createRequest)
	if err != nil {
		t.Fatalf("create match: %v", err)
	}
	var created struct {
		MatchID     string `json:"match_id"`
		Status      string `json:"status"`
		TimeControl string `json:"time_control"`
		InviteURL   string `json:"invite_url"`
		Credential  string `json:"recovery_credential"`
		Clock       struct {
			ActiveColor string `json:"active_color"`
		} `json:"clock"`
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		createResponse.Body.Close()
		t.Fatalf("decode create response: %v", err)
	}
	createResponse.Body.Close()
	if createResponse.StatusCode != http.StatusCreated || created.Status != "waiting" || created.TimeControl != "5+0" || created.Clock.ActiveColor != "" {
		t.Fatalf("create response status=%d body=%+v", createResponse.StatusCode, created)
	}
	if created.Credential == "" {
		t.Fatal("host did not receive a seat recovery credential")
	}
	if !strings.HasPrefix(created.InviteURL, server.URL+"/") {
		t.Fatalf("invitation URL = %q; expected it to stay on the server-served web origin %q", created.InviteURL, server.URL)
	}
	inviteURL, err := url.Parse(created.InviteURL)
	if err != nil {
		t.Fatalf("parse invitation URL: %v", err)
	}
	token := strings.TrimPrefix(inviteURL.Fragment, "token=")
	joinBody, _ := json.Marshal(map[string]string{"invite_token": token})
	joinResponse, err := http.Post(server.URL+"/matches/join", "application/json", bytes.NewReader(joinBody))
	if err != nil {
		t.Fatalf("join match: %v", err)
	}
	var joined struct {
		MatchID  string `json:"match_id"`
		Status   string `json:"status"`
		Color    string `json:"color"`
		Position string `json:"position"`
		Clock    struct {
			ActiveColor string `json:"active_color"`
			WhiteMS     int64  `json:"white_remaining_ms"`
			BlackMS     int64  `json:"black_remaining_ms"`
		} `json:"clock"`
	}
	if err := json.NewDecoder(joinResponse.Body).Decode(&joined); err != nil {
		joinResponse.Body.Close()
		t.Fatalf("decode join response: %v", err)
	}
	joinResponse.Body.Close()
	if joinResponse.StatusCode != http.StatusOK || joined.MatchID != created.MatchID || joined.Status != "active" || joined.Color != "black" {
		t.Fatalf("join response status=%d body=%+v", joinResponse.StatusCode, joined)
	}
	if joined.Clock.ActiveColor != "white" || joined.Clock.WhiteMS != 300_000 || joined.Clock.BlackMS != 300_000 {
		t.Fatalf("activated clock = %+v", joined.Clock)
	}
	stateRequest, err := http.NewRequest(http.MethodGet, server.URL+"/matches/"+created.MatchID, nil)
	if err != nil {
		t.Fatal(err)
	}
	stateRequest.Header.Set("Authorization", "Bearer "+created.Credential)
	stateResponse, err := http.DefaultClient.Do(stateRequest)
	if err != nil {
		t.Fatalf("host reads joined match: %v", err)
	}
	var hostState struct {
		Status string `json:"status"`
		Color  string `json:"color"`
		Clock  struct {
			ActiveColor string `json:"active_color"`
		} `json:"clock"`
	}
	if err := json.NewDecoder(stateResponse.Body).Decode(&hostState); err != nil {
		stateResponse.Body.Close()
		t.Fatalf("decode host state: %v", err)
	}
	stateResponse.Body.Close()
	if stateResponse.StatusCode != http.StatusOK || hostState.Status != "active" || hostState.Color != "white" || hostState.Clock.ActiveColor != "white" {
		t.Fatalf("host state status=%d body=%+v", stateResponse.StatusCode, hostState)
	}

	thirdResponse, err := http.Post(server.URL+"/matches/join", "application/json", bytes.NewReader(joinBody))
	if err != nil {
		t.Fatalf("third join attempt: %v", err)
	}
	thirdResponse.Body.Close()
	if thirdResponse.StatusCode != http.StatusConflict {
		t.Fatalf("third join status = %d, want %d", thirdResponse.StatusCode, http.StatusConflict)
	}
}
