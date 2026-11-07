package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"chess_clone/internal/match"
	"chess_clone/internal/store/postgres"
)

func TestExpiredClockPersistsTimeoutAndDrawsDeadPositions(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for timeout integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := postgres.NewMatchStore(pool)

	cases := []struct {
		name     string
		position string
		want     string
	}{
		{name: "mating-material", position: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", want: "0-1"},
		{name: "insufficient-material", position: "8/8/8/8/8/8/4k3/7K w - - 0 1", want: "1/2-1/2"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			control, _ := match.ParseTimeControl("3+2")
			id := fmt.Sprintf("timeout-%d", time.Now().UnixNano())
			game, white, err := match.NewMatch(id, id+"-invite", id+"-white", control)
			if err != nil {
				t.Fatal(err)
			}
			white.RecoveryCredentialDigest = []byte(id + "-white-recovery")
			if err := store.Create(ctx, game, white); err != nil {
				t.Fatal(err)
			}
			black := match.PlayerSeat{ID: id + "-black", MatchID: id, Color: match.ColorBlack, RecoveryCredentialDigest: []byte(id + "-black-recovery")}
			if _, err := store.Join(ctx, game.InviteTokenDigest, black); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `UPDATE matches SET position = $2, white_remaining_ms = 1,
				clock_started_at = clock_timestamp() - interval '2 seconds' WHERE id = $1`, id, test.position); err != nil {
				t.Fatal(err)
			}
			expired, err := store.FindExpired(ctx, time.Now())
			if err != nil || len(expired) == 0 {
				t.Fatalf("FindExpired = %v, error = %v", expired, err)
			}
			events, err := store.RecordTimeout(ctx, id, time.Now())
			if err != nil {
				t.Fatalf("RecordTimeout: %v", err)
			}
			if len(events) != 2 {
				t.Fatalf("timeout event count = %d, want 2", len(events))
			}
			for _, event := range events {
				var got struct {
					Type   string `json:"type"`
					Result string `json:"result"`
				}
				if err := json.Unmarshal(event, &got); err != nil {
					t.Fatal(err)
				}
				if got.Result != test.want {
					t.Errorf("timeout event %s result = %q, want %q", got.Type, got.Result, test.want)
				}
			}
			var status string
			var remaining int64
			if err := pool.QueryRow(ctx, `SELECT status, white_remaining_ms FROM matches WHERE id = $1`, id).Scan(&status, &remaining); err != nil {
				t.Fatal(err)
			}
			if status != "completed" || remaining != 0 {
				t.Fatalf("persisted timeout status/time = %s/%d", status, remaining)
			}
		})
	}
}
