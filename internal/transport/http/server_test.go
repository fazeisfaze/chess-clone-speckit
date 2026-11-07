package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"chess_clone/internal/config"
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
