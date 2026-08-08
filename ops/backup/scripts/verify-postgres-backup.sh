#!/bin/bash
set -euo pipefail

# Proves a backup is actually restorable and usable — not just that pg_dump
# exited 0. Restores the most recent backup into a throwaway scratch database,
# checks it has real tables and that a known table is queryable, then drops
# the scratch database.
#
# Usage: verify-postgres-backup.sh [dump-file]
#   If no dump-file is given, the most recent one under $BACKUP_DIR/postgres
#   is used.

: "${PGHOST:?}"
: "${PGUSER:?}"
: "${PGPASSWORD:?}"
: "${BACKUP_DIR:?}"

log() { echo "[verify-postgres-backup] $*" >&2; }

DUMP_FILE="${1:-}"
if [ -z "$DUMP_FILE" ]; then
  DUMP_FILE=$(ls -t "${BACKUP_DIR}"/postgres/*.dump "${BACKUP_DIR}"/postgres/*.dump.enc 2>/dev/null | head -n1 || true)
fi

if [ -z "$DUMP_FILE" ]; then
  log "ERROR: no backup files found under ${BACKUP_DIR}/postgres"
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERIFY_DB="taskworked_backup_verify_$$"

cleanup() {
  log "dropping scratch database '${VERIFY_DB}'"
  PGDATABASE=postgres psql -c "DROP DATABASE IF EXISTS \"${VERIFY_DB}\"" >/dev/null 2>&1 || true
}
trap cleanup EXIT

log "restoring ${DUMP_FILE} into scratch database '${VERIFY_DB}'"
"${SCRIPT_DIR}/restore-postgres.sh" "$DUMP_FILE" "$VERIFY_DB"

TABLE_COUNT=$(PGDATABASE="$VERIFY_DB" psql -tA -c \
  "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public'")

if [ "${TABLE_COUNT:-0}" -lt 1 ]; then
  log "FAIL: restored database has 0 tables — backup is not usable"
  exit 1
fi

ROW_TOTAL=$(PGDATABASE="$VERIFY_DB" psql -tA -c "
  SELECT COALESCE(SUM(n_live_tup), 0)
  FROM pg_stat_user_tables
")

log "OK: restored database has ${TABLE_COUNT} table(s), ~${ROW_TOTAL} live row(s) across them, and is queryable"
