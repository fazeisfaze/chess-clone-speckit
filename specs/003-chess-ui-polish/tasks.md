# Tasks: Chess UI Polish

**Input**: Design documents from `/specs/003-chess-ui-polish/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/ui.md`,
`quickstart.md`

**Tests**: Included because the project constitution mandates test-driven development.
For each behavior, run the focused test and capture the expected failure before changing
implementation.

## Phase 1: Setup

**Purpose**: Confirm the existing frontend test/build environment; no project initialization
or new dependency is needed.

- [X] T001 Verify the existing scripts and test dependencies in `web/package.json` before starting story work

---

## Phase 2: Foundational

**Purpose**: No new shared infrastructure is required. The existing React frontend, Vitest,
and browser test setup support both independent user stories.

**Checkpoint**: After T001, either story can be implemented independently.

---

## Phase 3: User Story 1 - Copy an Invitation Link (Priority: P1) 🎯 MVP

**Goal**: Let a host copy the complete invitation from the waiting lobby and receive
accessible, truthful success or failure feedback without navigating away.

**Independent Test**: In `web/src/features/match/MatchLobby.test.tsx`, verify the action is
absent when no invitation exists, copies the exact link on success, keeps the lobby visible,
announces success only on resolved clipboard write, and announces failure when clipboard
access is unavailable or rejected.

### Tests for User Story 1

> Write and run these tests first; confirm they fail for the missing/incorrect interaction.

- [X] T002 [US1] Add clipboard success, failure, absent-link, and no-navigation tests in `web/src/features/match/MatchLobby.test.tsx`

### Implementation for User Story 1

- [X] T003 [US1] Replace the waiting lobby's raw invitation anchor with a labeled copy button and accessible truthful feedback in `web/src/features/match/MatchLobby.tsx`
- [X] T004 [US1] Style the invitation copy action and its feedback for the existing lobby layout in `web/src/style.css`

**Checkpoint**: User Story 1 must pass its focused tests independently before proceeding.

---

## Phase 4: User Story 2 - Distinguish Piece Colors Consistently (Priority: P1)

**Goal**: Ensure White and Black pieces remain visually distinct on every square color and
from either board orientation, including after board-position changes.

**Independent Test**: In `web/src/features/game/GameBoard.test.tsx`, render both colors on
light/dark squares from White and Black perspectives and assert the side-specific classes;
include a changed position with promoted material where supported by the fixture.

### Tests for User Story 2

> Write and run these tests first; confirm they fail if ownership styling depends on square
> or orientation.

- [X] T005 [P] [US2] Add side-color rendering tests for both orientations and square colors in `web/src/features/game/GameBoard.test.tsx`

### Implementation for User Story 2

- [X] T006 [US2] Verify each rendered piece uses side-derived styling independently of square and orientation in `web/src/features/game/GameBoard.tsx`
- [X] T007 [US2] Verify existing White/Black contrast styles remain distinct via the side classes in `web/src/style.css`

**Checkpoint**: User Story 2 must pass its focused tests independently; verify changed
positions retain side-based presentation.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: Run shared validation and confirm the documented scenarios remain actionable.

- [X] T008 [P] Run all frontend unit tests and production type/build checks using scripts in `web/package.json`
- [X] T009 [P] Run the applicable browser end-to-end scenarios configured under `web/e2e/`
- [X] T010 Verify the final behavior against `specs/003-chess-ui-polish/quickstart.md` and update `specs/003-chess-ui-polish/test-evidence.md` with RED/GREEN/regression results

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 confirms existing test tooling; no feature dependencies.
- **Foundational (Phase 2)**: No tasks; the existing frontend infrastructure is sufficient.
- **User Stories (Phases 3-4)**: Both can begin after T001 and can proceed independently.
- **Polish (Phase 5)**: Depends on both story implementations being complete.

### User Story Dependencies

- **User Story 1 (P1)**: Independent; suggested MVP because the copy-link action is a
  self-contained user-visible improvement.
- **User Story 2 (P1)**: Independent of User Story 1; can be implemented in parallel.

### Within Each User Story

- Write and run its focused tests before implementation (TDD constitution requirement).
- For User Story 1, complete T002 before T003; T004 may be done after the button structure
  exists.
- For User Story 2, complete T005 before T006; T007 follows the rendered class contract.
- Run focused story tests at each checkpoint before cross-cutting validation.

## Parallel Opportunities

- T002 and T005 modify separate test files and can be authored in parallel after T001.
- After those red tests are recorded, User Story 1 (T003-T004) and User Story 2 (T006-T007)
  touch separate components and can proceed in parallel. Both modify `web/src/style.css`,
  so coordinate or combine CSS edits to avoid conflicts.
- T008 and T009 are separate validation activities and can run in parallel after both stories
  pass focused tests.

## Parallel Example: User Stories

```text
Contributor A: T002 -> T003 -> T004 (invitation copy)
Contributor B: T005 -> T006 -> T007 (piece colors)
After both: T008 and T009, then T010
```

## Implementation Strategy

### MVP First (User Story 1)

1. Complete T001.
2. Complete T002 and record the expected red result.
3. Implement T003-T004 and verify the lobby tests pass.
4. Stop and validate the invitation-copy story independently; it can be demonstrated as
   the first increment.

### Incremental Delivery

1. Add User Story 1 and validate it independently.
2. Add User Story 2 and validate board colors independently.
3. Run cross-cutting unit, build, and browser checks; record evidence.

## Format Validation

All actionable tasks use the required checkbox, sequential task ID, optional `[P]`, required
story labels in user-story phases, and an explicit file path in each description.
