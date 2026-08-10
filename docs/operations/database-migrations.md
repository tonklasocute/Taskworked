# Database Migrations

Status: implemented (P1.1). Replaces the old GORM `AutoMigrate` call
(`bootstrap.Migrate`, removed) with versioned SQL migrations via
[golang-migrate](https://github.com/golang-migrate/migrate). See
[docs/superpowers/specs/2026-08-10-p1-organization-architecture-audit.md](../superpowers/specs/2026-08-10-p1-organization-architecture-audit.md)
§9 for why.

## How it works

- Migration files live in `backend/migrations/*.up.sql` /
  `*.down.sql`, embedded into the binary at compile time
  (`backend/migrations/embed.go`) — no separate directory needs to be
  shipped alongside the Docker image.
- `backend/internal/platform/migrator` wraps golang-migrate with this
  project's specific needs: `Up`, `Version`, `LatestVersion`, `Force`.
- `backend/cmd/migrate` is a small CLI that applies pending migrations and
  exits — this is `docker-compose.yml`'s `migrate` service, which runs
  once before `api` starts (`api` depends on it with
  `condition: service_completed_successfully`).
- `cmd/api/main.go` no longer runs migrations itself. It does verify, at
  startup, that the database is (a) actually migrated at all, (b) not
  "dirty" (a previous migration failed partway), and (c) at the same
  version this binary's embedded migrations expect — and refuses to start
  otherwise. This is a safety net for the case where `api` is run without
  the `migrate` step having completed (e.g. outside Compose).

## Running migrations

**Via Docker Compose** (production/staging shape): automatic — `migrate`
runs before `api` on every `docker compose up`.

**Local dev, without Docker for app code**:
```
make infra      # postgres/redis/minio
make migrate    # applies pending migrations
make backend    # or: cd backend && go run ./cmd/api
```

**Directly**:
```
cd backend
DATABASE_URL=postgres://... go run ./cmd/migrate
```

## Writing a new migration

Add a new pair of files: `backend/migrations/000N_description.up.sql` and
`.down.sql`, where `N` is the next integer. Conventions established so far:

- Additive, reversible changes only in a given migration where possible;
  when a change is genuinely destructive (dropping a column with data),
  split it into an additive migration (deprecate/stop using it in code)
  followed by a later cleanup migration, so there's a safe window between
  the two in production.
- Every `up.sql` gets a real `down.sql` — even if the down-migration is
  "this can't be perfectly undone, here's the closest safe reversal."
- Data-migrating steps (backfills) should include a consistency check
  (see `0002_organizations.up.sql`'s `DO $$ ... END $$` block) that
  `RAISE EXCEPTION`s — and thus rolls back the whole migration transaction
  — if the backfill didn't fully cover what it was supposed to, rather than
  silently leaving inconsistent data for later code to trip over.

## One-time cutover for a pre-existing database

**This only applies once, to a database that was previously managed by the
old `AutoMigrate` and already has its schema** (tables/columns/indexes
matching exactly what `0001_baseline.up.sql` would create from scratch).
Running migrations normally against such a database fails on `0001` with
`relation "X" already exists`, because golang-migrate has no record of
anything being applied yet and doesn't know the tables already match.

Fix: mark migration `1` as applied without running its SQL (the SQL would
be a no-op anyway, since the schema already matches it exactly), then run
the rest of the pending migrations normally:

```
cd backend
DATABASE_URL=postgres://... go run ./cmd/migrate -force-baseline
```

`-force-baseline` calls `migrator.Force(dsn, 1)` (records "at version 1"
directly in golang-migrate's bookkeeping table, `schema_migrations`,
without executing `0001_baseline.up.sql`), then proceeds to run every
migration after that normally (`0002` onward) exactly as `migrate` always
does.

**Only ever run this against a database whose existing schema you've
confirmed matches `0001_baseline.up.sql` exactly** — if it doesn't (e.g. a
manually-hand-edited table, or a schema from a different, older version of
this app), forcing version 1 will hide a real mismatch instead of
surfacing it as the "relation already exists" error that's actually
telling you something's off. When in doubt, diff `information_schema`
between the two before forcing (see this repo's own cutover, verified this
way, in the P1.1 implementation report).

This repo's own dev/CI databases went through exactly this cutover as part
of implementing P1.1, since they already had an AutoMigrate-managed schema
from before this change — a fresh database, going forward, never needs
`-force-baseline` at all; it just runs `0001` through the latest migration
normally.
