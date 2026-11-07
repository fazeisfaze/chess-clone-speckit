# Implementation Plan: Chess UI Polish

**Branch**: `003-chess-ui-polish` | **Date**: 2026-11-07 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/003-chess-ui-polish/spec.md`

## Summary

Polish the existing chess UI with a clear invitation-copy action and consistent
side-based piece coloring. Use the existing invitation value and browser clipboard
capability, with explicit success/failure feedback. Preserve current chess, match, and
invitation behavior; no backend or persistent-data changes are expected.

## Technical Context

**Language/Version**: TypeScript, React 19; CSS; existing Go backend is unchanged.

**Primary Dependencies**: Existing React/Vite frontend and browser Clipboard API; no new
dependencies planned.

**Storage**: N/A; invitation link remains transient client state and piece ownership is
derived from the current position.

**Testing**: Vitest, React Testing Library, Playwright, and `npm run build`/TypeScript
checks. Begin behavior changes with focused failing tests per constitution.

**Target Platform**: Supported modern desktop and mobile web browsers.

**Project Type**: Web application; this feature changes only the browser presentation and
interaction layer.

**Performance Goals**: Copy feedback appears within 2 seconds on supported browsers;
rendering changes do not perceptibly delay board updates.

**Constraints**: Do not alter chess rules, server contracts, match lifecycle, or invitation
authorization. Clipboard failures must not report success. Side colors must not depend on
square color or board orientation. Do not expose credentials beyond the existing invitation
link behavior.

**Scale/Scope**: One waiting-lobby copy interaction and the existing 64-square game board;
no new service or data store.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **TDD (NON-NEGOTIABLE)**: PASS — add focused tests first for clipboard success/failure
  and piece side-color classes; demonstrate red before implementation, then green and
  regression results.
- **Chess Rules Correctness**: PASS — no rules or game-state behavior changes.
- **Integration Confidence**: PASS — test lobby clipboard behavior at component level and
  run relevant browser/E2E coverage where practical.
- **Simplicity and Maintainability**: PASS — reuse existing React components, state, CSS,
  and browser APIs; no new dependency or architectural layer.
- **Consistent Player Experience**: PASS — provide accessible, truthful copy feedback and
  stable side identification across orientations and responsive views.

## Project Structure

### Documentation (this feature)

```text
specs/003-chess-ui-polish/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    └── ui.md
```

### Source Code (repository root)

```text
web/src/features/match/
├── MatchLobby.tsx
└── MatchLobby.test.tsx
web/src/features/game/
├── GameBoard.tsx
├── GameBoard.test.tsx
└── GameView.tsx
web/src/style.css
web/e2e/
```

**Structure Decision**: Keep the feature within the existing React frontend, matching
responsibilities already present in the lobby and board. Backend and persistence paths are
not involved.

## Complexity Tracking

No constitution violations or additional project layers are proposed.

## Post-design Constitution Re-check

PASS — design adds no new data persistence, service, dependency, or rules behavior. UI
contracts and validation scenarios preserve clear user feedback and use test-first changes.
