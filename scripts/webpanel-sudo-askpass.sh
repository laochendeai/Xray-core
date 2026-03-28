#!/usr/bin/env bash
set -euo pipefail

PROMPT="${SUDO_ASKPASS_PROMPT:-WebPanel needs your password to continue.}"
TITLE="${SUDO_ASKPASS_TITLE:-WebPanel Privilege Setup}"

if command -v zenity >/dev/null 2>&1; then
  exec zenity --password --title="$TITLE" --text="$PROMPT"
fi

if command -v kdialog >/dev/null 2>&1; then
  exec kdialog --title "$TITLE" --password "$PROMPT"
fi

echo "No graphical askpass helper is available (need zenity or kdialog)." >&2
exit 1
