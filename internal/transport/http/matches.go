package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"chess_clone/internal/match"
	"chess_clone/internal/transport/realtime"
)

type matchEndpoints struct {
	service *match.Service
	origin  string
	hub     *realtime.Hub
}

type matchRequest struct {
	TimeControl string `json:"time_control"`
}

type joinRequest struct {
	InviteToken string `json:"invite_token"`
}

type recoveryRequest struct {
	RecoveryCredential string `json:"recovery_credential"`
	KnownVersion       *int64 `json:"known_version,omitempty"`
}

type moveRequest struct {
	ExpectedVersion int64  `json:"expected_version"`
	From            string `json:"from"`
	To              string `json:"to"`
	Promotion       string `json:"promotion"`
}

type clockResponse struct {
	WhiteRemainingMS int64   `json:"white_remaining_ms"`
	BlackRemainingMS int64   `json:"black_remaining_ms"`
	ActiveColor      *string `json:"active_color"`
	AsOf             string  `json:"as_of"`
}

type matchResponse struct {
	MatchID            string        `json:"match_id"`
	Status             string        `json:"status"`
	Color              string        `json:"color"`
	Position           string        `json:"position"`
	InviteURL          string        `json:"invite_url,omitempty"`
	RecoveryCredential string        `json:"recovery_credential,omitempty"`
	Version            int64         `json:"version"`
	TimeControl        string        `json:"time_control"`
	InCheck            bool          `json:"in_check"`
	EligibleDraws      []string      `json:"eligible_draws"`
	Clock              clockResponse `json:"clock"`
}

type moveResponse struct {
	Type string `json:"type,omitempty"`
	Move struct {
		Sequence  int64  `json:"sequence"`
		Color     string `json:"color"`
		From      string `json:"from"`
		To        string `json:"to"`
		Promotion string `json:"promotion,omitempty"`
		Notation  string `json:"notation"`
		Check     bool   `json:"check"`
	} `json:"move"`
	Position      string        `json:"position"`
	NextTurn      string        `json:"next_turn"`
	Version       int64         `json:"version"`
	InCheck       bool          `json:"in_check"`
	EligibleDraws []string      `json:"eligible_draws"`
	Clock         clockResponse `json:"clock"`
	Result        string        `json:"result,omitempty"`
}

type drawDecisionRequest struct {
	Decision string `json:"decision"`
}

type drawClaimRequest struct {
	Method string `json:"method"`
}

func (e matchEndpoints) create(w http.ResponseWriter, r *http.Request) {
	var body matchRequest
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "request body must contain a time_control")
		return
	}
	created, err := e.service.Create(r.Context(), body.TimeControl)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_time_control", "time control must be 3+2, 5+0, or 10+0")
		return
	}
	response := matchResponse{
		MatchID: created.Match.ID, Status: string(created.Match.Status), Color: string(created.Seat.Color),
		Position:           created.Match.Position,
		InviteURL:          strings.TrimRight(e.origin, "/") + "/join#token=" + created.InviteToken,
		RecoveryCredential: created.RecoveryCredential, Version: created.Match.Version,
		TimeControl: created.Match.TimeControl,
		Clock: clockResponse{
			WhiteRemainingMS: created.Match.WhiteRemaining.Milliseconds(),
			BlackRemainingMS: created.Match.BlackRemaining.Milliseconds(),
			ActiveColor:      nil, AsOf: time.Now().UTC().Format(time.RFC3339Nano),
		},
	}
	writeJSON(w, http.StatusCreated, response)
}

func (e matchEndpoints) join(w http.ResponseWriter, r *http.Request) {
	var body joinRequest
	if err := decodeJSON(w, r, &body); err != nil || body.InviteToken == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "invite_token is required")
		return
	}
	joined, err := e.service.Join(r.Context(), body.InviteToken)
	if errors.Is(err, match.ErrInvitationNotFound) {
		writeError(w, http.StatusNotFound, "invitation_not_found", "invitation is invalid or no longer available")
		return
	}
	if errors.Is(err, match.ErrMatchUnavailable) {
		writeError(w, http.StatusConflict, "match_unavailable", "match is full or unavailable")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "match_join_failed", "could not join match")
		return
	}
	activeColor := string(joined.Match.ClockActiveColor)
	response := matchResponse{
		MatchID: joined.Match.ID, Status: string(joined.Match.Status), Color: string(joined.Seat.Color),
		Position: joined.Match.Position, RecoveryCredential: joined.RecoveryCredential,
		Version: joined.Match.Version, TimeControl: joined.Match.TimeControl, EligibleDraws: joined.Match.EligibleDraws,
		Clock: clockResponse{
			WhiteRemainingMS: joined.Match.WhiteRemaining.Milliseconds(),
			BlackRemainingMS: joined.Match.BlackRemaining.Milliseconds(),
			ActiveColor:      &activeColor,
			AsOf:             joined.Match.ClockStartedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		},
	}
	writeJSON(w, http.StatusOK, response)
}

