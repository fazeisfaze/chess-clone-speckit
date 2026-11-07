# Feature Specification: Online Chess

**Feature Branch**: `001-online-chess`

**Created**: 2026-11-07

**Status**: Draft

**Input**: User description: "build a clone normal chess game online"

## Clarifications

### Session 2026-11-07

- Q: Should timed chess clocks be included in the first version of the game? → A: Include a server-authoritative clock in the first version.
- Q: What time control should the first version use? → A: Offer the 3+2, 5+0, and 10+0 presets (minutes per player plus increment in seconds per move).
- Q: When should White's clock start? → A: Start when the second player joins and the match becomes active.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Start an Online Match (Priority: P1)

A player starts a standard chess match and shares an invitation with another player.
The second player joins, and both players see the same initial board and know whose
turn it is.

**Why this priority**: A two-player online match is the core value of the feature.

**Independent Test**: Start a match in one browser session, join from a second session,
and verify both sessions show the same game and opposing colors.

**Acceptance Scenarios**:

1. **Given** a player without an active match, **When** they create a match,
   **Then** the game provides an invitation that another player can use to join.
2. **Given** an open match, **When** a second player joins using its invitation,
   **Then** both players see the standard starting position and the same active game.
3. **Given** a match has two players, **When** another player attempts to join,
   **Then** the match remains unchanged and the additional player is told it is unavailable.
4. **Given** a player creates a match, **When** they select one of the available clock presets,
   **Then** the invited opponent joins a match using that same preset and both players can see it.
5. **Given** the invited player joins, **When** the match becomes active,
   **Then** White's clock starts and Black's clock remains paused until Black's turn.

---

### User Story 2 - Play a Standard Chess Game (Priority: P1)

Players take turns making legal moves according to standard chess rules. Each player
can see the opponent's moves, the current game state, and both synchronized clocks.

**Why this priority**: Online matchmaking without correct, synchronized chess play is
not a usable chess game.

**Independent Test**: Join a match from two sessions, make legal and illegal move
attempts, and verify both sessions agree on the resulting board and turn.

**Acceptance Scenarios**:

1. **Given** it is a player's turn, **When** they make a legal move,
   **Then** both players see the updated board and the turn passes to the opponent.
2. **Given** a player attempts an illegal move or moves out of turn,
   **When** the move is submitted, **Then** the board remains unchanged and the player
   is informed that the move was not accepted.
3. **Given** a move causes check, **When** the board updates,
   **Then** both players can identify that the king is in check.
4. **Given** a pawn reaches its final rank, **When** the move is completed,
   **Then** the player can promote it to a queen, rook, bishop, or knight.
5. **Given** a move ends the game by checkmate, stalemate, or a recognized draw,
   **When** the move is accepted, **Then** both players see the final result and cannot
   make further moves in that game.
6. **Given** a timed game is active, **When** a legal move is accepted,
   **Then** both players see the same clock state and only the next player's clock runs.
7. **Given** a player's time expires, **When** the server confirms expiration,
   **Then** both players see the game result and further moves are rejected.

---

### User Story 3 - Recover or End a Match (Priority: P2)

A player can recover from a temporary connection loss and return to the current game.
Either player can resign, and players can agree to a draw.

**Why this priority**: Recovery and clear match endings make online games reliable,
without blocking the initial playable match.

**Independent Test**: Interrupt one player's connection during a match, reconnect,
verify the current position, then test resignation and an agreed draw in separate games.

**Acceptance Scenarios**:

1. **Given** a player disconnects during an active match, **When** they reconnect,
   **Then** they return to the latest accepted position and correct turn.
2. **Given** an active match, **When** a player resigns,
   **Then** the game ends and the opponent is declared the winner.
3. **Given** an active match, **When** one player offers a draw and the other accepts,
   **Then** the game ends as a draw.
4. **Given** a player offers a draw and the other declines,
   **Then** the game continues without changing the board or turn.

### Edge Cases

- A player submits duplicate or delayed moves; only a valid move for the current position
  and turn is accepted.
- Both players attempt to move at nearly the same time; at most one move is accepted for
  the current turn.
- A player disconnects while joining or while an opponent move is being processed; the
  game must not lose or duplicate an accepted move.
- An invitation refers to a match that is full, completed, or no longer available; joining
  is rejected with a clear explanation.
