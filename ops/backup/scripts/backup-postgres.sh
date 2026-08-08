#!/bin/bash
set -euo pipefail

# Dumps the Postgres database to a timestamped, compressed, custom-format file.
# Prints the resulting file path on stdout (and nothing else on stdout) so
# callers can capture it directly: FILE=$(./backup-postgres.sh)
#
# Required env (standard libpq vars, picked up automatically by pg_dump):
#   PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE
# Required env:
#   BACKUP_DIR — directory to write dumps into (a subdir "postgres" is created)
# Optional env:
#   BACKUP_ENCRYPTION_PASSPHRASE — if set, the dump is encrypted with
#     openssl aes-256-cbc and the plaintext dump is removed.

: "${BACKUP_DIR:?BACKUP_DIR must be set}"
: "${PGDATABASE:?PGDATABASE must be set}"

log() { echo "[backup-postgres] $*" >&2; }

TS="$(date -u +%Y%m%dT%H%M%SZ)"
OUT_DIR="${BACKUP_DIR}/postgres"
mkdir -p "$OUT_DIR"
DUMP_FILE="${OUT_DIR}/${PGDATABASE}-${TS}.dump"

log "starting dump of database '${PGDATABASE}' at ${TS}"

pg_dump --format=custom --compress=9 --no-owner --no-privileges \
  --file="${DUMP_FILE}.tmp"

mv "${DUMP_FILE}.tmp" "$DUMP_FILE"

if [ -n "${BACKUP_ENCRYPTION_PASSPHRASE:-}" ]; then
  log "encrypting dump"
  openssl enc -aes-256-cbc -pbkdf2 -salt \
    -pass "pass:${BACKUP_ENCRYPTION_PASSPHRASE}" \
    -in "$DUMP_FILE" -out "${DUMP_FILE}.enc"
  rm -f "$DUMP_FILE"
  DUMP_FILE="${DUMP_FILE}.enc"
fi

SIZE=$(stat -c%s "$DUMP_FILE" 2>/dev/null || wc -c < "$DUMP_FILE")
if [ "$SIZE" -lt 100 ]; then
  log "ERROR: dump file is suspiciously small (${SIZE} bytes) — treating as failed"
  exit 1
fi

log "wrote ${DUMP_FILE} (${SIZE} bytes)"
printf '%s\n' "$DUMP_FILE"