func (e matchEndpoints) get(w http.ResponseWriter, r *http.Request) {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		writeError(w, http.StatusUnauthorized, "unauthorized", "seat authorization is required")
		return
	}
	state, err := e.service.Read(r.Context(), r.PathValue("match_id"), parts[1])
	if errors.Is(err, match.ErrInvalidSeat) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "seat authorization is invalid")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "match_read_failed", "could not read match")
		return
	}
	var activeColor *string
	if state.Match.ClockActiveColor != "" {
		value := string(state.Match.ClockActiveColor)
		activeColor = &value
	}
	asOf := state.Match.ClockAsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	response := matchResponse{
		MatchID: state.Match.ID, Status: string(state.Match.Status), Color: string(state.Seat.Color),
		Position: state.Match.Position, Version: state.Match.Version, TimeControl: state.Match.TimeControl,
		InCheck:       state.Match.InCheck,
		EligibleDraws: state.Match.EligibleDraws,
		Clock: clockResponse{
			WhiteRemainingMS: state.Match.WhiteRemaining.Milliseconds(),
			BlackRemainingMS: state.Match.BlackRemaining.Milliseconds(),
			ActiveColor:      activeColor,
			AsOf:             asOf.UTC().Format(time.RFC3339Nano),
		},
	}
	writeJSON(w, http.StatusOK, response)
}

func (e matchEndpoints) recover(w http.ResponseWriter, r *http.Request) {
	var body recoveryRequest
	if err := decodeJSON(w, r, &body); err != nil || body.RecoveryCredential == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "recovery_credential is required")
		return
	}
	r.Header.Set("Authorization", "Bearer "+body.RecoveryCredential)
	e.get(w, r)
}

func (e matchEndpoints) move(w http.ResponseWriter, r *http.Request) {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		writeError(w, http.StatusUnauthorized, "unauthorized", "seat authorization is required")
		return
	}
	var body moveRequest
	if err := decodeJSON(w, r, &body); err != nil || body.From == "" || body.To == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "expected_version, from, and to are required")
		return
	}
	response, err := e.applyMove(r.Context(), r.PathValue("match_id"), parts[1], body)
	switch {
	case errors.Is(err, match.ErrInvalidSeat):
		writeError(w, http.StatusUnauthorized, "unauthorized", "seat authorization is invalid")
		return
	case errors.Is(err, match.ErrStaleVersion):
		writeError(w, http.StatusConflict, "stale_version", "match has changed; refresh the board")
		return
	case errors.Is(err, match.ErrMatchUnavailable):
		writeError(w, http.StatusConflict, "match_unavailable", "match is not active")
		return
	case errors.Is(err, match.ErrFlagFell):
		writeError(w, http.StatusConflict, "clock_expired", "the active clock has expired")
		return
	case errors.Is(err, match.ErrWrongTurn), errors.Is(err, match.ErrIllegalMove):
		writeError(w, http.StatusUnprocessableEntity, "illegal_move", err.Error())
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "move_failed", "could not accept move")
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (e matchEndpoints) applyMove(ctx context.Context, matchID, credential string, body moveRequest) (moveResponse, error) {
	game, accepted, color, err := e.service.ApplyMove(ctx, matchID, credential, body.ExpectedVersion, body.From, body.To, body.Promotion)
	if err != nil {
		return moveResponse{}, err
	}
	nextTurn := string(game.ClockActiveColor)
	if nextTurn == "" {
		fields := strings.Fields(game.Position)
		if len(fields) > 1 {
			if fields[1] == "w" {
				nextTurn = "white"
			} else {
				nextTurn = "black"
			}
		}
	}
	activeColor := nextTurn
	response := moveResponse{
		Type: "move_accepted", Position: game.Position, NextTurn: nextTurn, Version: game.Version, InCheck: game.InCheck, EligibleDraws: game.EligibleDraws,
		Clock: clockResponse{
			WhiteRemainingMS: game.WhiteRemaining.Milliseconds(),
			BlackRemainingMS: game.BlackRemaining.Milliseconds(),
			AsOf:             game.ClockAsOf.UTC().Format(time.RFC3339Nano),
		},
		Result: game.Result,
	}
	if game.Status == match.StatusActive {
		response.Clock.ActiveColor = &activeColor
	}
	response.Move.Sequence = accepted.Sequence
	response.Move.Color = string(color)
	response.Move.From = accepted.From
	response.Move.To = accepted.To
	response.Move.Promotion = accepted.Promotion
	response.Move.Notation = accepted.Notation
	response.Move.Check = accepted.GivesCheck
	if e.hub != nil {
		if data, err := json.Marshal(response); err == nil {
			e.hub.Publish(game.ID, data)
		}
		if game.Status == match.StatusCompleted {
			terminal, err := json.Marshal(map[string]any{
				"type": "game_ended", "match_id": game.ID, "version": game.Version,
				"result": game.Result, "reason": game.TerminationReason,
				"clock": response.Clock,
			})
			if err == nil {
				e.hub.Publish(game.ID, terminal)
			}
		}
	}
	return response, nil
}

