package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"chess_clone/internal/chess"
	"chess_clone/internal/match"
)

type MatchStore struct {
	Pool *pgxpool.Pool
}

func NewMatchStore(pool *pgxpool.Pool) *MatchStore {
	return &MatchStore{Pool: pool}
}

func (s *MatchStore) Create(ctx context.Context, game match.Match, white match.PlayerSeat) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin match creation: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `INSERT INTO matches (
		id, invite_token_digest, status, position, time_control, initial_time_ms, increment_ms,
		white_remaining_ms, black_remaining_ms, clock_active_color, clock_started_at, version
	) VALUES ($1, $2, 'waiting', $3, $4, $5, $6, $5, $5, NULL, NULL, 0)`,
		game.ID, []byte(game.InviteTokenDigest), game.Position, game.TimeControl,
		game.InitialTime.Milliseconds(), game.Increment.Milliseconds()); err != nil {
		return fmt.Errorf("insert match: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO player_seats (id, match_id, color, recovery_credential_digest)
		VALUES ($1, $2, 'white', $3)`, white.ID, game.ID, white.RecoveryCredentialDigest); err != nil {
		return fmt.Errorf("insert White seat: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE matches SET white_player_id = $2 WHERE id = $1`, game.ID, white.ID); err != nil {
		return fmt.Errorf("assign White seat: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit match creation: %w", err)
	}
	return nil
}

