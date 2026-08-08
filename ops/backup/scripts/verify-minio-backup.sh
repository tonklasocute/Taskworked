#!/bin/bash
set -euo pipefail

# Proves the most recent MinIO backup snapshot is actually restorable: mirrors
# it into a throwaway scratch bucket and compares file counts + spot-checks a
# checksum, then removes the scratch bucket.
#
# Usage: verify-minio-backup.sh [snapshot-dir]

: "${BACKUP_DIR:?}"
: "${MINIO_ENDPOINT:?}"
: "${MINIO_ACCESS_KEY:?}"
: "${MINIO_SECRET_KEY:?}"

log() { echo "[verify-minio-backup] $*" >&2; }

SNAPSHOT_DIR="${1:-}"
if [ -z "$SNAPSHOT_DIR" ]; then
  # No snapshot given explicitly: auto-discover the latest one, and it's fine
  # for none to exist yet (e.g. bucket was empty at backup time) — skip quietly.
  SNAPSHOT_DIR=$(ls -td "${BACKUP_DIR}"/minio/*/ 2>/dev/null | head -n1 || true)
  SNAPSHOT_DIR="${SNAPSHOT_DIR%/}"
  if [ -z "$SNAPSHOT_DIR" ] || [ ! -d "$SNAPSHOT_DIR" ]; then
    log "no MinIO snapshot found under ${BACKUP_DIR}/minio — skipping (nothing has been backed up yet)"
    exit 0
  fi
elif [ ! -d "$SNAPSHOT_DIR" ]; then
  # A snapshot path WAS given explicitly (e.g. by run-full-backup.sh, which
  # just produced it) and it doesn't exist — that's a real failure, not an
  # "nothing to verify yet" situation.
  log "ERROR: snapshot directory '${SNAPSHOT_DIR}' was specified but does not exist"
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERIFY_BUCKET="taskworked-backup-verify-$$"

cleanup() {
  log "removing scratch bucket '${VERIFY_BUCKET}'"
  mc alias set target "http://${MINIO_ENDPOINT}" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY" >/dev/null 2>&1 || true
  mc rb --force --dangerous "target/${VERIFY_BUCKET}" >/dev/null 2>&1 || true
}
trap cleanup EXIT

log "restoring ${SNAPSHOT_DIR} into scratch bucket '${VERIFY_BUCKET}'"
"${SCRIPT_DIR}/restore-minio.sh" "$SNAPSHOT_DIR" "$VERIFY_BUCKET" --yes-i-am-sure

SOURCE_COUNT=$(find "$SNAPSHOT_DIR" -type f | wc -l)
# `mc find` lists objects (S3 has no separate directory entries), no --type
# flag needed/supported — unlike GNU find.
RESTORED_COUNT=$(mc find "target/${VERIFY_BUCKET}" 2>/dev/null | wc -l)

if [ "$RESTORED_COUNT" -ne "$SOURCE_COUNT" ]; then
  log "FAIL: source snapshot had ${SOURCE_COUNT} file(s), restored bucket has ${RESTORED_COUNT}"
  exit 1
fi

if [ "$SOURCE_COUNT" -gt 0 ]; then
  SAMPLE_REL=$(find "$SNAPSHOT_DIR" -type f | head -n1 | sed "s|^${SNAPSHOT_DIR}/||")
  SOURCE_HASH=$(sha256sum "${SNAPSHOT_DIR}/${SAMPLE_REL}" | awk '{print $1}')
  RESTORED_HASH=$(mc cat "target/${VERIFY_BUCKET}/${SAMPLE_REL}" 2>/dev/null | sha256sum | awk '{print $1}')
  if [ "$SOURCE_HASH" != "$RESTORED_HASH" ]; then
    log "FAIL: checksum mismatch on sample object '${SAMPLE_REL}' (source=${SOURCE_HASH} restored=${RESTORED_HASH})"
    exit 1
  fi
  log "sample object '${SAMPLE_REL}' checksum verified (${SOURCE_HASH})"
fi

log "OK: ${RESTORED_COUNT} object(s) restored and verified present in scratch bucket"
