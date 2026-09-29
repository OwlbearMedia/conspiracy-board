# API Conventions

These patterns are established in `internal/` (the fullest example is `internal/auth` on the `auth` branch). Follow them for new features, and change this doc when you change the pattern.

## Feature package structure

Each feature (auth, boards, pins, …) is a package under `internal/<feature>/`, split by layer:

| File | Type | Responsibility |
|---|---|---|
| `store.go` | `Store` wrapping `*pgxpool.Pool` | SQL only. Maps pgx errors to package sentinel errors (`ErrNotFound`, `ErrEmailTaken`, …). Wraps everything else with `fmt.Errorf("<op>: %w", err)` |
| `service.go` | `Service` | Input normalization and validation, business rules, and authorization. Transport-agnostic, so REST handlers and the future WebSocket hub can share it ([architecture.md §10](../../docs/architecture.md#10-build-order)) |
| `http.go` | `Handler` with `Routes() http.Handler` | Decode → call service → map errors to HTTP. No SQL, no business rules |
| `middleware.go` | | Feature middleware, if any |

Wire it up in `internal/server/server.go` with `r.Mount("/<feature>", handler.Routes())` inside the `/api/v1` route. Construct dependencies explicitly (`NewStore(pool)` → `NewService(store, logger)` → `NewHandler(svc, logger, …)`); there's no DI framework and no globals.

## HTTP responses and errors

Use [`internal/httpx`](../internal/httpx/httpx.go) in every handler:

- `httpx.Decode(w, r, &in)` caps bodies at 1 MiB and rejects trailing data. On failure, respond `400 invalid_json`.
- `httpx.JSON(w, status, v)` for success bodies.
- `httpx.Error(w, status, code, message)` for every error. The envelope is:

  ```json
  { "error": { "code": "email_taken", "message": "that email is already registered" } }
  ```

  `code` is a stable snake_case identifier that clients branch on, so treat it as API surface. `message` is for humans.

Error codes in use: `invalid_json`, `invalid_input`, `email_taken`, `invalid_credentials`, `unauthenticated`, `csrf_mismatch`, `internal`.

Error mapping pattern:

- User-fixable validation → return a `ValidationError` from the service → `400 invalid_input` with the message.
- Known domain conditions → sentinel errors checked with `errors.Is` → a specific status and code.
- Anything else → log it with context and return `500 internal` with a generic message. **Never leak internal error text to clients.**

## Auth in handlers

Once `auth` is merged:

- Protect routes with `authHandler.RequireUser`, then read the caller with `auth.UserFrom(r.Context())`.
- Wrap every mutating route (POST/PATCH/PUT/DELETE) that acts on a session in `auth.RequireCSRF`.
- Board-level authorization (viewer/editor/owner) belongs in the **service** layer, following the matrix in [architecture.md §5](../../docs/architecture.md#5-data-model). Check `owner_id` first, then `board_members`.

## Security habits already in the code

Keep these:

- Tokens (sessions, invitations) are random 32-byte values. Only a SHA-256 hash is stored.
- Use constant-time comparison for secrets (`crypto/subtle`).
- Login returns the same error and does the same Argon2 work whether the email exists or not.
- Never log passwords, raw tokens, or cookie values. Log IDs.

## Logging

- `log/slog` with the JSON handler. Use the `...Context` variants (`logger.InfoContext(ctx, …)`) inside request paths.
- Structured key/value pairs with snake_case keys (`"user_id", id`), never string interpolation.
- The request logger middleware already logs method, path, status, bytes, duration, and `request_id`, so don't duplicate that in handlers.

## Configuration

Add new settings as fields on `config.Config`, loaded from env in `config.Load()`. Required values fail at startup with a clear error. Document every new variable in the [API README](../README.md#configuration).

## Testing

- Run `go test -race ./...`. CI does the same (with a Postgres service container and `DATABASE_URL` set).
- **Unit tests** go in the same package (`package auth`). Write them table-driven with `t.Run` and `t.Parallel()` where safe. Use `httptest.NewRecorder` for middleware.
- **Integration tests** go in the external test package (`package auth_test`), in `integration_test.go`. They build the real router with `server.New(...)`, serve it with `httptest.NewServer`, and use an `http.Client` with a cookie jar. They:
  - call `t.Skip` when `DATABASE_URL` is unset, so `go test` works without a DB;
  - run `migrations.Up` first;
  - `TRUNCATE users CASCADE` for a clean slate, which **destroys data** in whatever DB they point at.
- Tests in a package that truncates shouldn't run in parallel with other DB tests on the same database. Go runs packages in parallel by default, so give each DB-test package unique data, or use `-p 1` once there are several.
- The hub/WebSocket layer (step 4) is meant to be the flagship concurrency test suite ([architecture.md §9](../../docs/architecture.md#9-observability--testing)).

## Style

- Standard `gofmt` / `goimports` grouping: stdlib, third-party, then `github.com/dylanwhitney/conspiracy-board/...`.
- Keep dependencies minimal ([ADR-1](../../docs/architecture.md#adr-1-go-for-the-api--accepted)). Justify each new module.
- Comments explain *why* (security rationale, trade-offs), not what. Match the existing density.
