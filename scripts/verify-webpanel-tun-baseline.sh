#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./lib-webpanel.sh
source "$SCRIPT_DIR/lib-webpanel.sh"

usage() {
  cat <<'EOF'
usage: verify-webpanel-tun-baseline.sh [--config PATH] [--base-url URL] [--output-dir DIR] [--protected-url URL] [--proxy-target TAG] [--proxy-probe-url URL] [--quic-url URL] [--curl-timeout-sec N] [--allow-running]

Repeatable real-machine verification for the transparent-TUN correctness baseline.

This flow verifies:
  - direct egress reporting before and after TUN enablement
  - runtime DNS/routing diagnostics and generated runtime config markers
  - one representative protected/direct site while TUN is running
  - one proxy-path probe, plus an HTTP/3 probe when curl supports it

Examples:
  ./scripts/verify-webpanel-tun-baseline.sh --config /path/to/config.json
  ./scripts/verify-webpanel-tun-baseline.sh --config /path/to/config.json --protected-url https://hifly.cc/study/feature/index.html
EOF
}

log() {
  printf 'INFO %s\n' "$*"
}

warn() {
  printf 'WARN %s\n' "$*" >&2
}

die() {
  printf 'FAIL %s\n' "$*" >&2
  exit 1
}

CONFIG_PATH="$(webpanel_default_config_path)"
BASE_URL_OVERRIDE=""
OUTPUT_DIR="/tmp/webpanel-tun-baseline"
PROTECTED_URL=""
PROXY_TARGET=""
PROXY_PROBE_URL="https://www.gstatic.com/generate_204"
QUIC_URL="https://cloudflare-quic.com/"
CURL_TIMEOUT_SEC=15
ALLOW_RUNNING=0
BASE_URL_ARGS=()
PROXY_PROBE_ATTEMPTS=3

TOKEN=""
STARTED_BY_SCRIPT=0
SNAPSHOT_BEFORE=""
SNAPSHOT_RUNNING=""
SNAPSHOT_RESTORED=""

BEFORE_STATUS_JSON=""
SETTINGS_JSON=""
ACTIVE_POOL_JSON=""
RUNNING_STATUS_JSON=""
RESTORED_STATUS_JSON=""
PUBLIC_IP_BEFORE_JSON=""
PUBLIC_IP_RUNNING_JSON=""
PROTECTED_PROBE_JSON=""
PROXY_PROBE_JSON=""
QUIC_PROBE_JSON=""
RUNTIME_CHECKS_JSON=""
SUMMARY_JSON=""
SUMMARY_TXT=""

TMP_FILES=()

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
    --protected-url)
      PROTECTED_URL="$2"
      shift 2
      ;;
    --proxy-target)
      PROXY_TARGET="$2"
      shift 2
      ;;
    --proxy-probe-url)
      PROXY_PROBE_URL="$2"
      shift 2
      ;;
    --quic-url)
      QUIC_URL="$2"
      shift 2
      ;;
    --curl-timeout-sec)
      CURL_TIMEOUT_SEC="$2"
      shift 2
      ;;
    --allow-running)
      ALLOW_RUNNING=1
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

register_tmp() {
  local file="$1"
  TMP_FILES+=("$file")
}

new_tmp() {
  local file
  file="$(mktemp)"
  register_tmp "$file"
  printf '%s\n' "$file"
}

cleanup_tmp() {
  local file
  for file in "${TMP_FILES[@]:-}"; do
    [[ -n "$file" && -f "$file" ]] && rm -f "$file"
  done
  return 0
}

api_get_checked() {
  local path="$1"
  local output_file="$2"
  local status_code
  status_code="$(webpanel_api_get "$TOKEN" "$path" "$output_file")" || status_code="curl_failed"
  [[ "$status_code" == "200" ]] || die "GET $path failed: $status_code"
}

api_post_checked() {
  local path="$1"
  local output_file="$2"
  local payload="${3:-}"
  local status_code
  status_code="$(webpanel_api_post "$TOKEN" "$path" "$output_file" "$payload")" || status_code="curl_failed"
  [[ "$status_code" == "200" ]] || {
    cat "$output_file" >&2 || true
    die "POST $path failed: $status_code"
  }
}

