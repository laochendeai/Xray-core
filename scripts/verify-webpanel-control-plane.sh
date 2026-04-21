#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib-webpanel.sh
source "$SCRIPT_DIR/lib-webpanel.sh"

usage() {
  cat <<'EOF'
usage: verify-webpanel-control-plane.sh <preflight|snapshot|post-reboot> [--config PATH] [--base-url URL] [--output-dir DIR]

Read-only real-machine verification for the webpanel stability control plane.

Commands:
  preflight    Check local files, helper/runtime paths, API login, TUN status, and pool summary.
  snapshot     Capture API/state/log snapshots into an evidence directory.
  post-reboot  Assert the machine came back clean after a reboot.
EOF
}

PASS_COUNT=0
WARN_COUNT=0
FAIL_COUNT=0

pass() {
  PASS_COUNT=$((PASS_COUNT + 1))
  printf 'PASS %s\n' "$*"
}

warn() {
  WARN_COUNT=$((WARN_COUNT + 1))
  printf 'WARN %s\n' "$*"
}

fail() {
  FAIL_COUNT=$((FAIL_COUNT + 1))
  printf 'FAIL %s\n' "$*" >&2
}

info() {
  printf 'INFO %s\n' "$*"
}

assert_file() {
  local path="$1"
  local label="$2"
  if [[ -e "$path" ]]; then
    pass "$label exists: $path"
  else
    fail "$label missing: $path"
  fi
}

assert_login_and_fetch() {
  local token status_code
  TOKEN="$(webpanel_login)" || {
    fail "Unable to login to $WEBPANEL_BASE_URL with config credentials"
    return 1
  }
  pass "WebPanel login succeeded: $WEBPANEL_BASE_URL"

  TUN_STATUS_JSON="$(mktemp)"
  NODE_POOL_JSON="$(mktemp)"

  status_code="$(webpanel_api_get "$TOKEN" "/api/v1/tun/status" "$TUN_STATUS_JSON")" || status_code="curl_failed"
  if [[ "$status_code" != "200" ]]; then
    fail "GET /api/v1/tun/status failed: $status_code"
    return 1
  fi
  pass "Fetched /api/v1/tun/status"

  status_code="$(webpanel_api_get "$TOKEN" "/api/v1/node-pool" "$NODE_POOL_JSON")" || status_code="curl_failed"
  if [[ "$status_code" != "200" ]]; then
    fail "GET /api/v1/node-pool failed: $status_code"
    return 1
  fi
  pass "Fetched /api/v1/node-pool"
}

cleanup_tmp() {
  [[ -n "${TUN_STATUS_JSON:-}" && -f "${TUN_STATUS_JSON:-}" ]] && rm -f "$TUN_STATUS_JSON"
  [[ -n "${NODE_POOL_JSON:-}" && -f "${NODE_POOL_JSON:-}" ]] && rm -f "$NODE_POOL_JSON"
}

print_summary() {
  local machine_state last_reason running helper_exists elevation_ready allow_remote active_nodes min_active healthy
  machine_state="$(json_get "$TUN_STATUS_JSON" "machineState" 2>/dev/null || true)"
  last_reason="$(json_get "$TUN_STATUS_JSON" "lastStateReason" 2>/dev/null || true)"
  running="$(json_get "$TUN_STATUS_JSON" "running" 2>/dev/null || true)"
  helper_exists="$(json_get "$TUN_STATUS_JSON" "helperExists" 2>/dev/null || true)"
  elevation_ready="$(json_get "$TUN_STATUS_JSON" "elevationReady" 2>/dev/null || true)"
  allow_remote="$(json_get "$TUN_STATUS_JSON" "allowRemote" 2>/dev/null || true)"
  active_nodes="$(json_get "$NODE_POOL_JSON" "summary.activeNodes" 2>/dev/null || true)"
  min_active="$(json_get "$NODE_POOL_JSON" "summary.minActiveNodes" 2>/dev/null || true)"
  healthy="$(json_get "$NODE_POOL_JSON" "summary.healthy" 2>/dev/null || true)"

  info "machineState=${machine_state:-unknown} running=${running:-unknown} reason=${last_reason:-unknown}"
  info "helperExists=${helper_exists:-unknown} elevationReady=${elevation_ready:-unknown} allowRemote=${allow_remote:-unknown}"
  info "activeNodes=${active_nodes:-unknown} minActiveNodes=${min_active:-unknown} poolHealthy=${healthy:-unknown}"
}

