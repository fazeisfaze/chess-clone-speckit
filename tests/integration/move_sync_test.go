package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/jackc/pgx/v5/pgxpool"

	"chess_clone/internal/config"
	"chess_clone/internal/store/postgres"
	httpapi "chess_clone/internal/transport/http"
)

func TestMoveEndpointAcceptsLegalMoveAndRejectsIllegalOrStaleMove(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for move integration tests")
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

	createBody, _ := json.Marshal(map[string]string{"time_control": "3+2"})
	createResponse, err := http.Post(server.URL+"/matches", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	var created struct {
		MatchID    string `json:"match_id"`
		Credential string `json:"recovery_credential"`
		InviteURL  string `json:"invite_url"`
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		createResponse.Body.Close()
		t.Fatal(err)
	}
	createResponse.Body.Close()
	token := strings.TrimPrefix(strings.SplitN(created.InviteURL, "#token=", 2)[1], "")
	joinBody, _ := json.Marshal(map[string]string{"invite_token": token})
	joinResponse, err := http.Post(server.URL+"/matches/join", "application/json", bytes.NewReader(joinBody))
	if err != nil {
		t.Fatal(err)
	}
	var joined struct {
		Credential string `json:"recovery_credential"`
	}
	if err := json.NewDecoder(joinResponse.Body).Decode(&joined); err != nil {
		joinResponse.Body.Close()
		t.Fatal(err)
	}
	joinResponse.Body.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/matches/" + created.MatchID + "/events"
	wsContext, wsCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer wsCancel()
	ws, _, err := websocket.Dial(wsContext, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"http://localhost:5173"}},
	})
	if err != nil {
		t.Fatalf("connect realtime session: %v", err)
	}
	defer ws.Close(websocket.StatusNormalClosure, "test complete")
	if err := wsjson.Write(wsContext, ws, map[string]string{"type": "authenticate", "recovery_credential": joined.Credential}); err != nil {
		t.Fatal(err)
	}
	var snapshot map[string]any
	if err := wsjson.Read(wsContext, ws, &snapshot); err != nil {
		t.Fatalf("read match snapshot: %v", err)
	}
	if snapshot["type"] != "match_snapshot" {
		t.Fatalf("first realtime message = %v, want match_snapshot", snapshot)
	}
	var connectionEvent map[string]any
	if err := wsjson.Read(wsContext, ws, &connectionEvent); err != nil || connectionEvent["type"] != "opponent_connection" {
		t.Fatalf("initial presence event = %v, error = %v", connectionEvent, err)
	}

	submit := func(credential string, version int64, from, to string) (*http.Response, error) {
		body, _ := json.Marshal(map[string]any{"expected_version": version, "from": from, "to": to, "promotion": nil})
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/matches/"+created.MatchID+"/moves", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+credential)
		req.Header.Set("Content-Type", "application/json")
		return http.DefaultClient.Do(req)
	}
	response, err := submit(created.Credential, 1, "e2", "e4")
	if err != nil {
		t.Fatal(err)
	}
	var accepted struct {
		Version  int64  `json:"version"`
		Position string `json:"position"`
		NextTurn string `json:"next_turn"`
		Clock    struct {
			WhiteRemainingMS int64 `json:"white_remaining_ms"`
			BlackRemainingMS int64 `json:"black_remaining_ms"`
		} `json:"clock"`
	}
	if err := json.NewDecoder(response.Body).Decode(&accepted); err != nil {
		response.Body.Close()
		t.Fatalf("decode move response: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK || accepted.Version != 2 || accepted.NextTurn != "black" || accepted.Position == "" {
		t.Fatalf("accepted move status=%d response=%+v", response.StatusCode, accepted)
	}
	if accepted.Clock.WhiteRemainingMS <= 0 || accepted.Clock.WhiteRemainingMS > 182_000 || accepted.Clock.BlackRemainingMS != 180_000 {
		t.Fatalf("clock after 3+2 move = %+v", accepted.Clock)
	}
	var event map[string]any
	if err := wsjson.Read(wsContext, ws, &event); err != nil {
		t.Fatalf("read move event: %v", err)
	}
	if event["type"] != "move_accepted" || event["version"] != float64(2) {
		t.Fatalf("move event = %s", fmt.Sprint(event))
	}
	if err := wsjson.Write(wsContext, ws, map[string]any{
		"type": "move", "expected_version": 2, "from": "e7", "to": "e5", "promotion": nil,
	}); err != nil {
		t.Fatalf("submit WebSocket move: %v", err)
	}
	if err := wsjson.Read(wsContext, ws, &event); err != nil {
		t.Fatalf("read WebSocket move event: %v", err)
	}
	if event["type"] != "move_accepted" || event["version"] != float64(3) {
		t.Fatalf("WebSocket move event = %s", fmt.Sprint(event))
	}

	for _, attempt := range []struct {
		credential string
		version    int64
		from, to   string
		want       int
	}{
		{credential: joined.Credential, version: 1, from: "e7", to: "e5", want: http.StatusConflict},
		{credential: created.Credential, version: 3, from: "e4", to: "e6", want: http.StatusUnprocessableEntity},
	} {
		failed, err := submit(attempt.credential, attempt.version, attempt.from, attempt.to)
		if err != nil {
			t.Fatal(err)
		}
		failed.Body.Close()
		if failed.StatusCode != attempt.want {
			t.Errorf("move %s%s status=%d, want %d", attempt.from, attempt.to, failed.StatusCode, attempt.want)
		}
	}
	for _, move := range []struct {
		credential string
		version    int64
		from, to   string
	}{
		{credential: created.Credential, version: 3, from: "d1", to: "h5"},
		{credential: joined.Credential, version: 4, from: "b8", to: "c6"},
		{credential: created.Credential, version: 5, from: "f1", to: "c4"},
		{credential: joined.Credential, version: 6, from: "g8", to: "f6"},
	} {
		response, err := submit(move.credential, move.version, move.from, move.to)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("move %s%s status=%d", move.from, move.to, response.StatusCode)
		}
		if err := wsjson.Read(wsContext, ws, &event); err != nil || event["type"] != "move_accepted" {
			t.Fatalf("intermediate move event = %v, error = %v", event, err)
		}
	}
	finalResponse, err := submit(created.Credential, 7, "h5", "f7")
	if err != nil {
		t.Fatal(err)
	}
	var finalMove struct {
		Result  string `json:"result"`
		Version int64  `json:"version"`
		InCheck bool   `json:"in_check"`
		Move    struct {
			Check bool `json:"check"`
		} `json:"move"`
	}
	if err := json.NewDecoder(finalResponse.Body).Decode(&finalMove); err != nil {
		finalResponse.Body.Close()
		t.Fatal(err)
	}
	finalResponse.Body.Close()
	if finalResponse.StatusCode != http.StatusOK || finalMove.Result != "1-0" || finalMove.Version != 8 || !finalMove.InCheck || !finalMove.Move.Check {
		t.Fatalf("checkmate move status=%d result=%+v", finalResponse.StatusCode, finalMove)
	}
	if err := wsjson.Read(wsContext, ws, &event); err != nil || event["type"] != "move_accepted" {
		t.Fatalf("final move event = %v, error = %v", event, err)
	}
	if err := wsjson.Read(wsContext, ws, &event); err != nil || event["type"] != "game_ended" {
		t.Fatalf("terminal event = %v, error = %v", event, err)
	}
}
