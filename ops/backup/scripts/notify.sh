#!/bin/bash
set -euo pipefail

# Logs a backup-run outcome and, if BACKUP_WEBHOOK_URL is configured, POSTs a
# small JSON payload to it. The webhook is generic (any endpoint that accepts
# a JSON POST — e.g. an internal alerting relay) so this script doesn't hardcode
# any particular provider. A failed webhook POST is logged but never fails the
# backup job itself — notification delivery must not mask a real backup result.
#
# Usage: notify.sh <ok|fail> <message>

STATUS="${1:?usage: notify.sh <ok|fail> <message>}"
MESSAGE="${2:?usage: notify.sh <ok|fail> <message>}"
TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

echo "[notify] ${TS} status=${STATUS} ${MESSAGE}"

if [ -n "${BACKUP_WEBHOOK_URL:-}" ]; then
  ESCAPED_MESSAGE=$(printf '%s' "$MESSAGE" | sed 's/\\/\\\\/g; s/"/\\"/g')
  BODY=$(printf '{"service":"taskworked-backup","status":"%s","message":"%s","timestamp":"%s"}' \
    "$STATUS" "$ESCAPED_MESSAGE" "$TS")
  if ! curl -fsS -m 10 -X POST "$BACKUP_WEBHOOK_URL" \
      -H 'Content-Type: application/json' \
      -d "$BODY" >/dev/null; then
    echo "[notify] WARNING: webhook POST failed — see the status line above for the actual backup result" >&2
  fi
fi
