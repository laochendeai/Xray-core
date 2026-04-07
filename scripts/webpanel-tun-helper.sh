#!/usr/bin/env bash
set -euo pipefail

umask 022

ACTION="${1:-status}"
XRAY_BIN="${2:-${XRAY_BIN:-}}"
XRAY_CONFIG="${3:-${XRAY_CONFIG:-}}"
STATE_DIR="${4:-${STATE_DIR:-}}"
TUN_NAME="${5:-${TUN_NAME:-xray0}}"
if [[ $# -gt 5 ]]; then
  shift 5
  REMOTE_DNS="$*"
else
  REMOTE_DNS="${REMOTE_DNS:-1.1.1.1 8.8.8.8}"
fi

if [[ -z "$XRAY_BIN" || -z "$XRAY_CONFIG" || -z "$STATE_DIR" ]]; then
  echo "XRAY_BIN, XRAY_CONFIG and STATE_DIR are required." >&2
  exit 1
fi

PID_FILE="$STATE_DIR/xray-tun.pid"
LOG_FILE="$STATE_DIR/xray-tun.log"
STATE_FILE="$STATE_DIR/network-state.env"
ROUTE_TABLE_ID="${XRAY_TUN_ROUTE_TABLE_ID:-2027}"
UPSTREAM_RULE_PREF_BASE="${XRAY_TUN_UPSTREAM_RULE_PREF_BASE:-10000}"
BYPASS_RULE_PREF="${XRAY_TUN_BYPASS_RULE_PREF:-12000}"
BYPASS_UID_RANGE="${XRAY_TUN_BYPASS_UID_RANGE:-0-0}"
CAPTURE_ROUTE_TABLE_ID="${XRAY_TUN_CAPTURE_ROUTE_TABLE_ID:-2028}"
CAPTURE_DNS_RULE_PREF="${XRAY_TUN_CAPTURE_DNS_RULE_PREF:-12010}"
CAPTURE_UDP_443_RULE_PREF="${XRAY_TUN_CAPTURE_UDP_443_RULE_PREF:-12015}"
CAPTURE_TCP_RULE_PREF="${XRAY_TUN_CAPTURE_TCP_RULE_PREF:-12020}"

require_root() {
  if [[ ${EUID:-$(id -u)} -ne 0 ]]; then
    echo "This action requires root privileges." >&2
    exit 1
  fi
}

load_state() {
  if [[ -f "$STATE_FILE" ]]; then
    # shellcheck disable=SC1090
    source "$STATE_FILE"
  fi
}

active_tun_name() {
  load_state
  printf '%s\n' "${STATE_TUN_NAME:-$TUN_NAME}"
}

find_running_pid() {
  if [[ -f "$PID_FILE" ]]; then
    local pid
    pid="$(tr -d '[:space:]' <"$PID_FILE" 2>/dev/null || true)"
    if [[ "$pid" =~ ^[0-9]+$ ]] && ps -p "$pid" -o args= 2>/dev/null | grep -Fq "$XRAY_CONFIG"; then
      printf '%s\n' "$pid"
      return 0
    fi
    rm -f "$PID_FILE"
  fi

  local pid
  pid="$(pgrep -f -o -- "$XRAY_BIN run -c $XRAY_CONFIG" || true)"
  if [[ -n "$pid" ]]; then
    printf '%s\n' "$pid" >"$PID_FILE"
    chmod 0644 "$PID_FILE" 2>/dev/null || true
    printf '%s\n' "$pid"
    return 0
  fi

  return 1
}

extract_upstream_ips() {
  python3 - "$XRAY_CONFIG" <<'PY'
import ipaddress
import json
import socket
import sys
from pathlib import Path

config = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
skip_protocols = {"direct", "freedom", "blackhole", "block", "dns", "loopback"}
seen = set()

for outbound in config.get("outbounds", []):
    if outbound.get("protocol") in skip_protocols:
        continue
    settings = outbound.get("settings", {})
    for key in ("servers", "vnext"):
        for server in settings.get(key, []):
            address = server.get("address")
            if not address:
                continue
            try:
                ipaddress.ip_address(address)
                candidates = [address]
            except ValueError:
                try:
                    candidates = sorted(
                        {
                            item[4][0]
                            for item in socket.getaddrinfo(address, None, socket.AF_INET, socket.SOCK_STREAM)
                        }
                    )
                except OSError:
                    candidates = []
            for candidate in candidates:
                if candidate not in seen:
                    print(candidate)
                    seen.add(candidate)
PY
}

read_default_route() {
  python3 - <<'PY'
import subprocess

for line in subprocess.check_output(["ip", "-4", "route", "show", "default"], text=True).splitlines():
    parts = line.split()
    if "via" in parts and "dev" in parts:
        gateway = parts[parts.index("via") + 1]
        device = parts[parts.index("dev") + 1]
        print(gateway)
        print(device)
        break
PY
}

write_state() {
  local default_gw="$1"
  local default_dev="$2"
  local upstream_ips="$3"
  local dns_overridden="$4"
  local rp_filter_all_old="$5"
  local rp_filter_tun_old="$6"
  local tun_name="$7"

  mkdir -p "$STATE_DIR"
  cat >"$STATE_FILE" <<EOF
DEFAULT_GW='$default_gw'
DEFAULT_DEV='$default_dev'
UPSTREAM_IPS='$upstream_ips'
DNS_OVERRIDDEN='$dns_overridden'
RP_FILTER_OLD='$rp_filter_all_old'
RP_FILTER_ALL_OLD='$rp_filter_all_old'
RP_FILTER_TUN_OLD='$rp_filter_tun_old'
STATE_TUN_NAME='$tun_name'
ROUTE_TABLE_ID='$ROUTE_TABLE_ID'
UPSTREAM_RULE_PREF_BASE='$UPSTREAM_RULE_PREF_BASE'
BYPASS_RULE_PREF='$BYPASS_RULE_PREF'
BYPASS_UID_RANGE='$BYPASS_UID_RANGE'
CAPTURE_ROUTE_TABLE_ID='$CAPTURE_ROUTE_TABLE_ID'
CAPTURE_DNS_RULE_PREF='$CAPTURE_DNS_RULE_PREF'
CAPTURE_UDP_443_RULE_PREF='$CAPTURE_UDP_443_RULE_PREF'
CAPTURE_TCP_RULE_PREF='$CAPTURE_TCP_RULE_PREF'
EOF
  chmod 0644 "$STATE_FILE" 2>/dev/null || true
}

clear_split_routes() {
  local tun_name="$1"
  ip route del 0.0.0.0/1 dev "$tun_name" 2>/dev/null || true
  ip route del 128.0.0.0/1 dev "$tun_name" 2>/dev/null || true
}

clear_upstream_bypass_rules() {
  local upstream_ips="$1"
  local pref=0
  local idx=0
  local ip_addr
  for ip_addr in $upstream_ips; do
    pref=$((UPSTREAM_RULE_PREF_BASE + idx))
    ip -4 rule del pref "$pref" to "$ip_addr"/32 lookup main 2>/dev/null || true
    ip -4 rule del pref "$pref" 2>/dev/null || true
    ip -4 rule del to "$ip_addr"/32 lookup main 2>/dev/null || true
    idx=$((idx + 1))
  done
}

clear_policy_routes() {
  ip -4 rule del pref "$BYPASS_RULE_PREF" 2>/dev/null || true
  ip -4 rule del pref "$CAPTURE_DNS_RULE_PREF" 2>/dev/null || true
  ip -4 rule del pref "$CAPTURE_UDP_443_RULE_PREF" 2>/dev/null || true
  ip -4 rule del pref "$CAPTURE_TCP_RULE_PREF" 2>/dev/null || true
  ip route flush table "$ROUTE_TABLE_ID" 2>/dev/null || true
  ip route flush table "$CAPTURE_ROUTE_TABLE_ID" 2>/dev/null || true
}

prepare_bypass_route_table() {
  local tun_name="$1"
  local route_line=""

  ip route flush table "$ROUTE_TABLE_ID" 2>/dev/null || true

  while IFS= read -r route_line; do
    [[ -n "$route_line" ]] || continue
    [[ "$route_line" == *" dev ${tun_name}"* ]] && continue
    route_line="${route_line// linkdown/}"
    ip route replace table "$ROUTE_TABLE_ID" $route_line >/dev/null
  done < <(ip route show table main)

  ip -4 rule del pref "$BYPASS_RULE_PREF" 2>/dev/null || true
  ip -4 rule add pref "$BYPASS_RULE_PREF" uidrange "$BYPASS_UID_RANGE" lookup "$ROUTE_TABLE_ID"
}

copy_local_main_routes_to_table() {
  local target_table="$1"
  local tun_name="$2"
  local route_line=""

  while IFS= read -r route_line; do
    [[ -n "$route_line" ]] || continue
    [[ "$route_line" == default\ * ]] && continue
    [[ "$route_line" == *" dev ${tun_name}"* ]] && continue
    route_line="${route_line// linkdown/}"
    ip route replace table "$target_table" $route_line >/dev/null 2>&1 || true
  done < <(ip route show table main)
}

prepare_upstream_bypass_rules() {
  local upstream_ips="$1"
  local pref=0
  local idx=0
  local ip_addr

  for ip_addr in $upstream_ips; do
    pref=$((UPSTREAM_RULE_PREF_BASE + idx))
    ip -4 rule del pref "$pref" 2>/dev/null || true
    ip -4 rule add pref "$pref" to "$ip_addr"/32 lookup main
    idx=$((idx + 1))
  done
}

prepare_capture_route_table() {
  local tun_name="$1"

  ip route flush table "$CAPTURE_ROUTE_TABLE_ID" 2>/dev/null || true
  copy_local_main_routes_to_table "$CAPTURE_ROUTE_TABLE_ID" "$tun_name"
  ip route replace table "$CAPTURE_ROUTE_TABLE_ID" 0.0.0.0/1 dev "$tun_name"
  ip route replace table "$CAPTURE_ROUTE_TABLE_ID" 128.0.0.0/1 dev "$tun_name"

  ip -4 rule del pref "$CAPTURE_DNS_RULE_PREF" 2>/dev/null || true
  ip -4 rule add pref "$CAPTURE_DNS_RULE_PREF" ipproto udp dport 53 lookup "$CAPTURE_ROUTE_TABLE_ID"

  ip -4 rule del pref "$CAPTURE_UDP_443_RULE_PREF" 2>/dev/null || true
  ip -4 rule add pref "$CAPTURE_UDP_443_RULE_PREF" ipproto udp dport 443 lookup "$CAPTURE_ROUTE_TABLE_ID"

  ip -4 rule del pref "$CAPTURE_TCP_RULE_PREF" 2>/dev/null || true
  ip -4 rule add pref "$CAPTURE_TCP_RULE_PREF" ipproto tcp lookup "$CAPTURE_ROUTE_TABLE_ID"
}

resolved_link_has_dns_scope() {
  local default_dev="$1"

  if ! command -v resolvectl >/dev/null 2>&1; then
    return 1
  fi

  resolvectl status "$default_dev" 2>/dev/null | grep -Fq "Current Scopes: DNS"
}

link_uses_remote_dns_override() {
  local default_dev="$1"
  local status=""
  local dns_server=""

  if [[ -z "$default_dev" ]] || ! command -v resolvectl >/dev/null 2>&1; then
    return 1
  fi

  status="$(resolvectl status "$default_dev" 2>/dev/null || true)"
  [[ -n "$status" ]] || return 1

  if grep -Fq "DNS Domain: ~." <<<"$status"; then
    return 0
  fi

  for dns_server in $REMOTE_DNS; do
    grep -Eq "(Current DNS Server|DNS Servers):.*\\b${dns_server//./\\.}\\b" <<<"$status" && return 0
  done

  return 1
}

link_has_dns_config() {
  local default_dev="$1"

  if [[ -z "$default_dev" ]]; then
    return 1
  fi

  if resolved_link_has_dns_scope "$default_dev"; then
    return 0
  fi

  if command -v nmcli >/dev/null 2>&1; then
    nmcli device show "$default_dev" 2>/dev/null | grep -q 'IP4\.DNS' && return 0
  fi

  return 1
}

wait_for_dns_ready() {
  local default_dev="$1"

  for _ in $(seq 1 15); do
    if link_has_dns_config "$default_dev"; then
      return 0
    fi
    sleep 0.2
  done

  return 1
}

wait_for_dns_scope() {
  local default_dev="$1"

  for _ in $(seq 1 10); do
    if resolved_link_has_dns_scope "$default_dev"; then
      return 0
    fi
    sleep 0.2
  done

  return 1
}

set_nm_unmanaged() {
  local tun_name="$1"

  if [[ -z "$tun_name" ]] || ! command -v nmcli >/dev/null 2>&1; then
    return 0
  fi

  nmcli device set "$tun_name" managed no >/dev/null 2>&1 || true
}

networkmanager_connection_name() {
  local default_dev="$1"
  local connection_name

  if ! command -v nmcli >/dev/null 2>&1; then
    return 1
  fi

  connection_name="$(nmcli -g GENERAL.CONNECTION device show "$default_dev" 2>/dev/null | head -n1 | tr -d '\r')"
  if [[ -z "$connection_name" || "$connection_name" == "--" ]]; then
    return 1
  fi

  printf '%s\n' "$connection_name"
}

restore_dns() {
  local default_dev="$1"
  local connection_name=""

  if command -v resolvectl >/dev/null 2>&1 && [[ -n "$default_dev" ]]; then
    resolvectl revert "$default_dev" >/dev/null 2>&1 || true
  fi

  if [[ -z "$default_dev" ]] || ! command -v nmcli >/dev/null 2>&1; then
    return 0
  fi

  # NetworkManager-managed links may not automatically republish DNS scopes
  # after `resolvectl revert`, so reapply first and fall back to reconnecting.
  nmcli device reapply "$default_dev" >/dev/null 2>&1 || true
  if wait_for_dns_ready "$default_dev"; then
    return 0
  fi

  if connection_name="$(networkmanager_connection_name "$default_dev")"; then
    nmcli connection up "$connection_name" >/dev/null 2>&1 || true
    wait_for_dns_ready "$default_dev" || true
  fi
}

cleanup_network_state() {
  local default_gw=""
  local default_dev=""
  local upstream_ips=""
  local dns_overridden="0"
  local rp_filter_all_old=""
  local rp_filter_tun_old=""
  local tun_name
  tun_name="$(active_tun_name)"

  load_state
  default_gw="${DEFAULT_GW:-}"
  default_dev="${DEFAULT_DEV:-}"
  upstream_ips="${UPSTREAM_IPS:-}"
  dns_overridden="${DNS_OVERRIDDEN:-0}"
  rp_filter_all_old="${RP_FILTER_ALL_OLD:-${RP_FILTER_OLD:-}}"
  rp_filter_tun_old="${RP_FILTER_TUN_OLD:-}"
  tun_name="${STATE_TUN_NAME:-$tun_name}"
  ROUTE_TABLE_ID="${ROUTE_TABLE_ID:-2027}"
  UPSTREAM_RULE_PREF_BASE="${UPSTREAM_RULE_PREF_BASE:-10000}"
  BYPASS_RULE_PREF="${BYPASS_RULE_PREF:-12000}"
  BYPASS_UID_RANGE="${BYPASS_UID_RANGE:-0-0}"
  CAPTURE_ROUTE_TABLE_ID="${CAPTURE_ROUTE_TABLE_ID:-2028}"
  CAPTURE_DNS_RULE_PREF="${CAPTURE_DNS_RULE_PREF:-12010}"
  CAPTURE_UDP_443_RULE_PREF="${CAPTURE_UDP_443_RULE_PREF:-12015}"
  CAPTURE_TCP_RULE_PREF="${CAPTURE_TCP_RULE_PREF:-12020}"

  if [[ -z "$default_gw" || -z "$default_dev" ]]; then
    mapfile -t default_route < <(read_default_route)
    if [[ ${#default_route[@]} -ge 2 ]]; then
      default_gw="${default_route[0]}"
      default_dev="${default_route[1]}"
    fi
  fi

  if [[ -z "$upstream_ips" ]]; then
    upstream_ips="$(extract_upstream_ips | tr '\n' ' ' | xargs)"
  fi

  clear_split_routes "$tun_name"
  if [[ -n "$upstream_ips" ]]; then
    clear_upstream_bypass_rules "$upstream_ips"
  fi
  clear_policy_routes
  if [[ "$dns_overridden" == "1" ]] || link_uses_remote_dns_override "$default_dev"; then
    restore_dns "$default_dev"
  fi
  if [[ -n "$rp_filter_tun_old" ]] && ip link show "$tun_name" >/dev/null 2>&1; then
    sysctl -w "net.ipv4.conf.${tun_name}.rp_filter=$rp_filter_tun_old" >/dev/null 2>&1 || true
  fi
  if [[ -n "$rp_filter_all_old" ]]; then
    sysctl -w "net.ipv4.conf.all.rp_filter=$rp_filter_all_old" >/dev/null 2>&1 || true
  fi
  rm -f "$STATE_FILE"
}

stop_running_xray() {
  local pid
  if ! pid="$(find_running_pid)"; then
    rm -f "$PID_FILE"
    return 1
  fi

  kill "$pid" 2>/dev/null || true
  for _ in $(seq 1 20); do
    if ! ps -p "$pid" >/dev/null 2>&1; then
      rm -f "$PID_FILE"
      return 0
    fi
    sleep 0.5
  done

  kill -9 "$pid" 2>/dev/null || true
  rm -f "$PID_FILE"
  return 0
}

wait_for_link() {
  local tun_name="$1"
  for _ in $(seq 1 20); do
    if ip link show "$tun_name" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  return 1
}

start_xray() {
  mkdir -p "$STATE_DIR"
  : >"$LOG_FILE"
  "$XRAY_BIN" run -c "$XRAY_CONFIG" >>"$LOG_FILE" 2>&1 < /dev/null &
  local pid="$!"
  printf '%s\n' "$pid" >"$PID_FILE"
  chmod 0644 "$PID_FILE" 2>/dev/null || true
  chmod 0644 "$LOG_FILE" 2>/dev/null || true

  for _ in $(seq 1 20); do
    if ps -p "$pid" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  return 1
}

validate_runtime_config() {
  "$XRAY_BIN" run -test -c "$XRAY_CONFIG" >/dev/null
}

configure_network_state() {
  local tun_name="$1"

  mapfile -t default_route < <(read_default_route)
  if [[ ${#default_route[@]} -lt 2 ]]; then
    echo "Unable to determine the current IPv4 default route." >&2
    return 1
  fi

  local default_gw="${default_route[0]}"
  local default_dev="${default_route[1]}"
  local upstream_ips
  local dns_overridden="0"
  local rp_filter_all_old
  local rp_filter_tun_old

  upstream_ips="$(extract_upstream_ips | tr '\n' ' ' | xargs)"
  rp_filter_all_old="$(sysctl -n net.ipv4.conf.all.rp_filter 2>/dev/null || echo 0)"
  rp_filter_tun_old="$(sysctl -n "net.ipv4.conf.${tun_name}.rp_filter" 2>/dev/null || true)"
  sysctl -w net.ipv4.conf.all.rp_filter=0 >/dev/null
  if [[ -n "$rp_filter_tun_old" ]]; then
    # Linux uses the max(all, iface) rp_filter value. Leaving xray0 at 2 can
    # drop locally-originated packets before they ever reach the TUN runtime.
    sysctl -w "net.ipv4.conf.${tun_name}.rp_filter=0" >/dev/null
  fi

  set_nm_unmanaged "$tun_name"
  prepare_bypass_route_table "$tun_name"
  prepare_upstream_bypass_rules "$upstream_ips"
  prepare_capture_route_table "$tun_name"
  if link_uses_remote_dns_override "$default_dev"; then
    restore_dns "$default_dev"
  fi

  write_state "$default_gw" "$default_dev" "$upstream_ips" "$dns_overridden" "$rp_filter_all_old" "$rp_filter_tun_old" "$tun_name"
}

is_tun_active() {
  local tun_name
  tun_name="$(active_tun_name)"
  if ! find_running_pid >/dev/null 2>&1; then
    return 1
  fi
  if ! ip link show "$tun_name" >/dev/null 2>&1; then
    return 1
  fi
  if ip route show 0.0.0.0/1 dev "$tun_name" | grep -q .; then
    return 0
  fi
  if ip -4 rule show pref "$CAPTURE_TCP_RULE_PREF" 2>/dev/null | grep -Eq "ipproto tcp .*lookup $CAPTURE_ROUTE_TABLE_ID|ipproto tcp lookup $CAPTURE_ROUTE_TABLE_ID"; then
    return 0
  fi
  if ip -4 rule show pref "$CAPTURE_UDP_443_RULE_PREF" 2>/dev/null | grep -Eq "ipproto udp.* dport 443 .*lookup $CAPTURE_ROUTE_TABLE_ID|ipproto udp dport 443 lookup $CAPTURE_ROUTE_TABLE_ID"; then
    return 0
  fi
  ip route show table "$CAPTURE_ROUTE_TABLE_ID" 0.0.0.0/1 dev "$tun_name" | grep -q .
}

do_start() {
  require_root
  validate_runtime_config

  cleanup_network_state
  stop_running_xray >/dev/null 2>&1 || true

  if ! start_xray; then
    rm -f "$PID_FILE"
    echo "Failed to start Xray. Check $LOG_FILE." >&2
    exit 1
  fi

  if ! wait_for_link "$TUN_NAME"; then
    stop_running_xray >/dev/null 2>&1 || true
    echo "Xray started but $TUN_NAME did not appear." >&2
    exit 1
  fi

  if ! configure_network_state "$TUN_NAME"; then
    cleanup_network_state
    stop_running_xray >/dev/null 2>&1 || true
    echo "Failed to configure TUN routes or DNS." >&2
    exit 1
  fi

  local pid
  pid="$(find_running_pid)"
  printf 'Xray transparent TUN started.\n'
  printf 'PID: %s\n' "$pid"
  printf 'TUN: %s\n' "$TUN_NAME"
  printf 'Runtime config: %s\n' "$XRAY_CONFIG"
  printf 'Log: %s\n' "$LOG_FILE"
  printf 'ACTION=started\n'
}

do_stop() {
  require_root

  cleanup_network_state
  if stop_running_xray; then
    printf 'Xray transparent TUN stopped.\n'
    printf 'ACTION=stopped\n'
  else
    printf 'Xray transparent TUN was already stopped.\n'
    printf 'ACTION=already-stopped\n'
  fi
}

do_status() {
  if is_tun_active; then
    local pid
    pid="$(find_running_pid)"
    printf 'Xray transparent TUN is running.\n'
    printf 'PID: %s\n' "$pid"
    printf 'TUN: %s\n' "$(active_tun_name)"
    printf 'ACTION=status:running\n'
  else
    printf 'Xray transparent TUN is stopped.\n'
    printf 'ACTION=status:stopped\n'
  fi
}

do_toggle() {
  require_root

  if is_tun_active; then
    do_stop
  else
    do_start
  fi
}

case "$ACTION" in
  start)
    do_start
    ;;
  stop)
    do_stop
    ;;
  status)
    do_status
    ;;
  toggle)
    do_toggle
    ;;
  *)
    printf 'usage: %s [start|stop|toggle|status] [xray_bin] [xray_config] [state_dir] [tun_name] [remote_dns]\n' "$0" >&2
    exit 2
    ;;
esac