wait_for_running_state() {
  local expected="$1"
  local timeout_s="$2"
  local started_at now status_json running

  started_at="$(date +%s)"
  while true; do
    status_json="$(new_tmp)"
    if [[ "$(webpanel_api_get "$TOKEN" "/api/v1/tun/status" "$status_json" 2>/dev/null || true)" == "200" ]]; then
      running="$(json_get "$status_json" "running" 2>/dev/null || true)"
      if [[ "$running" == "$expected" ]]; then
        return 0
      fi
    fi
    now="$(date +%s)"
    if (( now - started_at >= timeout_s )); then
      return 1
    fi
    sleep 1
  done
}

derive_protected_url() {
  local settings_json="$1"
  python3 - "$settings_json" <<'PY'
import json
import re
import sys
from pathlib import Path

data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
rules = data.get("protectDomains") or []

domain_re = re.compile(r"^[A-Za-z0-9.-]+$")

for rule in rules:
    value = str(rule or "").strip()
    if not value:
        continue
    if value.startswith("full:"):
        print(f"https://{value[5:]}")
        raise SystemExit(0)
    if value.startswith("domain:"):
        host = value[7:]
        if host:
            print(f"https://{host}")
            raise SystemExit(0)
    if value.startswith("*."):
        host = value[2:]
        if host:
            print(f"https://{host}")
            raise SystemExit(0)
    if domain_re.match(value):
        print(f"https://{value}")
        raise SystemExit(0)
PY
}

select_proxy_target() {
  local active_pool_json="$1"
  python3 - "$active_pool_json" <<'PY'
import json
import sys
from pathlib import Path

data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
for node in data.get("nodes") or []:
    tag = (node.get("outboundTag") or "").strip()
    if tag:
        print(tag)
        raise SystemExit(0)
PY
}

select_balancer_target() {
  local balancer_tag="${1:-auto}"
  local output_file
  output_file="$(new_tmp)"
  if [[ "$(webpanel_api_get "$TOKEN" "/api/v1/routing/balancers/$balancer_tag" "$output_file" 2>/dev/null || true)" != "200" ]]; then
    return 1
  fi

  python3 - "$output_file" <<'PY'
import json
import sys
from pathlib import Path

data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
balancer = data.get("balancer") or {}
override = balancer.get("override") or {}
principal = ((balancer.get("principle_target") or {}).get("tag") or [])

target = override.get("target")
if target:
    print(target)
elif principal:
    print(principal[0])
PY
}

curl_probe_json() {
  local result_path="$1"
  local method="$2"
  local url="$3"
  shift 3

  local body_file meta_file exit_code
  body_file="$(new_tmp)"
  meta_file="$(new_tmp)"

  set +e
  if [[ "$method" == "HEAD" ]]; then
    curl -I -L --max-time "$CURL_TIMEOUT_SEC" -sS \
      -o /dev/null \
      -w '%{http_code}\t%{remote_ip}\t%{time_total}\t%{url_effective}' \
      "$@" \
      "$url" >"$meta_file"
  else
    curl -L --max-time "$CURL_TIMEOUT_SEC" -sS \
      -o "$body_file" \
      -w '%{http_code}\t%{remote_ip}\t%{time_total}\t%{url_effective}' \
      "$@" \
      "$url" >"$meta_file"
  fi
  exit_code=$?
  set -e

  python3 - "$result_path" "$meta_file" "$body_file" "$url" "$method" "$exit_code" <<'PY'
import json
import sys
from pathlib import Path

result_path = Path(sys.argv[1])
meta_path = Path(sys.argv[2])
body_path = Path(sys.argv[3])
url = sys.argv[4]
method = sys.argv[5]
exit_code = int(sys.argv[6])

meta = meta_path.read_text(encoding="utf-8", errors="replace").strip().split("\t")
meta += ["", "", "", ""]
http_code, remote_ip, time_total, final_url = meta[:4]

body = ""
if method != "HEAD" and body_path.exists():
    body = body_path.read_text(encoding="utf-8", errors="replace").strip()

payload = {
    "url": url,
    "method": method,
    "exitCode": exit_code,
    "httpCode": http_code,
    "remoteIp": remote_ip,
    "timeTotal": time_total,
    "effectiveUrl": final_url or url,
}
if body:
    payload["responseBody"] = body

result_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY
}

