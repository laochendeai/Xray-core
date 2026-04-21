#!/usr/bin/env bash
set -euo pipefail

require_root() {
  if [[ ${EUID:-$(id -u)} -ne 0 ]]; then
    echo "This installer must run as root." >&2
    exit 1
  fi
}

usage() {
  cat <<'EOF'
usage: install-webpanel-tun-sudoers.sh [--repo PATH] [--config PATH] [--user USER]
                                       [--xray-src PATH]
                                       [--helper-dst PATH] [--xray-dst PATH]

Installs a root-owned helper and xray binary, updates the webpanel TUN config,
and writes an exact-arguments sudoers rule for one-click TUN switching.
EOF
}

require_root

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_PATH="$REPO_ROOT/dev-config.current-nodes.json"
TARGET_USER="${SUDO_USER:-$(stat -c '%U' "$REPO_ROOT")}"
HELPER_SRC="$REPO_ROOT/scripts/webpanel-tun-helper.sh"
HELPER_DST="/usr/local/libexec/xray-webpanel-tun-helper"
XRAY_SRC="$REPO_ROOT/xray"
XRAY_DST="/usr/local/bin/xray-webpanel-xray"
SUDOERS_FILE="/etc/sudoers.d/xray-webpanel-tun"
TMP_SUDOERS_FILE=""

cleanup_temp_sudoers() {
  if [[ -n "$TMP_SUDOERS_FILE" && -f "$TMP_SUDOERS_FILE" ]]; then
    rm -f "$TMP_SUDOERS_FILE"
  fi
}

trap cleanup_temp_sudoers EXIT

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo)
      REPO_ROOT="$2"
      CONFIG_PATH="$REPO_ROOT/dev-config.current-nodes.json"
      HELPER_SRC="$REPO_ROOT/scripts/webpanel-tun-helper.sh"
      XRAY_SRC="$REPO_ROOT/xray"
      shift 2
      ;;
    --config)
      CONFIG_PATH="$2"
      shift 2
      ;;
    --user)
      TARGET_USER="$2"
      shift 2
      ;;
    --xray-src)
      XRAY_SRC="$2"
      shift 2
      ;;
    --helper-dst)
      HELPER_DST="$2"
      shift 2
      ;;
    --xray-dst)
      XRAY_DST="$2"
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

for required in "$HELPER_SRC" "$XRAY_SRC" "$CONFIG_PATH"; do
  if [[ ! -e "$required" ]]; then
    echo "Missing required file: $required" >&2
    exit 1
  fi
done

mkdir -p "$(dirname "$HELPER_DST")" "$(dirname "$XRAY_DST")"
install -o root -g root -m 0755 "$HELPER_SRC" "$HELPER_DST"
install -o root -g root -m 0755 "$XRAY_SRC" "$XRAY_DST"

config_owner="$(stat -c '%u:%g' "$CONFIG_PATH")"

python3 "$SCRIPT_DIR/normalize-webpanel-tun-config.py" "$CONFIG_PATH" "$HELPER_DST" "$XRAY_DST"

chown "$config_owner" "$CONFIG_PATH"

mapfile -t tun_info < <(python3 - "$CONFIG_PATH" <<'PY'
import json
import sys
from pathlib import Path

data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
tun = data.get("webpanel", {}).get("tun", {})

runtime_config = tun.get("runtimeConfigPath")
state_dir = tun.get("stateDir")
interface = tun.get("interfaceName", "xray0")
remote_dns = tun.get("remoteDns") or ["1.1.1.1", "8.8.8.8"]

if not runtime_config or not state_dir:
    raise SystemExit("webpanel.tun.runtimeConfigPath and stateDir are required")

print(runtime_config)
print(state_dir)
print(interface)
for item in remote_dns:
    print(item)
PY
)

if [[ ${#tun_info[@]} -lt 3 ]]; then
  echo "Failed to parse webpanel.tun settings from $CONFIG_PATH" >&2
  exit 1
fi

RUNTIME_CONFIG_PATH="${tun_info[0]}"
STATE_DIR="${tun_info[1]}"
INTERFACE_NAME="${tun_info[2]}"
REMOTE_DNS=("${tun_info[@]:3}")
if [[ ${#REMOTE_DNS[@]} -eq 0 ]]; then
  REMOTE_DNS=("1.1.1.1" "8.8.8.8")
fi

for value in "$HELPER_DST" "$XRAY_DST" "$RUNTIME_CONFIG_PATH" "$STATE_DIR" "$INTERFACE_NAME" "${REMOTE_DNS[@]}"; do
  if [[ "$value" =~ [[:space:]] ]]; then
    echo "Whitespace is not supported in sudoers-managed arguments: $value" >&2
    exit 1
  fi
done

cmd_suffix_start="start $XRAY_DST $RUNTIME_CONFIG_PATH $STATE_DIR $INTERFACE_NAME"
cmd_suffix_stop="stop $XRAY_DST $RUNTIME_CONFIG_PATH $STATE_DIR $INTERFACE_NAME"
cmd_suffix_toggle="toggle $XRAY_DST $RUNTIME_CONFIG_PATH $STATE_DIR $INTERFACE_NAME"
for dns in "${REMOTE_DNS[@]}"; do
  cmd_suffix_start+=" $dns"
  cmd_suffix_stop+=" $dns"
  cmd_suffix_toggle+=" $dns"
done

TMP_SUDOERS_FILE="$(mktemp)"
python3 "$SCRIPT_DIR/render-webpanel-tun-sudoers.py" \
  "$TARGET_USER" \
  "$HELPER_DST" \
  "$XRAY_DST" \
  "$RUNTIME_CONFIG_PATH" \
  "$STATE_DIR" \
  "$INTERFACE_NAME" \
  "${REMOTE_DNS[@]}" >"$TMP_SUDOERS_FILE"

chmod 0440 "$TMP_SUDOERS_FILE"
visudo -cf "$TMP_SUDOERS_FILE" >/dev/null
install -o root -g root -m 0440 "$TMP_SUDOERS_FILE" "$SUDOERS_FILE"
cleanup_temp_sudoers
TMP_SUDOERS_FILE=""

cat <<EOF
Installed helper: $HELPER_DST
Installed xray:   $XRAY_DST
Updated config:   $CONFIG_PATH
Installed sudoers: $SUDOERS_FILE

Exact sudo command now allowed:
  sudo -n $HELPER_DST $cmd_suffix_start
EOF
