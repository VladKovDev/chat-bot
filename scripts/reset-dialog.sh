#!/usr/bin/env sh
set -eu

if [ "${1:-}" = "" ]; then
  echo "usage: $0 <session-id> [reason]" >&2
  exit 2
fi

if [ "${ADMIN_RESET_TOKEN:-}" = "" ]; then
  echo "ADMIN_RESET_TOKEN is required" >&2
  exit 2
fi

SESSION_ID="$1"
REASON="${2:-manual reset from local script}"
DECISION_ENGINE_URL="${DECISION_ENGINE_URL:-http://localhost:8080}"
BODY="$(python3 -c 'import json, sys; print(json.dumps({"reason": sys.argv[1]}, ensure_ascii=False))' "$REASON")"

curl -fsS \
  -X POST \
  -H "Content-Type: application/json" \
  -H "X-Admin-Token: ${ADMIN_RESET_TOKEN}" \
  --data "$BODY" \
  "${DECISION_ENGINE_URL}/api/v1/admin/sessions/${SESSION_ID}/reset"