run_proxy_probe_with_retries() {
  local attempt exit_code log_path final_log_path
  final_log_path="$OUTPUT_DIR/proxy-probe.txt"

  for ((attempt = 1; attempt <= PROXY_PROBE_ATTEMPTS; attempt++)); do
    log_path="$OUTPUT_DIR/proxy-probe-attempt-${attempt}.txt"
    set +e
    "$SCRIPT_DIR/probe-webpanel-outbound.sh" \
      --config "$CONFIG_PATH" \
      "${BASE_URL_ARGS[@]}" \
      --target "$PROXY_TARGET" \
      --probe-url "$PROXY_PROBE_URL" >"$log_path" 2>&1
    exit_code=$?
    set -e

    cp "$log_path" "$final_log_path"
    if [[ "$exit_code" == "0" ]]; then
      printf '%s\t%s\t%s\n' "$exit_code" "$attempt" "$final_log_path"
      return 0
    fi
    sleep 1
  done

  printf '%s\t%s\t%s\n' "$exit_code" "$PROXY_PROBE_ATTEMPTS" "$final_log_path"
  return 0
}

verify_runtime_state() {
  local runtime_config_json="$1"
  local settings_json="$2"
  local running_status_json="$3"
  local result_json="$4"

  python3 - "$runtime_config_json" "$settings_json" "$running_status_json" "$result_json" <<'PY'
import json
import sys
from pathlib import Path

runtime_path = Path(sys.argv[1])
settings_path = Path(sys.argv[2])
status_path = Path(sys.argv[3])
result_path = Path(sys.argv[4])

runtime = json.loads(runtime_path.read_text(encoding="utf-8"))
settings = json.loads(settings_path.read_text(encoding="utf-8"))
status = json.loads(status_path.read_text(encoding="utf-8"))

rules = ((runtime.get("routing") or {}).get("rules") or [])
servers = ((runtime.get("dns") or {}).get("servers") or [])
protect_domains = settings.get("protectDomains") or []
remote_dns = settings.get("remoteDns") or []
diagnostics = {item.get("category"): item for item in status.get("routingDiagnostics") or [] if isinstance(item, dict)}

def inbound_has(tag, key, value):
    for rule in rules:
        if not isinstance(rule, dict):
            continue
        inbound_tags = rule.get("inboundTag") or []
        if inbound_tags == [tag] and rule.get(key) == value:
            return True
    return False

def has_dns_rule():
    for rule in rules:
        if not isinstance(rule, dict):
            continue
        inbound_tags = rule.get("inboundTag") or []
        if inbound_tags == ["tun-in"] and rule.get("port") == "53" and rule.get("outboundTag") == "dns-out":
            return True
    return False

def has_protect_direct_rule():
    if not protect_domains:
        return True
    expected = set(protect_domains)
    for rule in rules:
        if not isinstance(rule, dict):
            continue
        domains = rule.get("domain") or []
        if rule.get("outboundTag") == "direct" and expected.issubset(set(domains)):
            return True
    return False

def remote_dns_present():
    runtime_addresses = set()
    for item in servers:
        if isinstance(item, str):
            runtime_addresses.add(item)
            continue
        if isinstance(item, dict):
            address = item.get("address")
            if address:
                runtime_addresses.add(address)
    for resolver in remote_dns:
        if resolver in runtime_addresses or f"tcp://{resolver}" in runtime_addresses:
            return True
    return not remote_dns

checks = {
    "hasTunDnsInterceptionRule": has_dns_rule(),
    "hasProtectedDirectRule": has_protect_direct_rule(),
    "hasDnsDirectLocalRoute": inbound_has("dns-direct-local", "outboundTag", "direct"),
    "hasDnsRemoteRoute": inbound_has("dns-remote", "balancerTag", "node-pool-active"),
    "hasTunCatchAllRoute": inbound_has("tun-in", "balancerTag", "node-pool-active"),
    "hasRemoteDnsServer": remote_dns_present(),
    "hasProtectedRoutingDiagnostic": ("protected_direct_domains" in diagnostics) if protect_domains else True,
    "hasDefaultProxyDiagnostic": "default_proxy_domains" in diagnostics,
    "hasCnRoutingDiagnostic": "cn_direct_domains" in diagnostics,
}

required = [
    "hasTunDnsInterceptionRule",
    "hasProtectedDirectRule",
    "hasDnsDirectLocalRoute",
    "hasDnsRemoteRoute",
    "hasTunCatchAllRoute",
    "hasRemoteDnsServer",
    "hasProtectedRoutingDiagnostic",
    "hasDefaultProxyDiagnostic",
]

payload = {
    "runtimeConfigPath": str(runtime_path),
    "checks": checks,
    "passed": all(checks[key] for key in required),
}
result_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY
}

