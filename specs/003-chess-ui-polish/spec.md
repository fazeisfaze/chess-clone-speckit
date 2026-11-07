# Feature Specification: Chess UI Polish

**Feature Branch**: `003-chess-ui-polish`

**Created**: 2026-11-07

**Status**: Draft

**Input**: User description: "dong bo mau sac quan co, nang cap UI link moi thanh click vao de copy link"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Copy an Invitation Link (Priority: P1)

When a host has created an invitation match, they can use a clearly labeled copy action
to place the complete invitation link on the clipboard without navigating away from the
waiting screen. The page displays a short confirmation when copying succeeds.

**Why this priority**: The second player needs the invitation to join, and copying it
should be direct and reliable without exposing a long token string as page text.

**Independent Test**: Create a match, activate the copy action, and verify the clipboard
contains the exact invitation link, the page stays on the waiting screen, and success is
announced to the host.

**Acceptance Scenarios**:

1. **Given** a host has created a match, **When** they activate the invitation copy action,
   **Then** the complete invitation link is copied and the page does not navigate.
2. **Given** copying succeeds, **When** the host receives feedback,
   **Then** the UI announces that the invitation link was copied without displaying the
   raw token as the action label.
3. **Given** the browser denies clipboard access or copying fails,
   **When** the host activates the action, **Then** the UI reports failure and does not
   falsely claim that the link was copied.
4. **Given** no match invitation exists yet, **When** the host views the lobby,
   **Then** no copy action is offered for a missing link.

---

### User Story 2 - Distinguish Piece Colors Consistently (Priority: P1)

Either player can distinguish White's pieces from Black's pieces throughout the game.
Each side's pieces keep the same visual color on light and dark squares and when the board
is viewed from either player's orientation.

**Why this priority**: Consistent piece colors prevent confusion about ownership and make
the board easier to read during online play.

**Independent Test**: Render the starting position and representative positions containing
both sides on light and dark squares from both player orientations; verify each side retains
its assigned visual color.

**Acceptance Scenarios**:

1. **Given** pieces occupy both light and dark squares, **When** either player views the
   board, **Then** White's pieces share one consistent visual treatment and Black's pieces
   share a distinct treatment.
2. **Given** the board orientation changes between White's and Black's perspectives,
   **When** both players inspect the same position, **Then** piece colors still identify
   the same side regardless of square color or orientation.
3. **Given** a move, capture, or promotion changes the board, **When** the board updates,
   **Then** each remaining or promoted piece continues to use the visual treatment of its
   assigned side.

### Edge Cases

- Clipboard access is denied, unavailable, or invoked outside a browser-supported secure
  context; the UI reports failure without navigating or claiming success.
- The host activates copy repeatedly; each successful action copies the current match's
  invitation link and provides understandable feedback.
- A piece is rendered on a square whose background is close in brightness to the piece;
  the side remains visually distinguishable.
- A player views the board from the opposite orientation; the visual side assignment does
  not flip with the square arrangement.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide a clearly labeled invitation-copy action only when
  the host has an active invitation link.
- **FR-002**: Activating the invitation-copy action MUST copy the current match's complete
  invitation link without navigating the browser away from the lobby.
- **FR-003**: The system MUST announce successful copying and MUST report clipboard failure
  without showing a false success state.
- **FR-004**: The invitation-copy action MUST use a descriptive label rather than displaying
  the raw invitation token as its visible label.
- **FR-005**: The visual color treatment for a chess piece MUST be based on its assigned
  side, not on the background color of its square.
- **FR-006**: White and Black pieces MUST remain consistently distinguishable across board
  squares, player perspectives, moves, captures, and promotions.

### Key Entities *(include if feature involves data)*

- **Invitation Link**: The existing match invitation that grants the guest access to join
  the waiting match; copied as a complete value when requested by the host.
- **Piece Side**: The existing White or Black ownership of a piece, which determines its
  consistent visual treatment on the board.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In 20 invitation-copy trials on supported browsers, at least 19 copy the
  correct complete invitation link within 2 seconds without navigating away.
- **SC-002**: All White and Black pieces in the project's board-rendering fixture set retain
  the correct side-specific visual treatment on both square colors and both orientations.
- **SC-003**: At least 9 of 10 first-time test participants can copy and share an invitation
  link without help and can correctly distinguish the two piece sides within 5 seconds.
- **SC-004**: In every clipboard-denial test, the UI reports failure and never displays a
  success confirmation.

## Assumptions

- This is a visual and interaction refinement to the existing online standard chess game;
  it does not change match creation, invitation authorization, move rules, or clocks.
- The invitation-copy action copies the existing invitation link in full. Invitation-token
  placement and credential security remain governed by the separately scoped invitation
  link security work; the raw token is not displayed as the visible action label.
- The existing White/Black side assignments are authoritative and do not change when a
  player flips board perspective.
- Clipboard access is available on supported secure browser contexts; failures are still
  surfaced clearly.
