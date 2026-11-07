package match

import (
	"fmt"
	"time"
)

type Status string
type Color string

const (
	StatusWaiting   Status = "waiting"
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
	ColorWhite      Color  = "white"
	ColorBlack      Color  = "black"
)

type TimeControl struct {
	Code      string
	Initial   time.Duration
	Increment time.Duration
}

type Match struct {
	ID                string
	InviteTokenDigest string
	Status            Status
	WhitePlayerID     string
	BlackPlayerID     string
	Position          string
	TimeControl       string
	InitialTime       time.Duration
	Increment         time.Duration
	Version           int64
	WhiteRemaining    time.Duration
	BlackRemaining    time.Duration
	ClockActiveColor  Color
	ClockStartedAt    *time.Time
	ClockAsOf         time.Time
	Result            string
	TerminationReason string
	InCheck           bool
	EligibleDraws     []string
}

type PlayerSeat struct {
	ID                       string
	MatchID                  string
	Color                    Color
	RecoveryCredentialDigest []byte
	Connected                bool
}

func ParseTimeControl(code string) (TimeControl, error) {
	switch code {
	case "3+2":
		return TimeControl{Code: code, Initial: 3 * time.Minute, Increment: 2 * time.Second}, nil
	case "5+0":
		return TimeControl{Code: code, Initial: 5 * time.Minute}, nil
	case "10+0":
		return TimeControl{Code: code, Initial: 10 * time.Minute}, nil
	default:
		return TimeControl{}, fmt.Errorf("unsupported time control %q", code)
	}
}

func NewMatch(id, inviteDigest, whiteSeatID string, control TimeControl) (Match, PlayerSeat, error) {
	if id == "" || inviteDigest == "" || whiteSeatID == "" || control.Code == "" || control.Initial <= 0 {
		return Match{}, PlayerSeat{}, fmt.Errorf("match ID, invitation digest, White seat, and time control are required")
	}
	game := Match{
		ID:                id,
		InviteTokenDigest: inviteDigest,
		Status:            StatusWaiting,
		WhitePlayerID:     whiteSeatID,
		Position:          "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		TimeControl:       control.Code,
		InitialTime:       control.Initial,
		Increment:         control.Increment,
		WhiteRemaining:    control.Initial,
		BlackRemaining:    control.Initial,
	}
	white := PlayerSeat{ID: whiteSeatID, MatchID: id, Color: ColorWhite}
	return game, white, nil
}