write_quic_prereq_json() {
  local result_json="$1"
  local running_status_json="$2"
  local proxy_probe_json="$3"
  local http3_supported="$4"
  local http3_exit_code="$5"

  local udp443_capture="false"
  if command -v ip >/dev/null 2>&1 && ip -4 rule show | grep -Eq "^${WEBPANEL_CAPTURE_UDP_443_RULE_PREF}:.*ipproto udp.* dport 443 .*lookup ${WEBPANEL_CAPTURE_ROUTE_TABLE_ID}\b"; then
    udp443_capture="true"
  fi

  python3 - "$result_json" "$running_status_json" "$proxy_probe_json" "$udp443_capture" "$http3_supported" "$http3_exit_code" <<'PY'
import json
import sys
from pathlib import Path

result_path = Path(sys.argv[1])
status = json.loads(Path(sys.argv[2]).read_text(encoding="utf-8"))
proxy_probe = json.loads(Path(sys.argv[3]).read_text(encoding="utf-8"))
udp443_capture = sys.argv[4] == "true"
http3_supported = sys.argv[5] == "true"
http3_exit_code = int(sys.argv[6])

proxy_egress = status.get("proxyEgress") or {}
proxy_probe_exit_code = proxy_probe.get("exitCode")
proxy_probe_passed = int(proxy_probe_exit_code if proxy_probe_exit_code is not None else 1) == 0
prerequisites_passed = udp443_capture and proxy_egress.get("status") == "dynamic"
http3_passed = (not http3_supported) or http3_exit_code == 0
payload = {
    "udp443CaptureRuleActive": udp443_capture,
    "proxyEgressStatus": proxy_egress.get("status"),
    "proxyProbeExitCode": proxy_probe.get("exitCode"),
    "proxyProbePassed": proxy_probe_passed,
    "http3Supported": http3_supported,
    "http3ExitCode": http3_exit_code,
    "prerequisitesPassed": prerequisites_passed,
    "http3Passed": http3_passed,
}
payload["passed"] = prerequisites_passed
result_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY
}

