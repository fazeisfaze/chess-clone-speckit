# Research: Online Chess

## Go HTTP and real-time transport

- **Decision**: Use Go's `net/http` for HTTP endpoints and `github.com/coder/websocket`
  for WebSocket connections. WebSockets carry notifications and player actions, but are
  not the source of truth. Authenticate the player's seat, validate every action on the
  server, and broadcast only after a successful commit.
- **Rationale**: `net/http` is sufficient for a small service. A maintained WebSocket
  package avoids implementing the protocol. The server-authoritative approach prevents
  clients from deciding legality or game outcomes.
- **Alternatives considered**: Implementing WebSockets manually was rejected as unnecessary
  protocol complexity. Treating the socket as authoritative was rejected because sockets
  are transient and cannot by themselves support reliable reconnect recovery.
- **References**: [Go net/http](https://pkg.go.dev/net/http),
  [coder/websocket](https://github.com/coder/websocket).

## Durable match state and concurrency

- **Decision**: Use PostgreSQL through `github.com/jackc/pgx`. Persist a match's accepted
  move history and resulting position together in a transaction. Version each position
  and reject stale/duplicate commands. Serialize commands per match in the service and
  enforce state transitions in database transactions.
- **Rationale**: Durable snapshots and move history allow recovery after a disconnect or
  process restart. Atomic persistence avoids disagreement between a move log and current
  position. Database concurrency controls remain necessary even with in-process
  serialization.
- **Alternatives considered**: SQLite is viable for a local prototype, but PostgreSQL is
  the stronger default for concurrent matches and future deployment growth. Redis-based
  coordination is deferred until multiple application instances require cross-instance
  fanout or match ownership.
- **References**: [pgx](https://github.com/jackc/pgx),
  [PostgreSQL transaction isolation](https://www.postgresql.org/docs/current/transaction-iso.html),
   [PostgreSQL explicit locking](https://www.postgresql.org/docs/current/explicit-locking.html).

## Server-authoritative chess clocks

- **Decision**: Persist each clock's remaining milliseconds, active color, selected initial
  time/increment, and a server timestamp anchor. Start White when the second seat joins.
  Serialize move and timeout decisions for a match; calculate elapsed time on the server,
  add increment only after a legal accepted move, switch the active clock, and commit the
  move, board, clock state, result, and version atomically.
- **Rationale**: A persisted anchor avoids a write every second and lets the server recover
  the running clock after disconnects or process restarts. A single authoritative event
  time and match lock determine races consistently. Expiration must also be checked when
  processing commands/state and by a timeout worker; a timer callback alone can run late.
- **Client synchronization**: Send remaining times, active color, and `as_of` server time
  in snapshots and versioned events. Clients estimate server-time offset and render the
  display locally using a monotonic browser clock, correcting from newer snapshots. Never
  accept a client countdown or timestamp as authority. Reconnect receives a fresh snapshot.
- **Alternatives considered**: Persisting every second adds write load without improving
  authority. Trusting client clocks is insecure and vulnerable to clock skew. In-memory
  timers cannot recover after process restart. Broadcasting before commit can show an
  unpersisted move; commit first and consider an outbox if delivery across crashes requires
  stronger guarantees.
- **Race handling**: After acquiring the match row lock, use PostgreSQL `clock_timestamp()`
  as the authoritative event time and compare it with the active clock deadline. If time
  has expired, record the timeout and reject the move; otherwise validate the move, subtract
  elapsed time, apply increment, and switch clocks. Timeout processing uses the same lock
  and deadline check. A timeout worker publishes expiration when no command is arriving;
  requests also check deadlines so a late worker wakeup cannot accept a late move.
- **References**: Go [`time`](https://pkg.go.dev/time#hdr-Monotonic_Clocks) documents
  monotonic clock readings; PostgreSQL
  [`clock_timestamp()`](https://www.postgresql.org/docs/current/functions-datetime.html)
  supplies wall-clock time and
  [explicit locking](https://www.postgresql.org/docs/current/explicit-locking.html) supports
  serializing updates; [FIDE Laws of Chess, Article 6](https://handbook.fide.com/chapter/e012023)
  describes chess clocks and increments.

## Chess rules engine

- **Decision**: Evaluate `github.com/corentings/chess/v2` as the initial rules engine,
  pin the selected version, and isolate it behind an application-owned adapter. Verify
  required move legality and draw semantics with project tests before adoption.
- **Rationale**: A library documents legal moves, outcomes, FEN/PGN support, claimable
  threefold/fifty-move draws, and automatic fivefold/seventy-five-move draws. A backend
  adapter limits coupling and leaves room to replace the dependency. Its MIT license is
  permissive, while its own repository notes maintenance is primarily for the maintainer's
  use, so dependency health must be reviewed before locking it in.
- **Alternatives considered**: Implementing all chess rules internally was rejected for
  MVP due to edge-case and correctness risk. The archived `notnil/chess` repository is
  not a preferred new dependency; reconsider alternatives if the candidate fails tests or
  maintenance review.
- **Validation**: Test legal and illegal moves, castling, en passant, promotion, check,
  checkmate, stalemate, repetition and move-count draw claims, automatic draws, and
  serialization round trips against trusted positions.
- **References**: [corentings/chess](https://github.com/corentings/chess),
  [package documentation](https://pkg.go.dev/github.com/corentings/chess/v2),
  [archived notnil/chess](https://github.com/notnil/chess).

## React browser client

- **Decision**: Use React with TypeScript and Vite. Build the frontend as static assets
  served by the Go application. Keep state local to the app or a focused match-state hook;
  do not add a global state library without a demonstrated need. Use Vitest and React
  Testing Library for UI behavior and Playwright with two independent browser contexts for
  end-to-end online-match scenarios.
- **Rationale**: React provides a manageable UI model for board state, promotion, and live
  connection updates. Vite provides a small build/development setup and can proxy HTTP and
  WebSocket requests to the Go server. Serving static output keeps Go as the only backend.
  Accessible grid semantics and keyboard interaction must be verified alongside pointer
  interactions.
- **Alternatives considered**: Browser-native modules avoid dependencies but require more
  manual UI state coordination. A full-stack React framework adds server and routing
  features not needed for this single-game MVP. Redux-like global state and a specialized
  chessboard package are deferred unless requirements justify them; any board package must
  pass keyboard-accessibility and server-authority checks.

## Connection and invitation security

- **Decision**: Generate high-entropy unguessable invitation tokens and separate
  player-seat recovery credentials from invitation credentials. Validate WebSocket
  origins, enforce seat authorization, bound message sizes, and rate-limit sensitive
  endpoints.
- **Rationale**: Invitation links are shareable join capabilities, while reconnect
  credentials should not let anyone who later sees an old invitation impersonate a seat.
- **Alternatives considered**: Reusing an invitation token as a permanent player credential
  was rejected because it conflates joining with player authentication.
- **Reference**: [OWASP WebSocket Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/WebSocket_Security_Cheat_Sheet.html).

## Research follow-up

- Confirm the selected chess library's exact claim/automatic draw API and license at
  dependency selection time; its current documentation is informative but does not replace
  project-level conformance tests.
- Select the supported Go toolchain version when initializing the implementation, since
  the repository currently has no Go module or deployment target.
