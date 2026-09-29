# AGENTS.md

Guidance for AI coding agents working in this repo. Humans: start with [README.md](README.md).

## What this is

Conspiracy Board is a real-time collaborative cork board: users pin clues and connect them with strings. It's a portfolio project built to "enterprise" standards. That's deliberate, so preserve the rigor: security rationale in comments, tests, IaC, and documented decisions.

- Monorepo: `api/` (Go), `web/` (Vue 3 + TS), `infra/` (Terraform), `nginx/` (local only), `.github/` (Actions).
- **Before non-trivial work, read [docs/project-status.md](docs/project-status.md).** It tracks what's built, what's broken, and which branch holds in-progress work. The design is in [docs/architecture.md](docs/architecture.md).

## Doc map

| Need | Read |
|---|---|
| Current state, known issues, next steps | [docs/project-status.md](docs/project-status.md) |
| Design, ADRs, data model, REST/WS protocol, build order | [docs/architecture.md](docs/architecture.md) |
| Local stack, running outside Docker, troubleshooting | [docs/development.md](docs/development.md) |
| Workflows, Dependabot, deploys, previews | [docs/ci-cd.md](docs/ci-cd.md) |
| Go package structure, errors, logging, tests | [api/docs/conventions.md](api/docs/conventions.md) |
| Migrations, schema notes, SQL style | [api/docs/database.md](api/docs/database.md) |
| Frontend ↔ API contract (errors, CSRF, WS) | [web/docs/api-integration.md](web/docs/api-integration.md) |
| Terraform layout and state | [infra/README.md](infra/README.md) |
| Index of everything | [docs/README.md](docs/README.md) |

## Commands

```sh
make up / make down / make logs      # full local stack at http://localhost:8080
make test                            # cd api && go test -race ./...
make lint                            # cd api && go vet ./...
make typecheck                       # cd web && pnpm typecheck
cd web && pnpm build                 # type check + production build
docker build -f api/Dockerfile --target prod .   # from REPO ROOT (the image embeds web/)
```

To verify a change, run what CI runs: `go vet ./... && go build ./... && go test -race ./...` in `api/`, and `pnpm typecheck && pnpm build` in `web/`.

## Rules

### Safety: ask before doing these

- **Pushing to `main`** triggers the production Deploy workflow.
- **Pushing a `preview/*` branch** creates paid AWS infrastructure (about $9/month each).
- **Running `terraform apply` or `destroy`**, or any AWS-mutating command.
- **Editing a migration** in `api/internal/migrations/sql/` that already exists on `main`. Add a new numbered migration instead.
- **Running DB-backed tests with `DATABASE_URL` pointing at the dev database.** Integration tests run `TRUNCATE users CASCADE`. Use a separate `conspiracy_test` database ([docs/development.md](docs/development.md#working-outside-docker)).

### Things that must stay true

- `.github/workflows/ci.yml` stays **secret-free**, so Dependabot PRs can run it. AWS-needing jobs go in other workflows.
- `api/internal/web/dist/index.html` is a committed placeholder (a `.gitignore` exception) required by `//go:embed`. Don't delete it or commit a real build over it.
- The `infra/envs/prod` outputs are a contract consumed by `infra/envs/preview`. Keep their names stable.
- The preview slug sanitization is duplicated in `preview-deploy.yml` and `preview-teardown.yml` (twice). Change all copies together.
- Migrations must be backward-compatible with the previous release. They run before the new code rolls out.
- Commit `web/pnpm-lock.yaml` with any `web/package.json` change. CI installs with `--frozen-lockfile`.
- Don't commit `.env`, `.pnpm-store/`, `node_modules/`, `api/tmp/`, or Terraform state.

### Code conventions (details in the per-app docs)

- **Go:** feature packages under `api/internal/<feature>/` split into `store.go` (SQL), `service.go` (validation, rules, authorization; transport-agnostic), and `http.go` (handlers). Mount them in `internal/server/server.go`. Use `internal/httpx` for all JSON I/O and the `{"error":{"code","message"}}` envelope. Error `code`s are API surface. Use `slog` with snake_case keys. Keep dependencies minimal.
- **Security:** hash tokens at rest (SHA-256), compare secrets in constant time, never log secrets, tokens, or passwords, protect mutating routes with CSRF middleware, and return generic 500s.
- **SQL:** hand-written with pgx, `$n` placeholders, UUIDs selected as `::text`. (sqlc is in the design but not adopted; don't introduce it without a decision recorded in the architecture doc.)
- **Web:** Vue 3 `<script setup lang="ts">`, strict TS, Pinia for state, and every API call through one client module.
- **Tests:** table-driven Go tests with `t.Parallel()`. Integration tests in the `_test` package skip when `DATABASE_URL` is unset. Run with `-race`.
- **Comments** explain *why*, especially security and trade-off rationale. Match the existing style.

## Keeping docs honest

When a change makes a doc wrong, fix the doc in the same change. In particular:

- Finishing a build step, fixing a known issue, or deviating from the design → update [docs/project-status.md](docs/project-status.md).
- A new env var → the table in [api/README.md](api/README.md#configuration).
- A new error code → [api/docs/conventions.md](api/docs/conventions.md) and [web/docs/api-integration.md](web/docs/api-integration.md).
- A new architectural decision → add an ADR to [docs/architecture.md](docs/architecture.md).
- New docs go where [docs/README.md](docs/README.md#where-new-docs-go) says, and get linked from its index.