check_network_surface() {
  local running
  running="$(json_get "$TUN_STATUS_JSON" "running" 2>/dev/null || true)"
  if [[ "$running" != "true" ]]; then
    pass "Transparent mode is not currently running"
    return
  fi

  if ! command -v ip >/dev/null 2>&1; then
    warn "ip command is unavailable; skipping route/interface checks"
    return
  fi

  if ip link show "$WEBPANEL_TUN_INTERFACE" >/dev/null 2>&1; then
    pass "TUN interface is present while running: $WEBPANEL_TUN_INTERFACE"
  else
    fail "TUN status says running but interface is missing: $WEBPANEL_TUN_INTERFACE"
  fi

  if ip -4 rule show | grep -Eq "^${WEBPANEL_BYPASS_RULE_PREF}:.*lookup ${WEBPANEL_BYPASS_ROUTE_TABLE_ID}\b"; then
    pass "Bypass policy rule is active for the TUN helper traffic"
  else
    fail "Missing bypass policy rule for helper traffic"
  fi

  if ip -4 rule show | grep -Eq "^${WEBPANEL_CAPTURE_IPV4_RULE_PREF}:.*lookup ${WEBPANEL_CAPTURE_ROUTE_TABLE_ID}\b"; then
    pass "Strict IPv4 full-tunnel capture policy rule is active"
  else
    fail "Missing strict IPv4 full-tunnel capture policy rule"
  fi

  if ip -4 rule show | grep -Eq "^${WEBPANEL_LEGACY_CAPTURE_DNS_RULE_PREF}:.*lookup ${WEBPANEL_CAPTURE_ROUTE_TABLE_ID}\b"; then
    fail "Legacy UDP/53-only capture policy rule is still active"
  else
    pass "Legacy UDP/53-only capture policy rule is absent"
  fi

  if ip -4 rule show | grep -Eq "^${WEBPANEL_LEGACY_CAPTURE_UDP_443_RULE_PREF}:.*lookup ${WEBPANEL_CAPTURE_ROUTE_TABLE_ID}\b"; then
    fail "Legacy UDP/443-only capture policy rule is still active"
  else
    pass "Legacy UDP/443-only capture policy rule is absent"
  fi

  if ip -4 rule show | grep -Eq "^${WEBPANEL_LEGACY_CAPTURE_TCP_RULE_PREF}:.*ipproto tcp .*lookup ${WEBPANEL_CAPTURE_ROUTE_TABLE_ID}\b|^${WEBPANEL_LEGACY_CAPTURE_TCP_RULE_PREF}:.*ipproto tcp lookup ${WEBPANEL_CAPTURE_ROUTE_TABLE_ID}\b"; then
    fail "Legacy TCP-only capture policy rule is still active"
  else
    pass "Legacy TCP-only capture policy rule is absent"
  fi

  if ip route show table "$WEBPANEL_CAPTURE_ROUTE_TABLE_ID" | grep -Eq "^0\\.0\\.0\\.0/1 dev ${WEBPANEL_TUN_INTERFACE}\b"; then
    pass "Lower capture route is attached to $WEBPANEL_TUN_INTERFACE"
  else
    fail "Missing lower capture route on $WEBPANEL_TUN_INTERFACE"
  fi

  if ip route show table "$WEBPANEL_CAPTURE_ROUTE_TABLE_ID" | grep -Eq "^128\\.0\\.0\\.0/1 dev ${WEBPANEL_TUN_INTERFACE}\b"; then
    pass "Upper capture route is attached to $WEBPANEL_TUN_INTERFACE"
  else
    fail "Missing upper capture route on $WEBPANEL_TUN_INTERFACE"
  fi

  if [[ -d /proc/sys/net/ipv6/conf ]]; then
    if find /proc/sys/net/ipv6/conf -name disable_ipv6 -type f -exec sh -c 'for path do [ "$(cat "$path" 2>/dev/null)" = "1" ] || exit 1; done' sh {} +; then
      pass "IPv6 is disabled while transparent mode is running"
    else
      fail "IPv6 is not fully disabled while transparent mode is running"
    fi
  else
    pass "IPv6 sysctl tree is unavailable, so no IPv6 host path is exposed"
  fi
}

