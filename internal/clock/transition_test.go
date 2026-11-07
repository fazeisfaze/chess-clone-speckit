package clock

import (
	"errors"
	"testing"
	"time"
)

func TestApplyMoveDebitsElapsedTimeAndAddsOnlySelectedIncrement(t *testing.T) {
	started := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	state, err := Start(3*time.Minute, started)
	if err != nil {
		t.Fatal(err)
	}

	next, err := ApplyMove(state, 2*time.Second, started.Add(20*time.Second))
	if err != nil {
		t.Fatalf("ApplyMove: %v", err)
	}
	if next.WhiteRemaining != 162*time.Second || next.BlackRemaining != 3*time.Minute {
		t.Fatalf("remaining = %s/%s, want 2m42s/3m0s", next.WhiteRemaining, next.BlackRemaining)
	}
	if next.ActiveColor != "black" || !next.StartedAt.Equal(started.Add(20*time.Second)) {
		t.Fatalf("next clock = %+v", next)
	}

	blackMove, err := ApplyMove(next, 0, next.StartedAt.Add(time.Second))
	if err != nil {
		t.Fatalf("ApplyMove black: %v", err)
	}
	if blackMove.BlackRemaining != 3*time.Minute-time.Second || blackMove.WhiteRemaining != 162*time.Second {
		t.Fatalf("5+0-style transition changed wrong clock: %+v", blackMove)
	}
}

func TestApplyMoveRejectsMoveAtOrAfterFlagFall(t *testing.T) {
	started := time.Now().UTC()
	state, err := Start(time.Second, started)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ApplyMove(state, 2*time.Second, started.Add(time.Second))
	if !errors.Is(err, ErrFlagFell) {
		t.Fatalf("ApplyMove error = %v, want ErrFlagFell", err)
	}
}
