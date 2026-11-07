# Quickstart: Online Chess Validation

This guide describes the planned end-to-end validation flow. The repository does not yet
contain implementation code or a Go module, so these commands become runnable after the
implementation phase initializes the project.

## Prerequisites

- Supported stable Go toolchain.
- Supported Node.js package manager and browser-test prerequisites.
- PostgreSQL service and the project's configured local database.
- Two modern browser sessions (separate profiles or devices are suitable).
- The application configuration for database connection and local server address.

## Automated checks

1. From the repository root, run `go test ./...`.
   - **Expected**: focused domain, adapter, storage, transport, and integration tests pass.
2. From `web/`, install the frontend dependencies and run the configured frontend unit
   test script.
   - **Expected**: Vitest and React Testing Library checks pass.
3. From `web/`, run `npm run test:e2e` after the implementation establishes the browser
   test package.
   - **Expected**: both browser clients complete the scenarios below without diverging state.

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
12. In separate matches, verify resignation, draw acceptance, and draw decline.

**Expected outcome**: both players see identical accepted board state and clock control;
client clocks stay within 1 second of server time; invalid actions do not mutate board or
clock; timeout is resolved identically; reconnect returns the latest committed version and
correct running clock.
