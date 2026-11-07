# Quickstart: Chess UI Polish Validation

## Prerequisites

- Node.js/npm installed for the web project.
- The repository's web dependencies installed (`cd web && npm install`).
- For browser end-to-end checks, the application and its configured test services must be
  available as described by the existing Playwright setup.

## Validate invitation copy

1. Run focused lobby tests: `cd web && npx vitest run src/features/match/MatchLobby.test.tsx`.
2. Verify the host waiting state offers a descriptive copy button and no copy button appears
   before an invitation exists.
3. In a supported secure browser context, create a match and activate the copy button.
4. Paste into a temporary text field and verify the exact invitation link is present, the
   lobby remains visible, and success feedback is announced.
5. Deny or stub clipboard access and activate the button; verify an accessible failure
   appears and no success is announced.

## Validate piece colors

1. Run focused board tests: `cd web && npx vitest run src/features/game/GameBoard.test.tsx`.
2. Check representative white and black pieces on both square colors and in both player
   orientations; verify side presentation follows the piece and not the square.
3. Verify a changed position containing a promotion still uses the promoted piece's side.

## Regression checks

- Run all web unit tests: `cd web && npm test`.
- Run the frontend type-check and production build: `cd web && npm run build`.
- Run applicable browser scenarios: `cd web && npm run test:e2e`.

Expected outcome: all automated checks pass; clipboard success/failure behavior is truthful;
the invitation is copied without navigation; and piece colors remain side-consistent.
For full contract details, see [contracts/ui.md](contracts/ui.md); for the transient data
projection, see [data-model.md](data-model.md).
