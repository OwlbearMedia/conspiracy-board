# Project Status

_Last reviewed: 2026-09-28._ Update this doc when a build step lands or a known issue is fixed.

## Progress against the build order

The build order is defined in [architecture.md §10](architecture.md#10-build-order).

| # | Step | State | Where |
|---|---|---|---|
| 1 | Foundation: repo layout, Compose, migrations, healthz, CI skeleton | ✅ Done, with gaps (see known issues) | `main` |
| 2 | Auth & boards: register/login/sessions, CSRF, dashboard CRUD | 🚧 Auth API in progress, boards not started | `auth` branch + a stash |
| 3 | Board canvas (REST only): Konva, pins, strings, optimistic updates | ⏳ Not started | |
| 4 | Real-time: hub, WS endpoint, event protocol, reconnect/resync | ⏳ Not started | |
| 5 | Sharing: invitations, roles, authorization matrix | ⏳ Not started (schema exists) | |
| 6 | AWS: Terraform, OIDC deploy, CloudFront/S3, alarms | 🟡 Skeleton only | `infra/`, `.github/workflows/` |
| 7 | Polish: presence cursors, pin kinds, Playwright smoke | ⏳ Not started | |

### What exists on `main`

- **API** ([api/](../api/)): chi server with request-ID/logging/recover/timeout middleware, `/api/v1/healthz` and `/api/v1/readyz`, env-based config, `api migrate` subcommand with embedded migrations, `httpx` JSON/error helpers, and Argon2id password hashing + session token primitives in `internal/auth` (not yet wired to routes).
- **Database**: migration `0001_init` creates the full schema from the design (users, sessions, boards, board_members, invitations, pins, connections).
- **Web** ([web/](../web/)): Vue 3 + Pinia scaffold with a single page that shows API readiness. No router, stores, or canvas yet.
- **Local dev**: Compose stack (postgres, migrate, api with Air hot reload, vite, nginx).
- **CI/CD**: CI (vet/build/migrate/test, typecheck/build, docker build), a Deploy workflow with placeholder steps, preview deploy/teardown workflows, and Dependabot with auto-merge.
- **Infra**: `modules/preview` is written; `envs/prod` is a skeleton listing the outputs `envs/preview` depends on.

### What exists on the `auth` branch (not merged)

Commit `ecad26d` "Starting basic auth" adds:

- `internal/auth` split into `store.go` (pgx queries), `service.go` (validation and business rules), `http.go` (handlers and cookies), and `middleware.go` (`RequireUser`, `RequireCSRF`).
- Routes: `POST /api/v1/auth/register`, `/login`, `/logout` (CSRF-protected), and `GET /api/v1/me` (behind `RequireUser`).
- Sessions: 7-day TTL with sliding expiry (extended once past half-life), SHA-256-hashed tokens at rest, cookies `cb_session` (HttpOnly) and `cb_csrf` (readable by JS, echoed in `X-CSRF-Token`).
- Tests: table-driven unit tests for passwords, tokens, and CSRF middleware, plus HTTP integration tests (`TestAuthFlow`, `TestRegisterValidation`, `TestMeRequiresAuth`) that run against real Postgres when `DATABASE_URL` is set.
- `config.SecureCookies()`: the `Secure` flag is off only when `APP_ENV=local`.

`stash@{0}` ("WIP on auth") contains a generated `api/go.sum`, Go/dependency version bumps, a temporary `go mod tidy` CI step, and the `.pnpm-store` deletion. `main` now has all of these except the CI step, which is no longer needed, so the stash can probably be dropped.

**The branch forked before ~10 Dependabot merges on `main`**, so expect conflicts in `api/go.mod`, `api/Dockerfile`, and `web/`. Suggested path: rebase `auth` onto `main`, take `main`'s `go.mod`/`go.sum` in conflicts, then run `go mod tidy`.

Still to do for step 2: boards CRUD + dashboard, a janitor for expired sessions (`Store.DeleteExpiredSessions` exists but nothing calls it), frontend login/register/dashboard, and rate limiting on login (not in the design yet; worth an ADR note).

## Known issues

Roughly in the order they should be fixed.

1. **Dependabot PRs merged while CI was red.** Until `go.sum` was committed (2026-09-28), every CI run failed with `missing go.sum entry`, yet Dependabot auto-merge kept landing PRs (#18 through #44 merged since July). That means branch protection with required checks isn't configured, which is the precondition the comment at the top of `dependabot-automerge.yml` warns about. Once CI is green, set up branch protection on `main` requiring `api`, `web`, and `docker`, or disable auto-merge until you do.
2. **Deploy fails on every push to `main`.** There's no AWS role yet (`vars.AWS_DEPLOY_ROLE_ARN` is unset) and the rollout steps are placeholders. Until step 6, consider gating the job with `if: vars.AWS_DEPLOY_ROLE_ARN != ''` or switching it to `workflow_dispatch`.
3. **Preview Teardown fails every night** for the same reason (its schedule runs daily at 06:00 UTC). Gate it the same way.
4. **Node 20 is past end-of-life** (April 2026). It's used in `docker-compose.yml`, `api/Dockerfile` (`web-build` stage), and `ci.yml`. The `@v4` actions also print Node 20 deprecation warnings, and Dependabot PRs for newer action majors are open.
5. **Open Dependabot majors** (not auto-merged by design): pinia 3, TypeScript 7, vue-tsc 3, @vitejs/plugin-vue 6, AWS provider 6, node image, and several actions. Review these once CI is trustworthy. TypeScript 7 is the native Go-based compiler, so check vue-tsc compatibility first.
6. **Lint tooling from the design isn't set up.** The design lists golangci-lint, eslint, and `terraform fmt` in CI. Today `make lint` is `go vet` only, and there's no eslint config or Terraform check.
7. **Preview task can't reach its database yet.** `infra/modules/preview` injects `DATABASE_ADMIN_JSON`, but the app reads `DATABASE_URL`. The composing entrypoint or a per-preview secret is a TODO in that module.

## Deviations from the design

Deliberate or not, the code currently differs from [architecture.md](architecture.md) in these ways. Either update the design or change the code, but don't leave them silent.

- **sqlc isn't used.** ADR-1 calls for sqlc; the `auth` branch uses hand-written SQL through `pgxpool` in a `Store` type. Decide before step 3 adds many queries.
- **Go version.** The design says "Go 1.22+"; the module now targets Go 1.26 and the Docker images use 1.27.
- **IDs are strings.** UUIDs cross the Go boundary as `id::text` strings, with no uuid library (noted in `auth/store.go`).

## Open design questions

- **Same-origin vs cross-origin in production.** The design serves the SPA at `app.<domain>` (CloudFront) and the API at `api.<domain>` (ALB). That's same-site but *cross-origin*. The frontend currently fetches relative `/api/v1/...` paths, and `nginx.conf` says local same-origin "matches production." One of these has to give:
  - (a) Add a CloudFront behavior routing `/api/*` (and WebSockets) to the ALB, which keeps everything same-origin and matches local dev and previews.
  - (b) Go cross-origin: CORS with credentials, a cookie `Domain=.<domain>` so the SPA can read `cb_csrf`, and an absolute API base URL in the web build.

  Option (a) is simpler and keeps the three environments consistent. Record the decision as an ADR.
- **Rate limiting / brute-force protection** on `/auth/login` and `/auth/register` isn't addressed in the design.

## Suggested next steps

1. Confirm CI is green now that `go.sum` is committed.
2. Configure branch protection, and gate Deploy and Teardown until AWS exists (issues 1–3).
3. Rebase `auth` onto `main`, finish and merge the auth API.
4. Decide the same-origin question and the sqlc question, and record both in `architecture.md`.
5. Continue step 2: boards CRUD API, then the frontend auth and dashboard (add vue-router and a small API client, see [web/docs/api-integration.md](../web/docs/api-integration.md)).
