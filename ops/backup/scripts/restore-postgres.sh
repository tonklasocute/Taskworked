#!/bin/bash
set -euo pipefail

# Restores a Postgres dump (produced by backup-postgres.sh) into a target
# database. Refuses to restore over the primary database (PGDATABASE) unless
# --yes-i-am-sure is passed, to prevent an accidental production overwrite.
#
# Usage: restore-postgres.sh <dump-file> <target-db> [--yes-i-am-sure]
#
# Required env: PGHOST, PGPORT, PGUSER, PGPASSWORD
# Optional env: BACKUP_ENCRYPTION_PASSPHRASE (required if the dump is *.enc)

DUMP_FILE="${1:?usage: restore-postgres.sh <dump-file> <target-db> [--yes-i-am-sure]}"
TARGET_DB="${2:?usage: restore-postgres.sh <dump-file> <target-db> [--yes-i-am-sure]}"
CONFIRM="${3:-}"

: "${PGHOST:?}"
: "${PGUSER:?}"
: "${PGPASSWORD:?}"

log() { echo "[restore-postgres] $*" >&2; }

if [ "$TARGET_DB" = "${PGDATABASE:-}" ] && [ "$CONFIRM" != "--yes-i-am-sure" ]; then
  log "REFUSING to restore over primary database '${TARGET_DB}' without --yes-i-am-sure"
  exit 1
fi

if [ ! -f "$DUMP_FILE" ]; then
  log "ERROR: dump file not found: ${DUMP_FILE}"
  exit 1
fi

WORK_FILE="$DUMP_FILE"
CLEANUP_TMP=""
case "$DUMP_FILE" in
  *.enc)
    : "${BACKUP_ENCRYPTION_PASSPHRASE:?dump is encrypted; set BACKUP_ENCRYPTION_PASSPHRASE to decrypt}"
    WORK_FILE="/tmp/$(basename "${DUMP_FILE%.enc}")"
    log "decrypting ${DUMP_FILE}"
    openssl enc -d -aes-256-cbc -pbkdf2 \
      -pass "pass:${BACKUP_ENCRYPTION_PASSPHRASE}" \
      -in "$DUMP_FILE" -out "$WORK_FILE"
    CLEANUP_TMP="$WORK_FILE"
    ;;
esac
trap '[ -n "$CLEANUP_TMP" ] && rm -f "$CLEANUP_TMP"' EXIT

log "ensuring database '${TARGET_DB}' exists"
EXISTS=$(PGDATABASE=postgres psql -tA -c \
  "SELECT 1 FROM pg_database WHERE datname = '${TARGET_DB}'")
if [ "$EXISTS" != "1" ]; then
  PGDATABASE=postgres psql -v ON_ERROR_STOP=1 -c "CREATE DATABASE \"${TARGET_DB}\""
fi

log "restoring ${WORK_FILE} into '${TARGET_DB}'"
pg_restore --clean --if-exists --no-owner --no-privileges \
  --dbname="$TARGET_DB" "$WORK_FILE"

log "restore complete"
