package match

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"chess_clone/internal/clock"
)

var (
	ErrInvitationNotFound = errors.New("invitation not found")
	ErrMatchUnavailable   = errors.New("match is full or unavailable")
	ErrInvalidSeat        = errors.New("invalid player seat")
	ErrStaleVersion       = errors.New("match version is stale")
	ErrWrongTurn          = errors.New("it is not this player's turn")
	ErrIllegalMove        = errors.New("move is illegal")
	ErrFlagFell           = clock.ErrFlagFell
	ErrDrawOfferNotFound  = errors.New("draw offer not found")
	ErrDrawOfferPending   = errors.New("a draw offer is already pending")
)

type Store interface {
	Create(context.Context, Match, PlayerSeat) error
	Join(context.Context, string, PlayerSeat) (Match, error)
	ReadForSeat(context.Context, string, []byte) (Match, Color, error)
	ApplyMove(context.Context, string, []byte, int64, string, string, string) (Match, Move, Color, error)
	Resign(context.Context, string, []byte) (Match, error)
	CreateDrawOffer(context.Context, string, []byte, string) (DrawOffer, error)
	ResolveDrawOffer(context.Context, string, []byte, string, bool) (Match, DrawOffer, error)
	ClaimDraw(context.Context, string, []byte, string) (Match, error)
}

type Service struct {
	store Store
}

type MatchResult struct {
	Match              Match
	Seat               PlayerSeat
	InviteToken        string
	RecoveryCredential string
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, timeControlCode string) (MatchResult, error) {
	control, err := ParseTimeControl(timeControlCode)
	if err != nil {
		return MatchResult{}, err
	}
	matchID, err := randomToken(18)
	if err != nil {
		return MatchResult{}, fmt.Errorf("generate match ID: %w", err)
	}
	seatID, err := randomToken(18)
	if err != nil {
		return MatchResult{}, fmt.Errorf("generate seat ID: %w", err)
	}
	inviteToken, err := randomToken(32)
	if err != nil {
		return MatchResult{}, fmt.Errorf("generate invitation: %w", err)
	}
	recoveryCredential, err := randomToken(32)
	if err != nil {
		return MatchResult{}, fmt.Errorf("generate recovery credential: %w", err)
	}

	game, white, err := NewMatch(matchID, digest(inviteToken), seatID, control)
	if err != nil {
		return MatchResult{}, err
	}
	white.RecoveryCredentialDigest = digestBytes(recoveryCredential)
	if err := s.store.Create(ctx, game, white); err != nil {
		return MatchResult{}, err
	}
	return MatchResult{Match: game, Seat: white, InviteToken: inviteToken, RecoveryCredential: recoveryCredential}, nil
}

func (s *Service) Join(ctx context.Context, inviteToken string) (MatchResult, error) {
	if inviteToken == "" {
		return MatchResult{}, ErrInvitationNotFound
	}
	seatID, err := randomToken(18)
	if err != nil {
		return MatchResult{}, fmt.Errorf("generate seat ID: %w", err)
	}
	recoveryCredential, err := randomToken(32)
	if err != nil {
		return MatchResult{}, fmt.Errorf("generate recovery credential: %w", err)
	}
	black := PlayerSeat{ID: seatID, Color: ColorBlack, RecoveryCredentialDigest: digestBytes(recoveryCredential)}
	game, err := s.store.Join(ctx, digest(inviteToken), black)
	if err != nil {
		return MatchResult{}, err
	}
	black.MatchID = game.ID
	return MatchResult{Match: game, Seat: black, RecoveryCredential: recoveryCredential}, nil
}

func (s *Service) Read(ctx context.Context, matchID, recoveryCredential string) (MatchResult, error) {
	if matchID == "" || recoveryCredential == "" {
		return MatchResult{}, ErrInvalidSeat
	}
	game, color, err := s.store.ReadForSeat(ctx, matchID, digestBytes(recoveryCredential))
	if err != nil {
		return MatchResult{}, err
	}
	seatID := game.BlackPlayerID
	if color == ColorWhite {
		seatID = game.WhitePlayerID
	}
	return MatchResult{Match: game, Seat: PlayerSeat{ID: seatID, MatchID: game.ID, Color: color}}, nil
}

func (s *Service) ApplyMove(ctx context.Context, matchID, recoveryCredential string, expectedVersion int64, from, to, promotion string) (Match, Move, Color, error) {
	if matchID == "" || recoveryCredential == "" {
		return Match{}, Move{}, "", ErrInvalidSeat
	}
	return s.store.ApplyMove(ctx, matchID, digestBytes(recoveryCredential), expectedVersion, from, to, promotion)
}

func (s *Service) Resign(ctx context.Context, matchID, recoveryCredential string) (Match, error) {
	if matchID == "" || recoveryCredential == "" {
		return Match{}, ErrInvalidSeat
	}
	return s.store.Resign(ctx, matchID, digestBytes(recoveryCredential))
}

func (s *Service) OfferDraw(ctx context.Context, matchID, recoveryCredential string) (DrawOffer, error) {
	if matchID == "" || recoveryCredential == "" {
		return DrawOffer{}, ErrInvalidSeat
	}
	offerID, err := randomToken(18)
	if err != nil {
		return DrawOffer{}, fmt.Errorf("generate draw offer ID: %w", err)
	}
	return s.store.CreateDrawOffer(ctx, matchID, digestBytes(recoveryCredential), offerID)
}

func (s *Service) ResolveDrawOffer(ctx context.Context, matchID, recoveryCredential, offerID string, accept bool) (Match, DrawOffer, error) {
	if matchID == "" || recoveryCredential == "" || offerID == "" {
		return Match{}, DrawOffer{}, ErrInvalidSeat
	}
	return s.store.ResolveDrawOffer(ctx, matchID, digestBytes(recoveryCredential), offerID, accept)
}

func (s *Service) ClaimDraw(ctx context.Context, matchID, recoveryCredential, method string) (Match, error) {
	if matchID == "" || recoveryCredential == "" {
		return Match{}, ErrInvalidSeat
	}
	return s.store.ClaimDraw(ctx, matchID, digestBytes(recoveryCredential), method)
}

func randomToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func digest(value string) string {
	return hex.EncodeToString(digestBytes(value))
}

func digestBytes(value string) []byte {
	sum := sha256.Sum256([]byte(value))
	return sum[:]
}
