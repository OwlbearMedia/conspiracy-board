# Local Development

The short version is in the [root README](../README.md#quick-start). This doc covers how the local stack is wired, how to work outside Docker, and how to fix common problems.

## How the stack is wired

`docker-compose.yml` runs five services:

| Service | Image / build | Role |
|---|---|---|
| `postgres` | `postgres:16-alpine` | Database, exposed on host `5432`, data in the `pgdata` volume |
| `migrate` | `api/Dockerfile` target `dev` | Runs `go run ./cmd/api migrate` once, then exits |
| `api` | `api/Dockerfile` target `dev` | Go API under [Air](https://github.com/air-verse/air) hot reload, listening on `:8080` inside the network. Starts only after `migrate` succeeds |
| `web` | `node:20-alpine` | `pnpm install && pnpm dev` (Vite on `:5173` inside the network) |
| `nginx` | `nginx:1.27-alpine` | The only app entry point: host `8080`. `/api/*` goes to `api:8080` (including WebSocket upgrades), `/` goes to `web:5173` (including Vite HMR) |

Why nginx exists locally: it makes the browser see one origin, so cookies and relative `/api/...` URLs behave the way they will behind the ALB/CloudFront. Production doesn't use nginx ([ADR-2](architecture.md#adr-2-drop-nginx-from-production-s3--cloudfront-for-the-frontend--accepted)).

Source is bind-mounted: `./api` → `/app` in the api/migrate containers, and `./web` → `/web`. Go edits trigger an Air rebuild (it watches `.go` and `.sql`), and web edits trigger Vite HMR. `node_modules` and the Go module cache live in named volumes, so they don't collide with host installs.

### Environment variables

The API reads configuration from the environment ([api/README.md](../api/README.md#configuration) has the full list). Compose sets `DATABASE_URL` and `APP_ENV=local`. The only value `.env` overrides today is `POSTGRES_PASSWORD` (default `localdev`).

## Working outside Docker

Using Docker for Postgres and running the apps on the host gives faster test cycles and native debugging.

```sh
docker compose up -d postgres
export DATABASE_URL='postgres://conspiracy:localdev@localhost:5432/conspiracy?sslmode=disable'

cd api
go run ./cmd/api migrate   # apply migrations
go run ./cmd/api           # API on http://localhost:8080
go test -race ./...        # unit tests; DB-backed tests run only when DATABASE_URL is set
```

> ⚠️ DB-backed integration tests **truncate tables** (`TRUNCATE users CASCADE`). Pointing `DATABASE_URL` at your dev database while running tests wipes your local data. Use a separate test database:
>
> ```sh
> docker compose exec postgres createdb -U conspiracy conspiracy_test
> DATABASE_URL='postgres://conspiracy:localdev@localhost:5432/conspiracy_test?sslmode=disable' go test -race ./...
> ```

For the frontend, `cd web && pnpm install && pnpm dev` serves on http://localhost:5173. `vite.config.ts` has no dev proxy, so `/api` calls from that origin fail. For anything that calls the API, use the full stack at `:8080`, or add a `server.proxy` entry for `/api` to `vite.config.ts`.

## Database tasks

```sh
docker compose exec postgres psql -U conspiracy conspiracy   # psql shell
make migrate                                                  # apply pending migrations
docker compose down -v                                        # ⚠️ delete ALL local data (volumes), then `make up`
```

There's no `migrate down` command. Down migrations exist for completeness, but the `api` binary only runs `up`. See [api/docs/database.md](../api/docs/database.md).

## Troubleshooting

| Symptom | Cause / fix |
|---|---|
| `missing go.sum entry` from `migrate` or `api`, or the api container never starts | A dependency changed without `go.sum` being updated. Run `cd api && go mod tidy`, commit `go.mod` + `go.sum`, then `make up` |
| Page shows **API status: down** | `make logs` for API errors, and `docker compose ps` to check that `migrate` exited 0 and `api` is running |
| Port `8080` or `5432` already in use | Stop the other process, or change the host port in `docker-compose.yml` (don't commit that change) |
| Air doesn't pick up a change | Air watches `.go` and `.sql` only. Restart with `docker compose restart api` |
| Web dependencies act stale after a `package.json` change | `docker compose restart web` re-runs `pnpm install`. If it's still stale: `docker compose down && docker volume rm conspiracy-board_webmodules` |
| Postgres auth fails after changing `POSTGRES_PASSWORD` | The password is set only when the volume is first created. Use `docker compose down -v` to reset |
