package clock

import (
	"errors"
	"fmt"
	"time"
)

var ErrFlagFell = errors.New("clock expired")

type State struct {
	WhiteRemaining time.Duration
	BlackRemaining time.Duration
	ActiveColor    string
	StartedAt      time.Time
}

func Start(initial time.Duration, startedAt time.Time) (State, error) {
	if initial <= 0 || startedAt.IsZero() {
		return State{}, fmt.Errorf("initial time and start timestamp are required")
	}
	return State{
		WhiteRemaining: initial,
		BlackRemaining: initial,
		ActiveColor:    "white",
		StartedAt:      startedAt.UTC(),
	}, nil
}
