package clock

import (
	"fmt"
	"time"
)

func ApplyMove(state State, increment time.Duration, acceptedAt time.Time) (State, error) {
	if state.StartedAt.IsZero() || (state.ActiveColor != "white" && state.ActiveColor != "black") {
		return State{}, fmt.Errorf("running clock state is invalid")
	}
	if increment < 0 || acceptedAt.Before(state.StartedAt) {
		return State{}, fmt.Errorf("move timestamp or increment is invalid")
	}
	acceptedAt = acceptedAt.UTC()
	elapsed := acceptedAt.Sub(state.StartedAt)
	if state.ActiveColor == "white" {
		if elapsed >= state.WhiteRemaining {
			return State{}, ErrFlagFell
		}
		state.WhiteRemaining = state.WhiteRemaining - elapsed + increment
		state.ActiveColor = "black"
	} else {
		if elapsed >= state.BlackRemaining {
			return State{}, ErrFlagFell
		}
		state.BlackRemaining = state.BlackRemaining - elapsed + increment
		state.ActiveColor = "white"
	}
	state.StartedAt = acceptedAt
	return state, nil
}