func (e matchEndpoints) socketMessage(ctx context.Context, matchID, credential string, data []byte) error {
	var message struct {
		Type            string `json:"type"`
		ExpectedVersion int64  `json:"expected_version"`
		From            string `json:"from"`
		To              string `json:"to"`
		Promotion       string `json:"promotion"`
	}
	if err := json.Unmarshal(data, &message); err != nil || message.Type != "move" {
		return errors.New("unsupported realtime action")
	}
	_, err := e.applyMove(ctx, matchID, credential, moveRequest{
		ExpectedVersion: message.ExpectedVersion,
		From:            message.From,
		To:              message.To,
		Promotion:       message.Promotion,
	})
	return err
}

func (e matchEndpoints) snapshot(ctx context.Context, matchID, credential string) ([]byte, error) {
	state, err := e.service.Read(ctx, matchID, credential)
	if err != nil {
		return nil, err
	}
	var activeColor *string
	if state.Match.ClockActiveColor != "" {
		value := string(state.Match.ClockActiveColor)
		activeColor = &value
	}
	asOf := state.Match.ClockAsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	response := struct {
		Type          string        `json:"type"`
		MatchID       string        `json:"match_id"`
		Status        string        `json:"status"`
		Color         string        `json:"color"`
		Position      string        `json:"position"`
		NextTurn      string        `json:"next_turn"`
		Version       int64         `json:"version"`
		TimeControl   string        `json:"time_control"`
		InCheck       bool          `json:"in_check"`
		EligibleDraws []string      `json:"eligible_draws"`
		Clock         clockResponse `json:"clock"`
		Result        string        `json:"result,omitempty"`
	}{
		Type: "match_snapshot", MatchID: state.Match.ID, Status: string(state.Match.Status),
		Color: string(state.Seat.Color), Position: state.Match.Position,
		NextTurn: turnFromPosition(state.Match.Position), Version: state.Match.Version,
		TimeControl: state.Match.TimeControl, InCheck: state.Match.InCheck, EligibleDraws: state.Match.EligibleDraws, Result: state.Match.Result,
		Clock: clockResponse{
			WhiteRemainingMS: state.Match.WhiteRemaining.Milliseconds(),
			BlackRemainingMS: state.Match.BlackRemaining.Milliseconds(),
			ActiveColor:      activeColor, AsOf: asOf.UTC().Format(time.RFC3339Nano),
		},
	}
	return json.Marshal(response)
}

func (e matchEndpoints) resign(w http.ResponseWriter, r *http.Request) {
	credential, ok := bearerCredential(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "seat authorization is required")
		return
	}
	game, err := e.service.Resign(r.Context(), r.PathValue("match_id"), credential)
	if status, code, message, handled := terminationError(err); handled {
		writeError(w, status, code, message)
		return
	}
	response := terminalResponse(game)
	e.publishTerminal(game, response)
	writeJSON(w, http.StatusOK, response)
}

func (e matchEndpoints) offerDraw(w http.ResponseWriter, r *http.Request) {
	credential, ok := bearerCredential(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "seat authorization is required")
		return
	}
	seat, err := e.service.Read(r.Context(), r.PathValue("match_id"), credential)
	if err != nil {
		if status, code, message, handled := terminationError(err); handled {
			writeError(w, status, code, message)
			return
		}
	}
	offer, err := e.service.OfferDraw(r.Context(), r.PathValue("match_id"), credential)
	if status, code, message, handled := terminationError(err); handled {
		writeError(w, status, code, message)
		return
	}
	response := map[string]any{"offer_id": offer.ID, "status": offer.Status, "offering_color": seat.Seat.Color}
	if e.hub != nil {
		if event, marshalErr := json.Marshal(map[string]any{"type": "draw_offer", "offer_id": offer.ID, "offering_seat_id": offer.OfferingSeatID, "offering_color": seat.Seat.Color, "status": offer.Status}); marshalErr == nil {
			e.hub.Publish(offer.MatchID, event)
		}
	}
	writeJSON(w, http.StatusCreated, response)
}

