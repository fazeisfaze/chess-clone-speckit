package match

import (
	"testing"
	"time"
)

func TestActivateMatchStartsWhiteClockAndPausesBlack(t *testing.T) {
	control, _ := ParseTimeControl("3+2")
	game, _, err := NewMatch("match-1", "invite-digest", "white-seat", control)
	if err != nil {
		t.Fatal(err)
	}
	black := PlayerSeat{ID: "black-seat", MatchID: game.ID, Color: ColorBlack}
	startedAt := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)

	active, err := ActivateMatch(game, black, startedAt)
	if err != nil {
		t.Fatalf("ActivateMatch: %v", err)
	}
	if active.Status != StatusActive || active.BlackPlayerID != black.ID {
		t.Fatalf("activated match = %+v", active)
	}
	if active.ClockActiveColor != ColorWhite || active.ClockStartedAt == nil || !active.ClockStartedAt.Equal(startedAt) {
		t.Fatalf("White clock did not start at activation: %+v", active)
	}
	if active.WhiteRemaining != 3*time.Minute || active.BlackRemaining != 3*time.Minute {
		t.Fatalf("remaining clocks = %s/%s", active.WhiteRemaining, active.BlackRemaining)
	}
}

func TestActivateMatchRejectsWrongSeatAndRepeatedActivation(t *testing.T) {
	control, _ := ParseTimeControl("5+0")
	game, _, _ := NewMatch("match-1", "invite-digest", "white-seat", control)
	wrongSeat := PlayerSeat{ID: "other-black", MatchID: "another-match", Color: ColorBlack}
	if _, err := ActivateMatch(game, wrongSeat, time.Now()); err == nil {
		t.Fatal("ActivateMatch accepted a seat from a different match")
	}
	validSeat := PlayerSeat{ID: "black-seat", MatchID: game.ID, Color: ColorBlack}
	active, err := ActivateMatch(game, validSeat, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ActivateMatch(active, validSeat, time.Now()); err == nil {
		t.Fatal("ActivateMatch accepted an already active match")
	}
}