run_preflight() {
  assert_file "$WEBPANEL_CONFIG_PATH" "Config"
  assert_file "$WEBPANEL_HELPER_PATH" "TUN helper"

  if [[ -e "$WEBPANEL_BINARY_PATH" ]]; then
    pass "Xray binary exists: $WEBPANEL_BINARY_PATH"
  else
    warn "Configured Xray binary is missing or unresolved: $WEBPANEL_BINARY_PATH"
  fi

  if [[ -d "$WEBPANEL_STATE_DIR" ]]; then
    pass "TUN state directory exists: $WEBPANEL_STATE_DIR"
  else
    warn "TUN state directory does not exist yet: $WEBPANEL_STATE_DIR"
  fi

  assert_login_and_fetch || return
  print_summary

  local helper_exists elevation_ready allow_remote
  helper_exists="$(json_get "$TUN_STATUS_JSON" "helperExists" 2>/dev/null || true)"
  elevation_ready="$(json_get "$TUN_STATUS_JSON" "elevationReady" 2>/dev/null || true)"
  allow_remote="$(json_get "$TUN_STATUS_JSON" "allowRemote" 2>/dev/null || true)"

  if [[ "$helper_exists" == "true" ]]; then
    pass "API confirms helper exists"
  else
    fail "API reports helper missing"
  fi

  if [[ "$WEBPANEL_USE_SUDO" == "true" ]]; then
    if [[ "$elevation_ready" == "true" ]]; then
      pass "API confirms sudo elevation is ready"
    else
      fail "API reports sudo elevation is not ready"
    fi
  else
    pass "Config is set to direct execution without sudo"
  fi

  if [[ "$allow_remote" == "false" ]]; then
    pass "TUN control remains local-only"
  else
    fail "TUN control is remotely accessible"
  fi

  if [[ -f "$WEBPANEL_CONTROL_PLANE_STATE_PATH" ]]; then
    pass "Machine state file exists: $WEBPANEL_CONTROL_PLANE_STATE_PATH"
  else
    warn "Machine state file does not exist yet: $WEBPANEL_CONTROL_PLANE_STATE_PATH"
  fi

  if [[ -f "$WEBPANEL_NODE_POOL_STATE_PATH" ]]; then
    pass "Node pool state file exists: $WEBPANEL_NODE_POOL_STATE_PATH"
  else
    warn "Node pool state file does not exist yet: $WEBPANEL_NODE_POOL_STATE_PATH"
  fi

  check_network_surface
}

run_snapshot() {
  assert_login_and_fetch || return
  print_summary
  local snapshot_dir
  snapshot_dir="$(webpanel_capture_snapshot "$TOKEN" "$OUTPUT_DIR" "snapshot-$(date +%Y%m%d-%H%M%S)")"
  pass "Captured evidence snapshot: $snapshot_dir"
}

