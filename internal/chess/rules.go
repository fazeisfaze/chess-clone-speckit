package chess

import chesslib "github.com/corentings/chess/v2"

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

func (g *Game) CannotPossiblyCheckmate() bool {
	return g.game.Method() == chesslib.InsufficientMaterial
}
