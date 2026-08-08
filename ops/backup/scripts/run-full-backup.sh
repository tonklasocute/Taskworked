#!/bin/bash
set -euo pipefail

# Orchestrates one full backup run: postgres backup -> verify -> minio backup
# -> verify -> prune -> notify. This is the entrypoint cron calls on schedule,
# and what `make backup` runs on demand.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
STARTED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

fail() {
  local reason="$1"
  echo "[run-full-backup] FAILED: ${reason}" >&2
  "${SCRIPT_DIR}/notify.sh" fail "${reason}"
  exit 1
}

echo "[run-full-backup] starting full backup run at ${STARTED_AT}"

PG_DUMP_FILE=$("${SCRIPT_DIR}/backup-postgres.sh") || fail "postgres backup failed"
"${SCRIPT_DIR}/verify-postgres-backup.sh" "$PG_DUMP_FILE" || fail "postgres backup verification failed"

MINIO_SNAPSHOT=$("${SCRIPT_DIR}/backup-minio.sh") || fail "minio backup failed"
if [ -n "${MINIO_SNAPSHOT:-}" ]; then
  "${SCRIPT_DIR}/verify-minio-backup.sh" "$MINIO_SNAPSHOT" || fail "minio backup verification failed"
fi

"${SCRIPT_DIR}/prune-backups.sh" || fail "backup pruning failed"

SUMMARY="postgres=${PG_DUMP_FILE} minio=${MINIO_SNAPSHOT:-none}"
echo "[run-full-backup] completed successfully: ${SUMMARY}"
"${SCRIPT_DIR}/notify.sh" ok "backup completed: ${SUMMARY}"
