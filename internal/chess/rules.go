package chess

import (
	"fmt"
	"strings"

	chesslib "github.com/corentings/chess/v2"
)

type Game struct {
	game *chesslib.Game
}

func NewGame(fen string) (*Game, error) {
	option, err := chesslib.FEN(fen)
	if err != nil {
		return nil, err
	}
	return &Game{game: chesslib.NewGame(option)}, nil
}

func NewGameFromMoves(moves []string) (*Game, error) {
	game, err := NewGame("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if err != nil {
		return nil, err
	}
	for _, move := range moves {
		if _, err := game.PlayUCI(move); err != nil {
			return nil, err
		}
	}
	return game, nil
}

func (g *Game) PlayUCI(move string) (string, error) {
	position, _, _, err := g.PlayUCIResult(move)
	return position, err
}

func (g *Game) PlayUCIResult(move string) (string, string, bool, error) {
	decoded, err := (chesslib.UCINotation{}).Decode(g.game.Position(), move)
	if err != nil {
		return "", "", false, err
	}
	notation := (chesslib.AlgebraicNotation{}).Encode(g.game.Position(), decoded)
	if err := g.game.Move(decoded, nil); err != nil {
		return "", "", false, err
	}
	return g.game.FEN(), notation, len(notation) > 0 && (notation[len(notation)-1] == '+' || notation[len(notation)-1] == '#'), nil
}

func (g *Game) Outcome() string { return g.game.Outcome().String() }

func (g *Game) Method() string { return g.game.Method().String() }

func (g *Game) FEN() string { return g.game.FEN() }

func (g *Game) CannotPossiblyCheckmate() bool {
	return g.game.Method() == chesslib.InsufficientMaterial
}

func (g *Game) EligibleDraws() []string {
	methods := g.game.EligibleDraws()
	eligible := make([]string, 0, len(methods))
	for _, method := range methods {
		eligible = append(eligible, method.String())
	}
	return eligible
}

func (g *Game) Turn() string {
	if strings.Fields(g.game.FEN())[1] == "b" {
		return "black"
	}
	return "white"
}

func (g *Game) ClaimDraw(method string) error {
	var wanted chesslib.Method
	switch method {
	case "ThreefoldRepetition":
		wanted = chesslib.ThreefoldRepetition
	case "FiftyMoveRule":
		wanted = chesslib.FiftyMoveRule
	default:
		return fmt.Errorf("unsupported draw claim %q", method)
	}
	for _, eligible := range g.game.EligibleDraws() {
		if eligible == wanted {
			return g.game.Draw(wanted)
		}
	}
	return fmt.Errorf("draw claim is not eligible")
}
