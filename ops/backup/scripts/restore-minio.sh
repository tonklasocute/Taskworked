#!/bin/bash
set -euo pipefail

# Restores a MinIO backup snapshot directory back into a target bucket.
# Refuses to restore over the primary bucket (MINIO_BUCKET) unless
# --yes-i-am-sure is passed.
#
# Usage: restore-minio.sh <snapshot-dir> <target-bucket> [--yes-i-am-sure]
#
# Required env: MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY

SNAPSHOT_DIR="${1:?usage: restore-minio.sh <snapshot-dir> <target-bucket> [--yes-i-am-sure]}"
TARGET_BUCKET="${2:?usage: restore-minio.sh <snapshot-dir> <target-bucket> [--yes-i-am-sure]}"
CONFIRM="${3:-}"

: "${MINIO_ENDPOINT:?}"
: "${MINIO_ACCESS_KEY:?}"
: "${MINIO_SECRET_KEY:?}"

log() { echo "[restore-minio] $*" >&2; }

if [ "$TARGET_BUCKET" = "${MINIO_BUCKET:-}" ] && [ "$CONFIRM" != "--yes-i-am-sure" ]; then
  log "REFUSING to restore over primary bucket '${TARGET_BUCKET}' without --yes-i-am-sure"
  exit 1
fi

if [ ! -d "$SNAPSHOT_DIR" ]; then
  log "ERROR: snapshot directory not found: ${SNAPSHOT_DIR}"
  exit 1
fi

mc alias set target "http://${MINIO_ENDPOINT}" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY" >/dev/null
mc mb --ignore-existing "target/${TARGET_BUCKET}" >&2

log "mirroring ${SNAPSHOT_DIR} -> target/${TARGET_BUCKET}"
# mc writes its transfer-summary table to stdout even with --quiet; keep this
# script's stdout clean of it (some callers, e.g. verify-minio-backup.sh, treat
# unexpected stdout content as significant).
mc mirror --overwrite --quiet "$SNAPSHOT_DIR" "target/${TARGET_BUCKET}" >&2

log "restore complete"
