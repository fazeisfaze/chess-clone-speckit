# Test Evidence: Move History Panel

## User Story 1 — Review the Move History

- **RED (HTTP response)**: The focused Go HTTP test failed to compile because the match
  response had no `Moves` field or move-history response type.
- **RED (frontend)**: The GameView/recovery tests failed because there was no accessible
  move-history region.
- **GREEN**: Focused Go HTTP tests passed for authorized GET and recovery responses. A
  database-backed integration test applied `e2`–`e4`, recovered the seat, and verified the
  ordered move sequence, color, coordinates, and SAN notation.
- **GREEN**: Focused frontend move-history and recovery tests passed, including the empty
  state and ordered entries. The panel is scrollable on desktop and remains reachable below
  the board on narrow displays.

## User Story 2 — Identify the Latest Move

- **RED (reducer)**: Temporarily omitted appending the accepted move in the reducer; its
  focused test failed because the expected accepted history entry was absent.
- **GREEN (reducer)**: Restored version-checked history append; duplicate, stale, rejected,
  and gap events leave history unchanged. Snapshots restore history, with missing history
  defaulting to an empty list for compatibility with older snapshots.
- **RED (board)**: The focused board test failed because origin square `e7` lacked the
  `last-move-from` class.
- **GREEN (board/UI)**: Both accepted-move squares and the matching history item receive
  latest-move styling. Two-client browser coverage verifies both perspectives update.
- The accepted-move HTTP/WebSocket payload already contained sequence, color, coordinates,
  promotion, and notation; that contract was verified and reused.

## Regression and integration checks

- `npm test`: passed, 25 tests across 6 files.
- `npm run build`: passed (TypeScript and Vite production build).
- `DATABASE_URL=... go test ./...`: passed, including database-backed integration tests.
- `E2E_BASE_URL=http://localhost:5173 npm run test:e2e -- --workers=1`: passed, all 8
  Playwright scenarios, including move synchronization/highlights, recovery, clocks, and
  responsive layout.
