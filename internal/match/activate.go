package match

import (
	"time"

	"chess_clone/internal/clock"
)

func ActivateMatch(game Match, black PlayerSeat, startedAt time.Time) (Match, error) {
	if game.Status != StatusWaiting {
		return Match{}, ErrMatchUnavailable
	}
	if black.ID == "" || black.MatchID != game.ID || black.Color != ColorBlack {
		return Match{}, ErrInvalidSeat
	}
	clockState, err := clock.Start(game.InitialTime, startedAt)
	if err != nil {
		return Match{}, err
	}
	game.Status = StatusActive
	game.BlackPlayerID = black.ID
	game.ClockActiveColor = ColorWhite
	game.ClockStartedAt = &clockState.StartedAt
	game.WhiteRemaining = clockState.WhiteRemaining
	game.BlackRemaining = clockState.BlackRemaining
	game.Version++
	return game, nil
}
