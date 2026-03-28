#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib-webpanel.sh
source "$SCRIPT_DIR/lib-webpanel.sh"

usage() {
  cat <<'EOF'
usage: rehearse-webpanel-fallback.sh [--config PATH] [--base-url URL] [--output-dir DIR] [--apply] [--allow-running] [--keep-quarantine]

Rehearse the critical real-machine fallback path:
  clean -> proxied -> active pool drops below minimum -> automatic clean fallback

Default mode is dry-run. Use --apply to actually call the WebPanel API and change machine state.
EOF
}

log() {
  printf 'INFO %s\n' "$*"
}

die() {
  printf 'FAIL %s\n' "$*" >&2
  exit 1
}

CONFIG_PATH="$(webpanel_default_config_path)"
BASE_URL_OVERRIDE=""
OUTPUT_DIR="/tmp/webpanel-fallback-rehearsal"
APPLY=0
ALLOW_RUNNING=0
KEEP_QUARANTINE=0
TOKEN=""
TUN_STATUS_JSON=""
ACTIVE_POOL_JSON=""
STARTED_BY_SCRIPT=0
declare -a QUARANTINED_IDS=()

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
    --apply)
      APPLY=1
      shift
      ;;
    --allow-running)
      ALLOW_RUNNING=1
      shift
      ;;
    --keep-quarantine)
      KEEP_QUARANTINE=1
      shift
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

