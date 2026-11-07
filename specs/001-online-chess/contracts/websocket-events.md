# WebSocket Event Contract: Online Chess

## Connection

Connect to `/matches/{match_id}/events` over `wss://` in production. The first client frame
MUST be an `authenticate` message containing its seat recovery credential. The server
MUST authenticate within 5 seconds, send no match data before authentication, and close
unauthorized connections. It MUST validate the browser `Origin`, impose message-size and
rate limits, and never log credentials. Credentials MUST NOT appear in URLs.

The connection is a real-time channel, not durable storage. After a disconnect, clients
recover through the HTTP recovery operation and then reconnect. Every event includes a
monotonically increasing `version` so clients can discard duplicates and detect gaps.
Clock snapshots include both remaining values in milliseconds, active color, selected
control, and an `as_of` server timestamp. Clients render a local countdown from this anchor
and periodically correct it from server snapshots; client time is never authoritative.

## Client messages

### `authenticate`

`{"type":"authenticate","recovery_credential":"..."}`. On success, the server sends
the latest authorized match snapshot and version before live events. On failure, it closes
the connection without revealing match state.

### `move`

```json
{
  "type": "move",
  "expected_version": 12,
  "from": "e2",
  "to": "e4",
  "promotion": null
}
```

The server validates authorization, turn, expected version, clock deadline, and chess
legality. It commits the move, clock debit/increment, next active color, and version before
emitting `move_accepted` to both players. Invalid or late commands receive a private
`action_rejected` and do not change the board; a timeout may instead end the match.

### `draw_offer` / `draw_response` / `draw_claim` / `resign`

Draw response includes an offer identifier and `accept` or `decline`. A resignation or
accepted draw causes a terminal `game_ended` event after persistence. A draw claim is
accepted only when validated against authoritative position and move history.

## Server events

- `match_ready`: second seat joined; includes opposing colors, initial position, version,
  chosen clock control, initial remaining times, White as active color, server timestamp,
  check status, and claimable draw methods.
- `move_accepted`: accepted move, resulting position, next turn, version, whether the move
  gives check, claimable draw methods, updated clock snapshot, and optional result.
- `action_rejected`: private error code/message; no state change.
- `draw_offer`: offer identifier and offering color.
- `draw_resolved`: offer outcome and current match status.
- `draw_claim_resolved`: whether the claim succeeded; a successful claim also ends the game.
- `game_ended`: result and reason, with final version.
- `clock_expired`: timeout result and final clock snapshot; clients also receive `game_ended`.
- `opponent_connection`: opponent connected/disconnected status; does not alter chess state.
- `sync_required`: client version is too old or has a gap; recover through HTTP.

## Ordering and recovery

- Server events for one match MUST be ordered by committed version.
- A client MUST apply an event only if its version is the expected next version; otherwise
  it requests a state recovery.
- Simultaneous moves based on the same version result in at most one accepted move.
- Move acceptance and timeout expiration for a match are serialized against the same
  authoritative server time; a late move cannot win a race against an already expired clock.
- A flag fall is a win only if the opponent can possibly checkmate by a legal sequence;
  otherwise the game is drawn under standard chess rules.
- Reconnection MUST return the latest committed game state even if the client missed events.
