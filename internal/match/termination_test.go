package match

import (
	"errors"
	"testing"
	"time"

	"chess_clone/internal/chess"
)

func TestResignEndsMatchForOpponentAndStopsClock(t *testing.T) {
	game, white, black, started := activeTestMatch(t)
	ended, err := Resign(game, black, started.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if ended.Status != StatusCompleted || ended.Result != "1-0" || ended.TerminationReason != "resignation" {
		t.Fatalf("resignation result = %+v", ended)
	}
	if ended.ClockActiveColor != "" || ended.ClockStartedAt != nil || ended.WhitePlayerID != white.ID {
		t.Fatalf("resignation did not stop clock/preserve seats: %+v", ended)
	}
}

func TestDrawOfferCanOnlyBeResolvedByOpponent(t *testing.T) {
	game, white, black, started := activeTestMatch(t)
	offer, err := CreateDrawOffer(game, white, "offer-1", started)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ResolveDrawOffer(game, offer, white, true, started.Add(time.Second)); !errors.Is(err, ErrInvalidSeat) {
		t.Fatalf("offer resolved by offering player: %v", err)
	}
	continued, declined, err := ResolveDrawOffer(game, offer, black, false, started.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if continued.Status != StatusActive || declined.Status != DrawOfferDeclined || continued.Position != game.Position || continued.ClockActiveColor != game.ClockActiveColor {
		t.Fatalf("declined draw changed match: match=%+v offer=%+v", continued, declined)
	}
	ended, accepted, err := ResolveDrawOffer(game, offer, black, true, started.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if ended.Status != StatusCompleted || ended.Result != "1/2-1/2" || accepted.Status != DrawOfferAccepted {
		t.Fatalf("accepted draw = match %+v, offer %+v", ended, accepted)
	}
}

func TestClaimDrawRequiresAvailableRuleAndCurrentTurn(t *testing.T) {
	game, white, black, started := activeTestMatch(t)
	if _, err := ClaimDraw(game, white, nil, "ThreefoldRepetition", started); !errors.Is(err, ErrDrawNotClaimable) {
		t.Fatalf("unavailable claim error = %v", err)
	}
	history := []string{"g1f3", "g8f6", "f3g1", "f6g8", "g1f3", "g8f6", "f3g1", "f6g8"}
	position, err := chess.NewGameFromMoves(history)
	if err != nil {
		t.Fatal(err)
	}
	game.Position = position.FEN()
	if _, err := ClaimDraw(game, black, history, "ThreefoldRepetition", started); !errors.Is(err, ErrWrongTurn) {
		t.Fatalf("wrong-turn claim error = %v", err)
	}
	ended, err := ClaimDraw(game, white, history, "ThreefoldRepetition", started.Add(time.Second))
	if err != nil {
		t.Fatalf("valid repetition claim: %v", err)
	}
	if ended.Status != StatusCompleted || ended.Result != "1/2-1/2" || ended.TerminationReason != "ThreefoldRepetition" {
		t.Fatalf("claimed draw = %+v", ended)
	}
}
