package match

import (
	"fmt"
	"time"

	"chess_clone/internal/chess"
)

var ErrDrawNotClaimable = fmt.Errorf("draw cannot be claimed in this position")

type DrawOfferStatus string

const (
	DrawOfferPending  DrawOfferStatus = "pending"
	DrawOfferAccepted DrawOfferStatus = "accepted"
	DrawOfferDeclined DrawOfferStatus = "declined"
)

type DrawOffer struct {
	ID             string
	MatchID        string
	OfferingSeatID string
	Status         DrawOfferStatus
	CreatedAt      time.Time
	ResolvedAt     *time.Time
}

func Resign(game Match, seat PlayerSeat, at time.Time) (Match, error) {
	if game.Status != StatusActive {
		return Match{}, ErrMatchUnavailable
	}
	if !ownsSeat(game, seat) {
		return Match{}, ErrInvalidSeat
	}
	stopped, err := stopClock(game, at)
	if err != nil {
		return Match{}, err
	}
	stopped.Status = StatusCompleted
	stopped.TerminationReason = "resignation"
	if seat.Color == ColorWhite {
		stopped.Result = "0-1"
	} else {
		stopped.Result = "1-0"
	}
	stopped.Version++
	return stopped, nil
}

func CreateDrawOffer(game Match, seat PlayerSeat, id string, at time.Time) (DrawOffer, error) {
	if game.Status != StatusActive {
		return DrawOffer{}, ErrMatchUnavailable
	}
	if id == "" || !ownsSeat(game, seat) {
		return DrawOffer{}, ErrInvalidSeat
	}
	return DrawOffer{ID: id, MatchID: game.ID, OfferingSeatID: seat.ID, Status: DrawOfferPending, CreatedAt: at.UTC()}, nil
}

func ResolveDrawOffer(game Match, offer DrawOffer, seat PlayerSeat, accept bool, at time.Time) (Match, DrawOffer, error) {
	if game.Status != StatusActive || offer.MatchID != game.ID || offer.Status != DrawOfferPending {
		return Match{}, offer, ErrMatchUnavailable
	}
	if !ownsSeat(game, seat) || offer.OfferingSeatID == seat.ID {
		return Match{}, offer, ErrInvalidSeat
	}
	at = at.UTC()
	offer.ResolvedAt = &at
	if !accept {
		offer.Status = DrawOfferDeclined
		return game, offer, nil
	}
	stopped, err := stopClock(game, at)
	if err != nil {
		return Match{}, offer, err
	}
	stopped.Status = StatusCompleted
	stopped.Result = "1/2-1/2"
	stopped.TerminationReason = "draw_offer"
	stopped.Version++
	offer.Status = DrawOfferAccepted
	return stopped, offer, nil
}

func ClaimDraw(game Match, seat PlayerSeat, history []string, method string, at time.Time) (Match, error) {
	if game.Status != StatusActive {
		return Match{}, ErrMatchUnavailable
	}
	if !ownsSeat(game, seat) {
		return Match{}, ErrInvalidSeat
	}
	position, err := chess.NewGameFromMoves(history)
	if err != nil || position.FEN() != game.Position {
		return Match{}, fmt.Errorf("%w: match history does not match position", ErrDrawNotClaimable)
	}
	if position.Turn() != string(seat.Color) {
		return Match{}, ErrWrongTurn
	}
	if err := position.ClaimDraw(method); err != nil {
		return Match{}, fmt.Errorf("%w: %s", ErrDrawNotClaimable, err)
	}
	stopped, err := stopClock(game, at)
	if err != nil {
		return Match{}, err
	}
	stopped.Status = StatusCompleted
	stopped.Result = "1/2-1/2"
	stopped.TerminationReason = method
	stopped.Version++
	return stopped, nil
}

func ownsSeat(game Match, seat PlayerSeat) bool {
	return seat.MatchID == game.ID && ((seat.Color == ColorWhite && seat.ID == game.WhitePlayerID) || (seat.Color == ColorBlack && seat.ID == game.BlackPlayerID))
}

func stopClock(game Match, at time.Time) (Match, error) {
	if game.ClockStartedAt == nil || (game.ClockActiveColor != ColorWhite && game.ClockActiveColor != ColorBlack) {
		return Match{}, ErrFlagFell
	}
	if at.Before(*game.ClockStartedAt) {
		return Match{}, fmt.Errorf("termination timestamp predates active clock")
	}
	elapsed := at.Sub(*game.ClockStartedAt)
	if game.ClockActiveColor == ColorWhite {
		if elapsed >= game.WhiteRemaining {
			return Match{}, ErrFlagFell
		}
		game.WhiteRemaining -= elapsed
	} else {
		if elapsed >= game.BlackRemaining {
			return Match{}, ErrFlagFell
		}
		game.BlackRemaining -= elapsed
	}
	game.ClockActiveColor = ""
	game.ClockStartedAt = nil
	game.ClockAsOf = at.UTC()
	return game, nil
}
