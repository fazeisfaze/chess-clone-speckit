package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"chess_clone/internal/config"
	"chess_clone/internal/match"
)

func TestHealthzReturnsJSON(t *testing.T) {
	handler := NewHandler(config.Config{})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got := recorder.Body.String(); got != "{\"status\":\"ok\"}\n" {
		t.Fatalf("body = %q, want health response", got)
	}
}

func TestHealthzRejectsNonGetMethods(t *testing.T) {
	handler := NewHandler(config.Config{})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/healthz", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}

func TestNewServerUsesConfiguredAddressAndTimeouts(t *testing.T) {
	server := NewServer(config.Config{HTTPAddr: ":9081"}, NewHandler(config.Config{}))
	if server.Addr != ":9081" {
		t.Fatalf("Addr = %q, want :9081", server.Addr)
	}
	if server.ReadHeaderTimeout <= 0 || server.IdleTimeout <= 0 {
		t.Fatalf("server timeouts must be configured: %+v", server)
	}
}

func TestWriteErrorUsesSharedEnvelope(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeError(recorder, http.StatusConflict, "match_full", "match is full")
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body.Error.Code != "match_full" || body.Error.Message != "match is full" {
		t.Fatalf("error body = %+v", body.Error)
	}
}

func TestMatchResponseSerializesOrderedMoveHistoryDetails(t *testing.T) {
	response := matchResponse{Moves: []moveHistoryResponse{
		{Sequence: 1, Color: "white", From: "e2", To: "e4", Notation: "e4"},
		{Sequence: 2, Color: "black", From: "e7", To: "e5", Notation: "e5"},
	}}
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal match response: %v", err)
	}
	var body struct {
		Moves []moveHistoryResponse `json:"moves"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("unmarshal match response: %v", err)
	}
	if len(body.Moves) != 2 || body.Moves[0].Notation != "e4" || body.Moves[1].Color != "black" {
		t.Fatalf("moves = %+v, want ordered e4/e5 history", body.Moves)
	}
}

type moveHistoryReadStore struct {
	match.Store
	game  match.Match
	color match.Color
	moves []match.Move
}

func (s moveHistoryReadStore) ReadForSeat(context.Context, string, []byte) (match.Match, match.Color, []match.Move, error) {
	return s.game, s.color, s.moves, nil
}

func TestAuthorizedMatchReadAndRecoveryReturnMoveHistory(t *testing.T) {
	store := moveHistoryReadStore{
		game:  match.Match{ID: "match-1", Status: match.StatusActive, Position: "start", TimeControl: "5+0", Version: 3},
		color: match.ColorWhite,
		moves: []match.Move{
			{Sequence: 1, Color: match.ColorWhite, From: "e2", To: "e4", Notation: "e4"},
			{Sequence: 2, Color: match.ColorBlack, From: "e7", To: "e5", Notation: "e5"},
		},
	}
	endpoints := matchEndpoints{service: match.NewService(store)}
	for _, tc := range []struct {
		name string
		hand func(http.ResponseWriter, *http.Request)
		req  *http.Request
	}{
		{name: "read", hand: endpoints.get, req: httptest.NewRequest(http.MethodGet, "/matches/match-1", nil)},
		{name: "recover", hand: endpoints.recover, req: httptest.NewRequest(http.MethodPost, "/matches/match-1/recover", strings.NewReader(`{"recovery_credential":"seat-secret"}`))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.req.SetPathValue("match_id", "match-1")
			tc.req.Header.Set("Authorization", "Bearer seat-secret")
			recorder := httptest.NewRecorder()
			tc.hand(recorder, tc.req)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
			}
			var body struct {
				Moves []moveHistoryResponse `json:"moves"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if len(body.Moves) != 2 || body.Moves[0].Sequence != 1 || body.Moves[0].Notation != "e4" || body.Moves[1].Color != "black" {
				t.Fatalf("moves = %+v, want ordered e4/e5 history", body.Moves)
			}
		})
	}
}

func TestServeShutsDownWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	server := NewServer(config.Config{HTTPAddr: "127.0.0.1:0"}, NewHandler(config.Config{}))
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, server) }()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not stop after context cancellation")
	}
}