func (s *MatchStore) Join(ctx context.Context, inviteDigest string, black match.PlayerSeat) (match.Match, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return match.Match{}, fmt.Errorf("begin match join: %w", err)
	}
	defer tx.Rollback(ctx)

	var game match.Match
	var status string
	var initialMS, incrementMS, whiteMS, blackMS int64
	err = tx.QueryRow(ctx, `SELECT id, status, white_player_id, position, time_control, initial_time_ms, increment_ms,
		white_remaining_ms, black_remaining_ms, version
		FROM matches WHERE invite_token_digest = $1 FOR UPDATE`, []byte(inviteDigest)).Scan(
		&game.ID, &status, &game.WhitePlayerID, &game.Position, &game.TimeControl, &initialMS, &incrementMS,
		&whiteMS, &blackMS, &game.Version,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return match.Match{}, match.ErrInvitationNotFound
	}
	if err != nil {
		return match.Match{}, fmt.Errorf("load invited match: %w", err)
	}
	if status != string(match.StatusWaiting) {
		return match.Match{}, match.ErrMatchUnavailable
	}
	game.Status = match.StatusWaiting
	game.InitialTime = time.Duration(initialMS) * time.Millisecond
	game.Increment = time.Duration(incrementMS) * time.Millisecond
	game.WhiteRemaining = time.Duration(whiteMS) * time.Millisecond
	game.BlackRemaining = time.Duration(blackMS) * time.Millisecond
	black.MatchID = game.ID

	var startedAt time.Time
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&startedAt); err != nil {
		return match.Match{}, fmt.Errorf("read activation timestamp: %w", err)
	}
	active, err := match.ActivateMatch(game, black, startedAt)
	if err != nil {
		return match.Match{}, err
	}
	active.ClockAsOf = startedAt
	if _, err := tx.Exec(ctx, `INSERT INTO player_seats (id, match_id, color, recovery_credential_digest)
		VALUES ($1, $2, 'black', $3)`, black.ID, game.ID, black.RecoveryCredentialDigest); err != nil {
		return match.Match{}, fmt.Errorf("insert Black seat: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE matches SET status = 'active', black_player_id = $2,
		clock_active_color = 'white', clock_started_at = $3, version = $4, updated_at = $3
		WHERE id = $1`, active.ID, black.ID, startedAt, active.Version); err != nil {
		return match.Match{}, fmt.Errorf("activate match: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return match.Match{}, fmt.Errorf("commit match join: %w", err)
	}
	return active, nil
}

func (s *MatchStore) ReadForSeat(ctx context.Context, matchID string, credentialDigest []byte) (match.Match, match.Color, error) {
	var game match.Match
	var color, status string
	var activeColor pgtype.Text
	var startedAt pgtype.Timestamptz
	var initialMS, incrementMS, whiteMS, blackMS int64
	err := s.Pool.QueryRow(ctx, `SELECT m.id, m.status, m.white_player_id, m.black_player_id,
		m.position, m.time_control, m.initial_time_ms, m.increment_ms,
		CASE WHEN m.clock_active_color = 'white' THEN GREATEST(m.white_remaining_ms -
			FLOOR(EXTRACT(EPOCH FROM (instant.value - m.clock_started_at)) * 1000)::bigint, 0)
			ELSE m.white_remaining_ms END,
		CASE WHEN m.clock_active_color = 'black' THEN GREATEST(m.black_remaining_ms -
			FLOOR(EXTRACT(EPOCH FROM (instant.value - m.clock_started_at)) * 1000)::bigint, 0)
			ELSE m.black_remaining_ms END,
		m.clock_active_color, m.clock_started_at, instant.value, m.version, s.color,
		COALESCE((SELECT gives_check FROM moves WHERE match_id = m.id ORDER BY sequence DESC LIMIT 1), false)
		FROM matches m
		JOIN player_seats s ON s.match_id = m.id
		CROSS JOIN (SELECT clock_timestamp() AS value) AS instant
		WHERE m.id = $1 AND s.recovery_credential_digest = $2`, matchID, credentialDigest).Scan(
		&game.ID, &status, &game.WhitePlayerID, &game.BlackPlayerID, &game.Position,
		&game.TimeControl, &initialMS, &incrementMS, &whiteMS, &blackMS,
		&activeColor, &startedAt, &game.ClockAsOf, &game.Version, &color, &game.InCheck,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return match.Match{}, "", match.ErrInvalidSeat
	}
	if err != nil {
		return match.Match{}, "", fmt.Errorf("read match for seat: %w", err)
	}
	game.Status = match.Status(status)
	game.InitialTime = time.Duration(initialMS) * time.Millisecond
	game.Increment = time.Duration(incrementMS) * time.Millisecond
	game.WhiteRemaining = time.Duration(whiteMS) * time.Millisecond
	game.BlackRemaining = time.Duration(blackMS) * time.Millisecond
	if activeColor.Valid {
		game.ClockActiveColor = match.Color(activeColor.String)
	}
	if startedAt.Valid {
		game.ClockStartedAt = &startedAt.Time
	}
	if game.Status == match.StatusActive {
		history, err := loadMoveHistory(ctx, s.Pool, game.ID)
		if err != nil {
			return match.Match{}, "", err
		}
		rules, err := chess.NewGameFromMoves(history)
		if err != nil || rules.FEN() != game.Position {
			return match.Match{}, "", fmt.Errorf("persisted move history does not match current position")
		}
		game.EligibleDraws = rules.EligibleDraws()
	}
	return game, match.Color(color), nil
}

func (s *MatchStore) ApplyMove(ctx context.Context, matchID string, credentialDigest []byte, expectedVersion int64, from, to, promotion string) (match.Match, match.Move, match.Color, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return match.Match{}, match.Move{}, "", fmt.Errorf("begin move transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var game match.Match
	var seat match.PlayerSeat
	var status, color string
	var activeColor pgtype.Text
	var startedAt pgtype.Timestamptz
	var initialMS, incrementMS, whiteMS, blackMS int64
	err = tx.QueryRow(ctx, `SELECT m.id, m.status, m.white_player_id, m.black_player_id,
		m.position, m.time_control, m.initial_time_ms, m.increment_ms,
		m.white_remaining_ms, m.black_remaining_ms, m.clock_active_color,
		m.clock_started_at, m.version, s.id, s.color
		FROM matches m JOIN player_seats s ON s.match_id = m.id
		WHERE m.id = $1 AND s.recovery_credential_digest = $2
		FOR UPDATE OF m`, matchID, credentialDigest).Scan(
		&game.ID, &status, &game.WhitePlayerID, &game.BlackPlayerID, &game.Position,
		&game.TimeControl, &initialMS, &incrementMS, &whiteMS, &blackMS,
		&activeColor, &startedAt, &game.Version, &seat.ID, &color,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return match.Match{}, match.Move{}, "", match.ErrInvalidSeat
	}
	if err != nil {
		return match.Match{}, match.Move{}, "", fmt.Errorf("load match for move: %w", err)
	}
	game.Status = match.Status(status)
	game.InitialTime = time.Duration(initialMS) * time.Millisecond
	game.Increment = time.Duration(incrementMS) * time.Millisecond
	game.WhiteRemaining = time.Duration(whiteMS) * time.Millisecond
	game.BlackRemaining = time.Duration(blackMS) * time.Millisecond
	if activeColor.Valid {
		game.ClockActiveColor = match.Color(activeColor.String)
	}
	if startedAt.Valid {
		game.ClockStartedAt = &startedAt.Time
	}
	seat.MatchID = game.ID
	seat.Color = match.Color(color)
	var acceptedAt time.Time
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&acceptedAt); err != nil {
		return match.Match{}, match.Move{}, "", fmt.Errorf("read move acceptance time: %w", err)
	}
	updated, accepted, err := match.ApplyMove(game, seat, expectedVersion, from, to, promotion, acceptedAt)
	if err != nil {
		return match.Match{}, match.Move{}, "", err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO moves (
		match_id, sequence, player_seat_id, from_square, to_square, promotion, notation,
		position_after, accepted_at, gives_check
	) VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8, $9, $10)`,
		accepted.MatchID, accepted.Sequence, accepted.PlayerSeatID, accepted.From, accepted.To,
		accepted.Promotion, accepted.Notation, accepted.PositionAfter, accepted.AcceptedAt, accepted.GivesCheck); err != nil {
		return match.Match{}, match.Move{}, "", fmt.Errorf("insert accepted move: %w", err)
	}
	history, err := loadMoveHistory(ctx, tx, updated.ID)
	if err != nil {
		return match.Match{}, match.Move{}, "", err
	}
	rules, err := chess.NewGameFromMoves(history)
	if err != nil || rules.FEN() != updated.Position {
		return match.Match{}, match.Move{}, "", fmt.Errorf("persisted move history does not match resulting position")
	}
	updated.EligibleDraws = rules.EligibleDraws()
	var result any
	if updated.Result != "" {
		data, marshalErr := json.Marshal(map[string]string{"outcome": updated.Result, "reason": updated.TerminationReason})
		if marshalErr != nil {
			return match.Match{}, match.Move{}, "", fmt.Errorf("encode match result: %w", marshalErr)
		}
		result = string(data)
	}
	var active any
	var clockStarted any
	if updated.Status == match.StatusActive {
		active = string(updated.ClockActiveColor)
		clockStarted = updated.ClockStartedAt
	}
	if _, err := tx.Exec(ctx, `UPDATE matches SET status = $2, position = $3,
		white_remaining_ms = $4, black_remaining_ms = $5, clock_active_color = $6,
		clock_started_at = $7, version = $8, result = $9::jsonb, updated_at = $10
		WHERE id = $1`, updated.ID, updated.Status, updated.Position,
		updated.WhiteRemaining.Milliseconds(), updated.BlackRemaining.Milliseconds(), active,
		clockStarted, updated.Version, result, accepted.AcceptedAt); err != nil {
		return match.Match{}, match.Move{}, "", fmt.Errorf("update match after move: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return match.Match{}, match.Move{}, "", fmt.Errorf("commit accepted move: %w", err)
	}
	return updated, accepted, seat.Color, nil
}

type moveHistoryQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func loadMoveHistory(ctx context.Context, queryer moveHistoryQuerier, matchID string) ([]string, error) {
	rows, err := queryer.Query(ctx, `SELECT from_square || to_square || COALESCE(promotion, '')
		FROM moves WHERE match_id = $1 ORDER BY sequence`, matchID)
	if err != nil {
		return nil, fmt.Errorf("load move history: %w", err)
	}
	defer rows.Close()
	var history []string
	for rows.Next() {
		var move string
		if err := rows.Scan(&move); err != nil {
			return nil, fmt.Errorf("scan move history: %w", err)
		}
		history = append(history, move)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate move history: %w", err)
	}
	return history, nil
}

func (s *MatchStore) FindExpired(ctx context.Context, _ time.Time) ([]string, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id FROM matches
		WHERE status = 'active' AND clock_active_color IS NOT NULL AND clock_started_at IS NOT NULL
		AND clock_started_at + CASE WHEN clock_active_color = 'white' THEN white_remaining_ms
			ELSE black_remaining_ms END * interval '1 millisecond' <= clock_timestamp()`)
	if err != nil {
		return nil, fmt.Errorf("find expired clocks: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan expired match: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expired matches: %w", err)
	}
	return ids, nil
}

func (s *MatchStore) RecordTimeout(ctx context.Context, matchID string, _ time.Time) ([][]byte, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin timeout transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	var status string
	var position string
	var activeColor pgtype.Text
	var startedAt pgtype.Timestamptz
	var whiteMS, blackMS int64
	if err := tx.QueryRow(ctx, `SELECT status, position, clock_active_color, clock_started_at,
		white_remaining_ms, black_remaining_ms FROM matches WHERE id = $1 FOR UPDATE`, matchID).
		Scan(&status, &position, &activeColor, &startedAt, &whiteMS, &blackMS); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("lock expired match: %w", err)
	}
	if status != string(match.StatusActive) || !activeColor.Valid || !startedAt.Valid {
		return nil, nil
	}
	var eventTime time.Time
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&eventTime); err != nil {
		return nil, fmt.Errorf("read timeout adjudication time: %w", err)
	}
	elapsed := eventTime.Sub(startedAt.Time)
	remaining := whiteMS
	if activeColor.String == "black" {
		remaining = blackMS
	}
	if elapsed < time.Duration(remaining)*time.Millisecond {
		return nil, tx.Commit(ctx)
	}

	rules, err := chess.NewGame(position)
	if err != nil {
		return nil, fmt.Errorf("load position for timeout adjudication: %w", err)
	}
	outcome := "1-0"
	if activeColor.String == "white" {
		outcome = "0-1"
	}
	reason := "timeout"
	if rules.CannotPossiblyCheckmate() {
		outcome = "1/2-1/2"
		reason = "timeout_insufficient_material"
	}
	resultJSON, err := json.Marshal(map[string]string{"outcome": outcome, "reason": reason})
	if err != nil {
		return nil, fmt.Errorf("encode timeout result: %w", err)
	}
	if activeColor.String == "white" {
		whiteMS = 0
	} else {
		blackMS = 0
	}
	var version int64
	if err := tx.QueryRow(ctx, `UPDATE matches SET status = 'completed', white_remaining_ms = $2,
		black_remaining_ms = $3, clock_active_color = NULL, clock_started_at = NULL,
		version = version + 1, result = $4::jsonb, updated_at = $5
		WHERE id = $1 RETURNING version`, matchID, whiteMS, blackMS, string(resultJSON), eventTime).Scan(&version); err != nil {
		return nil, fmt.Errorf("record timeout result: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit timeout result: %w", err)
	}
	clockSnapshot := map[string]any{
		"white_remaining_ms": whiteMS, "black_remaining_ms": blackMS,
		"active_color": nil, "as_of": eventTime.UTC().Format(time.RFC3339Nano),
	}
	base := map[string]any{
		"match_id": matchID, "version": version, "result": outcome,
		"reason": reason, "clock": clockSnapshot,
	}
	clockEvent := make(map[string]any, len(base)+1)
	for key, value := range base {
		clockEvent[key] = value
	}
	clockEvent["type"] = "clock_expired"
	endedEvent := make(map[string]any, len(base)+1)
	for key, value := range base {
		endedEvent[key] = value
	}
	endedEvent["type"] = "game_ended"
	first, _ := json.Marshal(clockEvent)
	second, _ := json.Marshal(endedEvent)
	return [][]byte{first, second}, nil
}

func (s *MatchStore) Resign(ctx context.Context, matchID string, credentialDigest []byte) (match.Match, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return match.Match{}, fmt.Errorf("begin resignation: %w", err)
	}
	defer tx.Rollback(ctx)
	game, seat, eventTime, err := s.lockMatchForSeat(ctx, tx, matchID, credentialDigest)
	if err != nil {
		return match.Match{}, err
	}
	ended, err := match.Resign(game, seat, eventTime)
	if err != nil {
		return match.Match{}, err
	}
	if err := persistMatchState(ctx, tx, ended); err != nil {
		return match.Match{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return match.Match{}, fmt.Errorf("commit resignation: %w", err)
	}
	return ended, nil
}

func (s *MatchStore) CreateDrawOffer(ctx context.Context, matchID string, credentialDigest []byte, offerID string) (match.DrawOffer, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return match.DrawOffer{}, fmt.Errorf("begin draw offer: %w", err)
	}
	defer tx.Rollback(ctx)
	game, seat, eventTime, err := s.lockMatchForSeat(ctx, tx, matchID, credentialDigest)
	if err != nil {
		return match.DrawOffer{}, err
	}
	offer, err := match.CreateDrawOffer(game, seat, offerID, eventTime)
	if err != nil {
		return match.DrawOffer{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO draw_offers (id, match_id, offering_seat_id, status, created_at)
		VALUES ($1, $2, $3, 'pending', $4)`, offer.ID, offer.MatchID, offer.OfferingSeatID, eventTime); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "one_pending_draw_offer_per_match" {
			return match.DrawOffer{}, match.ErrDrawOfferPending
		}
		return match.DrawOffer{}, fmt.Errorf("insert draw offer: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return match.DrawOffer{}, fmt.Errorf("commit draw offer: %w", err)
	}
	return offer, nil
}

