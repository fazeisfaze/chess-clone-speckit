package realtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func sessionTestServer(handler http.Handler) *httptest.Server {
	mux := http.NewServeMux()
	mux.Handle("GET /matches/{match_id}/events", handler)
	return httptest.NewServer(mux)
}

func dialTestSocket(t *testing.T, server *httptest.Server, origin string) (*websocket.Conn, context.Context) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/matches/match-1/events"
	header := http.Header{"Origin": []string{origin}}
	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		cancel()
		t.Fatalf("dial websocket: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close(websocket.StatusNormalClosure, "test complete")
		cancel()
	})
	return conn, ctx
}

func TestSessionRejectsUntrustedOrigin(t *testing.T) {
	server := sessionTestServer(NewSessionHandler(SessionOptions{
		AllowedOrigin:         "http://allowed.test",
		AuthenticationTimeout: time.Second,
		Authorize:             func(context.Context, string, string) bool { return true },
	}))
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/matches/match-1/events"
	_, response, err := websocket.Dial(context.Background(), url, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"http://untrusted.test"}},
	})
	if err == nil {
		t.Fatal("websocket connection from untrusted origin succeeded")
	}
	if response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("handshake response = %v, want HTTP 403", response)
	}
}

func TestSessionRequiresAuthenticationBeforeSendingData(t *testing.T) {
	server := sessionTestServer(NewSessionHandler(SessionOptions{
		AllowedOrigin:         "http://allowed.test",
		AuthenticationTimeout: time.Second,
		Authorize: func(_ context.Context, matchID, credential string) bool {
			return matchID == "match-1" && credential == "seat-secret"
		},
	}))
	defer server.Close()
	conn, ctx := dialTestSocket(t, server, "http://allowed.test")

	type readResult struct {
		message map[string]string
		err     error
	}
	firstMessage := make(chan readResult, 1)
	go func() {
		var message map[string]string
		err := wsjson.Read(ctx, conn, &message)
		firstMessage <- readResult{message: message, err: err}
	}()
	select {
	case got := <-firstMessage:
		t.Fatalf("server sent %v before authentication (err %v)", got.message, got.err)
	case <-time.After(20 * time.Millisecond):
	}

	if err := wsjson.Write(ctx, conn, map[string]string{"type": "authenticate", "recovery_credential": "seat-secret"}); err != nil {
		t.Fatalf("send authentication: %v", err)
	}
	select {
	case got := <-firstMessage:
		if got.err != nil {
			t.Fatalf("read authenticated response: %v", got.err)
		}
		if got.message["type"] != "authenticated" {
			t.Fatalf("first authenticated message = %v", got.message)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not acknowledge authentication")
	}
}

func TestSessionClosesWhenAuthenticationTimesOut(t *testing.T) {
	server := sessionTestServer(NewSessionHandler(SessionOptions{
		AllowedOrigin:         "http://allowed.test",
		AuthenticationTimeout: 20 * time.Millisecond,
		Authorize:             func(context.Context, string, string) bool { return true },
	}))
	defer server.Close()
	conn, ctx := dialTestSocket(t, server, "http://allowed.test")
	_, _, err := conn.Read(ctx)
	if err == nil {
		t.Fatal("connection remained open after authentication timeout")
	}
}

func TestSessionRejectsOversizedAuthenticationMessage(t *testing.T) {
	server := sessionTestServer(NewSessionHandler(SessionOptions{
		AllowedOrigin:         "http://allowed.test",
		AuthenticationTimeout: time.Second,
		MaxMessageBytes:       32,
		Authorize:             func(context.Context, string, string) bool { return true },
	}))
	defer server.Close()
	conn, ctx := dialTestSocket(t, server, "http://allowed.test")
	large := map[string]string{"type": "authenticate", "recovery_credential": strings.Repeat("x", 100)}
	if err := wsjson.Write(ctx, conn, large); err != nil {
		t.Fatalf("send oversized authentication: %v", err)
	}
	_, _, err := conn.Read(ctx)
	if websocket.CloseStatus(err) != websocket.StatusMessageTooBig {
		t.Fatalf("close status = %v, want message too big (err %v)", websocket.CloseStatus(err), err)
	}
}
