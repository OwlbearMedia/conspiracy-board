# API

Go HTTP + (planned) WebSocket API for Conspiracy Board. Uses the chi router, the pgx Postgres driver, golang-migrate with embedded SQL, and `log/slog` JSON logs. A single static binary serves traffic and runs migrations.

Design context: [docs/architecture.md](../docs/architecture.md) (§6 covers the REST/WS contract).

## Run it

The easiest way is the full stack from the repo root: `make up`, then the API is at http://localhost:8080/api/v1/healthz. See the [root README](../README.md).

To run on the host (Go 1.26+), with Postgres from Compose:

```sh
docker compose up -d postgres                     # from repo root
export DATABASE_URL='postgres://conspiracy:localdev@localhost:5432/conspiracy?sslmode=disable'

go run ./cmd/api migrate   # apply migrations and exit
go run ./cmd/api           # serve on :8080
```

## Configuration

All configuration comes from environment variables ([internal/config/config.go](internal/config/config.go)).

| Variable | Default | Meaning |
|---|---|---|
| `DATABASE_URL` | **required** | Postgres URL, e.g. `postgres://user:pass@host:5432/db?sslmode=disable` |
| `PORT` | `8080` | HTTP listen port |
| `APP_ENV` | `local` | `local` \| `preview` \| `production`. Anything other than `local` sets `Secure` on cookies |
| `SERVE_STATIC` | unset | `1` serves the embedded SPA from `internal/web/dist`. Used by single-container preview deploys only |

## Commands

```sh
go test -race ./...   # all tests (DB-backed tests skip unless DATABASE_URL is set)
go vet ./...          # the current "lint"
go build ./...
```

From the repo root, `make test` and `make lint` run the same commands.

> ⚠️ DB-backed tests truncate tables. Point `DATABASE_URL` at a throwaway database, not your dev data. See [docs/development.md](../docs/development.md#working-outside-docker).

## Layout

```
cmd/api/main.go          entrypoint: config, `migrate` subcommand, server start + graceful shutdown
internal/
  config/                env → Config
  server/                router assembly, middleware, health/readiness, route mounting
  httpx/                 JSON response + error envelope + bounded request decoding
  auth/                  password hashing, session tokens (+ handlers/service/store on the auth branch)
  migrations/            embedded golang-migrate runner
    sql/                 NNNN_name.up.sql / .down.sql
  web/                   embedded SPA handler for SERVE_STATIC
    dist/index.html      committed placeholder; the Docker build overwrites dist/ with the real bundle
Dockerfile               targets: dev (Air), web-build, build, prod (distroless). Build from the REPO ROOT
.air.toml                hot-reload config used by the dev container
```

## Endpoints

| Method | Path | Status |
|---|---|---|
| GET | `/api/v1/healthz` | ✅ Liveness: returns `ok` |
| GET | `/api/v1/readyz` | ✅ Readiness: pings the DB, 503 if unreachable |
| POST | `/api/v1/auth/register`, `/login`, `/logout` | 🚧 `auth` branch |
| GET | `/api/v1/me` | 🚧 `auth` branch |
| … | boards, members, invitations, pins, connections, WS | ⏳ Designed in [architecture.md §6](../docs/architecture.md#6-api--real-time-design) |

## Docker image

```sh
docker build -f api/Dockerfile --target prod .   # from the repo root, NOT from api/
```

The build context is the repo root so the `web-build` stage can compile the SPA and embed it into the binary (used when `SERVE_STATIC=1`). The final image is `distroless/static:nonroot` and exposes `8080`.

## More docs

- [docs/conventions.md](docs/conventions.md): package structure, errors, logging, testing patterns
- [docs/database.md](docs/database.md): schema notes, writing migrations, query style
