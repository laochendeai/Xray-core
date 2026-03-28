#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
usage: install-webpanel-systemd-service.sh [--config PATH] [--user USER] [--xray-bin PATH] [--service-name NAME]

Installs a systemd service that runs the Xray config containing the embedded WebPanel.
This is the operational step that makes startup clean-state enforcement happen on every boot.
EOF
}

require_root() {
  if [[ ${EUID:-$(id -u)} -ne 0 ]]; then
    echo "This installer must run as root." >&2
    exit 1
  fi
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_PATH="$REPO_ROOT/dev-config.current-nodes.json"
TARGET_USER="${SUDO_USER:-$(stat -c '%U' "$REPO_ROOT")}"
XRAY_BIN="/usr/local/bin/xray-webpanel-xray"
SERVICE_NAME="xray-webpanel"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --config)
      CONFIG_PATH="$2"
      shift 2
      ;;
    --user)
      TARGET_USER="$2"
      shift 2
      ;;
    --xray-bin)
      XRAY_BIN="$2"
      shift 2
      ;;
    --service-name)
      SERVICE_NAME="$2"
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

require_root

for required in "$CONFIG_PATH" "$XRAY_BIN"; do
  if [[ ! -e "$required" ]]; then
    echo "Missing required file: $required" >&2
    exit 1
  fi
done

WORK_DIR="$(cd "$(dirname "$CONFIG_PATH")" && pwd)"
UNIT_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
TARGET_GROUP="$(id -gn "$TARGET_USER")"
TARGET_HOME="$(getent passwd "$TARGET_USER" | cut -d: -f6)"

if [[ -z "$TARGET_HOME" ]]; then
  echo "Unable to resolve home directory for user: $TARGET_USER" >&2
  exit 1
fi

for value in "$CONFIG_PATH" "$XRAY_BIN" "$WORK_DIR" "$TARGET_USER" "$TARGET_GROUP" "$TARGET_HOME"; do
  if [[ "$value" =~ [[:space:]] ]]; then
    echo "Whitespace is not supported in systemd installer arguments: $value" >&2
    exit 1
  fi
done

cat >"$UNIT_FILE" <<EOF
[Unit]
Description=Xray WebPanel Control Plane
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$TARGET_USER
Group=$TARGET_GROUP
WorkingDirectory=$WORK_DIR
ExecStart=$XRAY_BIN run -c $CONFIG_PATH
Restart=on-failure
RestartSec=2
LimitNOFILE=1048576
Environment=HOME=$TARGET_HOME

[Install]
WantedBy=multi-user.target
EOF

chmod 0644 "$UNIT_FILE"
systemctl daemon-reload
systemctl enable --now "$SERVICE_NAME"
systemctl --no-pager --full status "$SERVICE_NAME" || true

cat <<EOF
Installed systemd unit: $UNIT_FILE
Enabled and started:    $SERVICE_NAME

Next verification steps:
  1. $REPO_ROOT/scripts/verify-webpanel-control-plane.sh preflight --config $CONFIG_PATH
  2. $REPO_ROOT/scripts/rehearse-webpanel-fallback.sh --config $CONFIG_PATH --apply
  3. reboot
  4. $REPO_ROOT/scripts/verify-webpanel-control-plane.sh post-reboot --config $CONFIG_PATH
EOF
