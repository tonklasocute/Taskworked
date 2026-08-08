# P0 Hardening Plan — Taskworked

Date: 2026-08-08
Source: [2026-08-08-production-readiness-audit.md](../specs/2026-08-08-production-readiness-audit.md)
Status: P0.1 in progress

Each phase below is independently implementable, testable, and revertable without
touching unrelated phases. No phase introduces microservices or rewrites existing
architecture. Existing APIs/frontend behavior are preserved unless a phase's own
notes say otherwise (only P0.6 — token storage — intentionally changes an API
contract, and that's called out explicitly when it's reached).

| Phase | Scope | Touches app code? | Touches schema? |
|---|---|---|---|
| **P0.1** | Backup & recovery (Postgres + MinIO) | No | No |
| P0.2 | Panic recovery middleware + rate limiting | Yes (middleware only) | No |
| P0.3 | Kanban/list pagination correctness bug | Yes (frontend + 1 new backend endpoint) | No |
| P0.4 | Versioned migrations + FK constraints + soft deletes | Yes (models) | Yes |
| P0.5 | DB connection pool limits + transactions on multi-step writes | Yes (platform + services) | No |
| P0.6 | Token storage hardening (httpOnly cookies) + WS token off URL | Yes (auth + realtime + frontend) | No |
| P0.7 | Gamification integrity (EXP race + farming exploit) | Yes (gamification + task) | Yes (1 column) |
| P0.8 | Security headers + CORS allowlist + graceful shutdown | Yes (bootstrap/main only) | No |
| P0.9 | Audit logging | Yes (new module + call sites) | Yes (new table) |
| P0.10 | AI rate limiting/cost control | Yes (ai module + middleware) | No |
| P0.11 | TLS proxy + environment separation (ops only) | No | No |
| P0.12 | CI security scanning + deploy pipeline | No | No |

Each phase, when implemented, gets its own "Files changed / DB changes / API
changes / Security impact / Tests added / Tests passed / Remaining risks" report
posted in the conversation, and requires explicit approval before the next phase
starts.

---

## P0.1 — Backup & Recovery

### Goal
Automated, scheduled, retained, and **actually-verified** backup/restore for
Postgres and MinIO, deployable alongside the existing Docker Compose stack, with
no secrets in source control and a documented RPO/RTO.

### Affected files (all new — no existing file is modified except `docker-compose.yml`,
`.env.example`, and `Makefile`, all additive changes)
- `ops/backup/Dockerfile` — small Alpine image: `postgresql16-client`, `mc`
  (MinIO client), `gzip`, `openssl` (optional encryption), `curl` (webhook
  notify), busybox `crond` for scheduling.
- `ops/backup/entrypoint.sh` — renders crontab from `BACKUP_SCHEDULE_CRON`, runs
  `crond` in the foreground.
- `ops/backup/scripts/backup-postgres.sh` — `pg_dump` (custom format, compressed),
  optional `openssl enc` encryption, timestamped filename.
- `ops/backup/scripts/restore-postgres.sh` — restores a given dump into a target
  database (requires explicit `--target-db`, refuses to silently overwrite the
  primary DB without `--yes-i-am-sure`).
- `ops/backup/scripts/verify-postgres-backup.sh` — restores the latest backup into
  a throwaway scratch database and runs a smoke query; this is the actual
  backup→restore→usable proof, not just an exit-code check.
- `ops/backup/scripts/backup-minio.sh` / `restore-minio.sh` / `verify-minio-backup.sh`
  — same pattern via `mc mirror`, verified by checksum comparison and a sample
  object read-back.
- `ops/backup/scripts/prune-backups.sh` — enforces `BACKUP_RETENTION_DAYS`.
- `ops/backup/scripts/notify.sh` — shared helper: structured log line + optional
  webhook POST on failure (and optionally success), no hardcoded endpoint.
- `ops/backup/scripts/run-full-backup.sh` — orchestrates backup → verify → prune →
  notify; this is what cron invokes.
- `docker-compose.yml` — new `backup` service (additive block only).
- `.env.example` — new optional env vars, all with safe defaults or empty/off.
- `Makefile` — `make backup`, `make backup-verify`, `make restore-postgres`,
  `make restore-minio` convenience targets.
- `docs/operations/backup-recovery.md` — RPO/RTO, schedule, retention, restore
  runbook, verification evidence from this implementation.

### Database changes
None. This phase is purely operational tooling around the existing schema.

### Migration requirements
None.

### Rollback strategy
Everything added in this phase is additive (a new `ops/backup/` directory, a new
compose service block, new optional env vars, a new Makefile target, new docs).
No existing file's behavior changes. Rollback is: remove the `backup` service
block from `docker-compose.yml` (or simply don't start it) and/or delete
`ops/backup/`. Zero impact on the running API/frontend/DB in either direction.

### Tests required
- **Functional, not just exit-code**: seed a marker row in the running dev
  Postgres → run backup → run verify (restore into a scratch DB, `SELECT` the
  marker row back) → tear down scratch DB. Same pattern for MinIO: upload a
  marker object → backup → verify (restore into a scratch location, read the
  object back, compare checksum).
- Retention: create synthetic old-dated backup files, run prune, confirm only
  files older than `BACKUP_RETENTION_DAYS` are removed and recent ones survive.
- Existing suites per standing instruction: `go build`, `go vet`, `go test ./...`,
  e2e suite, frontend `lint`+`build` — expected to be unaffected since no app code
  is touched, but run to confirm.

### Risks
- **Offsite replication is out of this environment's reach**: I have no real
  production offsite target (S3 bucket, second server) to push backups to. I'll
  implement local backup + full verification (which satisfies "automated backup /
  retention / verification / restore") and add a documented, pluggable offsite-sync
  hook (rclone-compatible) that an operator must point at real infrastructure
  before going live. I will not claim offsite replication is solved — it isn't,
  and can't be from here.
- Encryption is opt-in via `BACKUP_ENCRYPTION_PASSPHRASE` (openssl symmetric) —
  off by default for local/dev ergonomics, documented as required for production.
- `mc mirror` needs the target bucket to exist; scripts must handle a
  not-yet-bootstrapped bucket without failing the whole job.