func (s *MatchStore) ResolveDrawOffer(ctx context.Context, matchID string, credentialDigest []byte, offerID string, accept bool) (match.Match, match.DrawOffer, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return match.Match{}, match.DrawOffer{}, fmt.Errorf("begin draw resolution: %w", err)
	}
	defer tx.Rollback(ctx)
	game, seat, eventTime, err := s.lockMatchForSeat(ctx, tx, matchID, credentialDigest)
	if err != nil {
		return match.Match{}, match.DrawOffer{}, err
	}
	var offer match.DrawOffer
	var status string
	err = tx.QueryRow(ctx, `SELECT id, match_id, offering_seat_id, status, created_at, resolved_at
		FROM draw_offers WHERE id = $1 AND match_id = $2 FOR UPDATE`, offerID, matchID).
		Scan(&offer.ID, &offer.MatchID, &offer.OfferingSeatID, &status, &offer.CreatedAt, &offer.ResolvedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return match.Match{}, match.DrawOffer{}, match.ErrDrawOfferNotFound
	}
	if err != nil {
		return match.Match{}, match.DrawOffer{}, fmt.Errorf("load draw offer: %w", err)
	}
	offer.Status = match.DrawOfferStatus(status)
	updated, resolved, err := match.ResolveDrawOffer(game, offer, seat, accept, eventTime)
	if err != nil {
		return match.Match{}, match.DrawOffer{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE draw_offers SET status = $2, resolved_at = $3 WHERE id = $1`, resolved.ID, resolved.Status, resolved.ResolvedAt); err != nil {
		return match.Match{}, match.DrawOffer{}, fmt.Errorf("update draw offer: %w", err)
	}
	if updated.Status == match.StatusCompleted {
		if err := persistMatchState(ctx, tx, updated); err != nil {
			return match.Match{}, match.DrawOffer{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return match.Match{}, match.DrawOffer{}, fmt.Errorf("commit draw resolution: %w", err)
	}
	return updated, resolved, nil
}

func (s *MatchStore) ClaimDraw(ctx context.Context, matchID string, credentialDigest []byte, method string) (match.Match, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return match.Match{}, fmt.Errorf("begin draw claim: %w", err)
	}
	defer tx.Rollback(ctx)
	game, seat, eventTime, err := s.lockMatchForSeat(ctx, tx, matchID, credentialDigest)
	if err != nil {
		return match.Match{}, err
	}
	rows, err := tx.Query(ctx, `SELECT from_square || to_square || COALESCE(promotion, '')
		FROM moves WHERE match_id = $1 ORDER BY sequence`, matchID)
	if err != nil {
		return match.Match{}, fmt.Errorf("load move history for draw claim: %w", err)
	}
	var history []string
	for rows.Next() {
		var move string
		if err := rows.Scan(&move); err != nil {
			rows.Close()
			return match.Match{}, fmt.Errorf("scan move history: %w", err)
		}
		history = append(history, move)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return match.Match{}, fmt.Errorf("iterate move history: %w", err)
	}
	rows.Close()
	ended, err := match.ClaimDraw(game, seat, history, method, eventTime)
	if err != nil {
		return match.Match{}, err
	}
	if err := persistMatchState(ctx, tx, ended); err != nil {
		return match.Match{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return match.Match{}, fmt.Errorf("commit draw claim: %w", err)
	}
	return ended, nil
}

func (s *MatchStore) lockMatchForSeat(ctx context.Context, tx pgx.Tx, matchID string, credentialDigest []byte) (match.Match, match.PlayerSeat, time.Time, error) {
	var game match.Match
	var seat match.PlayerSeat
	var status, color string
	var activeColor pgtype.Text
	var startedAt pgtype.Timestamptz
	var initialMS, incrementMS, whiteMS, blackMS int64
	var eventTime time.Time
	err := tx.QueryRow(ctx, `SELECT m.id, m.status, m.white_player_id, m.black_player_id, m.position,
		m.time_control, m.initial_time_ms, m.increment_ms, m.white_remaining_ms, m.black_remaining_ms,
		m.clock_active_color, m.clock_started_at, m.version, s.id, s.color,
		COALESCE((SELECT gives_check FROM moves WHERE match_id = m.id ORDER BY sequence DESC LIMIT 1), false)
		FROM matches m JOIN player_seats s ON s.match_id = m.id
		WHERE m.id = $1 AND s.recovery_credential_digest = $2 FOR UPDATE OF m`, matchID, credentialDigest).Scan(
		&game.ID, &status, &game.WhitePlayerID, &game.BlackPlayerID, &game.Position,
		&game.TimeControl, &initialMS, &incrementMS, &whiteMS, &blackMS,
		&activeColor, &startedAt, &game.Version, &seat.ID, &color, &game.InCheck,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return match.Match{}, match.PlayerSeat{}, time.Time{}, match.ErrInvalidSeat
	}
	if err != nil {
		return match.Match{}, match.PlayerSeat{}, time.Time{}, fmt.Errorf("lock match for seat action: %w", err)
	}
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&eventTime); err != nil {
		return match.Match{}, match.PlayerSeat{}, time.Time{}, fmt.Errorf("read serialized action time: %w", err)
	}
	game.Status = match.Status(status)
	game.InitialTime = time.Duration(initialMS) * time.Millisecond
	game.Increment = time.Duration(incrementMS) * time.Millisecond
	game.WhiteRemaining = time.Duration(whiteMS) * time.Millisecond
	game.BlackRemaining = time.Duration(blackMS) * time.Millisecond
	game.ClockAsOf = eventTime
	if activeColor.Valid {
		game.ClockActiveColor = match.Color(activeColor.String)
	}
	if startedAt.Valid {
		game.ClockStartedAt = &startedAt.Time
	}
	seat.MatchID = game.ID
	seat.Color = match.Color(color)
	return game, seat, eventTime, nil
}

func persistMatchState(ctx context.Context, tx pgx.Tx, game match.Match) error {
	resultJSON, err := json.Marshal(map[string]string{"outcome": game.Result, "reason": game.TerminationReason})
	if err != nil {
		return fmt.Errorf("encode terminal result: %w", err)
	}
	var active any
	var startedAt any
	if game.Status == match.StatusActive {
		active = string(game.ClockActiveColor)
		startedAt = game.ClockStartedAt
	}
	if _, err := tx.Exec(ctx, `UPDATE matches SET status = $2, position = $3,
		white_remaining_ms = $4, black_remaining_ms = $5, clock_active_color = $6,
		clock_started_at = $7, version = $8, result = $9::jsonb, updated_at = $10
		WHERE id = $1`, game.ID, game.Status, game.Position,
		game.WhiteRemaining.Milliseconds(), game.BlackRemaining.Milliseconds(), active,
		startedAt, game.Version, string(resultJSON), game.ClockAsOf); err != nil {
		return fmt.Errorf("persist terminal match: %w", err)
	}
	return nil
}
