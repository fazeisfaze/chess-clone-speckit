# Data Model: Online Chess

## Match

Represents one invitation-based two-player game.

| Field | Meaning and validation |
|---|---|
| `id` | Unique opaque match identifier. |
| `invite_token_digest` | Digest of high-entropy invitation secret; never store or expose the raw secret. |
| `status` | `waiting`, `active`, `completed`, or `unavailable`; transitions are validated by the service. |
| `white_player_id` | Seat identifier for White; set at creation. |
| `black_player_id` | Seat identifier for Black; set once when the invitation is accepted. |
| `position` | Canonical chess position including side to move, castling rights, en-passant target, and move counters. |
| `time_control` | Immutable match control: 3+2, 5+0, or 10+0 (minutes per player plus seconds of increment per move). |
| `initial_time_ms`, `increment_ms` | Time-control values in milliseconds, fixed at match creation. |
| `white_remaining_ms`, `black_remaining_ms` | Remaining time persisted at the latest clock transition. |
| `clock_active_color` | `white`, `black`, or null when the clock is stopped. |
| `clock_started_at` | Authoritative timestamp anchor for the active clock; null while stopped. |
| `version` | Monotonically increasing accepted-state version used to reject stale commands. |
| `result` | Empty while active; otherwise winner/draw outcome and termination reason. |
| `created_at`, `updated_at` | Timestamps used for lifecycle tracking. |

### State transitions

- `waiting` → `active` when the invited second seat successfully joins.
- On transition to `active`, White's clock starts at the chosen initial time and Black's
  clock remains paused.
- `waiting` → `unavailable` when the match is explicitly invalidated or expires under
  the eventual retention policy.
- `active` → `completed` on checkmate, stalemate/draw, resignation, or accepted draw.
- Clock expiration completes the match with a timeout win, unless the opponent cannot
  possibly checkmate by any legal sequence, in which case the result is a draw.
- `completed` and `unavailable` are terminal; moves and draw offers are rejected.

## Player Seat

Represents one guest player's authority within one match.

| Field | Meaning and validation |
|---|---|
| `id` | Unique opaque seat identifier. |
| `match_id` | Parent match. |
| `color` | `white` or `black`; each match has at most one seat of each color. |
| `recovery_credential_digest` | Digest of a separate high-entropy credential used to resume this seat. |
| `connection_status` | Derived transient status; a disconnect does not erase the seat or match. |

The invitation token grants a waiting guest permission to claim the empty seat; it is not
the credential for recovering that seat later. Seat authorization is required for all
mutating actions and match-private state.

## Move

Represents an accepted action and its place in authoritative match history.

| Field | Meaning and validation |
|---|---|
| `match_id` | Parent match. |
| `sequence` | Unique, contiguous sequence within the match; tied to match version. |
| `player_seat_id` | Must be the seat whose turn it is. |
| `from`, `to` | Valid board squares; accepted only if legal in the prior position. |
| `promotion` | Optional; if present, only queen, rook, bishop, or knight. |
| `notation` | Human-readable move notation generated from the accepted move. |
| `position_after` | Resulting canonical position committed atomically with the move. |
| `accepted_at` | Server timestamp. |

Rejected, duplicate, out-of-turn, stale-version, and post-completion move commands do not
create a `Move` or change the match version.

For a timed match, an accepted move atomically records the board change and clock transition:
charge elapsed time to the mover, apply that control's increment to the mover, switch the
active color, and reset the timestamp anchor. An expired clock ends the match and the move
is rejected. A flag fall is a win unless the opponent cannot possibly checkmate by any
legal sequence, in which case it is a draw.

## Clock snapshot

A clock snapshot is part of the match state and every versioned update. It contains both
remaining millisecond values at an `as_of` server timestamp, the active color, and the
selected time control. The active player's current display value is derived from this
persisted anchor; clients never submit authoritative time values.

## Draw Offer

Represents a player's offer to end an active match as a draw.

| Field | Meaning and validation |
|---|---|
| `match_id` | Parent active match. |
| `offering_seat_id` | One of the two match seats. |
| `status` | `pending`, `accepted`, or `declined`. |
| `created_at`, `resolved_at` | Offer and resolution timestamps. |

Only the opposing seat may accept or decline a pending offer. Accepting completes the
match as a draw; declining leaves position and turn unchanged. A later offer supersedes a
previously declined offer.

## Consistency invariants

- A match has no more than two seats and cannot be active until both seats exist.
- Every accepted move and resulting position/version are committed atomically.
- Every accepted move and its clock debit, increment, active color, and timestamp anchor are
  committed atomically.
- Only one command may advance a given match version; competing commands based on the
  same version cannot both be accepted.
- The persisted position, move history, current turn, clock state, and terminal result must
  agree.
- Match lifecycle, legal move, and draw outcome decisions are server authoritative.
