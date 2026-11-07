# Implementation Plan: Online Chess

**Branch**: `001-online-chess` | **Date**: 2026-11-07 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-online-chess/spec.md`; user confirms
a web application with a Go backend and React frontend.

## Summary

Deliver a browser-based, invitation-only standard chess game for two guest players.
Use a Go server as the authoritative owner of match state and legal moves, HTTP for
match setup and recovery, and WebSockets for live updates. Persist accepted moves and
match state in PostgreSQL so reconnects restore the latest committed position. Server-
authoritative clocks use the selected 3+2, 5+0, or 10+0 control and start White's clock
when the second player joins. Build the browser client with React, TypeScript, and Vite,
served as static assets by the Go application. Put chess rules behind a small adapter to
a verified Go chess library rather than implementing the rules engine from scratch.

## Technical Context

**Language/Version**: Go, supported stable release selected when the project is initialized;
TypeScript with React for the browser client

**Primary Dependencies**: Go `net/http`; `github.com/coder/websocket`; PostgreSQL driver
`github.com/jackc/pgx`; evaluate and pin `github.com/corentings/chess/v2` behind an
application-owned chess-rules adapter; React, TypeScript, and Vite for the frontend.

**Storage**: PostgreSQL for match snapshots, seats, append-only accepted move history, and
durable clock anchors/state.

**Testing**: Go standard `testing` package and `go test ./...`; Vitest and React Testing
Library for frontend behavior; focused rules-adapter, service, persistence, and
HTTP/WebSocket integration tests; Playwright two-browser end-to-end tests. Use failing
tests first, verify the expected red failure, then implement.

**Target Platform**: Modern desktop and mobile web browsers; Go server on a supported
server platform; PostgreSQL

**Project Type**: Web application (Go backend and browser frontend)

**Performance Goals**: Match creation and joining within 60 seconds in at least 95% of
stable-connection validation attempts; accepted moves visible to both players within
2 seconds in at least 99% of a 20-move test game, per the feature spec.

**Constraints**: Server validates every move and owns canonical game state. Commit an
accepted move, resulting position, and clock update atomically. Use a persisted timestamp
anchor to calculate elapsed time; client countdowns are display-only. Resolve move/timeout
races at one serialized server-side ordering point. WebSocket messages are not durable;
reconnect recovers board and clock state from persistence. Validate browser origin and
player-seat credentials; never use an invitation token as a long-lived reconnect
credential. Initial release has no public matchmaking, user accounts, or multi-instance
coordination layer.

**Scale/Scope**: Two players per invited match; MVP targets a single application deployment
with durable PostgreSQL state. No total user or match-volume target is specified.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **TDD (NON-NEGOTIABLE)**: PASS — implementation work must start with a focused failing
  test, demonstrate the expected failure, then proceed through minimal implementation and
  green verification before refactoring.
- **Chess Rules Correctness**: PASS — use a rules adapter and verify special moves,
  check/checkmate, stalemate, and claimable versus automatic draw behavior with tests.
- **Integration Confidence**: PASS — plan includes persistence and multi-client integration
  coverage for move/clock atomicity, broadcasts, reconnect, timeout, and game completion.
- **Simplicity and Maintainability**: PASS — one Go service, one React frontend, and
  no Redis or speculative account features for the initial scope.
- **Consistent Player Experience**: PASS — browser game communicates turn, errors, and
  match status; end-to-end scenarios cover both players.

**Post-design re-check**: PASS — the data model preserves server-authoritative board and
clock state; contracts cover authenticated real-time updates, timer synchronization, and
reconnect; the quickstart defines two-client clock/move checks. The selected scope adds no
constitution exception.

## Project Structure

### Documentation (this feature)

```text
specs/001-online-chess/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    ├── http-api.md
    └── websocket-events.md
```

### Source Code (repository root)

```text
cmd/server/                 # Go HTTP/WebSocket server entry point
internal/
├── chess/                  # Rules adapter and domain tests
├── match/                  # Match lifecycle and application services
├── transport/http/         # Match, move, draw, and recovery endpoints
├── transport/realtime/     # Authenticated WebSocket sessions and broadcasts
└── store/postgres/         # PostgreSQL persistence and transaction tests
web/
├── src/
│   ├── components/         # Board, promotion, match status, and game actions
│   ├── features/match/      # Match state and HTTP/WebSocket client
│   └── tests/               # Vitest and React Testing Library tests
├── e2e/                    # Playwright two-browser scenarios
└── package.json             # Frontend scripts and dependencies
migrations/                 # PostgreSQL schema migrations
```

**Structure Decision**: Use one Go backend with internal packages separated by domain,
transport, and storage responsibilities, plus a React/TypeScript frontend built by Vite
into static assets. Serve those assets from the Go application in production; use Vite's
development proxy for local HTTP and WebSocket traffic. Avoid a second application server
or global client-state framework for the MVP.

## Complexity Tracking

No constitution violations or additional project layers are proposed.
