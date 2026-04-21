#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"

log() {
  printf '\n[check] %s\n' "$1"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf '[check] missing required command: %s\n' "$1" >&2
    exit 1
  fi
}

require_cmd go
require_cmd node
require_cmd npm
require_cmd python3
require_cmd git

cd "$ROOT_DIR"

log "Go webpanel tests"
go test ./app/webpanel/...

log "Locale sync"
bash scripts/check-locale-sync.sh

log "Local-state guard"
bash scripts/check-local-state-guard.sh

if [[ ! -d "$ROOT_DIR/web/node_modules" ]]; then
  log "Install web dependencies"
  (
    cd web
    npm ci
  )
fi

log "Web tests"
(
  cd web
  npm run test
)

log "Web build"
(
  cd web
  npm run build
)

log "Web smoke test"
PYTHONDONTWRITEBYTECODE=1 python3 tests/test_web_smoke.py

log "All checks passed"
