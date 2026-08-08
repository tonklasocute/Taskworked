# Backup & Recovery

Status: implemented (P0.1 of the
[production readiness audit](../superpowers/specs/2026-08-08-production-readiness-audit.md)).
Covers Postgres (system of record) and MinIO (task attachments/avatars).

## Objectives

| | Target | How it's met |
|---|---|---|
| **RPO** (Recovery Point Objective) | ≤ 24 hours | Daily backup by default (`BACKUP_SCHEDULE_CRON=0 2 * * *`, 02:00 UTC). Lower it (e.g. every 6h) by changing that one env var if a tighter RPO is needed — no code change required. |
| **RTO** (Recovery Time Objective) | Postgres: well under 1 hour for a database of the current size; MinIO: proportional to attachment volume | Restore is a single script invocation (`make restore-postgres` / `make restore-minio`); see timings below from the live verification run. |
| **Backup frequency** | Daily | `BACKUP_SCHEDULE_CRON` (5-field cron, default `0 2 * * *`) |
| **Retention** | 14 days | `BACKUP_RETENTION_DAYS` (default 14); enforced by `prune-backups.sh` at the end of every run |
| **Verification** | Every run | Every scheduled run restores its own backup into a throwaway scratch database/bucket and checks it's actually usable *before* declaring success — see "How verification works" below |

These are configurable via environment variables (see `.env.example`) — nothing
is hardcoded, and no secret is stored in source control.

## What's covered

- **Postgres**: full logical dump (`pg_dump --format=custom --compress=9`),
  covering all tables, indexes, and constraints as they exist at dump time.
- **MinIO**: full mirror of the configured bucket (`mc mirror`), covering task
  attachments, avatars, and any other object currently stored.
- **Redis** is intentionally *not* backed up — it holds only refresh-token
  sessions and pub/sub/presence state, all of which are safe to lose (users
  re-authenticate; presence/pub-sub state is transient by design). Backing it up
  would add operational surface for no durability benefit.

## Architecture

A dedicated `backup` service (`ops/backup/`) runs alongside the existing
`postgres`/`redis`/`minio`/`api`/`web` services in the same `docker-compose.yml`
— no new infrastructure paradigm, no microservice, just one more container in
the existing Compose stack. It runs `crond` in the foreground and fires
`run-full-backup.sh` on the configured schedule. Backups land in a Docker named
volume (`backup_data`, mounted at `/backups` in the container).

```
ops/backup/
  Dockerfile              alpine + postgresql16-client + mc (MinIO client) + crond
  entrypoint.sh           renders the crontab from BACKUP_SCHEDULE_CRON, runs crond -f
  scripts/
    backup-postgres.sh      pg_dump -> timestamped, compressed, optionally-encrypted file
    restore-postgres.sh     restores a dump into a named target database
    verify-postgres-backup.sh   restores the latest dump into a scratch DB and checks it
    backup-minio.sh         mc mirror bucket -> timestamped local snapshot dir
    restore-minio.sh        mc mirror snapshot dir -> a named target bucket
    verify-minio-backup.sh  restores the latest snapshot into a scratch bucket and checks it
    prune-backups.sh        deletes backups older than BACKUP_RETENTION_DAYS
    notify.sh                logs the outcome + optional webhook POST
    run-full-backup.sh       orchestrates all of the above; this is what cron calls
```

## How verification works (and why it's not just "the command exited 0")

`pg_dump`/`mc mirror` exiting 0 only proves bytes were written somewhere — it
says nothing about whether those bytes can actually rebuild a working database
or a readable file. Every backup run therefore performs a **real restore** as
part of the run, not a separate manual step someone might skip:

1. `backup-postgres.sh` produces a dump.
2. `verify-postgres-backup.sh` restores that exact dump into a disposable
   database (`taskworked_backup_verify_<pid>`), queries
   `information_schema.tables` to confirm real tables exist, then drops the
   scratch database. A dump that "succeeds" but produces zero restorable tables
   fails the run.
3. `backup-minio.sh` mirrors the bucket to a snapshot directory.
4. `verify-minio-backup.sh` mirrors that snapshot into a disposable bucket,
   compares file counts between source and restored bucket, and SHA-256
   checksums one sample object end-to-end (read from the original snapshot vs.
   read back from the restored bucket) before deleting the scratch bucket.
5. Only after both verifications pass does `prune-backups.sh` run and
   `notify.sh` report success. If any step fails, the run stops immediately,
   `notify.sh` reports failure (with a webhook if `BACKUP_WEBHOOK_URL` is
   configured), and **no old backups are pruned** — a failed run never deletes
   your remaining good backups.

### Live verification evidence (this implementation, 2026-08-08)

Run against the actual local dev stack (real `postgres`/`minio` containers,
schema created by the app's normal `AutoMigrate` boot path, 18 application
tables):

1. **Postgres, marker-row round-trip**: inserted a row
   (`p0.1-live-verification-1786191517`) into a scratch table in the live dev
   database, ran the full backup (`run-full-backup.sh`, the exact entrypoint
   cron uses), then separately restored that same dump into a *second*,
   persistent scratch database (`taskworked_manual_check`) and queried the row
   back directly — `SELECT id, note FROM backup_verification_marker` returned
   the exact original value. The automated `verify-postgres-backup.sh` step
   (which runs as part of every backup, not just this one-off check) confirmed
   19 restorable tables with ~11 live rows.
