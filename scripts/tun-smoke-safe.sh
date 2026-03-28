#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
# shellcheck source=./lib-webpanel.sh
source "$SCRIPT_DIR/lib-webpanel.sh"

CONFIG_PATH="${CONFIG_PATH:-$ROOT_DIR/dev-config.current-nodes.json}"
RESTORE_SCRIPT="${RESTORE_SCRIPT:-/home/leo-cy/桌面/restore-network-clean.sh}"
PANEL_URL="${PANEL_URL:-}"
STATE_DIR="${STATE_DIR:-$ROOT_DIR/runtime/tun}"
WATCHDOG_SECONDS="${WATCHDOG_SECONDS:-90}"

mkdir -p "$STATE_DIR"
LOG_FILE="${LOG_FILE:-$STATE_DIR/smoke-$(date +%Y%m%d-%H%M%S).log}"

log() {
  printf '[smoke] %s\n' "$*" | tee -a "$LOG_FILE"
}

snapshot_state() {
  local label="$1"
  {
    printf '\n== %s ==\n' "$label"
    date
    ip link show "$WEBPANEL_TUN_INTERFACE" || true
    printf '\nRULES\n'
    ip -4 rule show || true
    printf '\nROUTES MAIN\n'
    ip route show table main || true
    printf '\nROUTES BYPASS (%s)\n' "$WEBPANEL_BYPASS_ROUTE_TABLE_ID"
    ip route show table "$WEBPANEL_BYPASS_ROUTE_TABLE_ID" || true
    printf '\nROUTES CAPTURE (%s)\n' "$WEBPANEL_CAPTURE_ROUTE_TABLE_ID"
    ip route show table "$WEBPANEL_CAPTURE_ROUTE_TABLE_ID" || true
    printf '\nDNS\n'
    resolvectl status | sed -n '1,80p' || true
  } >>"$LOG_FILE" 2>&1
}

run_curl_probe() {
  local label="$1"
  local url="$2"
  shift 2
  {
    printf '\n== PROBE %s ==\n' "$label"
    curl -I -L --max-time 15 -sS "$@" \
      -o /dev/null \
      -w 'http_code=%{http_code} remote_ip=%{remote_ip} time_total=%{time_total} url=%{url_effective}\n' \
      "$url"
  } >>"$LOG_FILE" 2>&1 || {
    log "probe failed: $label"
    return 1
  }
}

api_post_checked() {
  local path="$1"
  local label="$2"
  local output_file status_code

  output_file="$(mktemp)"
  status_code="$(webpanel_api_post "$TOKEN" "$path" "$output_file" 2>/dev/null || true)"
  {
    printf '\n== API %s ==\n' "$label"
    printf 'http_status=%s\n' "${status_code:-curl_failed}"
    cat "$output_file"
    printf '\n'
  } >>"$LOG_FILE" 2>&1

  if [[ "$status_code" != "200" ]]; then
    rm -f "$output_file"
    log "api call failed: $label http_status=${status_code:-curl_failed}"
    return 1
  fi

  rm -f "$output_file"
}

fetch_tun_status() {
  local label="$1"
  local output_file status_code

  output_file="$(mktemp)"
  status_code="$(webpanel_api_get "$TOKEN" "/api/v1/tun/status" "$output_file" 2>/dev/null || true)"
  {
    printf '\n== STATUS %s ==\n' "$label"
    printf 'http_status=%s\n' "${status_code:-curl_failed}"
    cat "$output_file"
    printf '\n'
  } >>"$LOG_FILE" 2>&1

  if [[ "$status_code" != "200" ]]; then
    rm -f "$output_file"
    log "status fetch failed: $label http_status=${status_code:-curl_failed}"
    return 1
  fi

  TUN_RUNNING="$(json_get "$output_file" "running" 2>/dev/null || true)"
  TUN_MACHINE_STATE="$(json_get "$output_file" "machineState" 2>/dev/null || true)"
  TUN_REASON="$(json_get "$output_file" "lastStateReason" 2>/dev/null || true)"
  rm -f "$output_file"
}

cleanup() {
  local exit_code="$1"
  log "cleanup start (exit=$exit_code)"
  if [[ -n "${WATCHDOG_PID:-}" ]]; then
    kill "$WATCHDOG_PID" >/dev/null 2>&1 || true
  fi
  bash "$RESTORE_SCRIPT" --config "$CONFIG_PATH" >>"$LOG_FILE" 2>&1 || true
  snapshot_state "after-restore"
  log "cleanup done"
}

trap 'cleanup $?' EXIT

webpanel_load_config "$CONFIG_PATH" "$PANEL_URL"
if ! TOKEN="$(webpanel_login)"; then
  log "failed to login to web panel"
  exit 1
fi

log "using panel $WEBPANEL_BASE_URL"
log "watchdog $WATCHDOG_SECONDS seconds"

(
  sleep "$WATCHDOG_SECONDS"
  {
    printf '[watchdog] timeout reached, forcing restore\n'
    bash "$RESTORE_SCRIPT" --config "$CONFIG_PATH"
  } >>"$LOG_FILE" 2>&1 || true
) &
WATCHDOG_PID="$!"

fetch_tun_status "before-start"
snapshot_state "before-start"

log "starting transparent mode"
api_post_checked "/api/v1/tun/start" "start"
sleep 5
fetch_tun_status "after-start"
if [[ "${TUN_RUNNING:-}" != "true" ]]; then
  log "transparent mode did not report running after start"
  exit 1
fi
snapshot_state "after-start"

run_curl_probe "qq-http1" "https://www.qq.com" --http1.1
run_curl_probe "baidu-http1" "https://www.baidu.com" --http1.1
run_curl_probe "gov-http1" "https://www.gov.cn" --http1.1
run_curl_probe "example-http1" "https://example.com" --http1.1

log "stopping transparent mode"
api_post_checked "/api/v1/tun/restore-clean" "restore-clean"
sleep 3
fetch_tun_status "after-stop"
if [[ "${TUN_RUNNING:-}" != "false" ]]; then
  log "transparent mode still reports running after restore-clean"
  exit 1
fi

log "smoke test complete"
