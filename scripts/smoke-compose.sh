#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROJECT_NAME="${COMPOSE_PROJECT_NAME:-chat-bot-smoke}"
READY_URL="${READY_URL:-http://localhost:${DECISION_ENGINE_PORT:-8080}/api/v1/ready}"

cd "$ROOT_DIR"

cleanup() {
  if [[ "${KEEP_COMPOSE:-0}" != "1" ]]; then
    docker compose -p "$PROJECT_NAME" down -v --remove-orphans
  fi
}
trap cleanup EXIT

docker compose -p "$PROJECT_NAME" down -v --remove-orphans
docker compose -p "$PROJECT_NAME" up --build -d

for attempt in $(seq 1 60); do
  status="$(curl -fsS -o /tmp/chat-bot-ready.json -w '%{http_code}' "$READY_URL" || true)"
  if [[ "$status" == "200" ]]; then
    cat /tmp/chat-bot-ready.json
    printf '\n'
    exit 0
  fi
  printf 'readiness attempt %s returned %s\n' "$attempt" "${status:-curl-error}"
  sleep 2
done

docker compose -p "$PROJECT_NAME" ps
docker compose -p "$PROJECT_NAME" logs --no-color --tail=200 decision-migrate decision-engine nlp-service postgres
exit 1
