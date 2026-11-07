# HTTP API Contract: Online Chess

This contract describes user-visible HTTP operations. All JSON uses UTF-8. Responses
MUST NOT expose raw credentials or another player's recovery secret. Error responses use
`{"error":{"code":"...","message":"..."}}`.

## Create a match

`POST /matches`

Creates a waiting match and the creator's White seat.

Request body: `{"time_control":"3+2"}`. Supported values are `3+2`, `5+0`, and `10+0`.
The selected control is immutable for that match.

**Success — `201 Created`**

```json
{
  "match_id": "opaque-match-id",
  "status": "waiting",
  "color": "white",
  "invite_url": "https://example.invalid/join#token=opaque-secret",
  "recovery_credential": "one-time-secret",
  "version": 0,
  "time_control": "3+2",
  "clock": {
    "white_remaining_ms": 180000,
    "black_remaining_ms": 180000,
    "active_color": null,
    "as_of": "2026-11-07T12:00:00Z"
  }
}
```

The invitation secret is high entropy. The recovery credential is separate and returned
only to its seat owner. Production deployment MUST use HTTPS.

## Join an invited match

`POST /matches/join`

Request body: `{"invite_token":"..."}`. Claims the empty seat in an available waiting
match. The server assigns the opposite color and starts White's clock when the match
becomes active. The client reads the token from the
invitation URL fragment and submits it in the request body so it is not sent in URL logs.

**Success — `200 OK`**: returns `match_id`, `status`, `color`, initial `position`,
`version`, selected `time_control`, initial clock snapshot, and a new seat-specific
`recovery_credential`.

**Errors**: `404` invalid invitation; `409` match already full or unavailable.

## Recover a seat

`POST /matches/{match_id}/recover`

Request body: `{"recovery_credential":"..."}`. On success, returns the seat's color,
current match status, canonical position, current version, result if completed, and moves
after an optional supplied `known_version`, plus a fresh clock snapshot. Invalid credentials
MUST NOT reveal match state. Remaining clock values are calculated from the server timestamp
anchor; a reconnect never resets or pauses the active clock.

## Read match state

`GET /matches/{match_id}`

Requires seat authorization. Returns match status, caller's color, current position, turn,
version, result, and move history. A waiting match may be read by its creator seat only.
The response includes a clock snapshot with remaining times, active color, time control, and
server `as_of` timestamp.

## Submit a move

`POST /matches/{match_id}/moves`

Requires seat authorization.

```json
{
  "expected_version": 12,
  "from": "e2",
  "to": "e4",
  "promotion": null
}
```

Success returns the accepted move, resulting position, next turn, version, check status,
updated clock snapshot, and game result if terminal. Invalid/illegal/out-of-turn moves return `422`; stale
versions return `409`. The server determines move-vs-timeout order after serializing the
command for the match; client timestamps are ignored.
Neither error may change match state. A successful move is persisted before success is
returned or an event is broadcast.

## Draw actions

- `POST /matches/{match_id}/draw-offers` creates a pending offer from the caller's seat.
- `POST /matches/{match_id}/draw-offers/{offer_id}/resolve` with
  `{"decision":"accept"}` or `{"decision":"decline"}` resolves the opponent's offer.
- `POST /matches/{match_id}/draw-claims` claims a rule-based draw when the current position
  and move history make a claim available to the caller. The server validates the claim
  against authoritative history; an invalid claim leaves the match unchanged.

An accepted offer or valid draw claim completes the match as a draw. Declining leaves the
board and turn unchanged.

## Resign

`POST /matches/{match_id}/resign` ends an active match and records the opponent as winner.

## Common errors

- `401`: missing or invalid seat authorization.
- `404`: unknown match/resource or invalid invitation.
- `409`: invalid lifecycle transition, stale version, duplicate action, or occupied seat.
- `422`: malformed or rule-illegal action.
- `429`: request rate limit exceeded.

Errors MUST be actionable without revealing secrets or another player's credentials.
