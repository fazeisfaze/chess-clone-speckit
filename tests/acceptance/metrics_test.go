package acceptance

import (
	"bytes"
	"context"
	"encoding/json"
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
	chesslib "github.com/corentings/chess/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"chess_clone/internal/config"
	"chess_clone/internal/store/postgres"
	httpapi "chess_clone/internal/transport/http"
)

const metricOrigin = "http://localhost:5173"

var twentyPlyOpening = []string{
	"e2e4", "e7e5", "g1f3", "b8c6", "f1b5", "a7a6", "b5a4", "g8f6", "e1g1", "f8e7",
	"f1e1", "b7b5", "a4b3", "d7d6", "c2c3", "e8g8", "h2h3", "c8b7", "d2d4", "e5d4",
}

type matchCredentials struct {
	id     string
	invite string
	white  string
	black  string
}

type realtimeEvent struct {
	Type     string          `json:"type"`
	Version  int64           `json:"version"`
	Position string          `json:"position"`
	NextTurn string          `json:"next_turn"`
	Result   string          `json:"result"`
	Clock    json.RawMessage `json:"clock"`
}

func TestAcceptanceMetrics(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for acceptance metrics")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
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
	app := httpapi.NewApplication(config.Config{AllowedOrigin: metricOrigin}, pool)
	server := httptest.NewServer(app.Handler)
	defer server.Close()

	engine := chesslib.NewGame()
	for _, uci := range twentyPlyOpening {
		move, err := (chesslib.UCINotation{}).Decode(engine.Position(), uci)
		if err != nil {
			t.Fatalf("acceptance opening move %s is illegal: %v", uci, err)
		}
		if err := engine.Move(move, nil); err != nil {
			t.Fatalf("apply acceptance opening move %s: %v", uci, err)
		}
	}
	if engine.Outcome() != chesslib.NoOutcome {
		t.Fatalf("acceptance sequence ends early with %s", engine.Outcome())
	}

	var successfulJoins, timelyMoves, totalMoves, successfulRecoveries, sameTimeouts int
	for gameIndex := 0; gameIndex < 20; gameIndex++ {
		started := time.Now()
		credentials := createAndJoin(t, ctx, server.URL)
		if time.Since(started) <= 60*time.Second {
			successfulJoins++
		}

		whiteSocket := connectSeat(t, ctx, server.URL, credentials.id, credentials.white)
		blackSocket := connectSeat(t, ctx, server.URL, credentials.id, credentials.black)
		for ply, uci := range twentyPlyOpening {
			version := int64(ply + 1)
			moveSocket := whiteSocket
			if ply%2 == 1 {
				moveSocket = blackSocket
			}
			from, to, promotion := splitUCI(uci)
			latencyStart := time.Now()
			if err := wsjson.Write(ctx, moveSocket, map[string]any{
				"type": "move", "expected_version": version, "from": from, "to": to, "promotion": promotion,
			}); err != nil {
				t.Fatalf("game %d ply %d submit move: %v", gameIndex, ply, err)
			}
			whiteEvent := readMoveEvent(t, ctx, whiteSocket, version+1)
			blackEvent := readMoveEvent(t, ctx, blackSocket, version+1)
			latency := time.Since(latencyStart)
			totalMoves++
			if latency <= 2*time.Second {
				timelyMoves++
			}
			if whiteEvent.Position != blackEvent.Position || whiteEvent.Version != blackEvent.Version || string(whiteEvent.Clock) != string(blackEvent.Clock) {
				t.Fatalf("clients diverged at game %d ply %d: white=%+v black=%+v", gameIndex, ply, whiteEvent, blackEvent)
			}
		}
		_ = whiteSocket.Close(websocket.StatusNormalClosure, "reconnect validation")
		recovered := recoverSeat(t, ctx, server.URL, credentials.id, credentials.white)
		if recovered.Status == "active" && recovered.Position != "" && recovered.Version == 21 && recovered.Clock.ActiveColor == "white" {
			successfulRecoveries++
		}
		whiteSocket = connectSeat(t, ctx, server.URL, credentials.id, credentials.white)

		if _, err := pool.Exec(ctx, `UPDATE matches SET white_remaining_ms = 1,
			clock_started_at = clock_timestamp() - interval '2 seconds' WHERE id = $1`, credentials.id); err != nil {
			t.Fatalf("force timeout for metric trial: %v", err)
		}
		events, err := app.MatchStore.RecordTimeout(ctx, credentials.id, time.Now())
		if err != nil || len(events) != 2 {
			t.Fatalf("record timeout events=%d error=%v", len(events), err)
		}
		for _, event := range events {
			app.Hub.Publish(credentials.id, event)
		}
		whiteTimeout := readTimeoutEvent(t, ctx, whiteSocket, "game_ended")
		blackTimeout := readTimeoutEvent(t, ctx, blackSocket, "game_ended")
		if whiteTimeout.Result == "0-1" && whiteTimeout.Result == blackTimeout.Result {
			sameTimeouts++
		}
		_ = whiteSocket.Close(websocket.StatusNormalClosure, "metric complete")
		_ = blackSocket.Close(websocket.StatusNormalClosure, "metric complete")
	}
	if successfulJoins < 19 {
		t.Errorf("SC-001 joined %d/20 pairs within 60 seconds, want at least 19", successfulJoins)
	}
	if timelyMoves < 396 || totalMoves != 400 {
		t.Errorf("SC-002 had %d/%d moves within 2 seconds, want at least 396/400", timelyMoves, totalMoves)
	}
	if successfulRecoveries < 19 {
		t.Errorf("SC-004 recovered %d/20 seats within five minutes, want at least 19", successfulRecoveries)
	}
	if sameTimeouts != 20 {
		t.Errorf("SC-006 clients agreed on %d/20 timeout results, want 20", sameTimeouts)
	}
}

