package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestHubPreservesCommittedEventOrderPerMatch(t *testing.T) {
	hub := NewHub()
	matchEvents, unsubscribe := hub.Subscribe("match-1")
	defer unsubscribe()
	otherMatchEvents, unsubscribeOther := hub.Subscribe("match-2")
	defer unsubscribeOther()

	hub.Publish("match-1", []byte(`{"type":"move_accepted","version":2}`))
	hub.Publish("match-1", []byte(`{"type":"move_accepted","version":3}`))
	for _, version := range []int{2, 3} {
		select {
		case event := <-matchEvents:
			var decoded struct {
				Version int `json:"version"`
			}
			if err := json.Unmarshal(event, &decoded); err != nil || decoded.Version != version {
				t.Fatalf("event = %s, want version %d (error %v)", event, version, err)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for published event")
		}
	}
	select {
	case event := <-otherMatchEvents:
		t.Fatalf("event for another match leaked: %s", event)
	default:
	}
}

func TestAuthenticatedSessionSendsSnapshotBeforeVersionedEvents(t *testing.T) {
	hub := NewHub()
	server := sessionTestServer(NewSessionHandler(SessionOptions{
		AllowedOrigin:         "http://allowed.test",
		AuthenticationTimeout: time.Second,
		Authorize:             func(context.Context, string, string) bool { return true },
		Hub:                   hub,
		Snapshot: func(context.Context, string, string) ([]byte, error) {
			return []byte(`{"type":"match_snapshot","version":1}`), nil
		},
	}))
	defer server.Close()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/matches/match-1/events"
	conn, _, err := websocket.Dial(context.Background(), url, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"http://allowed.test"}},
	})
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "test complete")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := wsjson.Write(ctx, conn, map[string]string{"type": "authenticate", "recovery_credential": "seat-secret"}); err != nil {
		t.Fatal(err)
	}
	readType := func() (map[string]any, error) {
		var event map[string]any
		err := wsjson.Read(ctx, conn, &event)
		return event, err
	}
	snapshot, err := readType()
	if err != nil || snapshot["type"] != "match_snapshot" || snapshot["version"] != float64(1) {
		t.Fatalf("snapshot = %v, error = %v", snapshot, err)
	}
	presence, err := readType()
	if err != nil || presence["type"] != "opponent_connection" {
		t.Fatalf("presence = %v, error = %v", presence, err)
	}
	hub.Publish("match-1", []byte(`{"type":"move_accepted","version":2}`))
	hub.Publish("match-1", []byte(`{"type":"move_accepted","version":3}`))
	for _, version := range []float64{2, 3} {
		event, err := readType()
		if err != nil || event["type"] != "move_accepted" || event["version"] != version {
			t.Fatalf("event = %v, want version %.0f (error %v)", event, version, err)
		}
	}
}

func TestAuthenticatedSessionBroadcastsOpponentPresenceChanges(t *testing.T) {
	hub := NewHub()
	observer, unsubscribe := hub.Subscribe("match-1")
	defer unsubscribe()
	server := sessionTestServer(NewSessionHandler(SessionOptions{
		AllowedOrigin: "http://allowed.test",
		Authorize:     func(context.Context, string, string) bool { return true },
		Hub:           hub,
		Snapshot: func(context.Context, string, string) ([]byte, error) {
			return []byte(`{"type":"match_snapshot","match_id":"match-1","color":"white","version":1}`), nil
		},
	}))
	defer server.Close()
	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/matches/match-1/events"
	conn, _, err := websocket.Dial(context.Background(), url, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"http://allowed.test"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := wsjson.Write(ctx, conn, map[string]string{"type": "authenticate", "recovery_credential": "seat-secret"}); err != nil {
		t.Fatal(err)
	}
	var snapshot map[string]any
	if err := wsjson.Read(ctx, conn, &snapshot); err != nil || snapshot["type"] != "match_snapshot" {
		t.Fatalf("snapshot = %v, error = %v", snapshot, err)
	}
	readPresence := func(want bool) {
		t.Helper()
		select {
		case raw := <-observer:
			var event struct {
				Type      string `json:"type"`
				Color     string `json:"color"`
				Connected bool   `json:"connected"`
			}
			if err := json.Unmarshal(raw, &event); err != nil || event.Type != "opponent_connection" || event.Color != "white" || event.Connected != want {
				t.Fatalf("presence = %+v, err=%v", event, err)
			}
		case <-time.After(time.Second):
			t.Fatal("presence event was not broadcast")
		}
	}
	readPresence(true)
	_ = conn.Close(websocket.StatusNormalClosure, "test disconnect")
	readPresence(false)
}