- The game reaches castling, en passant, promotion, checkmate, stalemate, or a draw by
  repetition, insufficient material, or the fifty-move rule; the applicable standard
  chess rule is applied consistently.
- A player attempts to move a pinned piece or leave their king in check; the move is rejected.
- A move arrives near the active clock's expiration; the server resolves it using its
  authoritative event time so both players receive the same outcome.
- A player disconnects during a timed game; the server clock continues independently of
  that player's browser and the recovered state includes both remaining times.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow a player to create an online match and provide a way
  to invite one opponent.
- **FR-002**: The system MUST allow an invited opponent to join an available match and
  MUST prevent more than two players from joining as players.
- **FR-003**: The system MUST assign opposing colors and show both players the same
  starting position and current game state.
- **FR-004**: The system MUST enforce alternating turns and accept only moves legal under
  standard chess rules, including castling, en passant, and promotion.
- **FR-005**: The system MUST offer promotion choices of queen, rook, bishop, and knight.
- **FR-006**: The system MUST identify check, checkmate, stalemate, and standard draw
  conditions, including repetition, insufficient material, and the fifty-move rule.
- **FR-007**: The system MUST communicate accepted moves and game-state changes to both
  players and keep their displayed positions consistent.
- **FR-008**: The system MUST allow a player to resign an active match and MUST record the
  opponent as winner.
- **FR-009**: The system MUST allow a player to offer a draw and the opponent to accept
  or decline it.
- **FR-010**: The system MUST allow a disconnected player to return to an active match
  and continue from the latest accepted position.
- **FR-011**: The system MUST reject moves submitted out of turn, illegal moves, and
  moves submitted after a game has ended without changing the game state.
- **FR-012**: The system MUST clearly show when a match is waiting for an opponent, active,
  completed, or unavailable.
- **FR-013**: The system MUST maintain both players' remaining times authoritatively on the
  server, run only the active player's clock, and synchronize clock state with accepted
  moves and match updates. Client clocks MUST be display-only; expiration MUST be resolved
  by the server and propagated to both players.
- **FR-014**: The system MUST let the match creator select one of three time controls:
  3 minutes per player with a 2-second increment per accepted move (3+2), 5 minutes per
  player with no increment (5+0), or 10 minutes per player with no increment (10+0). The
  selected control MUST remain the same for both players for the duration of the match.
- **FR-015**: The system MUST start White's clock when the second player joins and the match
  becomes active. Only White's clock runs until White's first accepted move; afterward the
  clock MUST switch to the player whose turn it is.

### Key Entities *(include if feature involves data)*

- **Player**: A participant in a match, with an assigned color and connection status.
- **Match**: An invited two-player game with a lifecycle, current position, turn, and result.
- **Move**: A proposed or accepted chess action associated with a player and a position
  in the match history.
- **Clock**: Both players' remaining time, the selected time control, and which player's
  clock is currently running.
- **Draw Offer**: An offer made by one player and its pending, accepted, or declined outcome.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Two players can create and join a match and see the starting board within
  60 seconds in at least 95% of validation attempts on a stable connection.
- **SC-002**: In a 20-move test game, both players' boards show the same accepted move
  within 2 seconds in at least 99% of moves on a stable connection.
- **SC-003**: All legal and illegal moves in the standard chess rules test set are
  respectively accepted and rejected, with no disagreement between the two players' views.
- **SC-004**: A player who reconnects within 5 minutes of a temporary disconnection can
  resume the same active match at its latest accepted position in at least 95% of tests.
- **SC-005**: At least 90% of first-time test participants can start a match, make a legal
  move, and identify whose turn follows without assistance.
- **SC-006**: In timed-match validation, both players' displayed clocks remain within
  1 second of the server-authoritative remaining time, and both clients show the same
  expiration result in 100% of tested timeouts.

## Assumptions

- The requested game is standard chess; the previously discussed dice-based variant is
  not included in this feature.
- The initial experience targets players using a supported web browser on desktop or
  mobile-sized screens.
- Players can join as guests; account creation, persistent profiles, rankings, chat,
  spectators, tournaments, and computer opponents are outside this feature's scope.
- Players use an invitation to join a match; public matchmaking is not required for the
  initial version.
- Timed matches use server-authoritative clocks and offer 3+2, 5+0, and 10+0 presets.
  White's clock starts when the second player joins; a disconnected player's clock
  continues to run. The player may resume an active match for as long as the match remains
  available.
- The game follows standard chess rules, including recognized draw conditions.
