package match

import (
	"errors"
	"testing"
	"time"
)

func activeTestMatch(t *testing.T) (Match, PlayerSeat, PlayerSeat, time.Time) {
	t.Helper()
	control, err := ParseTimeControl("3+2")
	if err != nil {
		t.Fatal(err)
	}
	game, white, err := NewMatch("match-1", "invite-digest", "white-seat", control)
	if err != nil {
		t.Fatal(err)
	}
	black := PlayerSeat{ID: "black-seat", MatchID: game.ID, Color: ColorBlack}
	started := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	game, err = ActivateMatch(game, black, started)
	if err != nil {
		t.Fatal(err)
	}
	return game, white, black, started
}

func TestApplyMoveCommitsLegalMoveAndClockForCurrentSeat(t *testing.T) {
	game, white, _, started := activeTestMatch(t)
	updated, accepted, err := ApplyMove(game, white, game.Version, "e2", "e4", "", started.Add(10*time.Second))
	if err != nil {
		t.Fatalf("ApplyMove: %v", err)
	}
	if accepted.UCI != "e2e4" || accepted.PlayerSeatID != white.ID {
		t.Fatalf("accepted move = %+v", accepted)
	}
	if updated.Position == game.Position || updated.Version != game.Version+1 || updated.ClockActiveColor != ColorBlack {
		t.Fatalf("updated match = %+v", updated)
	}
	if updated.WhiteRemaining != 2*time.Minute+52*time.Second || updated.BlackRemaining != 3*time.Minute {
		t.Fatalf("clock after move = %s/%s", updated.WhiteRemaining, updated.BlackRemaining)
	}
}

func TestApplyMoveRejectsWrongSeatStaleVersionAndIllegalMove(t *testing.T) {
	game, white, black, started := activeTestMatch(t)
	if _, _, err := ApplyMove(game, black, game.Version, "e7", "e5", "", started.Add(time.Second)); !errors.Is(err, ErrWrongTurn) {
		t.Fatalf("wrong-seat error = %v, want ErrWrongTurn", err)
	}
	if _, _, err := ApplyMove(game, white, game.Version-1, "e2", "e4", "", started.Add(time.Second)); !errors.Is(err, ErrStaleVersion) {
		t.Fatalf("stale-version error = %v, want ErrStaleVersion", err)
	}
	if _, _, err := ApplyMove(game, white, game.Version, "e2", "e5", "", started.Add(time.Second)); !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("illegal-move error = %v, want ErrIllegalMove", err)
	}
}

func TestApplyMoveRejectsMoveAfterClockExpires(t *testing.T) {
	game, white, _, started := activeTestMatch(t)
	_, _, err := ApplyMove(game, white, game.Version, "e2", "e4", "", started.Add(3*time.Minute))
	if !errors.Is(err, ErrFlagFell) {
		t.Fatalf("expired-move error = %v, want ErrFlagFell", err)
	}
}
