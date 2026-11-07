package clock

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

type timeoutTestStore struct {
	mu       sync.Mutex
	expired  []string
	flagged  chan string
	findCall int
}

func (s *timeoutTestStore) FindExpired(context.Context, time.Time) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.findCall++
	if s.findCall == 1 {
		return append([]string(nil), s.expired...), nil
	}
	return nil, nil
}

func (s *timeoutTestStore) RecordTimeout(_ context.Context, matchID string, _ time.Time) ([][]byte, error) {
	s.flagged <- matchID
	return [][]byte{
		[]byte(`{"type":"clock_expired","result":"1-0"}`),
		[]byte(`{"type":"game_ended","result":"1-0"}`),
	}, nil
}

func TestTimeoutWorkerFlagsExpiredMatchWithoutPlayerCommands(t *testing.T) {
	store := &timeoutTestStore{expired: []string{"match-1"}, flagged: make(chan string, 1)}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	published := make(chan []byte, 2)
	go func() {
		done <- RunTimeoutWorker(ctx, 5*time.Millisecond, store, func(_ string, event []byte) {
			published <- event
		})
	}()
	select {
	case matchID := <-store.flagged:
		if matchID != "match-1" {
			t.Fatalf("flagged match = %q", matchID)
		}
	case <-time.After(time.Second):
		cancel()
		t.Fatal("worker did not process expired clock without player commands")
	}
	select {
	case event := <-published:
		var got map[string]string
		if err := json.Unmarshal(event, &got); err != nil || got["type"] != "clock_expired" || got["result"] != "1-0" {
			t.Fatalf("published timeout event = %s (error %v)", event, err)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not publish the timeout result")
	}
	select {
	case event := <-published:
		var got map[string]string
		if err := json.Unmarshal(event, &got); err != nil || got["type"] != "game_ended" || got["result"] != "1-0" {
			t.Fatalf("published game-ended event = %s (error %v)", event, err)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not publish game_ended")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("worker returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not stop after cancellation")
	}
}
