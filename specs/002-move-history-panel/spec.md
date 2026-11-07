# Feature Specification: Move History Panel

**Feature Branch**: `002-move-history-panel`

**Created**: 2026-11-07

**Status**: Draft

**Input**: User description: "nâng cấp UI, bên phải bàn cờ hiển thị lịch sử các nước đi, highlight nước vừa đi"

## Clarifications

### Session 2026-11-07

- Q: Should the no-token-in-URL requirement be part of the current move-history UI change, or a separate security change to invitation links? → A: Treat it as a separate security change; keep it outside this feature's scope.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Review the Move History (Priority: P1)

During an online match, either player can review the moves already accepted in the
game. The move list identifies the move number, moving color, and standard chess
notation. On desktop, players can see the history beside the board; on narrow screens,
they can reach it without obscuring the board or game controls.

**Why this priority**: Players need context to understand how the current position arose
and to follow the game without relying on memory.

**Independent Test**: Play several alternating moves from two browser sessions and verify
that both clients show the same complete history in the correct order and can read it
while the game board remains usable.

**Acceptance Scenarios**:

1. **Given** a newly started match with no accepted moves, **When** either player views
   the game, **Then** the move history shows a clear empty state and does not invent moves.
2. **Given** a match with accepted moves, **When** a player views the history,
   **Then** moves appear in game order with move number, color, and notation.
3. **Given** a player scrolls through a long game, **When** earlier moves are reviewed,
   **Then** the full history remains reachable and the current board and controls remain
   available.

---

### User Story 2 - Identify the Latest Move (Priority: P1)

After either player makes a move, both players can immediately see which piece moved by
looking at the origin and destination squares highlighted on the board. The matching
latest move is also identifiable in the history panel.

**Why this priority**: A clear last-move indicator connects the move list to the live board
and helps both players follow online play.

**Independent Test**: In a two-browser match, make legal moves alternately and verify that
both clients highlight the same origin and destination squares and mark the corresponding
latest history entry.

**Acceptance Scenarios**:

1. **Given** a legal move is accepted, **When** both clients receive the updated game,
   **Then** they highlight the move's origin and destination squares and mark that move in
   the history.
2. **Given** another legal move is accepted, **When** the board updates,
   **Then** the highlight moves from the previous move to the newly accepted move.
3. **Given** a move is rejected or submitted twice, **When** the game state is unchanged,
   **Then** the history and last-move highlight remain unchanged.
4. **Given** a player reconnects or reloads an active game, **When** the latest match
   snapshot is restored, **Then** the history and last-move highlight match the latest
   accepted position.

### Edge Cases

- A move captures a piece on the destination square; both origin and destination remain
  highlighted even though the captured piece is no longer present.
- A promotion, castling move, or en-passant capture is shown as one accepted move, with
  its actual origin and destination squares highlighted.
- The game has not started, has no moves, or has ended; the history and highlight reflect
  the exact accepted history without adding placeholder moves.
- A client receives a duplicate, stale, or out-of-order update; it does not add a duplicate
  history entry or restore an older highlight.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST show every accepted move for the active match in chronological
  order, including its move number, moving color, and standard chess notation.
- **FR-002**: The system MUST keep both players' displayed move histories consistent with
  the server's accepted move history and update them when an accepted move arrives.
- **FR-003**: The system MUST highlight the origin and destination squares of the latest
  accepted move for both players.
- **FR-004**: The system MUST associate the latest accepted move with its corresponding
  entry in the move history.
- **FR-005**: The system MUST leave history and highlights unchanged for rejected moves,
  duplicate events, and stale or out-of-order events.
- **FR-006**: The system MUST restore move history and the latest-move highlight from the
  authoritative match state after a player reconnects or reloads the game.
- **FR-007**: The system MUST provide a clear empty state when a match contains no accepted
  moves and MUST make earlier moves reachable in a long game.
- **FR-008**: The move history MUST remain accessible alongside the board on supported
  desktop displays and remain reachable on narrow/mobile-sized displays without covering
  the board or essential match actions.

### Key Entities *(include if feature involves data)*

- **Move History Entry**: The existing accepted move associated with a match, including
  order, player color, notation, origin square, destination square, and resulting position.
- **Latest-Move Highlight**: A display state derived from the most recent accepted move's
  origin and destination squares; it is not an independent game action.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Across 400 accepted moves in two-client validation games, both players see
  the corresponding history entry and last-move squares within 2 seconds for at least
  396 moves.
- **SC-002**: Every accepted move in the project-maintained chess conformance fixtures
  appears exactly once and in the same order on both clients; rejected and duplicate
  moves appear zero times.
- **SC-003**: Across 20 reconnect/reload trials during active matches, at least 19 restore
  the complete move history and correct last-move highlight within 5 seconds.
- **SC-004**: At supported desktop widths, at least 90% of first-time participants can
  identify the latest move from the history and board highlight within 5 seconds without
  assistance.

## Assumptions

- This is an enhancement to the existing standard online chess game; it does not add a new
  chess variant or change move legality, clocks, or game-ending rules.
- The current server-accepted move history is the source of truth; the browser does not
  infer history from locally attempted moves.
- Standard algebraic notation and the application's existing White/Black terminology are
  used consistently.
- On desktop, the history panel is positioned to the right of the chessboard. On narrow
  screens, it may move below the board while remaining reachable.
- Invitation-token URL visibility and invitation-link credential behavior are unchanged by
  this feature and require a separate feature specification.
