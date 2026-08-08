#!/bin/bash
set -euo pipefail

# Enforces the retention policy: removes Postgres dump files and MinIO
# snapshot directories older than BACKUP_RETENTION_DAYS (default 14).

: "${BACKUP_DIR:?}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-14}"

log() { echo "[prune-backups] $*" >&2; }

log "removing backups older than ${RETENTION_DAYS} day(s) under ${BACKUP_DIR}"

if [ -d "${BACKUP_DIR}/postgres" ]; then
  find "${BACKUP_DIR}/postgres" -maxdepth 1 -type f -mtime "+${RETENTION_DAYS}" -print -delete
fi

if [ -d "${BACKUP_DIR}/minio" ]; then
  find "${BACKUP_DIR}/minio" -mindepth 1 -maxdepth 1 -type d -mtime "+${RETENTION_DAYS}" -print -exec rm -rf {} +
fi

log "done"
