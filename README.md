# Online Chess

> **Educational / vibe-coded project:** This app was built with AI-assisted “vibe coding”
> to learn and demonstrate the Spec-Kit workflow—from specifications and plans to tasks,
> implementation, and validation. It is a learning project, not a production-audited chess
> service. Review security, configuration, and usability before deploying it publicly.

A two-player online chess app with invitation links, server-authoritative legal moves and
clocks, reconnect/recovery, draw and resign actions, move history, and latest-move
highlighting. The Go server serves the React app and its HTTP/WebSocket API from one origin.



## Run with Docker Compose (recommended)

### Prerequisites

- Docker Engine or Docker Desktop
- Docker Compose v2 (`docker compose`)
- Docker Buildx plugin is recommended for image builds (install `docker-buildx-plugin` on
  Linux distributions that package it separately).

### Start

```sh
cp .env.example .env
docker compose up --build -d
```

Open **http://localhost:8080**. The first startup builds the React assets and Go server,
starts PostgreSQL, waits for the database health check, and applies database migrations
automatically before serving the app.

Check service status and logs:

```sh
docker compose ps
docker compose logs -f app
docker compose logs -f db
```

The application health endpoint is available at
[http://localhost:8080/healthz](http://localhost:8080/healthz). A healthy response is
`{"status":"ok"}`.

Stop the containers while preserving the database:

```sh
docker compose down
```

To also delete the local database volume and all saved matches, run
`docker compose down -v`. **This permanently deletes local database data.**

### Configuration

Compose reads settings from `.env` in the repository root. Available values:

| Variable | Default | Purpose |
|---|---|---|
| `POSTGRES_PASSWORD` | `chess_dev_password` | Password for the local PostgreSQL container and application connection. |
| `POSTGRES_PORT` | `5432` | PostgreSQL host port; the database is bound to localhost. |
| `HTTP_PORT` | `8080` | Host port for the app and health endpoint. |
| `ALLOWED_ORIGIN` | `http://localhost:8080` | Browser origin allowed for realtime connections. |
| `DATABASE_URL` | Built from the PostgreSQL defaults | Optional full connection string override for the app container. |

If you change `HTTP_PORT`, set `ALLOWED_ORIGIN` to the matching browser URL as well. Use
`http://localhost:<port>` when opening the app via localhost. The sample password is for
local development only; choose a strong secret before deploying anywhere shared or public.
If the password contains reserved characters, set `DATABASE_URL` explicitly and
URL-encode the password component in the connection string.

## Run locally without Docker

### Prerequisites

- Go version matching `go.mod`
- Node.js 22 or later and npm
- PostgreSQL

Start PostgreSQL and create a database/user, then export a connection string. In one
terminal, build the frontend and start the backend:

```sh
cd web
npm ci
npm run build
cd ..

export DATABASE_URL='postgres://chess:YOUR_PASSWORD@localhost:5432/chess?sslmode=disable'
export HTTP_ADDR='127.0.0.1:8080'
export WEB_DIR='web/dist'
export ALLOWED_ORIGIN='http://localhost:8080'
go run ./cmd/server
```

Open http://localhost:8080. The server applies migrations at startup. For frontend hot
reload during development, start `npm run dev -- --host 127.0.0.1` from `web/` and run the
backend on port 8080; Vite proxies API and WebSocket traffic to it. With the default Vite
port, open http://localhost:5173 and set `ALLOWED_ORIGIN=http://localhost:5173` for the Go
server.

## Tests and validation

Frontend tests and production build:

```sh
cd web
npm ci
npm test
npm run build
```

Go unit tests:

```sh
go test ./...
```

Database-backed integration, acceptance, and security tests require a running PostgreSQL
database and `DATABASE_URL`:

```sh
DATABASE_URL='postgres://chess:YOUR_PASSWORD@localhost:5432/chess?sslmode=disable' go test ./...
```

Playwright end-to-end tests require the application to be running and Chromium installed for
Playwright. Start the Go server and Vite dev server as described above, then run:

```sh
cd web
npx playwright install chromium
E2E_BASE_URL=http://localhost:5173 npm run test:e2e -- --workers=1
```

## Project structure

- `cmd/server/` — Go server entry point.
- `internal/chess/`, `internal/match/`, `internal/clock/` — chess rules adapter, match
  lifecycle, and clock behavior.
- `internal/store/postgres/` — PostgreSQL persistence and migrations.
- `internal/transport/http/`, `internal/transport/realtime/` — HTTP and WebSocket APIs.
- `web/` — React/TypeScript client, unit tests, and Playwright scenarios.
- `specs/` — feature specifications, plans, tasks, contracts, and test evidence.

## Notes

- The server is authoritative for legal moves, accepted match state, and clocks; client
  clocks are display-only.
- Invitation tokens are part of the current invitation-link flow. Changing token placement
  or invitation security is separate from the UI features documented here.
- The 10-participant first-use usability study is still pending; see
  [`specs/001-online-chess/usability-validation.md`](specs/001-online-chess/usability-validation.md).
