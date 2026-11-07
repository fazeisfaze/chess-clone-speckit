# Research: Chess UI Polish

## Invitation copy interaction

- **Decision**: Use the browser's asynchronous clipboard write capability from an explicit
  user action. Show a concise accessible success message only after the write resolves;
  show a failure message if it rejects or is unavailable. Do not navigate when copying.
- **Rationale**: Clipboard writing is a browser-native operation that works with user
  activation and secure-context policies, and promise resolution provides a truthful success
  signal. This meets the requirement without adding dependencies or exposing the URL as an
  interactive navigation link.
- **Alternatives considered**: Continue rendering the invite URL as a link (navigates rather
  than copying); add a clipboard library (unnecessary for this single action); silently
  report success before the clipboard operation resolves (misleading on denial).

## Piece side presentation

- **Decision**: Continue deriving ownership from the existing piece code and apply
  side-specific presentation classes independent of square styling and board orientation.
  Verify all represented piece types in both perspectives with component tests.
- **Rationale**: The current board already maps piece codes to White/Black and emits
  `piece-white`/`piece-black` classes. Styling those ownership classes is simpler and
  preserves the separation between chess position and display orientation.
- **Alternatives considered**: Infer piece color from square background (incorrect and
  fragile); use separate board implementations per perspective (duplicative); introduce an
  external chessboard UI package (unnecessary scope and dependency).

## Validation approach

- **Decision**: Use Vitest and React Testing Library for interaction and rendered class
  behavior, existing Playwright coverage for relevant browser flows, and the project build
  for type checking.
- **Rationale**: These are the project's established frontend validation tools and directly
  exercise user-observable behavior.
- **Alternatives considered**: Add a new test framework or rely only on manual inspection;
  neither is warranted for the current application.

No unresolved technical clarifications remain.
