#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

CONFIG_PATH="$TMP_DIR/config.json"
cat >"$CONFIG_PATH" <<'JSON'
{
  "webpanel": {
    "tun": {
      "runtimeConfigPath": "/tmp/xray-runtime/tun/config.json",
      "stateDir": "/tmp/xray-runtime/tun",
      "interfaceName": "xray0",
      "remoteDns": [
        "1.1.1.1",
        "8.8.8.8:53",
        "udp://9.9.9.9",
        "https://dns.google/dns-query",
        "tcp+local://119.29.29.29",
        "[2606:4700:4700::1111]:53"
      ]
    }
  }
}
JSON

python3 "$ROOT_DIR/scripts/normalize-webpanel-tun-config.py" \
  "$CONFIG_PATH" \
  "/usr/local/libexec/xray-webpanel-tun-helper" \
  "/usr/local/bin/xray-webpanel-xray"

python3 - "$CONFIG_PATH" <<'PY'
import json
import sys
from pathlib import Path

data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
tun = data["webpanel"]["tun"]

expected_remote_dns = [
    "https://cloudflare-dns.com/dns-query",
    "https://dns.google/dns-query",
    "https://dns.quad9.net/dns-query",
    "https://doh.pub/dns-query",
    "https://[2606:4700:4700::1111]/dns-query",
]

assert tun["helperPath"] == "/usr/local/libexec/xray-webpanel-tun-helper"
assert tun["binaryPath"] == "/usr/local/bin/xray-webpanel-xray"
assert tun["useSudo"] is True
assert tun["remoteDns"] == expected_remote_dns, tun["remoteDns"]
PY
