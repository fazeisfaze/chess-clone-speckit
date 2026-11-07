# UI Contract: Chess UI Polish

## Invitation copy action

- In the host waiting lobby, when a non-empty invitation link exists, provide a button with
  a descriptive accessible name such as **Copy invitation link**. Do not render the raw
  invitation value as the button label.
- Activating the button attempts to copy the exact current invitation link and does not
  navigate or open the link.
- Announce success only after clipboard writing completes. If clipboard access is missing
  or rejected, provide a visible and accessible failure message; do not show success.
- Do not offer the action when no invitation value is available.

## Board piece colors

- Every rendered White piece uses the White side presentation, and every Black piece uses
  the distinct Black side presentation.
- Piece side presentation is independent of light/dark square classes and player
  orientation.
- Moves, captures, and promotions continue to derive piece side from the resulting board
  position.

## Compatibility

No HTTP, WebSocket, match-state, or persistence contract changes are introduced. Existing
invitation URL format and authorization semantics are unchanged by this feature.
