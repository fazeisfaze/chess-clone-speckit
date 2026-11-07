package match

import "testing"

func TestParseTimeControlAcceptsOnlySupportedPresets(t *testing.T) {
	cases := []struct {
		code        string
		initialMS   int64
		incrementMS int64
	}{
		{code: "3+2", initialMS: 180_000, incrementMS: 2_000},
		{code: "5+0", initialMS: 300_000, incrementMS: 0},
		{code: "10+0", initialMS: 600_000, incrementMS: 0},
	}
	for _, test := range cases {
		t.Run(test.code, func(t *testing.T) {
			got, err := ParseTimeControl(test.code)
			if err != nil {
				t.Fatalf("ParseTimeControl(%q): %v", test.code, err)
			}
			if got.Initial.Milliseconds() != test.initialMS || got.Increment.Milliseconds() != test.incrementMS {
				t.Fatalf("control = %+v, want %dms + %dms", got, test.initialMS, test.incrementMS)
			}
		})
	}
	if _, err := ParseTimeControl("1+0"); err == nil {
		t.Fatal("ParseTimeControl accepted an unsupported preset")
	}
}

func TestNewMatchCreatesWaitingMatchWithWhiteSeat(t *testing.T) {
	control, err := ParseTimeControl("5+0")
	if err != nil {
		t.Fatal(err)
	}
	game, white, err := NewMatch("match-1", "invite-digest", "seat-white", control)
	if err != nil {
		t.Fatalf("NewMatch: %v", err)
	}
	if game.Status != StatusWaiting || game.WhitePlayerID != white.ID || white.Color != ColorWhite {
		t.Fatalf("match=%+v white=%+v", game, white)
	}
	if game.ClockActiveColor != "" || game.ClockStartedAt != nil {
		t.Fatalf("waiting match must have a stopped clock: %+v", game)
	}
	if game.WhiteRemaining.Milliseconds() != 300_000 || game.BlackRemaining.Milliseconds() != 300_000 {
		t.Fatalf("initial times = %s/%s", game.WhiteRemaining, game.BlackRemaining)
	}
}
