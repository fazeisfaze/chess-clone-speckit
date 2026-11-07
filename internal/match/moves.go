package match

import (
	"fmt"
	"strings"
	"time"

	"chess_clone/internal/chess"
	"chess_clone/internal/clock"
)

type Move struct {
	MatchID       string
	Sequence      int64
	PlayerSeatID  string
	From          string
	To            string
	Promotion     string
	UCI           string
	Notation      string
	PositionAfter string
	AcceptedAt    time.Time
	GivesCheck    bool
}

func ApplyMove(game Match, seat PlayerSeat, expectedVersion int64, from, to, promotion string, acceptedAt time.Time) (Match, Move, error) {
	if game.Status != StatusActive {
		return Match{}, Move{}, ErrMatchUnavailable
	}
	if expectedVersion != game.Version {
		return Match{}, Move{}, ErrStaleVersion
	}
	if seat.MatchID != game.ID || (seat.ID != game.WhitePlayerID && seat.ID != game.BlackPlayerID) {
		return Match{}, Move{}, ErrInvalidSeat
	}
	if game.ClockActiveColor != seat.Color {
		return Match{}, Move{}, ErrWrongTurn
	}
	if game.ClockStartedAt == nil {
		return Match{}, Move{}, ErrFlagFell
	}
	if len(from) != 2 || len(to) != 2 || from[0] < 'a' || from[0] > 'h' || from[1] < '1' || from[1] > '8' || to[0] < 'a' || to[0] > 'h' || to[1] < '1' || to[1] > '8' {
		return Match{}, Move{}, ErrIllegalMove
	}
	if promotion != "" && (len(promotion) != 1 || !strings.Contains("qrbn", promotion)) {
		return Match{}, Move{}, ErrIllegalMove
	}
	clockState := clock.State{
		WhiteRemaining: game.WhiteRemaining,
		BlackRemaining: game.BlackRemaining,
		ActiveColor:    string(game.ClockActiveColor),
		StartedAt:      *game.ClockStartedAt,
	}
	updatedClock, err := clock.ApplyMove(clockState, game.Increment, acceptedAt)
	if err != nil {
		return Match{}, Move{}, err
	}
	engine, err := chess.NewGame(game.Position)
	if err != nil {
		return Match{}, Move{}, fmt.Errorf("load current chess position: %w", err)
	}
	uci := from + to + promotion
	positionAfter, notation, givesCheck, err := engine.PlayUCIResult(uci)
	if err != nil {
		return Match{}, Move{}, ErrIllegalMove
	}
	move := Move{
		MatchID: game.ID, Sequence: game.Version, PlayerSeatID: seat.ID,
		From: from, To: to, Promotion: promotion, UCI: uci, Notation: notation,
		PositionAfter: positionAfter, AcceptedAt: acceptedAt.UTC(), GivesCheck: givesCheck,
	}
	game.Position = positionAfter
	game.InCheck = givesCheck
	game.WhiteRemaining = updatedClock.WhiteRemaining
	game.BlackRemaining = updatedClock.BlackRemaining
	game.ClockActiveColor = Color(updatedClock.ActiveColor)
	game.ClockStartedAt = &updatedClock.StartedAt
	game.ClockAsOf = acceptedAt.UTC()
	game.Version++
	if engine.Outcome() != "*" {
		game.Status = StatusCompleted
		game.Result = engine.Outcome()
		game.TerminationReason = engine.Method()
		game.ClockActiveColor = ""
		game.ClockStartedAt = nil
	}
	return game, move, nil
}