func createAndJoin(t *testing.T, ctx context.Context, baseURL string) matchCredentials {
	t.Helper()
	createRequest, _ := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/matches", bytes.NewBufferString(`{"time_control":"3+2"}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createdResponse, err := http.DefaultClient.Do(createRequest)
	if err != nil {
		t.Fatal(err)
	}
	var created struct {
		ID         string `json:"match_id"`
		InviteURL  string `json:"invite_url"`
		Credential string `json:"recovery_credential"`
	}
	if err := json.NewDecoder(createdResponse.Body).Decode(&created); err != nil {
		createdResponse.Body.Close()
		t.Fatal(err)
	}
	createdResponse.Body.Close()
	inviteParts := strings.SplitN(created.InviteURL, "#token=", 2)
	if len(inviteParts) != 2 {
		t.Fatalf("invitation URL is invalid: %q", created.InviteURL)
	}
	joinBody, _ := json.Marshal(map[string]string{"invite_token": inviteParts[1]})
	joinRequest, _ := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/matches/join", bytes.NewReader(joinBody))
	joinRequest.Header.Set("Content-Type", "application/json")
	joinedResponse, err := http.DefaultClient.Do(joinRequest)
	if err != nil {
		t.Fatal(err)
	}
	var joined struct {
		Credential string `json:"recovery_credential"`
	}
	if err := json.NewDecoder(joinedResponse.Body).Decode(&joined); err != nil {
		joinedResponse.Body.Close()
		t.Fatal(err)
	}
	joinedResponse.Body.Close()
	if createdResponse.StatusCode != http.StatusCreated || joinedResponse.StatusCode != http.StatusOK {
		t.Fatalf("create/join status=%d/%d", createdResponse.StatusCode, joinedResponse.StatusCode)
	}
	return matchCredentials{id: created.ID, invite: inviteParts[1], white: created.Credential, black: joined.Credential}
}

func connectSeat(t *testing.T, ctx context.Context, baseURL, matchID, credential string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(baseURL, "http") + "/matches/" + matchID + "/events"
	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{metricOrigin}}})
	if err != nil {
		t.Fatalf("dial seat websocket: %v", err)
	}
	if err := wsjson.Write(ctx, conn, map[string]string{"type": "authenticate", "recovery_credential": credential}); err != nil {
		t.Fatalf("authenticate seat websocket: %v", err)
	}
	for {
		var event realtimeEvent
		if err := wsjson.Read(ctx, conn, &event); err != nil {
			_ = conn.Close(websocket.StatusInternalError, "snapshot missing")
			t.Fatalf("read seat snapshot: %v", err)
		}
		if event.Type == "match_snapshot" {
			return conn
		}
	}
}

func readMoveEvent(t *testing.T, ctx context.Context, conn *websocket.Conn, version int64) realtimeEvent {
	t.Helper()
	for {
		var event realtimeEvent
		if err := wsjson.Read(ctx, conn, &event); err != nil {
			t.Fatalf("read move event version %d: %v", version, err)
		}
		if event.Type == "move_accepted" && event.Version == version {
			return event
		}
	}
}

func readTimeoutEvent(t *testing.T, ctx context.Context, conn *websocket.Conn, eventType string) realtimeEvent {
	t.Helper()
	for {
		var event realtimeEvent
		if err := wsjson.Read(ctx, conn, &event); err != nil {
			t.Fatalf("read timeout event %s: %v", eventType, err)
		}
		if event.Type == eventType {
			return event
		}
	}
}

func recoverSeat(t *testing.T, ctx context.Context, baseURL, matchID, credential string) struct {
	Status   string `json:"status"`
	Position string `json:"position"`
	Version  int64  `json:"version"`
	Clock    struct {
		ActiveColor string `json:"active_color"`
	} `json:"clock"`
} {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"recovery_credential": credential, "known_version": 0})
	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/matches/"+matchID+"/recover", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var state struct {
		Status   string `json:"status"`
		Position string `json:"position"`
		Version  int64  `json:"version"`
		Clock    struct {
			ActiveColor string `json:"active_color"`
		} `json:"clock"`
	}
	if err := json.NewDecoder(response.Body).Decode(&state); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("recover status=%d", response.StatusCode)
	}
	return state
}

func splitUCI(uci string) (string, string, string) {
	from, to := uci[:2], uci[2:4]
	if len(uci) == 5 {
		return from, to, uci[4:]
	}
	return from, to, ""
}
