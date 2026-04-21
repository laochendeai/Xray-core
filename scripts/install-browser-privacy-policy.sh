#!/usr/bin/env bash
set -euo pipefail

POLICY_FILE_NAME="${XRAY_BROWSER_POLICY_FILE_NAME:-xray-privacy-policy.json}"
ACTION="install"

usage() {
  cat <<'EOF'
usage: scripts/install-browser-privacy-policy.sh [--install|--remove]

Installs Linux Chromium-family managed policies that prevent daily browsers from
opening non-proxied WebRTC UDP paths. Browser-native DoH is disabled so DNS uses
the system path, where Xray strict TUN routes DNS to encrypted remote resolvers.

After installing, fully close and reopen Chrome/Chromium/Edge/Brave, then verify
the policy on chrome://policy or edge://policy.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --install)
      ACTION="install"
      ;;
    --remove)
      ACTION="remove"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      printf 'unknown argument: %s\n' "$1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

if [[ ${EUID:-$(id -u)} -ne 0 ]]; then
  echo "This script must be run as root because browser managed policies live under /etc." >&2
  exit 1
fi

if [[ -n "${XRAY_BROWSER_POLICY_DIRS:-}" ]]; then
  policy_dir_list="${XRAY_BROWSER_POLICY_DIRS//$'\n'/:}"
  policy_dir_list="${policy_dir_list//,/:}"
  IFS=':' read -r -a policy_dirs <<<"$policy_dir_list"
else
  policy_dirs=(
    "/etc/opt/chrome/policies/managed"
    "/etc/chromium/policies/managed"
    "/etc/opt/edge/policies/managed"
    "/etc/brave/policies/managed"
    "/etc/opt/brave.com/brave/policies/managed"
    "/etc/opt/vivaldi/policies/managed"
  )
fi

if [[ ${#policy_dirs[@]} -eq 0 ]]; then
  echo "No browser policy directories configured." >&2
  exit 1
fi

write_policy() {
  local dir="$1"
  local target="$dir/$POLICY_FILE_NAME"
  local tmp

  mkdir -p "$dir"
  tmp="$(mktemp "$dir/.${POLICY_FILE_NAME}.XXXXXX")"
  cat >"$tmp" <<'JSON'
{
  "WebRtcIPHandling": "disable_non_proxied_udp",
  "DnsOverHttpsMode": "off"
}
JSON
  chmod 0644 "$tmp"
  mv "$tmp" "$target"
  printf 'installed %s\n' "$target"
}

remove_policy() {
  local dir="$1"
  local target="$dir/$POLICY_FILE_NAME"

  if [[ -e "$target" ]]; then
    rm -f "$target"
    printf 'removed %s\n' "$target"
  fi
}

case "$ACTION" in
  install)
    for dir in "${policy_dirs[@]}"; do
      write_policy "$dir"
    done
    ;;
  remove)
    for dir in "${policy_dirs[@]}"; do
      remove_policy "$dir"
    done
    ;;
esac

cat <<'EOF'
Next steps:
1. Fully close every Chrome/Chromium-family browser window and background process.
2. Reopen the browser and check chrome://policy or edge://policy.
3. Confirm WebRtcIPHandling=disable_non_proxied_udp and DnsOverHttpsMode=off are active.
EOF
