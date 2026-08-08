#!/bin/bash
set -euo pipefail

# Mirrors the MinIO bucket to a timestamped local snapshot directory.
# Prints the resulting snapshot directory path on stdout.
#
# Required env: BACKUP_DIR, MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY,
#   MINIO_BUCKET
# Optional env: BACKUP_MINIO_ENABLED (default true) — set to "false" to skip
#   MinIO backup entirely (e.g. deployments not using object storage).

: "${BACKUP_DIR:?}"
: "${MINIO_ENDPOINT:?}"
: "${MINIO_ACCESS_KEY:?}"
: "${MINIO_SECRET_KEY:?}"
: "${MINIO_BUCKET:?}"

log() { echo "[backup-minio] $*" >&2; }

if [ "${BACKUP_MINIO_ENABLED:-true}" != "true" ]; then
  log "BACKUP_MINIO_ENABLED=false, skipping"
  exit 0
fi

mc alias set source "http://${MINIO_ENDPOINT}" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY" >/dev/null

if ! mc ls "source/${MINIO_BUCKET}" >/dev/null 2>&1; then
  log "bucket '${MINIO_BUCKET}' does not exist yet — nothing to back up"
  exit 0
fi

TS="$(date -u +%Y%m%dT%H%M%SZ)"
OUT_DIR="${BACKUP_DIR}/minio/${TS}"
mkdir -p "$OUT_DIR"

log "mirroring source/${MINIO_BUCKET} -> ${OUT_DIR}"
# mc writes its transfer-summary table to stdout even with --quiet; redirect it
# to stderr so stdout only ever carries the snapshot path this script returns.
mc mirror --overwrite --quiet "source/${MINIO_BUCKET}" "${OUT_DIR}" >&2

FILE_COUNT=$(find "$OUT_DIR" -type f | wc -l)
log "mirrored ${FILE_COUNT} object(s) to ${OUT_DIR}"

if [ "$FILE_COUNT" -eq 0 ]; then
  # Empty bucket is legitimate (fresh install) — remove the empty snapshot dir
  # so verify/restore scripts don't treat it as a real backup to check.
  rmdir "$OUT_DIR" 2>/dev/null || true
  log "bucket is empty, no snapshot retained"
  exit 0
fi

printf '%s\n' "$OUT_DIR"
