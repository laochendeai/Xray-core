#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib-webpanel.sh
source "$SCRIPT_DIR/lib-webpanel.sh"

usage() {
  cat <<'EOF'
usage: probe-webpanel-outbound.sh --target TAG [--config PATH] [--base-url URL] [--balancer-tag TAG] [--probe-url URL] [--method HEAD|GET] [--socks-addr HOST:PORT] [--timeout-sec N]

Temporarily overrides a routing balancer target, probes traffic through the local SOCKS
inbound, then restores the previous balancer target.

Examples:
  ./scripts/probe-webpanel-outbound.sh --target proxy-01
  ./scripts/probe-webpanel-outbound.sh --target pool_db7e14360e59 --method GET
EOF
}

CONFIG_PATH="$(webpanel_default_config_path)"
BASE_URL_OVERRIDE=""
BALANCER_TAG="auto"
TARGET_TAG=""
PROBE_URL="https://www.gstatic.com/generate_204"
METHOD="HEAD"
SOCKS_ADDR="127.0.0.1:11080"
TIMEOUT_SEC=12

while [[ $# -gt 0 ]]; do
  case "$1" in
    --config)
      CONFIG_PATH="$2"
      shift 2
      ;;
    --base-url)
      BASE_URL_OVERRIDE="$2"
      shift 2
      ;;
    --balancer-tag)
      BALANCER_TAG="$2"
      shift 2
      ;;
    --target)
      TARGET_TAG="$2"
      shift 2
      ;;
    --probe-url)
      PROBE_URL="$2"
      shift 2
      ;;
    --method)
      METHOD="$(printf '%s' "$2" | tr '[:lower:]' '[:upper:]')"
      shift 2
      ;;
    --socks-addr)
      SOCKS_ADDR="$2"
      shift 2
      ;;
    --timeout-sec)
      TIMEOUT_SEC="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$TARGET_TAG" ]]; then
  echo "--target is required" >&2
  usage >&2
  exit 2
fi

if [[ "$METHOD" != "HEAD" && "$METHOD" != "GET" ]]; then
  echo "--method must be HEAD or GET" >&2
  exit 2
fi

require_command curl
require_command python3

webpanel_load_config "$CONFIG_PATH" "$BASE_URL_OVERRIDE"
TOKEN="$(webpanel_login)"

CURRENT_TARGET=""
RESTORE_NEEDED=false

get_balancer_target() {
  local output_file code
  output_file="$(mktemp)"
  code="$(webpanel_api_get "$TOKEN" "/api/v1/routing/balancers/$BALANCER_TAG" "$output_file")" || code="curl_failed"
  if [[ "$code" != "200" ]]; then
    cat "$output_file" >&2 || true
    rm -f "$output_file"
    echo "Failed to read balancer $BALANCER_TAG: $code" >&2
    exit 1
  fi

  python3 - "$output_file" <<'PY'
import json
import sys
from pathlib import Path

data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
balancer = data.get("balancer") or {}
override = balancer.get("override") or {}
principal = ((balancer.get("principle_target") or {}).get("tag") or [])
if override.get("target"):
    print(override["target"])
elif principal:
    print(principal[0])
PY
  rm -f "$output_file"
}

set_balancer_target() {
  local target="$1" output_file payload code
  output_file="$(mktemp)"
  payload="$(python3 - "$target" <<'PY'
import json
import sys
print(json.dumps({"target": sys.argv[1]}))
PY
)"

  code="$(webpanel_http_json "$output_file" \
    -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -X PUT \
    -d "$payload" \
    "$WEBPANEL_BASE_URL/api/v1/routing/balancers/$BALANCER_TAG")" || code="curl_failed"

  if [[ "$code" != "200" ]]; then
    cat "$output_file" >&2 || true
    rm -f "$output_file"
    echo "Failed to override balancer $BALANCER_TAG to $target: $code" >&2
    exit 1
  fi

  rm -f "$output_file"
}

restore_balancer() {
  if [[ "$RESTORE_NEEDED" != "true" || -z "$CURRENT_TARGET" ]]; then
    return
  fi
  if set_balancer_target "$CURRENT_TARGET"; then
    printf 'INFO restored balancer %s -> %s\n' "$BALANCER_TAG" "$CURRENT_TARGET"
  else
    printf 'WARN failed to restore balancer %s -> %s\n' "$BALANCER_TAG" "$CURRENT_TARGET" >&2
  fi
}

trap restore_balancer EXIT

CURRENT_TARGET="$(get_balancer_target)"
printf 'INFO current balancer %s target: %s\n' "$BALANCER_TAG" "${CURRENT_TARGET:-<empty>}"

set_balancer_target "$TARGET_TAG"
RESTORE_NEEDED=true
printf 'INFO probing via balancer %s target: %s\n' "$BALANCER_TAG" "$TARGET_TAG"

set +e
if [[ "$METHOD" == "HEAD" ]]; then
  curl -I -sS --max-time "$TIMEOUT_SEC" --socks5-hostname "$SOCKS_ADDR" "$PROBE_URL"
  CURL_EXIT=$?
else
  curl -sS -D - -o /dev/null --max-time "$TIMEOUT_SEC" --socks5-hostname "$SOCKS_ADDR" "$PROBE_URL"
  CURL_EXIT=$?
fi
set -e

printf 'INFO curl exit: %s\n' "$CURL_EXIT"
exit "$CURL_EXIT"
