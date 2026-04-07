#!/usr/bin/env bash

set -euo pipefail

require_command() {
  local name="$1"
  if ! command -v "$name" >/dev/null 2>&1; then
    echo "Missing required command: $name" >&2
    exit 1
  fi
}

webpanel_default_config_path() {
  local script_dir repo_root
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  repo_root="$(cd "$script_dir/.." && pwd)"
  printf '%s\n' "$repo_root/dev-config.current-nodes.json"
}

webpanel_load_config() {
  local config_path="$1"
  local base_url_override="${2:-}"

  if [[ ! -f "$config_path" ]]; then
    echo "Config file not found: $config_path" >&2
    exit 1
  fi

  mapfile -t WEBPANEL_CONFIG_VALUES < <(python3 - "$config_path" "$base_url_override" <<'PY'
import json
import sys
from pathlib import Path

config_path = Path(sys.argv[1]).resolve()
base_url_override = sys.argv[2]
data = json.loads(config_path.read_text(encoding="utf-8"))
webpanel = data.get("webpanel", {})
tun = webpanel.get("tun", {})

listen = webpanel.get("listen") or "127.0.0.1:9527"
if base_url_override:
    base_url = base_url_override.rstrip("/")
elif listen.startswith("http://") or listen.startswith("https://"):
    base_url = listen.rstrip("/")
else:
    base_url = f"http://{listen}"

config_dir = config_path.parent
state_dir = Path(tun.get("stateDir") or config_dir / "runtime" / "tun")
runtime_config = Path(tun.get("runtimeConfigPath") or state_dir / "config.json")
helper_path = Path(tun.get("helperPath") or config_dir / "scripts" / "webpanel-tun-helper.sh")
binary_path = Path(tun.get("binaryPath") or "xray")
interface_name = tun.get("interfaceName") or "xray0"
remote_dns = tun.get("remoteDns") or ["1.1.1.1", "8.8.8.8"]
use_sudo = tun.get("useSudo")
if use_sudo is None:
    use_sudo = True

values = [
    str(config_path),
    str(config_dir),
    webpanel.get("username") or "admin",
    webpanel.get("password") or "admin123",
    base_url,
    str(helper_path),
    str(state_dir),
    str(runtime_config),
    interface_name,
    "true" if bool(use_sudo) else "false",
    str(binary_path),
    str(config_dir / "control_plane_state.json"),
    str(config_dir / "node_pool_state.json"),
]
values.extend(str(item) for item in remote_dns)
for value in values:
    print(value)
PY
)

  if [[ ${#WEBPANEL_CONFIG_VALUES[@]} -lt 13 ]]; then
    echo "Failed to parse webpanel config: $config_path" >&2
    exit 1
  fi

  WEBPANEL_CONFIG_PATH="${WEBPANEL_CONFIG_VALUES[0]}"
  WEBPANEL_CONFIG_DIR="${WEBPANEL_CONFIG_VALUES[1]}"
  WEBPANEL_USERNAME="${WEBPANEL_CONFIG_VALUES[2]}"
  WEBPANEL_PASSWORD="${WEBPANEL_CONFIG_VALUES[3]}"
  WEBPANEL_BASE_URL="${WEBPANEL_CONFIG_VALUES[4]}"
  WEBPANEL_HELPER_PATH="${WEBPANEL_CONFIG_VALUES[5]}"
  WEBPANEL_STATE_DIR="${WEBPANEL_CONFIG_VALUES[6]}"
  WEBPANEL_RUNTIME_CONFIG_PATH="${WEBPANEL_CONFIG_VALUES[7]}"
  WEBPANEL_TUN_INTERFACE="${WEBPANEL_CONFIG_VALUES[8]}"
  WEBPANEL_USE_SUDO="${WEBPANEL_CONFIG_VALUES[9]}"
  WEBPANEL_BINARY_PATH="${WEBPANEL_CONFIG_VALUES[10]}"
  WEBPANEL_CONTROL_PLANE_STATE_PATH="${WEBPANEL_CONFIG_VALUES[11]}"
  WEBPANEL_NODE_POOL_STATE_PATH="${WEBPANEL_CONFIG_VALUES[12]}"
  WEBPANEL_REMOTE_DNS=("${WEBPANEL_CONFIG_VALUES[@]:13}")
  WEBPANEL_BYPASS_ROUTE_TABLE_ID="${XRAY_TUN_ROUTE_TABLE_ID:-2027}"
  WEBPANEL_BYPASS_RULE_PREF="${XRAY_TUN_BYPASS_RULE_PREF:-12000}"
  WEBPANEL_CAPTURE_ROUTE_TABLE_ID="${XRAY_TUN_CAPTURE_ROUTE_TABLE_ID:-2028}"
  WEBPANEL_CAPTURE_DNS_RULE_PREF="${XRAY_TUN_CAPTURE_DNS_RULE_PREF:-12010}"
  WEBPANEL_CAPTURE_UDP_443_RULE_PREF="${XRAY_TUN_CAPTURE_UDP_443_RULE_PREF:-12015}"
  WEBPANEL_CAPTURE_TCP_RULE_PREF="${XRAY_TUN_CAPTURE_TCP_RULE_PREF:-12020}"
}

json_get() {
  local json_file="$1"
  local path="$2"
  python3 - "$json_file" "$path" <<'PY'
import json
import sys
from pathlib import Path

json_file = Path(sys.argv[1])
path = [part for part in sys.argv[2].split(".") if part]

data = json.loads(json_file.read_text(encoding="utf-8"))
cur = data
for part in path:
    if isinstance(cur, list):
        try:
            cur = cur[int(part)]
        except (ValueError, IndexError):
            cur = None
    elif isinstance(cur, dict):
        cur = cur.get(part)
    else:
        cur = None
    if cur is None:
        break

if cur is None:
    sys.exit(1)
if isinstance(cur, bool):
    print("true" if cur else "false")
elif isinstance(cur, (dict, list)):
    print(json.dumps(cur, ensure_ascii=False))
else:
    print(cur)
PY
}

json_print_active_nodes() {
  local json_file="$1"
  python3 - "$json_file" <<'PY'
import json
import sys
from pathlib import Path

data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
nodes = data.get("nodes", [])
for node in nodes:
    if node.get("status") != "active":
        continue
    print("\t".join([
        node.get("id", ""),
        node.get("remark", ""),
        node.get("address", ""),
    ]))
PY
}

webpanel_http_json() {
  local output_file="$1"
  shift
  curl -sS -o "$output_file" -w '%{http_code}' "$@"
}

webpanel_login() {
  local output_file http_code payload token
  output_file="$(mktemp)"
  payload="$(python3 - "$WEBPANEL_USERNAME" "$WEBPANEL_PASSWORD" <<'PY'
import json
import sys
print(json.dumps({"username": sys.argv[1], "password": sys.argv[2]}))
PY
)"

  http_code="$(webpanel_http_json "$output_file" \
    -H 'Content-Type: application/json' \
    -X POST \
    -d "$payload" \
    "$WEBPANEL_BASE_URL/api/v1/auth/login")" || {
    rm -f "$output_file"
    echo "curl_failed"
    return 1
  }

  if [[ "$http_code" != "200" ]]; then
    rm -f "$output_file"
    echo "login_http_$http_code"
    return 1
  fi

  token="$(python3 - "$output_file" <<'PY'
import json
import sys
from pathlib import Path

data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
print(data.get("token", ""))
PY
)"
  rm -f "$output_file"

  if [[ -z "$token" ]]; then
    echo "missing_token"
    return 1
  fi

  printf '%s\n' "$token"
}

webpanel_api_get() {
  local token="$1"
  local path="$2"
  local output_file="$3"
  webpanel_http_json "$output_file" \
    -H "Authorization: Bearer $token" \
    "$WEBPANEL_BASE_URL$path"
}

webpanel_api_post() {
  local token="$1"
  local path="$2"
  local output_file="$3"
  local payload="${4:-}"
  if [[ -n "$payload" ]]; then
    webpanel_http_json "$output_file" \
      -H "Authorization: Bearer $token" \
      -H 'Content-Type: application/json' \
      -X POST \
      -d "$payload" \
      "$WEBPANEL_BASE_URL$path"
    return
  fi

  webpanel_http_json "$output_file" \
    -H "Authorization: Bearer $token" \
    -X POST \
    "$WEBPANEL_BASE_URL$path"
}

webpanel_capture_snapshot() {
  local token="$1"
  local output_dir="$2"
  local label="$3"
  local snapshot_dir tun_out pool_out

  snapshot_dir="$output_dir/$label"
  mkdir -p "$snapshot_dir"

  tun_out="$snapshot_dir/tun-status.json"
  pool_out="$snapshot_dir/node-pool.json"

  webpanel_api_get "$token" "/api/v1/tun/status" "$tun_out" >/dev/null
  webpanel_api_get "$token" "/api/v1/node-pool" "$pool_out" >/dev/null

  if [[ -f "$WEBPANEL_CONTROL_PLANE_STATE_PATH" ]]; then
    cp "$WEBPANEL_CONTROL_PLANE_STATE_PATH" "$snapshot_dir/control_plane_state.json"
  fi
  if [[ -f "$WEBPANEL_NODE_POOL_STATE_PATH" ]]; then
    cp "$WEBPANEL_NODE_POOL_STATE_PATH" "$snapshot_dir/node_pool_state.json"
  fi
  if [[ -f "$WEBPANEL_STATE_DIR/xray-tun.log" ]]; then
    cp "$WEBPANEL_STATE_DIR/xray-tun.log" "$snapshot_dir/xray-tun.log"
  fi
  if [[ -f "$WEBPANEL_RUNTIME_CONFIG_PATH" ]]; then
    cp "$WEBPANEL_RUNTIME_CONFIG_PATH" "$snapshot_dir/runtime-config.json"
  fi
  if command -v ip >/dev/null 2>&1; then
    ip link show >"$snapshot_dir/ip-link.txt" 2>&1 || true
    ip -4 rule show >"$snapshot_dir/ip-rule.txt" 2>&1 || true
    ip route show >"$snapshot_dir/ip-route.txt" 2>&1 || true
    ip route show table main >"$snapshot_dir/ip-route-main.txt" 2>&1 || true
    ip route show table "$WEBPANEL_BYPASS_ROUTE_TABLE_ID" >"$snapshot_dir/ip-route-bypass.txt" 2>&1 || true
    ip route show table "$WEBPANEL_CAPTURE_ROUTE_TABLE_ID" >"$snapshot_dir/ip-route-capture.txt" 2>&1 || true
  fi
  if command -v resolvectl >/dev/null 2>&1; then
    resolvectl status >"$snapshot_dir/resolvectl-status.txt" 2>&1 || true
  fi

  printf '%s\n' "$snapshot_dir"
}
