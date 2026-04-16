#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

collect_blocked_paths() {
  local output
  if ! output="$($@ 2>/dev/null)"; then
    return 1
  fi

  if [[ -z "$output" ]]; then
    return 0
  fi

  while IFS= read -r path; do
    [[ -z "$path" ]] && continue

    case "$path" in
      control_plane_state.json|node_pool_state.json|dev-config.current-nodes.json)
        printf '%s\n' "$path"
        ;;
      runtime/*|backups/*|output/*)
        printf '%s\n' "$path"
        ;;
      xray|webpanel.test.exe)
        printf '%s\n' "$path"
        ;;
      *.log|*.bak|*.tmp|debug.*|*/debug.*)
        printf '%s\n' "$path"
        ;;
    esac
  done <<< "$output"
}

resolve_ci_base() {
  local payload_base=""

  if [[ -n "${GITHUB_EVENT_PATH:-}" ]] && [[ -f "${GITHUB_EVENT_PATH}" ]]; then
    payload_base="$(python3 - <<'PY' "${GITHUB_EVENT_PATH}"
import json
import os
import sys
from pathlib import Path

payload = json.loads(Path(sys.argv[1]).read_text())
event_name = os.environ.get('GITHUB_EVENT_NAME', '')
if event_name == 'pull_request':
    print(payload.get('pull_request', {}).get('base', {}).get('sha', ''))
elif event_name == 'push':
    print(payload.get('before', ''))
else:
    print('')
PY
)"
    if [[ -n "$payload_base" ]] && [[ "$payload_base" != "0000000000000000000000000000000000000000" ]] && git rev-parse --verify "${payload_base}^{commit}" >/dev/null 2>&1; then
      printf '%s\n' "$payload_base"
      return 0
    fi
  fi

  if [[ "${GITHUB_EVENT_NAME:-}" == "pull_request" ]] && [[ -n "${GITHUB_BASE_REF:-}" ]]; then
    local remote_ref="origin/${GITHUB_BASE_REF}"
    if git rev-parse --verify "$remote_ref" >/dev/null 2>&1; then
      git merge-base HEAD "$remote_ref"
      return 0
    fi
  fi

  if [[ "${GITHUB_EVENT_NAME:-}" == "push" ]] && [[ -n "${GITHUB_EVENT_BEFORE:-}" ]] && [[ "${GITHUB_EVENT_BEFORE}" != "0000000000000000000000000000000000000000" ]]; then
    if git rev-parse --verify "${GITHUB_EVENT_BEFORE}^{commit}" >/dev/null 2>&1; then
      printf '%s\n' "$GITHUB_EVENT_BEFORE"
      return 0
    fi
  fi

  return 1
}

resolve_local_base() {
  if git rev-parse --verify '@{upstream}' >/dev/null 2>&1; then
    git merge-base HEAD '@{upstream}'
    return 0
  fi

  if git rev-parse --verify 'HEAD~1' >/dev/null 2>&1; then
    git rev-parse 'HEAD~1'
    return 0
  fi

  return 1
}

base=""
range_label=""
blocked_paths=""

staged_paths="$(collect_blocked_paths git diff --cached --name-only || true)"
unstaged_paths="$(collect_blocked_paths git diff --name-only || true)"

if base="$(resolve_ci_base)"; then
  range_label="${base}...HEAD + index+worktree"
  range_paths="$(collect_blocked_paths git diff --name-only "${base}...HEAD")"
  blocked_paths="$(printf '%s\n%s\n%s\n' "$range_paths" "$staged_paths" "$unstaged_paths" | sed '/^$/d' | sort -u)"
elif base="$(resolve_local_base)"; then
  range_label="${base}...HEAD + index+worktree"
  range_paths="$(collect_blocked_paths git diff --name-only "${base}...HEAD")"
  blocked_paths="$(printf '%s\n%s\n%s\n' "$range_paths" "$staged_paths" "$unstaged_paths" | sed '/^$/d' | sort -u)"
else
  range_label="index+worktree"
  blocked_paths="$(printf '%s\n%s\n' "$staged_paths" "$unstaged_paths" | sed '/^$/d' | sort -u)"
fi

if [[ -z "$blocked_paths" ]]; then
  printf '[local-state-guard] no blocked local-state or build artifacts detected in %s\n' "$range_label"
  exit 0
fi

printf '[local-state-guard] remove local runtime state, backups, debug output, or built binaries from the change set before pushing\n' >&2
printf '[local-state-guard] diff range: %s\n' "$range_label" >&2
while IFS= read -r path; do
  [[ -z "$path" ]] && continue
  printf '[local-state-guard] blocked: %s\n' "$path" >&2
done <<< "$blocked_paths"
exit 1