run_post_reboot() {
  assert_login_and_fetch || return
  print_summary

  local machine_state last_reason running
  machine_state="$(json_get "$TUN_STATUS_JSON" "machineState" 2>/dev/null || true)"
  last_reason="$(json_get "$TUN_STATUS_JSON" "lastStateReason" 2>/dev/null || true)"
  running="$(json_get "$TUN_STATUS_JSON" "running" 2>/dev/null || true)"

  if [[ "$machine_state" == "clean" ]]; then
    pass "Machine state after reboot is clean"
  else
    fail "Expected machineState=clean after reboot, got ${machine_state:-unknown}"
  fi

  if [[ "$running" == "false" ]]; then
    pass "Transparent mode is not running after reboot"
  else
    fail "Transparent mode is still running after reboot"
  fi

  if [[ "$last_reason" == "startup_default_clean" ]]; then
    pass "Startup clean reason is visible"
  else
    fail "Expected lastStateReason=startup_default_clean after reboot, got ${last_reason:-unknown}"
  fi

  if command -v ip >/dev/null 2>&1; then
    if ip link show "$WEBPANEL_TUN_INTERFACE" >/dev/null 2>&1; then
      fail "TUN interface still exists after reboot"
    else
      pass "TUN interface is absent after reboot"
    fi

    if ip -4 rule show | grep -Eq "^${WEBPANEL_CAPTURE_IPV4_RULE_PREF}:.*lookup ${WEBPANEL_CAPTURE_ROUTE_TABLE_ID}\b"; then
      fail "Strict IPv4 full-tunnel capture rule still exists after reboot"
    else
      pass "Strict IPv4 full-tunnel capture rule is absent after reboot"
    fi

    if ip -4 rule show | grep -Eq "^${WEBPANEL_LEGACY_CAPTURE_DNS_RULE_PREF}:.*lookup ${WEBPANEL_CAPTURE_ROUTE_TABLE_ID}\b"; then
      fail "Legacy UDP/53 capture rule still exists after reboot"
    else
      pass "Legacy UDP/53 capture rule is absent after reboot"
    fi

    if ip -4 rule show | grep -Eq "^${WEBPANEL_LEGACY_CAPTURE_UDP_443_RULE_PREF}:.*lookup ${WEBPANEL_CAPTURE_ROUTE_TABLE_ID}\b"; then
      fail "Legacy UDP/443 capture rule still exists after reboot"
    else
      pass "Legacy UDP/443 capture rule is absent after reboot"
    fi

    if ip -4 rule show | grep -Eq "^${WEBPANEL_LEGACY_CAPTURE_TCP_RULE_PREF}:.*lookup ${WEBPANEL_CAPTURE_ROUTE_TABLE_ID}\b"; then
      fail "Legacy TCP capture rule still exists after reboot"
    else
      pass "Legacy TCP capture rule is absent after reboot"
    fi

    if ip -4 rule show | grep -Eq "^${WEBPANEL_BYPASS_RULE_PREF}:.*lookup ${WEBPANEL_BYPASS_ROUTE_TABLE_ID}\b"; then
      fail "Bypass rule still exists after reboot"
    else
      pass "Bypass rule is absent after reboot"
    fi

    if ip route show table "$WEBPANEL_CAPTURE_ROUTE_TABLE_ID" 2>/dev/null | grep -Eq "dev ${WEBPANEL_TUN_INTERFACE}\b"; then
      fail "Capture routing table still points to the TUN interface after reboot"
    else
      pass "Capture routing table no longer points to the TUN interface after reboot"
    fi
  fi

  local snapshot_dir
  snapshot_dir="$(webpanel_capture_snapshot "$TOKEN" "$OUTPUT_DIR" "post-reboot-$(date +%Y%m%d-%H%M%S)")"
  pass "Captured post-reboot evidence: $snapshot_dir"
}

COMMAND="${1:-}"
if [[ "$COMMAND" == "-h" || "$COMMAND" == "--help" ]]; then
  usage
  exit 0
fi
if [[ -z "$COMMAND" ]]; then
  usage >&2
  exit 2
fi
shift

CONFIG_PATH="$(webpanel_default_config_path)"
BASE_URL_OVERRIDE=""
OUTPUT_DIR="/tmp/webpanel-control-plane-verification"

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
    --output-dir)
      OUTPUT_DIR="$2"
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

require_command python3
require_command curl
mkdir -p "$OUTPUT_DIR"
webpanel_load_config "$CONFIG_PATH" "$BASE_URL_OVERRIDE"
trap cleanup_tmp EXIT

case "$COMMAND" in
  preflight)
    run_preflight
    ;;
  snapshot)
    run_snapshot
    ;;
  post-reboot)
    run_post_reboot
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac

info "summary: pass=$PASS_COUNT warn=$WARN_COUNT fail=$FAIL_COUNT"
if [[ $FAIL_COUNT -gt 0 ]]; then
  exit 1
fi
