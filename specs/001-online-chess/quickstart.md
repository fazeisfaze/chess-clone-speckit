# Quickstart: Online Chess Validation

This guide covers local setup and end-to-end validation for the implemented web application.

## Prerequisites

- Supported stable Go toolchain.
- Supported Node.js package manager and browser-test prerequisites.
- PostgreSQL service and the project's configured local database.
- Two modern browser sessions (separate profiles or devices are suitable).
- The application configuration for database connection and local server address.

## Local setup

1. Copy `.env.example` to `.env`, adjust local values if needed, and export them in the
   current shell with `set -a; . ./.env; set +a`.
2. Start PostgreSQL with `make dev-db`.
3. Apply migrations with `make migrate`.
4. Start the Go server with `make backend-dev` in one terminal. The server applies pending
   migrations on startup; `make migrate` applies them and exits without starting the server.
5. From `web/`, run `npm ci` once, then `npm run dev` in another terminal. The Vite server
   proxies HTTP and WebSocket traffic to Go.
6. For Go-served static frontend validation, run `npm run build` in `web/` and open the Go
   server origin directly. Invitation links remain on the same origin.

## Automated checks

1. With `DATABASE_URL` exported, run `go test ./...` from the repository root.
   - **Expected**: focused domain, adapter, storage, transport, security, and integration tests pass.
2. Run `go test ./tests/acceptance -count=1`.
   - **Expected**: 20 match attempts, 400 move broadcasts, 20 seat recoveries, and 20
     timeout outcomes meet the measured thresholds in SC-001, SC-002, SC-004, and SC-006.
3. From `web/`, install the frontend dependencies and run the configured frontend unit
   test script.
   - **Expected**: Vitest and React Testing Library checks pass.
4. From `web/`, run `npm run build` and `npm run test:e2e`.
   - **Expected**: both browser clients complete the scenarios below without diverging state.
5. To verify the UI served directly by Go, run
   `E2E_BASE_URL=http://127.0.0.1:8080 npm run test:e2e -- e2e/invitation.spec.ts e2e/gameplay.spec.ts`.
6. Run the clock accuracy browser metric with `npm run test:e2e -- e2e/acceptance-metrics.spec.ts`.
   - **Expected**: both clients' four clock readings per match remain within one second of
     the corresponding server snapshot in 20 timed matches.

Per the project constitution, every behavior change must first be covered by a focused test
that fails for the expected reason. Record the failing output before implementation, passing
output afterward, and regression results in [test-evidence.md](test-evidence.md).

## Two-player match validation

1. Start the server and apply the documented database migrations.
2. In browser session A, create a match and retain the invitation and seat recovery details.
3. In browser session B, join using the invitation.
4. Verify both sessions show the initial board, opposing colors, and the same active turn.
5. Create test matches with each preset (3+2, 5+0, and 10+0); verify both clients show the
   selected control and the same initial clock values.
6. Verify White's clock starts when the second player joins; Black's clock stays paused.
7. Make a legal move in A; verify B shows the same move, next turn, and updated clock within
   2 seconds. Verify the 3+2 increment is applied to the mover exactly once.
8. Attempt an illegal, out-of-turn, duplicate, and late move; verify none can alter the
   board or improperly change a clock.
9. Exercise castling, en passant, promotion, non-terminal check, checkmate, stalemate, and
   draw behavior using the documented test positions; verify both clients see check status.
10. Disconnect and reconnect one session during an active clock; verify it recovers the
    latest position and correct server-derived remaining times without pausing the game.
11. Allow a clock to expire with both clients connected; verify both receive the same result
    and subsequent moves are rejected. Verify timeout is a draw if the opponent has no
    possible legal sequence that can deliver checkmate.
12. In separate matches, verify resignation, draw acceptance/decline, and a threefold
    repetition claim when it is available.

**Expected outcome**: both players see identical accepted board state and clock control;
client clocks stay within 1 second of server time; invalid actions do not mutate board or
clock; timeout is resolved identically; reconnect returns the latest committed version and
correct running clock.

## Validation record

Validated on 2026-11-07 with local PostgreSQL:

- `DATABASE_URL=... go test ./... -count=1` — passed, including PostgreSQL integration and
  security tests.
- `DATABASE_URL=... go test ./tests/acceptance -count=1` — passed: 20/20 matches joined,
  400/400 move updates arrived within 2 seconds, 20/20 recoveries succeeded, and 20/20
  timeout outcomes agreed between both realtime clients.
- `npm test` and `npm run build` — passed.
- `npm run test:e2e` — all 7 Chromium scenarios passed; the separate 20-match clock
  accuracy scenario passed.
- The responsive layout check verifies a board/control sidebar on desktop and a stacked
  layout on mobile; all 8 current Chromium scenarios pass.
- `E2E_BASE_URL=http://127.0.0.1:8080 npm run test:e2e -- e2e/invitation.spec.ts e2e/gameplay.spec.ts`
  — both Go-served frontend scenarios passed, including the second player's move.
- The first-use 10-participant usability study remains pending; see
  [usability-validation.md](usability-validation.md).
