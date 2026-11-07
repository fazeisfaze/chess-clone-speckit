# Tasks: Move History Panel

**Input**: Requirements from `/specs/002-move-history-panel/spec.md` and current repository
source structure.

**Prerequisites**: `spec.md`. The existing `plan.md` is still an unfilled template; these
tasks use the established Go/React project structure and should be checked against a
completed plan before implementation.

**Tests**: Included to comply with the project constitution's mandatory TDD workflow. For
each behavior, run its focused test and record the expected red result before implementation.

**Organization**: Tasks are grouped by the two independently prioritized user stories.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Verify the existing backend/frontend test tooling; no new project or dependency
is required.

- [X] T001 Verify existing Go and frontend test commands in `go.mod` and `web/package.json`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: No new shared infrastructure is needed. Accepted move persistence, server
versioning, HTTP recovery, and WebSocket synchronization already exist and will be extended
within their story-specific tasks.

**Checkpoint**: After T001, User Story 1 can begin. User Story 2 follows User Story 1 because
its latest history entry must correspond to the history panel's displayed move.

---

## Phase 3: User Story 1 - Review the Move History (Priority: P1) 🎯 MVP

**Goal**: Show every accepted move in chronological order with move number, moving color,
and standard notation; provide an empty state and responsive access to earlier moves.

**Independent Test**: In frontend tests, load a match with no moves and then with multiple
accepted moves; verify empty-state behavior, move order/number/color/notation, and history
panel availability beside/below the board. Exercise the HTTP read/recovery responses to
verify that persisted accepted moves are available to both seats after reload.

### Tests for User Story 1

> Write tests first and confirm they fail because match history is not currently returned
> or rendered.

- [X] T002 [P] [US1] Add HTTP tests asserting authorized match-read and recovery responses include ordered accepted move history in `internal/transport/http/server_test.go`
- [X] T003 [P] [US1] Add match client/recovery tests for parsing and restoring move-history entries in `web/src/features/game/MatchRecovery.test.tsx`
- [X] T004 [P] [US1] Add move-list rendering tests for empty state, chronology, move numbers, colors, and notation in `web/src/features/game/GameView.test.tsx`

### Implementation for User Story 1

- [X] T005 [US1] Load complete accepted move history for authorized reads and recoveries using persisted history in `internal/store/postgres/matches.go` and `internal/transport/http/matches.go`
- [X] T006 [US1] Define the client move-history entry type and restore server-provided history in `web/src/features/match/matchClient.ts` and `web/src/features/game/useMatchSocket.ts`
- [X] T007 [US1] Add the chronological move-history panel and clear empty state beside the board in `web/src/features/game/GameView.tsx`
- [X] T008 [US1] Style a scrollable desktop history panel and a reachable narrow-screen layout without covering board controls in `web/src/style.css`

**Checkpoint**: User Story 1 must pass independently with histories matching accepted server moves after load/recovery.

---

## Phase 4: User Story 2 - Identify the Latest Move (Priority: P1)

**Goal**: Show the latest accepted move on both clients by highlighting its origin and
destination squares and marking the matching move-history entry; restore that state after
reconnect/reload and ignore stale/duplicate events.

**Independent Test**: In reducer and board tests, accept successive move events and verify
both latest squares and the corresponding list item update together; verify duplicate,
stale, rejected, and reloaded states do not produce an incorrect highlight. Confirm two
browsers converge on the same last move.

### Tests for User Story 2

> Write tests first and confirm they fail when accepted moves do not carry or retain the
> origin/destination and when history entries have no latest marker.

- [X] T009 [P] [US2] Add reducer tests for recording accepted move details and preserving state on duplicate, stale, and rejected events in `web/src/features/game/matchReducer.test.ts`
- [X] T010 [P] [US2] Add board tests for highlighting origin and destination squares in `web/src/features/game/GameBoard.test.tsx`
- [X] T011 [P] [US2] Add integration assertions that the latest accepted move is highlighted on both clients in `web/e2e/gameplay.spec.ts`

