# Tasks: Online Chess

**Input**: Design documents from `specs/001-online-chess/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, and `contracts/`

**Tests**: Required by the project constitution. Every behavior change starts with a
focused failing test; verify its expected failure before implementation, then verify it
passes before refactoring. Record the RED output, GREEN output, and relevant regression
results for every behavior change in `specs/001-online-chess/test-evidence.md`.

**Organization**: Tasks are grouped by user story so each story can be implemented and
validated as an increment.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Parallelizable after listed phase dependencies are met; no shared-file conflict.
- **[Story]**: User story label from `spec.md`.
- Every task names the exact file path(s) to create or update.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Initialize the Go backend, React frontend, and local development environment.

- [X] T001 Initialize the Go module and server entry point in `go.mod` and `cmd/server/main.go`, selecting an installed supported stable Go release and adding `github.com/coder/websocket`, `github.com/jackc/pgx`, and the evaluated chess-rules dependency.
- [X] T002 [P] Scaffold React, TypeScript, Vite, Vitest, React Testing Library, and Playwright dependencies/scripts in `web/package.json`, `web/index.html`, `web/tsconfig.json`, `web/vite.config.ts`, and `web/src/main.tsx`.
- [X] T003 [P] Add root `.gitignore` entries for `.env`, Go binaries, `web/node_modules/`, `web/dist/`, and frontend test output.
- [X] T004 [P] Add a PostgreSQL development service and persistent local volume in `compose.yaml`.
- [X] T005 [P] Document required local server and database variables in `.env.example` without real credentials.
- [X] T006 Add `Makefile` targets for backend tests, frontend tests, development startup, and database migrations using the paths established by this plan.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish shared configuration, persistence, server, and authenticated realtime transport.

- [X] T007 [P] Write failing configuration tests for missing required values and invalid database settings in `internal/config/config_test.go`.
- [X] T008 Implement validated environment configuration loading in `internal/config/config.go` and make `internal/config/config_test.go` pass.
- [X] T009 [P] Write failing PostgreSQL migration tests for applying the schema and safely detecting an already-applied migration in `internal/store/postgres/migrate_test.go`.
- [X] T010 Implement the SQL migration runner in `internal/store/postgres/migrate.go` and schemas in `migrations/001_initial.sql` and `migrations/002_move_check_flag.sql`; enforce match statuses `waiting`, `active`, `completed`, `unavailable`, seat colors `white` or `black` with at most one seat of each color, time controls `3+2`, `5+0`, or `10+0`, Draw Offer statuses `pending`, `accepted`, or `declined`, and persist each move's `gives_check` state.
- [X] T011 [P] Write failing HTTP server tests for startup configuration and `GET /healthz` behavior in `internal/transport/http/server_test.go`.
- [X] T012 Implement HTTP server construction, health response, graceful shutdown, and shared error responses in `internal/transport/http/server.go` and `cmd/server/main.go`; make `internal/transport/http/server_test.go` pass.
- [X] T013 [P] Write failing WebSocket tests for origin validation, authentication timeout, no data before authentication, and message-size rejection in `internal/transport/realtime/session_test.go`.
- [X] T014 Implement authenticated WebSocket session setup and origin/message limits in `internal/transport/realtime/session.go`; make `internal/transport/realtime/session_test.go` pass.
- [X] T015 [P] Write a failing integration test that the Go server serves the built React entry point and static assets in `tests/integration/static_site_test.go`.
- [X] T016 Serve Vite build output from the Go HTTP server and configure the local Vite HTTP/WebSocket proxy in `internal/transport/http/static.go` and `web/vite.config.ts`; make `tests/integration/static_site_test.go` pass.

**Checkpoint**: Go and React projects start locally; PostgreSQL migrations, health checks,
and authenticated realtime transport are ready for story work.

---

## Phase 3: User Story 1 - Start an Online Match (Priority: P1) 🎯 MVP increment

**Goal**: A host creates an invitation match, chooses a clock preset, and a guest joins;
the game activates with White's clock running.

**Independent Test**: Create a match in browser A, join it in browser B, and verify both
players see opposing colors, the same initial board and selected clock preset, and White's
clock starts only after the second seat joins.

### Tests for User Story 1

> Write each test first and verify the expected behavior failure before implementation.

- [X] T017 [P] [US1] Write failing domain tests for match creation, allowed presets `3+2`, `5+0`, `10+0`, White seat assignment, and waiting status in `internal/match/create_test.go`.
- [X] T018 [P] [US1] Write failing integration tests for creating a match, joining once by invitation, and rejecting a third player in `tests/integration/match_lifecycle_test.go`.
- [X] T019 [P] [US1] Write failing activation tests proving White's clock starts on second-seat join, Black's clock remains paused, and no clock runs while waiting in `internal/match/activation_test.go`.
- [X] T020 [P] [US1] Write failing React tests for preset selection, invitation display, join status, assigned color, and the same initial board/clock view for both seats in `web/src/features/match/MatchLobby.test.tsx`.
- [X] T021 [P] [US1] Write a failing two-browser invitation flow test in `web/e2e/invitation.spec.ts`.

### Implementation for User Story 1

- [X] T022 [US1] Implement match creation, random invitation/recovery credentials, and validation in `internal/match/create.go` and `internal/match/service.go`; allow only the three documented presets and create a White seat with a waiting match.
- [X] T023 [US1] Implement match and seat persistence in `internal/store/postgres/matches.go`; enforce invitation secrets are high-entropy, stored only as digests, and never reused as recovery credentials.
- [X] T024 [US1] Implement create/join and authenticated match-state HTTP handlers in `internal/transport/http/matches.go` using the request/response contract in `contracts/http-api.md`; validate requests, call match services, assign the opposite seat, and map full/unavailable match errors without owning clock activation.
- [X] T025 [US1] Implement the match activation service and clock initialization in `internal/clock/clock.go` and `internal/match/activate.go`; atomically set the match active, initialize both preset times, and start White when the second player joins.
- [X] T026 [US1] Implement match lobby, preset picker, invitation link, and join feedback in `web/src/features/match/MatchLobby.tsx` and `web/src/features/match/matchClient.ts`; make `web/src/features/match/MatchLobby.test.tsx` pass.
- [X] T027 [US1] Integrate match creation, invitation joining, and authenticated host-state polling into `web/src/App.tsx` and `web/src/features/match/MatchLobby.tsx`; verify `web/e2e/invitation.spec.ts` passes in two independent browser contexts.

**Checkpoint**: A guest match can be created and joined independently; clock configuration
and initial activation are visible to both players.

---

## Phase 4: User Story 2 - Play a Standard Chess Game (Priority: P1)

**Goal**: Players make legal standard chess moves with synchronized board state and clocks,
including increments and server-adjudicated timeouts.

**Independent Test**: Join an active match in two browser contexts, exercise legal and
illegal moves, special rules, simultaneous/stale commands, increments, and timeout; verify
both clients converge on the same move, turn, clock, and result.

### Tests for User Story 2

> Write each test first and verify the expected behavior failure before implementation.

- [X] T028 [P] [US2] Create project-maintained standard-chess conformance fixtures with expected legal moves and outcomes in `internal/chess/testdata/conformance_positions.json`, then write failing adapter tests for every fixture in `internal/chess/rules_test.go`; cover ordinary legal/illegal moves, check, checkmate, stalemate, castling, en passant, promotion, and recognized draw rules.
- [X] T029 [P] [US2] Write failing clock transition tests for elapsed-time debit, 3+2 increment exactly once, 5+0/10+0 no increment, and active-color switching in `internal/clock/transition_test.go`.
- [X] T030 [P] [US2] Write failing move service tests for wrong turn, stale/duplicate version, illegal moves, and move-versus-timeout races in `internal/match/moves_test.go`.
- [X] T031 [P] [US2] Write failing HTTP/WebSocket contract tests for accepted moves, rejection responses, ordered versioned events, and clock snapshots in `tests/integration/move_sync_test.go`.
- [X] T032 [P] [US2] Write failing React tests for board selection, legal move submission, promotion choices, turn display, and clock rendering in `web/src/features/game/GameBoard.test.tsx`.
- [X] T033 [P] [US2] Write a failing two-browser test for synchronized moves, 3+2 increments, and illegal-move rejection in `web/e2e/gameplay.spec.ts`; verify both clients converge on each accepted move and clock update.
- [X] T034 [P] [US2] Write failing timeout worker tests for expiration without incoming player commands and consistent timeout result in `internal/clock/worker_test.go`.

### Implementation for User Story 2

- [X] T035 [US2] Implement the application-owned chess rules adapter in `internal/chess/rules.go`; accept a move only when legal in the prior position and permit promotion only to queen, rook, bishop, or knight.
- [X] T036 [US2] Implement persisted clock transitions in `internal/clock/transition.go`; debit only the active player's elapsed time, apply the selected increment after a legal accepted move, and switch the active color.
- [X] T037 [US2] Implement authoritative move processing in `internal/match/moves.go` and `internal/store/postgres/matches.go`; commit move history, resulting position, check status, clock state, and version atomically, and reject expired, stale, duplicate, illegal, out-of-turn, and post-game moves without changing board state.
- [X] T038 [US2] Implement HTTP move handling in `internal/transport/http/matches.go` and realtime move commands/events in `internal/transport/realtime/session.go` according to `contracts/http-api.md` and `contracts/websocket-events.md`.
- [X] T039 [US2] Implement versioned clock snapshots and event fan-out in `internal/transport/realtime/hub.go` and `internal/transport/http/matches.go`; publish only after commit and require clients to resync on version gaps.
- [X] T040 [US2] Implement the accessible React chessboard, promotion chooser, check/turn status display, and clock display in `web/src/features/game/GameBoard.tsx` and `web/src/features/game/ClockDisplay.tsx`; make `web/src/features/game/GameBoard.test.tsx` pass.
- [X] T041 [US2] Implement React match event state and WebSocket synchronization in `web/src/features/game/useMatchSocket.ts` and `web/src/features/game/matchReducer.ts`; derive display countdowns from server timestamps and never send client time as authority.
- [X] T042 [US2] Implement timeout adjudication and notification in `internal/clock/worker.go`; serialize timeout checks with moves and apply a draw when the opponent cannot possibly checkmate by any legal sequence.
- [X] T043 [US2] Integrate the board, move controls, and realtime clock state into `web/src/features/game/GameView.tsx` and `web/src/App.tsx`; make `web/e2e/gameplay.spec.ts` and `tests/integration/move_sync_test.go` pass.

**Checkpoint**: Both clients can complete a legal timed chess game and agree on every
accepted move, clock transition, rejection, and terminal result.

---

## Phase 5: User Story 3 - Recover or End a Match (Priority: P2)

**Goal**: Players recover an active game with its running clock, resign, or offer and
respond to draw actions.

**Independent Test**: Interrupt a player's connection during a timed match and reconnect;
then separately verify resignation, draw offer acceptance/decline, and valid/invalid draw
claims while preserving authoritative board and clock state.

### Tests for User Story 3

> Write each test first and verify the expected behavior failure before implementation.

- [X] T044 [P] [US3] Write failing recovery tests for valid/invalid seat credentials, current board/version, and server-derived remaining clock values in `tests/integration/recovery_test.go`.
- [X] T045 [P] [US3] Write failing termination tests for resignation, draw offer acceptance/decline, rule-based draw claims, and rejection after completion in `internal/match/termination_test.go` and `tests/integration/termination_test.go`.
- [X] T046 [P] [US3] Write failing React tests for reconnect loading, opponent connection status, game result, and draw offer/draw-claim actions in `web/src/features/game/MatchRecovery.test.tsx`.
- [X] T047 [P] [US3] Write a failing browser test for reconnect during a running clock and restoration of the latest accepted state in `web/e2e/recovery.spec.ts`.
- [X] T048 [P] [US3] Write a failing browser test for resignation, draw acceptance, and draw decline in `web/e2e/termination.spec.ts`.

### Implementation for User Story 3

- [X] T049 [US3] Implement seat recovery in `internal/match/service.go` and `internal/transport/http/matches.go`; validate the separate high-entropy recovery credential and derive current time from the persisted server anchor without pausing the clock.
- [X] T050 [US3] Implement resign, draw offer/response, and chess draw-claim handling in `internal/match/termination.go`, `internal/store/postgres/matches.go`, and `internal/transport/http/matches.go`; only the opposing seat can resolve a pending draw offer and terminal games reject further actions.
- [X] T051 [US3] Add recovery snapshots and connection status events to `internal/transport/realtime/session.go` and `internal/transport/http/matches.go`; send current board, version, result, and clock state before resuming live events.
- [X] T052 [US3] Implement React recovery, connection status, resign, draw offer, draw-claim, and result UI in `web/src/features/match/MatchLobby.tsx`, `web/src/features/game/GameActions.tsx`, and `web/src/features/game/GameView.tsx`; make `web/src/features/game/MatchRecovery.test.tsx` pass.
- [X] T053 [US3] Integrate recovery and termination controls into `web/src/features/game/GameView.tsx`; make `web/e2e/recovery.spec.ts` and `web/e2e/termination.spec.ts` pass.

**Checkpoint**: Reconnected players resume the latest durable game and clock state; every
match ending is consistent for both players.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Verify accessibility, security, and all acceptance criteria across stories.

- [X] T054 [P] Add security regression tests for invitation/recovery credential redaction, WebSocket origin validation, rate limits, and unauthenticated state disclosure in `tests/security/session_security_test.go` and `internal/transport/realtime/session_test.go`.
- [X] T055 Harden credential handling, request limits, and WebSocket error behavior in `internal/transport/http/ratelimit.go`, `internal/transport/http/matches.go`, and `internal/transport/realtime/session.go`; make security tests pass.
- [X] T056 [P] Add keyboard and screen-reader interaction tests for the chess grid and live status announcements in `web/src/features/game/accessibility.test.tsx`.
- [X] T057 Implement keyboard-operable square selection and accessible turn, check, error, and clock announcements in `web/src/features/game/GameBoard.tsx`, `web/src/features/game/GameView.tsx`, and `web/src/features/game/ClockDisplay.tsx`; make accessibility tests pass.
- [X] T058 [P] Add and run acceptance-metric checks for SC-001, SC-002, SC-004, and SC-006 in `tests/acceptance/metrics_test.go` and `web/e2e/acceptance-metrics.spec.ts`; assert at least 19/20 joins within 60 seconds, at least 396/400 move updates within 2 seconds, at least 19/20 reconnect trials initiated within 5 minutes restore the latest position and clock state, and all 20 clock displays stay within 1 second with identical timeout results.
- [ ] T059 [P] Create and execute a first-use usability protocol with 10 participants, recording whether at least 9 can start a match, make a legal move, and identify the next turn without help in `specs/001-online-chess/usability-validation.md`.
- [X] T060 Run Go tests, frontend tests, browser end-to-end tests, and formatting checks; record results in `specs/001-online-chess/quickstart.md` and RED/GREEN/regression evidence in `specs/001-online-chess/test-evidence.md`.
- [X] T061 Validate all documented local setup and two-player timed-match scenarios in `specs/001-online-chess/quickstart.md` and fix any discrepancies in the referenced files.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies; establishes Go, React/Vite, PostgreSQL, and local configuration.
- **Foundational (Phase 2)**: Depends on Setup; blocks all user stories with shared persistence, server, and realtime transport.
- **User Story 1 (Phase 3)**: Depends on Foundational; creates invited matches and starts the selected clock.
- **User Story 2 (Phase 4)**: Depends on User Story 1's active match lifecycle; implements legal moves and synchronized board/clock behavior.
- **User Story 3 (Phase 5)**: Depends on User Stories 1 and 2; recovers and terminates active games.
- **Polish (Phase 6)**: Depends on all selected stories.

### User Story Dependencies

- **US1 (P1)**: Can begin after Phase 2; establishes the match that later stories exercise.
- **US2 (P1)**: Begins after US1's match creation/join interfaces exist; tests can use isolated active-match fixtures.
- **US3 (P2)**: Begins after US1 and US2 provide active matches, accepted moves, and clock state.

### Parallel Opportunities

- After Phase 1, configuration tests, migration tests, server tests, and WebSocket tests (T007, T009, T011, T013) can be authored in parallel because they target separate files.
- Within US1, domain, integration, activation-clock, and React tests (T017–T020) can be authored in parallel after Phase 2.
- Within US2, chess rules, clock transition, move service, transport, and React tests (T028–T034) can be authored in parallel; implementation tasks then follow their respective tests.
- Within US3, recovery, termination, and UI tests (T044–T048) can be authored in parallel.
- Do not parallelize tasks that update the same file or begin a story before its stated phase dependency is complete.

## Parallel Example: User Story 1

After Phase 2, independently author the red tests in T017 (`internal/match/create_test.go`),
T018 (`tests/integration/match_lifecycle_test.go`), T019
(`internal/match/activation_test.go`), and T020
(`web/src/features/match/MatchLobby.test.tsx`). Verify each fails for the expected missing
behavior before starting its corresponding implementation task.

## Implementation Strategy

### MVP First

1. Complete Phase 1 Setup and Phase 2 Foundational.
2. Complete US1 as the first demonstrable increment: create/join an invited match with a
   shared clock preset and White's clock starting on activation.
3. A usable playable MVP requires both US1 and US2; stop after US2 to validate legal moves,
   synchronized clocks, and complete timed games before adding recovery and draw/resign
   flows from US3.

### Incremental Delivery

1. Setup + Foundational → service can start and persist data.
2. US1 → two guests can create/join a timed match.
3. US2 → both guests can play and finish a standard timed chess game.
4. US3 → players can reconnect and use draw/resignation flows.
5. Polish → validate accessibility, security, and the full quickstart.

## Notes

- Every behavior task follows Red–Green–Refactor: its preceding test task must be observed failing for the expected reason before implementation begins.
- Record each behavior change's RED, GREEN, and regression results in `specs/001-online-chess/test-evidence.md`, including the test command and observed outcome.
- [P] tasks touch separate files and have no unfinished dependency within their phase.
- Story labels map directly to User Stories 1–3 in `spec.md`.
- Commit after each logical task group; keep the working tree scoped to this project repository.