func (e matchEndpoints) resolveDraw(w http.ResponseWriter, r *http.Request) {
	credential, ok := bearerCredential(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "seat authorization is required")
		return
	}
	var body drawDecisionRequest
	if err := decodeJSON(w, r, &body); err != nil || (body.Decision != "accept" && body.Decision != "decline") {
		writeError(w, http.StatusBadRequest, "invalid_request", "decision must be accept or decline")
		return
	}
	game, offer, err := e.service.ResolveDrawOffer(r.Context(), r.PathValue("match_id"), credential, r.PathValue("offer_id"), body.Decision == "accept")
	if status, code, message, handled := terminationError(err); handled {
		writeError(w, status, code, message)
		return
	}
	response := map[string]any{"offer_id": offer.ID, "status": offer.Status, "match_id": game.ID, "match_status": game.Status, "result": game.Result}
	if e.hub != nil {
		if event, marshalErr := json.Marshal(map[string]any{"type": "draw_resolved", "offer_id": offer.ID, "status": offer.Status, "version": game.Version}); marshalErr == nil {
			e.hub.Publish(game.ID, event)
		}
		if game.Status == match.StatusCompleted {
			e.publishTerminal(game, terminalResponse(game))
		}
	}
	writeJSON(w, http.StatusOK, response)
}

func (e matchEndpoints) claimDraw(w http.ResponseWriter, r *http.Request) {
	credential, ok := bearerCredential(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "seat authorization is required")
		return
	}
	var body drawClaimRequest
	if err := decodeJSON(w, r, &body); err != nil || body.Method == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "method is required")
		return
	}
	game, err := e.service.ClaimDraw(r.Context(), r.PathValue("match_id"), credential, body.Method)
	if status, code, message, handled := terminationError(err); handled {
		writeError(w, status, code, message)
		return
	}
	response := terminalResponse(game)
	e.publishTerminal(game, response)
	writeJSON(w, http.StatusOK, response)
}

func bearerCredential(r *http.Request) (string, bool) {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	return parts[1], true
}

func terminationError(err error) (int, string, string, bool) {
	switch {
	case err == nil:
		return 0, "", "", false
	case errors.Is(err, match.ErrInvalidSeat):
		return http.StatusUnauthorized, "unauthorized", "seat authorization is invalid", true
	case errors.Is(err, match.ErrDrawOfferNotFound):
		return http.StatusNotFound, "draw_offer_not_found", "draw offer not found", true
	case errors.Is(err, match.ErrDrawOfferPending):
		return http.StatusConflict, "draw_offer_pending", "a draw offer is already pending", true
	case errors.Is(err, match.ErrMatchUnavailable):
		return http.StatusConflict, "match_unavailable", "match is not active", true
	case errors.Is(err, match.ErrFlagFell):
		return http.StatusConflict, "clock_expired", "the active clock has expired", true
	case errors.Is(err, match.ErrWrongTurn), errors.Is(err, match.ErrDrawNotClaimable):
		return http.StatusUnprocessableEntity, "draw_action_rejected", err.Error(), true
	default:
		return http.StatusInternalServerError, "termination_failed", "could not complete match action", true
	}
}

func terminalResponse(game match.Match) map[string]any {
	return map[string]any{
		"match_id": game.ID, "status": game.Status, "result": game.Result,
		"reason": game.TerminationReason, "position": game.Position,
		"version": game.Version, "in_check": game.InCheck,
		"clock": map[string]any{
			"white_remaining_ms": game.WhiteRemaining.Milliseconds(),
			"black_remaining_ms": game.BlackRemaining.Milliseconds(),
			"active_color":       nil,
			"as_of":              game.ClockAsOf.UTC().Format(time.RFC3339Nano),
		},
	}
}

func (e matchEndpoints) publishTerminal(game match.Match, response map[string]any) {
	if e.hub == nil {
		return
	}
	event := make(map[string]any, len(response)+1)
	for key, value := range response {
		event[key] = value
	}
	event["type"] = "game_ended"
	if data, err := json.Marshal(event); err == nil {
		e.hub.Publish(game.ID, data)
	}
}

func turnFromPosition(position string) string {
	fields := strings.Fields(position)
	if len(fields) > 1 && fields[1] == "b" {
		return "black"
	}
	return "white"
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("request must contain one JSON value")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