write_summary_files() {
  python3 - \
    "$SUMMARY_JSON" \
    "$SUMMARY_TXT" \
    "$BEFORE_STATUS_JSON" \
    "$RUNNING_STATUS_JSON" \
    "$RESTORED_STATUS_JSON" \
    "$PUBLIC_IP_BEFORE_JSON" \
    "$PUBLIC_IP_RUNNING_JSON" \
    "$PROTECTED_PROBE_JSON" \
    "$PROXY_PROBE_JSON" \
    "$QUIC_PROBE_JSON" \
    "$RUNTIME_CHECKS_JSON" \
    "$PROTECTED_URL" \
    "$PROXY_TARGET" \
    "$PROXY_PROBE_URL" \
    "$SNAPSHOT_BEFORE" \
    "$SNAPSHOT_RUNNING" \
    "$SNAPSHOT_RESTORED" <<'PY'
import json
import sys
from pathlib import Path

(
    summary_json_path,
    summary_txt_path,
    before_status_path,
    running_status_path,
    restored_status_path,
    public_ip_before_path,
    public_ip_running_path,
    protected_probe_path,
    proxy_probe_path,
    quic_probe_path,
    runtime_checks_path,
    protected_url,
    proxy_target,
    proxy_probe_url,
    snapshot_before,
    snapshot_running,
    snapshot_restored,
) = sys.argv[1:]

def load_json(path):
    return json.loads(Path(path).read_text(encoding="utf-8"))

before = load_json(before_status_path)
running = load_json(running_status_path)
restored = load_json(restored_status_path)
public_ip_before = load_json(public_ip_before_path)
public_ip_running = load_json(public_ip_running_path)
protected_probe = load_json(protected_probe_path)
proxy_probe = load_json(proxy_probe_path)
quic_probe = load_json(quic_probe_path)
runtime_checks = load_json(runtime_checks_path)

def annotate_same_user_path_probe(payload):
    annotated = dict(payload)
    annotated["publicIp"] = payload.get("responseBody")
    annotated["ipEchoEndpoint"] = payload.get("effectiveUrl") or payload.get("url")
    return annotated

def annotate_protected_target_probe(payload):
    annotated = dict(payload)
    annotated["targetRemoteIp"] = payload.get("remoteIp")
    return annotated

same_user_path_before_tun = annotate_same_user_path_probe(public_ip_before)
same_user_path_with_tun = annotate_same_user_path_probe(public_ip_running)
protected_target_probe = annotate_protected_target_probe(protected_probe)
before_direct = before.get("directEgress") or {}
running_direct = running.get("directEgress") or {}
running_proxy = running.get("proxyEgress") or {}

summary = {
    "before": before,
    "running": running,
    "restored": restored,
    "sameUserPathBeforeTun": same_user_path_before_tun,
    "sameUserPathWithTun": same_user_path_with_tun,
    "protectedTargetProbe": protected_target_probe,
    "proxyProbe": proxy_probe,
    "quicProbe": quic_probe,
    "runtimeChecks": runtime_checks,
    "protectedUrl": protected_url,
    "proxyTarget": proxy_target,
    "proxyProbeUrl": proxy_probe_url,
    "ipSemantics": {
        "directEgressIpBeforeTun": before_direct.get("ip"),
        "directEgressIpWithTun": running_direct.get("ip"),
        "sameUserPathPublicIpBeforeTun": same_user_path_before_tun.get("publicIp"),
        "sameUserPathPublicIpWithTun": same_user_path_with_tun.get("publicIp"),
        "sameUserPathIpEchoEndpoint": same_user_path_with_tun.get("ipEchoEndpoint"),
        "protectedTargetRemoteIp": protected_target_probe.get("targetRemoteIp"),
    },
    "snapshots": {
        "before": snapshot_before,
        "running": snapshot_running,
        "restored": snapshot_restored,
    },
}
Path(summary_json_path).write_text(json.dumps(summary, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

default_proxy_diag = "-"
for item in running.get("routingDiagnostics") or []:
    if item.get("category") == "default_proxy_domains":
        default_proxy_diag = item.get("route", "-")
        break

lines = [
    f"SUMMARY before running={before.get('running')} machineState={before.get('machineState')} direct={before_direct.get('status')} direct_egress_ip={before_direct.get('ip', '-')}",
    f"SUMMARY running running={running.get('running')} machineState={running.get('machineState')} direct={running_direct.get('status')} direct_egress_ip={running_direct.get('ip', '-')} proxy_path={running_proxy.get('status')}",
    f"SUMMARY same_user_path_public_ip before_tun={same_user_path_before_tun.get('publicIp', '-')} with_tun={same_user_path_with_tun.get('publicIp', '-')} ip_echo_endpoint={same_user_path_with_tun.get('ipEchoEndpoint', '-')}",
    f"SUMMARY routing runtime_passed={runtime_checks.get('passed')} default_proxy_diag={default_proxy_diag}",
    f"SUMMARY protected_target url={protected_url} exit={protected_target_probe.get('exitCode')} http_code={protected_target_probe.get('httpCode')} target_remote_ip={protected_target_probe.get('targetRemoteIp', '-')}",
    f"SUMMARY proxy_probe target={proxy_target} exit={proxy_probe.get('exitCode')} passed={proxy_probe.get('proxyProbePassed', proxy_probe.get('exitCode') == 0)} probe_url={proxy_probe_url}",
    f"SUMMARY quic supported={quic_probe.get('http3Supported')} http3_passed={quic_probe.get('http3Passed')} prerequisites_passed={quic_probe.get('prerequisitesPassed')}",
    f"SUMMARY restored running={restored.get('running')} machineState={restored.get('machineState')} reason={restored.get('lastStateReason')}",
    f"SUMMARY evidence before={snapshot_before} running={snapshot_running} restored={snapshot_restored}",
    f"SUMMARY summary_json={summary_json_path}",
]
Path(summary_txt_path).write_text("\n".join(lines) + "\n", encoding="utf-8")
print("\n".join(lines))
PY
}

cleanup() {
  local exit_code="$1"
  local output_file status_json running

  if [[ $STARTED_BY_SCRIPT -eq 1 && -n "$TOKEN" ]]; then
    status_json="$(new_tmp)"
    if [[ "$(webpanel_api_get "$TOKEN" "/api/v1/tun/status" "$status_json" 2>/dev/null || true)" == "200" ]]; then
      running="$(json_get "$status_json" "running" 2>/dev/null || true)"
      if [[ "$running" == "true" ]]; then
        warn "cleanup restoring clean mode after non-terminal exit"
        output_file="$(new_tmp)"
        webpanel_api_post "$TOKEN" "/api/v1/tun/restore-clean" "$output_file" >/dev/null 2>&1 || true
      fi
    fi
  fi

  cleanup_tmp
  exit "$exit_code"
}

trap 'cleanup $?' EXIT

require_command curl
require_command python3
mkdir -p "$OUTPUT_DIR"

webpanel_load_config "$CONFIG_PATH" "$BASE_URL_OVERRIDE"
TOKEN="$(webpanel_login)" || die "Unable to login to $WEBPANEL_BASE_URL"
if [[ -n "$BASE_URL_OVERRIDE" ]]; then
  BASE_URL_ARGS=(--base-url "$BASE_URL_OVERRIDE")
fi

log "running control-plane preflight"
if ! "$SCRIPT_DIR/verify-webpanel-control-plane.sh" preflight --config "$CONFIG_PATH" "${BASE_URL_ARGS[@]}" | tee "$OUTPUT_DIR/preflight.txt"; then
  die "preflight verification failed"
fi

BEFORE_STATUS_JSON="$OUTPUT_DIR/status-before.json"
SETTINGS_JSON="$OUTPUT_DIR/tun-settings.json"
ACTIVE_POOL_JSON="$OUTPUT_DIR/node-pool-active.json"
RUNNING_STATUS_JSON="$OUTPUT_DIR/status-running.json"
RESTORED_STATUS_JSON="$OUTPUT_DIR/status-restored.json"
PUBLIC_IP_BEFORE_JSON="$OUTPUT_DIR/public-ip-before.json"
PUBLIC_IP_RUNNING_JSON="$OUTPUT_DIR/public-ip-running.json"
PROTECTED_PROBE_JSON="$OUTPUT_DIR/protected-probe.json"
PROXY_PROBE_JSON="$OUTPUT_DIR/proxy-probe.json"
QUIC_PROBE_JSON="$OUTPUT_DIR/quic-probe.json"
RUNTIME_CHECKS_JSON="$OUTPUT_DIR/runtime-checks.json"
SUMMARY_JSON="$OUTPUT_DIR/baseline-summary.json"
SUMMARY_TXT="$OUTPUT_DIR/baseline-summary.txt"

api_get_checked "/api/v1/tun/status" "$BEFORE_STATUS_JSON"
api_get_checked "/api/v1/tun/settings" "$SETTINGS_JSON"
api_get_checked "/api/v1/node-pool?status=active" "$ACTIVE_POOL_JSON"

INITIAL_RUNNING="$(json_get "$BEFORE_STATUS_JSON" "running" 2>/dev/null || true)"
if [[ "$INITIAL_RUNNING" == "true" && $ALLOW_RUNNING -ne 1 ]]; then
  die "Transparent mode is already running. Restore clean mode first or rerun with --allow-running."
fi
if [[ "$INITIAL_RUNNING" == "true" && $ALLOW_RUNNING -eq 1 ]]; then
  log "initial state is running; restoring clean baseline before verification"
  RESTORE_OUTPUT="$(new_tmp)"
  api_post_checked "/api/v1/tun/restore-clean" "$RESTORE_OUTPUT"
  wait_for_running_state "false" 30 || die "Timed out waiting for restore-clean before baseline capture"
  api_get_checked "/api/v1/tun/status" "$BEFORE_STATUS_JSON"
  INITIAL_RUNNING="false"
fi

if [[ -z "$PROTECTED_URL" ]]; then
  PROTECTED_URL="$(derive_protected_url "$SETTINGS_JSON" || true)"
fi
[[ -n "$PROTECTED_URL" ]] || die "Unable to derive a protected direct URL from tun settings. Use --protected-url."

if [[ -z "$PROXY_TARGET" ]]; then
  PROXY_TARGET="$(select_balancer_target auto || true)"
fi
if [[ -z "$PROXY_TARGET" ]]; then
  PROXY_TARGET="$(select_proxy_target "$ACTIVE_POOL_JSON" || true)"
fi
[[ -n "$PROXY_TARGET" ]] || die "No active node with outboundTag is available for proxy-path probing."

SNAPSHOT_BEFORE="$(webpanel_capture_snapshot "$TOKEN" "$OUTPUT_DIR" "before-$(date +%Y%m%d-%H%M%S)")"
log "captured baseline snapshot: $SNAPSHOT_BEFORE"

curl_probe_json "$PUBLIC_IP_BEFORE_JSON" "GET" "https://ipv4.icanhazip.com"

DIRECT_BEFORE_IP="$(json_get "$BEFORE_STATUS_JSON" "directEgress.ip" 2>/dev/null || true)"
PUBLIC_IP_BEFORE="$(json_get "$PUBLIC_IP_BEFORE_JSON" "responseBody" 2>/dev/null || true)"
[[ -n "$DIRECT_BEFORE_IP" ]] || die "Baseline directEgress IP is unavailable before TUN start."
[[ "$PUBLIC_IP_BEFORE" == "$DIRECT_BEFORE_IP" ]] || die "Baseline public IP ($PUBLIC_IP_BEFORE) does not match directEgress IP ($DIRECT_BEFORE_IP)."

IFS=$'\t' read -r PROXY_PROBE_EXIT PROXY_PROBE_ATTEMPT_USED PROXY_PROBE_LOG_PATH <<<"$(run_proxy_probe_with_retries)"
python3 - "$PROXY_PROBE_JSON" "$PROXY_TARGET" "$PROXY_PROBE_URL" "$PROXY_PROBE_EXIT" "$PROXY_PROBE_LOG_PATH" "$PROXY_PROBE_ATTEMPT_USED" <<'PY'
import json
import sys
from pathlib import Path

exit_code = int(sys.argv[4])
payload = {
    "target": sys.argv[2],
    "probeUrl": sys.argv[3],
    "exitCode": exit_code,
    "proxyProbePassed": exit_code == 0,
    "logPath": sys.argv[5],
    "attempt": int(sys.argv[6]),
}
Path(sys.argv[1]).write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY
if [[ "$PROXY_PROBE_EXIT" != "0" ]]; then
  warn "Forced proxy-path probe failed for target $PROXY_TARGET; keeping it as advisory evidence only."
fi

if [[ "$INITIAL_RUNNING" != "true" ]]; then
  START_OUTPUT="$(new_tmp)"
  api_post_checked "/api/v1/tun/start" "$START_OUTPUT"
  STARTED_BY_SCRIPT=1
fi

wait_for_running_state "true" 30 || die "Timed out waiting for transparent mode to report running"
api_get_checked "/api/v1/tun/status" "$RUNNING_STATUS_JSON"
SNAPSHOT_RUNNING="$(webpanel_capture_snapshot "$TOKEN" "$OUTPUT_DIR" "running-$(date +%Y%m%d-%H%M%S)")"
log "captured running snapshot: $SNAPSHOT_RUNNING"

RUNNING_DIRECT_IP="$(json_get "$RUNNING_STATUS_JSON" "directEgress.ip" 2>/dev/null || true)"
RUNNING_PROXY_STATUS="$(json_get "$RUNNING_STATUS_JSON" "proxyEgress.status" 2>/dev/null || true)"
[[ "$RUNNING_DIRECT_IP" == "$DIRECT_BEFORE_IP" ]] || die "Running directEgress IP ($RUNNING_DIRECT_IP) no longer matches baseline direct IP ($DIRECT_BEFORE_IP)."
[[ "$RUNNING_PROXY_STATUS" == "dynamic" ]] || die "Expected proxyEgress.status=dynamic while TUN is running, got ${RUNNING_PROXY_STATUS:-unknown}."

verify_runtime_state "$SNAPSHOT_RUNNING/runtime-config.json" "$SETTINGS_JSON" "$RUNNING_STATUS_JSON" "$RUNTIME_CHECKS_JSON"
if [[ "$(json_get "$RUNTIME_CHECKS_JSON" "passed" 2>/dev/null || true)" != "true" ]]; then
  die "Runtime DNS/routing verification failed. See $RUNTIME_CHECKS_JSON."
fi

curl_probe_json "$PUBLIC_IP_RUNNING_JSON" "GET" "https://ipv4.icanhazip.com"
PUBLIC_IP_RUNNING="$(json_get "$PUBLIC_IP_RUNNING_JSON" "responseBody" 2>/dev/null || true)"
if [[ "$PUBLIC_IP_RUNNING" == "$DIRECT_BEFORE_IP" ]]; then
  die "TUN-on public IP probe still matched the direct egress IP; proxy path was not observed."
fi

curl_probe_json "$PROTECTED_PROBE_JSON" "HEAD" "$PROTECTED_URL" --http1.1
if [[ "$(json_get "$PROTECTED_PROBE_JSON" "exitCode" 2>/dev/null || true)" != "0" ]]; then
  die "Protected direct probe failed for $PROTECTED_URL."
fi

HTTP3_SUPPORTED="false"
HTTP3_EXIT_CODE=0
if curl -V 2>/dev/null | grep -Eq '(^|[[:space:]])HTTP3($|[[:space:]])'; then
  HTTP3_SUPPORTED="true"
  set +e
  curl_probe_json "$OUTPUT_DIR/http3-probe.json" "GET" "$QUIC_URL" --http3-only
  HTTP3_EXIT_CODE="$(json_get "$OUTPUT_DIR/http3-probe.json" "exitCode" 2>/dev/null || true)"
  set -e
fi
write_quic_prereq_json "$QUIC_PROBE_JSON" "$RUNNING_STATUS_JSON" "$PROXY_PROBE_JSON" "$HTTP3_SUPPORTED" "${HTTP3_EXIT_CODE:-0}"
if [[ "$HTTP3_SUPPORTED" == "true" && "$HTTP3_EXIT_CODE" != "0" ]]; then
  warn "HTTP/3 probe did not pass; keeping proxy-path prerequisites as the baseline evidence."
fi

RESTORE_OUTPUT="$(new_tmp)"
api_post_checked "/api/v1/tun/restore-clean" "$RESTORE_OUTPUT"
wait_for_running_state "false" 30 || die "Timed out waiting for restore-clean to stop transparent mode"
api_get_checked "/api/v1/tun/status" "$RESTORED_STATUS_JSON"
SNAPSHOT_RESTORED="$(webpanel_capture_snapshot "$TOKEN" "$OUTPUT_DIR" "restored-$(date +%Y%m%d-%H%M%S)")"
log "captured restored snapshot: $SNAPSHOT_RESTORED"

RESTORED_RUNNING="$(json_get "$RESTORED_STATUS_JSON" "running" 2>/dev/null || true)"
RESTORED_STATE="$(json_get "$RESTORED_STATUS_JSON" "machineState" 2>/dev/null || true)"
[[ "$RESTORED_RUNNING" == "false" ]] || die "restore-clean completed but TUN still reports running"
[[ "$RESTORED_STATE" == "clean" ]] || die "restore-clean completed but machineState is ${RESTORED_STATE:-unknown}"
STARTED_BY_SCRIPT=0

write_summary_files