### Implementation for User Story 2

- [X] T012 [US2] Verify accepted move sequence, color, origin, destination, and notation are included in the existing move response and realtime event in `internal/transport/http/matches.go`
- [X] T013 [US2] Extend match state/reducer and snapshot restoration to retain ordered history and the latest accepted move without regressing version checks in `web/src/features/game/matchReducer.ts` and `web/src/features/game/useMatchSocket.ts`
- [X] T014 [US2] Pass latest-move squares to the board and mark the corresponding final history entry in `web/src/features/game/GameView.tsx` and `web/src/features/game/GameBoard.tsx`
- [X] T015 [US2] Add distinct latest-square and latest-history-entry highlight styles in `web/src/style.css`
- [X] T016 [US2] Verify the accepted-move HTTP/WebSocket contract includes actual move coordinates and notation in `internal/transport/http/server_test.go` and `web/src/features/game/matchReducer.test.ts`

**Checkpoint**: User Story 2 must pass focused tests and the two-client move/recovery scenario.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Verify history and latest-move behavior across the feature's full integration surface.

- [X] T017 [P] Run the complete Go test suite with `go test ./...` and database-backed integration coverage
- [X] T018 [P] Run all frontend tests and production type/build checks using `web/package.json`
- [X] T019 [P] Run applicable Playwright scenarios configured under `web/e2e/`
- [X] T020 Verify all scenarios in `specs/002-move-history-panel/spec.md`, then record RED/GREEN/regression results in `specs/002-move-history-panel/test-evidence.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 confirms existing test/build commands.
- **Foundational (Phase 2)**: No additional tasks; existing infrastructure is sufficient.
- **User Story 1 (Phase 3)**: Starts after T001; delivers history and forms the MVP.
- **User Story 2 (Phase 4)**: Depends on User Story 1's history entry model and view, so it
  follows US1 in this implementation plan.
- **Polish (Phase 5)**: Depends on both user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: Independent after setup; no dependency on latest-move highlighting.
- **User Story 2 (P1)**: Depends on US1 to associate the latest move with its visible history
  entry; board-square highlighting itself remains independently testable.

### Within Each User Story

- Tests MUST be written and run before the associated implementation (constitution TDD gate).
- US1 HTTP, client recovery, and UI tests (T002-T004) can be authored in parallel; implement
  server response before client restoration, then render and style the panel.
- US2 reducer, board, and two-client tests (T009-T011) can be authored in parallel; complete
  the move event contract before changing client reducer/event consumption.
- Do not parallelize tasks editing the same files (`GameView.tsx`, `GameBoard.tsx`,
  `matchReducer.ts`, `useMatchSocket.ts`, or `style.css`).

## Parallel Opportunities

- **US1**: T002, T003, and T004 target separate test files and can be developed in parallel.
- **US2**: T009, T010, and T011 target separate files and can be developed in parallel.
- **Polish**: T017-T019 are separate validation commands and may run concurrently after both
  stories pass their focused tests.
- The two user stories are not fully parallel because the latest entry must integrate with
  the history model introduced by US1.

## Parallel Example: User Story 1

```text
Contributor A: T002 (authorized HTTP read/recovery contract tests)
Contributor B: T003 (client history restoration tests)
Contributor C: T004 (move history panel behavior tests)
Then: T005 -> T006 -> T007 -> T008, respecting file dependencies
```

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete T001 and write/run T002-T004 to capture the expected red tests.
2. Complete T005-T008 to provide authoritative chronological history with responsive UI.
3. Stop and validate the empty state, accepted history, both seats, and reload recovery.
4. Deliver the move history panel as the first independently useful increment.

### Incremental Delivery

1. Deliver US1 and verify it independently.
2. Add US2 coordinates/highlights while retaining version and duplicate-event protections.
3. Run Go, frontend, and two-browser regression checks; record evidence.

## Format Validation

All actionable tasks use the required checkbox, sequential task ID, optional `[P]`, required
`[US1]`/`[US2]` labels in story phases, and explicit repository-relative file paths.
