# Database

PostgreSQL 16. Schema design and the ER diagram are in [architecture.md §5](../../docs/architecture.md#5-data-model). This doc covers working with the schema in code.

## Migrations

- Files: `internal/migrations/sql/NNNN_short_name.up.sql` + `NNNN_short_name.down.sql`, zero-padded and sequential (next: `0002_…`).
- They're embedded into the binary (`//go:embed sql/*.sql`) and applied by `api migrate` (golang-migrate, pgx/v5 driver). The runner rewrites `postgres://` to `pgx5://` internally, so keep using normal `postgres://` URLs everywhere.
- Where they run:
  - locally, the `migrate` Compose service on `make up` (or run `make migrate`);
  - in CI, before tests;
  - in production, a one-off ECS task *before* the service rolls out.
- Only `up` is exposed. Write a correct `down` anyway (reverse order, `IF EXISTS`).

**Rules:**

1. **Never edit a migration that's on `main`.** Add a new one. (`0001_init` can still change only while no shared environment has applied it. As of now none has, but prefer new migrations.)
2. **Stay backward-compatible with the previous release.** Migrations run before new code is live, so use expand/contract: add a nullable column → deploy code that writes it → backfill → tighten constraints later.
3. One logical change per migration. Include the indexes that change needs.

## Schema notes

- UUID primary keys come from `gen_random_uuid()` (`pgcrypto`). Emails are `citext` (case-insensitive unique). The service also lowercases them.
- Every board child table has `ON DELETE CASCADE` from `boards`. Deleting a pin cascades to its connections. Deleting a user cascades to their sessions and owned boards.
- `pins.version` is the optimistic-concurrency counter. Updates must use `WHERE id = $1 AND version = $2` and increment it ([ADR-4](../../docs/architecture.md#adr-4-server-authoritative-real-time-with-a-per-board-hub--accepted)).
- `pins.content` is JSONB so pin kinds can evolve without migrations. Validate its shape in the service layer per `kind`.
- `connections` forbids self-loops (`CHECK`) and duplicate pairs in either direction (unique index on `LEAST/GREATEST` of the pin IDs).
- Enum-like columns (`role`, `kind`) are `text` + `CHECK`, not Postgres enums, so adding values is a simple migration.
- The board owner isn't a row in `board_members`.

## Query style

- Hand-written SQL through `pgxpool` in each feature's `store.go` (see [conventions.md](conventions.md)). sqlc is in the design (ADR-1) but not adopted yet. Decide before the boards/pins work adds many queries ([project-status.md](../../docs/project-status.md#deviations-from-the-design)).
- Select UUIDs as `id::text` and `citext` as `::text`, because IDs are `string` in Go.
- Always use `$n` placeholders. Never build SQL with string formatting.
- Map `pgx.ErrNoRows` → `ErrNotFound`, and unique violation `23505` → a domain sentinel.
- Filter expired rows in SQL (`expires_at > now()`) rather than in Go.