cleanup() {
  local output status_code status_json running

  if [[ $APPLY -eq 1 && $KEEP_QUARANTINE -eq 0 && ${#QUARANTINED_IDS[@]} -gt 0 && -n "$TOKEN" ]]; then
    for node_id in "${QUARANTINED_IDS[@]}"; do
      output="$(mktemp)"
      status_code="$(webpanel_api_post "$TOKEN" "/api/v1/node-pool/$node_id/promote" "$output" 2>/dev/null || true)"
      rm -f "$output"
      if [[ "$status_code" == "200" ]]; then
        printf 'INFO restored node to active pool: %s\n' "$node_id"
      else
        printf 'WARN failed to restore node to active pool: %s\n' "$node_id" >&2
      fi
    done
  fi

  if [[ $APPLY -eq 1 && $STARTED_BY_SCRIPT -eq 1 && -n "$TOKEN" ]]; then
    status_json="$(mktemp)"
    status_code="$(webpanel_api_get "$TOKEN" "/api/v1/tun/status" "$status_json" 2>/dev/null || true)"
    if [[ "$status_code" == "200" ]]; then
      running="$(json_get "$status_json" "running" 2>/dev/null || true)"
      if [[ "$running" == "true" ]]; then
        output="$(mktemp)"
        webpanel_api_post "$TOKEN" "/api/v1/tun/restore-clean" "$output" >/dev/null 2>&1 || true
        rm -f "$output"
        printf 'INFO attempted restore-clean during cleanup\n'
      fi
    fi
    rm -f "$status_json"
  fi

  [[ -n "${TUN_STATUS_JSON:-}" && -f "${TUN_STATUS_JSON:-}" ]] && rm -f "$TUN_STATUS_JSON"
  [[ -n "${ACTIVE_POOL_JSON:-}" && -f "${ACTIVE_POOL_JSON:-}" ]] && rm -f "$ACTIVE_POOL_JSON"
}

fetch_state() {
  TUN_STATUS_JSON="$(mktemp)"
  ACTIVE_POOL_JSON="$(mktemp)"

  local status_code
  status_code="$(webpanel_api_get "$TOKEN" "/api/v1/tun/status" "$TUN_STATUS_JSON")"
  [[ "$status_code" == "200" ]] || die "GET /api/v1/tun/status failed: $status_code"

  status_code="$(webpanel_api_get "$TOKEN" "/api/v1/node-pool?status=active" "$ACTIVE_POOL_JSON")"
  [[ "$status_code" == "200" ]] || die "GET /api/v1/node-pool?status=active failed: $status_code"
}

wait_for_terminal_state() {
  local timeout_s="$1"
  local started_at now state reason running status_json status_code
  started_at="$(date +%s)"

  while true; do
    status_json="$(mktemp)"
    status_code="$(webpanel_api_get "$TOKEN" "/api/v1/tun/status" "$status_json")"
    if [[ "$status_code" != "200" ]]; then
      rm -f "$status_json"
      sleep 1
      continue
    fi

    state="$(json_get "$status_json" "machineState" 2>/dev/null || true)"
    reason="$(json_get "$status_json" "lastStateReason" 2>/dev/null || true)"
    running="$(json_get "$status_json" "running" 2>/dev/null || true)"

    if [[ "$state" == "clean" && "$running" == "false" ]]; then
      printf '%s\t%s\t%s\n' "$state" "$reason" "$running"
      rm -f "$status_json"
      return 0
    fi
    if [[ "$state" == "degraded" ]]; then
      printf '%s\t%s\t%s\n' "$state" "$reason" "$running"
      rm -f "$status_json"
      return 0
    fi
    rm -f "$status_json"

    now="$(date +%s)"
    if (( now - started_at >= timeout_s )); then
      return 1
    fi
    sleep 1
  done
}

select_quarantine_ids() {
  local active_count min_active quarantine_count
  active_count="$(json_get "$ACTIVE_POOL_JSON" "summary.activeCount" 2>/dev/null || true)"
  min_active="$(json_get "$ACTIVE_POOL_JSON" "summary.minActiveNodes" 2>/dev/null || true)"

  [[ -n "$active_count" ]] || die "Active pool count is unavailable"
  [[ -n "$min_active" ]] || die "Minimum active node count is unavailable"

  if (( active_count < min_active )); then
    die "Active pool is already below minimum: active=$active_count minimum=$min_active"
  fi
  if (( min_active < 1 )); then
    die "Minimum active node count must be at least 1 for this rehearsal"
  fi

  quarantine_count=$((active_count - min_active + 1))
  if (( quarantine_count < 1 )); then
    die "Computed quarantine count is invalid: $quarantine_count"
  fi

  mapfile -t ACTIVE_NODE_LINES < <(json_print_active_nodes "$ACTIVE_POOL_JSON")
  if (( ${#ACTIVE_NODE_LINES[@]} < quarantine_count )); then
    die "Not enough active nodes to select quarantine set"
  fi

  QUARANTINE_COUNT="$quarantine_count"
  ACTIVE_COUNT="$active_count"
  MIN_ACTIVE="$min_active"
}

trap cleanup EXIT

require_command python3
require_command curl
mkdir -p "$OUTPUT_DIR"
webpanel_load_config "$CONFIG_PATH" "$BASE_URL_OVERRIDE"
TOKEN="$(webpanel_login)" || die "Unable to login to $WEBPANEL_BASE_URL"
fetch_state
select_quarantine_ids

INITIAL_RUNNING="$(json_get "$TUN_STATUS_JSON" "running" 2>/dev/null || true)"
INITIAL_MACHINE_STATE="$(json_get "$TUN_STATUS_JSON" "machineState" 2>/dev/null || true)"

log "baseUrl=$WEBPANEL_BASE_URL"
log "activeNodes=$ACTIVE_COUNT minimumRequired=$MIN_ACTIVE quarantineNeeded=$QUARANTINE_COUNT"
log "initial machineState=${INITIAL_MACHINE_STATE:-unknown} running=${INITIAL_RUNNING:-unknown}"

if [[ "$INITIAL_RUNNING" == "true" && $ALLOW_RUNNING -ne 1 ]]; then
  die "Transparent mode is already running. Use --allow-running only if that is intentional."
fi

for ((i = 0; i < QUARANTINE_COUNT; i++)); do
  IFS=$'\t' read -r node_id node_remark node_address <<<"${ACTIVE_NODE_LINES[$i]}"
  QUARANTINED_IDS+=("$node_id")
  log "selected node for quarantine: id=$node_id remark=${node_remark:-"-"} address=${node_address:-"-"}"
done

if [[ $APPLY -ne 1 ]]; then
  log "dry-run only. Re-run with --apply to execute the fallback rehearsal."
  exit 0
fi

SNAPSHOT_DIR="$(webpanel_capture_snapshot "$TOKEN" "$OUTPUT_DIR" "before-$(date +%Y%m%d-%H%M%S)")"
log "captured initial snapshot: $SNAPSHOT_DIR"

if [[ "$INITIAL_RUNNING" != "true" ]]; then
  START_OUTPUT="$(mktemp)"
  START_STATUS="$(webpanel_api_post "$TOKEN" "/api/v1/tun/start" "$START_OUTPUT")"
  if [[ "$START_STATUS" != "200" ]]; then
    cat "$START_OUTPUT" >&2 || true
    rm -f "$START_OUTPUT"
    die "Failed to start transparent mode: HTTP $START_STATUS"
  fi
  rm -f "$START_OUTPUT"
  STARTED_BY_SCRIPT=1
  log "transparent mode start requested"
else
  log "transparent mode was already running; rehearsal will not claim ownership of startup"
fi

for node_id in "${QUARANTINED_IDS[@]}"; do
  OUTPUT_FILE="$(mktemp)"
  STATUS_CODE="$(webpanel_api_post "$TOKEN" "/api/v1/node-pool/$node_id/quarantine" "$OUTPUT_FILE")"
  if [[ "$STATUS_CODE" != "200" ]]; then
    cat "$OUTPUT_FILE" >&2 || true
    rm -f "$OUTPUT_FILE"
    die "Failed to quarantine node $node_id: HTTP $STATUS_CODE"
  fi
  rm -f "$OUTPUT_FILE"
  log "quarantined node: $node_id"
done

SNAPSHOT_DIR="$(webpanel_capture_snapshot "$TOKEN" "$OUTPUT_DIR" "after-quarantine-$(date +%Y%m%d-%H%M%S)")"
log "captured post-quarantine snapshot: $SNAPSHOT_DIR"

if ! RESULT="$(wait_for_terminal_state 30)"; then
  die "Timed out waiting for automatic fallback result"
fi

IFS=$'\t' read -r FINAL_STATE FINAL_REASON FINAL_RUNNING <<<"$RESULT"
log "terminal machineState=$FINAL_STATE reason=$FINAL_REASON running=$FINAL_RUNNING"

SNAPSHOT_DIR="$(webpanel_capture_snapshot "$TOKEN" "$OUTPUT_DIR" "after-result-$(date +%Y%m%d-%H%M%S)")"
log "captured terminal snapshot: $SNAPSHOT_DIR"

if [[ "$FINAL_STATE" == "clean" && "$FINAL_RUNNING" == "false" && "$FINAL_REASON" == "automatic_fallback_min_active_not_met" ]]; then
  log "fallback rehearsal passed"
  exit 0
fi

if [[ "$FINAL_STATE" == "degraded" ]]; then
  die "Fallback rehearsal reached degraded state instead of clean fallback. reason=$FINAL_REASON"
fi

die "Unexpected terminal state after rehearsal: state=$FINAL_STATE reason=$FINAL_REASON running=$FINAL_RUNNING"