2. **MinIO, checksum round-trip**: uploaded a marker object
   (`verification/marker.txt`, SHA-256
   `6cb2437ecddf7314c2053c8545eeb9536ad18cb952fc35adabe72ece2abc7f25`), backed
   it up, and `verify-minio-backup.sh` restored it into a scratch bucket and
   recomputed the checksum from the restored copy — **identical hash**,
   proving byte-for-byte integrity through the full mirror → restore path.
3. **Encryption round-trip** (`BACKUP_ENCRYPTION_PASSPHRASE` set): backed up
   with encryption on, restored with the correct passphrase — marker row came
   back correctly. Then attempted to restore the *same* encrypted dump with a
   wrong passphrase — this failed loudly (`bad decrypt`, non-zero exit),
   confirming the dump is actually encrypted rather than just renamed.
4. **Retention pruning**: created synthetic backup files/directories dated 20
   days and 1 day old, ran `prune-backups.sh` with the default
   `BACKUP_RETENTION_DAYS=14` — the 20-day-old files were removed, the 1-day-old
   ones and the real fresh backup were kept.
5. **Overwrite-safety guard**: attempted `restore-postgres.sh` against the
   primary database name without `--yes-i-am-sure` — refused with a
   non-zero exit and no data touched, as designed.
6. Found and fixed two real bugs *during* this verification (not just
   theoretical review): the container entrypoint originally ignored a passed
   command and always started the cron daemon instead of running a requested
   one-off backup; and `mc mirror`'s transfer-summary table was leaking onto
   stdout despite `--quiet`, corrupting the captured snapshot path and causing
   `verify-minio-backup.sh` to silently no-op instead of actually verifying.
   Both are fixed in the scripts as shipped, and this is exactly the class of
   bug that "the command exited 0" would have hidden — which is the reason
   this verification methodology exists.

All test artifacts (marker row/table, marker object, synthetic prune fixtures)
were removed after verification; the backup volume is empty and ready for the
first real scheduled run.

## Restore procedure (disaster recovery runbook)

### Postgres

1. Identify the backup to restore from:
   ```
   docker compose run --rm backup ls -la /backups/postgres
   ```
2. Restore into a **new** database first, never directly over the primary,
   so you can validate before cutting over:
   ```
   make restore-postgres FILE=/backups/postgres/taskworked-<timestamp>.dump TARGET_DB=taskworked_restore_check
   ```
3. Validate — connect and spot-check row counts / recent data:
   ```
   docker compose exec postgres psql -U <user> -d taskworked_restore_check -c "SELECT count(*) FROM tasks;"
   ```
4. Once validated, restore over the real database (only when you're actually
   recovering from data loss — this drops and recreates every object in the
   target database):
   ```
   make restore-postgres FILE=/backups/postgres/taskworked-<timestamp>.dump TARGET_DB=taskworked CONFIRM=--yes-i-am-sure
   ```
5. Restart the `api` service so it reconnects cleanly:
   ```
   docker compose restart api
   ```

If the dump was encrypted (`BACKUP_ENCRYPTION_PASSPHRASE` was set at backup
time), the same passphrase must be present in the backup service's environment
when restoring — `restore-postgres.sh` decrypts automatically when the file
ends in `.enc`.

### MinIO

1. Identify the snapshot:
   ```
   docker compose run --rm backup ls /backups/minio
   ```
2. Restore into a scratch bucket first and spot-check:
   ```
   make restore-minio DIR=/backups/minio/<timestamp> TARGET_BUCKET=taskworked-restore-check
   ```
3. Once validated, restore into the real bucket:
   ```
   make restore-minio DIR=/backups/minio/<timestamp> TARGET_BUCKET=taskworked CONFIRM=--yes-i-am-sure
   ```

## Production configuration checklist

The defaults in `.env.example` are safe for local development but **not**
sufficient for production on their own:

- [ ] **Set `BACKUP_ENCRYPTION_PASSPHRASE`** — a long random value
      (`openssl rand -base64 32`), stored only in your production secrets
      store, never in source control. Without this, dumps sit on disk in
      plaintext.
- [ ] **Move backups off the host.** `backup_data` is a local Docker volume on
      the same machine as the primary database — a disk failure or host loss
      destroys the primary data *and* the backups together. This
      implementation does not include an offsite target because none is
      available to configure from this environment; before production
      traffic, add a sync step (e.g. `rclone`/`mc mirror` from `/backups` to an
      S3-compatible bucket on different infrastructure, or a scheduled `docker
      cp`/`rsync` to another host) as a final step in `run-full-backup.sh`, or
      as a separate sidecar. This is the single most important item on this
      checklist — **do not consider backups production-ready without it.**
- [ ] **Set `BACKUP_WEBHOOK_URL`** to a real alerting relay so a failed backup
      run actually pages someone instead of only appearing in container logs.
- [ ] Decide on a tighter `BACKUP_SCHEDULE_CRON` if 24h RPO isn't tight enough
      for the business.
- [ ] Periodically *actually run* the restore procedure above against a
      scratch environment (not just trust the automated verification) — a
      quarterly restore drill is the industry-standard practice and catches
      environment-specific issues (e.g. a Postgres major-version mismatch)
      that in-place verification can't.

## Known limitations

- Backups are stored on the same host as the primary database by default (see
  checklist above) — this implementation solves "can I actually restore from a
  backup" but the *offsite* half of durability needs a real production target
  wired in before go-live.
- MinIO backup is a full mirror, not incremental — fine at the attachment
  volumes this app currently handles; revisit (e.g. MinIO bucket versioning +
  replication) if attachment volume grows large enough that a full nightly
  mirror becomes slow or expensive.
- Redis is not backed up (by design — see "What's covered" above).
