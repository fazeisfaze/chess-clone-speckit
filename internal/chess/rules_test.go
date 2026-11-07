package chess

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type conformanceFixture struct {
	Name         string   `json:"name"`
	FEN          string   `json:"fen"`
	LegalMoves   []string `json:"legal_moves"`
	IllegalMoves []string `json:"illegal_moves"`
	Sequence     []string `json:"sequence"`
	Outcome      string   `json:"outcome"`
	Method       string   `json:"method"`
}

func TestProjectConformanceFixtures(t *testing.T) {
	fixturePath := filepath.Join("testdata", "conformance_positions.json")
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []conformanceFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatalf("decode fixtures: %v", err)
	}
	if len(fixtures) < 7 {
		t.Fatalf("fixture count = %d, want at least 7", len(fixtures))
	}

	for _, fixture := range fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			for _, move := range fixture.LegalMoves {
				game, err := NewGame(fixture.FEN)
				if err != nil {
					t.Fatalf("NewGame: %v", err)
				}
				if _, err := game.PlayUCI(move); err != nil {
					t.Errorf("legal move %s rejected: %v", move, err)
				}
			}
			for _, move := range fixture.IllegalMoves {
				game, err := NewGame(fixture.FEN)
				if err != nil {
					t.Fatalf("NewGame: %v", err)
				}
				if _, err := game.PlayUCI(move); err == nil {
					t.Errorf("illegal move %s was accepted", move)
				}
			}
			if fixture.Outcome == "" {
				return
			}
			game, err := NewGame(fixture.FEN)
			if err != nil {
				t.Fatalf("NewGame: %v", err)
			}
			for _, move := range fixture.Sequence {
				if _, err := game.PlayUCI(move); err != nil {
					t.Fatalf("PlayUCI(%s): %v", move, err)
				}
			}
			if got := game.Outcome(); got != fixture.Outcome {
				t.Errorf("outcome = %q, want %q", got, fixture.Outcome)
			}
			if got := game.Method(); got != fixture.Method {
				t.Errorf("method = %q, want %q", got, fixture.Method)
			}
		})
	}
}

func TestPlayUCIResultReportsNonTerminalCheck(t *testing.T) {
	game, err := NewGame("4k3/8/8/8/8/8/4R3/K7 w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	_, notation, inCheck, err := game.PlayUCIResult("e2e1")
	if err != nil {
		t.Fatalf("PlayUCIResult: %v", err)
	}
	if !inCheck || notation != "Re1+" {
		t.Fatalf("check result = %q/%v, want Re1+/true", notation, inCheck)
	}
	if game.Outcome() != "*" {
		t.Fatalf("check move outcome = %q, want ongoing game", game.Outcome())
	}
}
