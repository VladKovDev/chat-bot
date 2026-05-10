#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR"

run() {
  printf '\n==> %s\n' "$*"
  "$@"
}

run_shell() {
  printf '\n==> %s\n' "$1"
  bash -c "$1"
}

run git diff --check

run_shell "ruby - <<'RUBY'
require 'json'
require 'psych'

json_files = Dir.glob('**/*.json', File::FNM_DOTMATCH).reject do |path|
  path.start_with?('.git/', '.beads/') || path.include?('/node_modules/') || path.include?('/__pycache__/')
end
yaml_files = Dir.glob('**/*.{yaml,yml}', File::FNM_DOTMATCH).reject do |path|
  path.start_with?('.git/', '.beads/') || path.include?('/node_modules/') || path.include?('/__pycache__/')
end

json_files.each do |path|
  JSON.parse(File.read(path))
end

yaml_files.each do |path|
  Psych.load_file(path)
end

puts \"validated #{json_files.length} JSON and #{yaml_files.length} YAML files\"
RUBY"

run_shell "! rg -n 'llmClient\\.Decide|/llm/decide|Ollama|GigaChat' services/decision-engine/internal services/transport-adapters/website/backend/internal services/transport-adapters/website/frontend/assets/js services/nlp-service/app"

UV_ENV_DIR="$(mktemp -d "${TMPDIR:-/tmp}/chat-bot-nlp-core-venv.XXXXXX")"
trap 'rm -rf "$UV_ENV_DIR"' EXIT

run_shell "cd services/decision-engine && go test -count=1 ./..."
run_shell "cd services/nlp-service && PYTHONDONTWRITEBYTECODE=1 UV_PROJECT_ENVIRONMENT='$UV_ENV_DIR' uv run --extra test python -m pytest -p no:cacheprovider"
run_shell "cd services/transport-adapters/website/backend && go test -count=1 ./..."
run_shell "node --test services/transport-adapters/website/frontend/assets/js/*.test.mjs"
