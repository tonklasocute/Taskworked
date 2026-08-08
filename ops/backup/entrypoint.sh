#!/bin/bash
set -euo pipefail

# Renders the backup schedule into a crontab and runs it in the foreground.
# Set BACKUP_SCHEDULE_CRON to a standard 5-field cron expression (default: daily
# at 02:00 UTC). Set RUN_BACKUP_ON_START=true to also run one backup immediately
# on container start (useful for first deploy / local verification).

: "${BACKUP_SCHEDULE_CRON:=0 2 * * *}"

# If a command was passed (e.g. `docker compose run backup /scripts/run-full-backup.sh`
# for an on-demand run, or `make backup`/`make backup-verify`), run exactly that
# and exit — don't fall through into the long-lived cron daemon. Only start
# crond when the container is run with no override, i.e. as the scheduled
# background service in `docker compose up`.
if [ "$#" -gt 0 ]; then
  exec "$@"
fi

echo "[entrypoint] backup service starting, schedule: ${BACKUP_SCHEDULE_CRON}"

# crond needs the job's environment explicitly exported into the crontab line,
# since cron jobs otherwise run with a near-empty environment.
env | grep -E '^(PG|MINIO_|BACKUP_)' > /etc/backup.env || true

{
  echo "${BACKUP_SCHEDULE_CRON} . /etc/backup.env; /scripts/run-full-backup.sh >> /proc/1/fd/1 2>> /proc/1/fd/2"
} > /etc/crontabs/root

if [ "${RUN_BACKUP_ON_START:-false}" = "true" ]; then
  echo "[entrypoint] RUN_BACKUP_ON_START=true, running an initial backup now"
  /scripts/run-full-backup.sh || echo "[entrypoint] initial backup failed, cron will retry on schedule" >&2
fi

exec crond -f -l 2
