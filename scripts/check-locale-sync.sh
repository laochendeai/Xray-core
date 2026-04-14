#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

ZH_FILE="web/src/i18n/locales/zh-CN.json"
EN_FILE="web/src/i18n/locales/en.json"

collect_locale_changes() {
  local mode="$1"
  shift

  local output
  if ! output="$($@ 2>/dev/null)"; then
    return 1
  fi

  if [[ -z "$output" ]]; then
    return 0
  fi

  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    case "$line" in
      "$ZH_FILE"|"$EN_FILE")
        printf '%s\n' "$line"
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
changes=""

staged_changes="$(collect_locale_changes staged git diff --cached --name-only || true)"
unstaged_changes="$(collect_locale_changes unstaged git diff --name-only || true)"

if base="$(resolve_ci_base)"; then
  range_label="${base}...HEAD + index+worktree"
  range_changes="$(collect_locale_changes diff-range git diff --name-only "${base}...HEAD")"
  changes="$(printf '%s\n%s\n%s\n' "$range_changes" "$staged_changes" "$unstaged_changes" | sed '/^$/d' | sort -u)"
elif base="$(resolve_local_base)"; then
  range_label="${base}...HEAD + index+worktree"
  range_changes="$(collect_locale_changes diff-range git diff --name-only "${base}...HEAD")"
  changes="$(printf '%s\n%s\n%s\n' "$range_changes" "$staged_changes" "$unstaged_changes" | sed '/^$/d' | sort -u)"
else
  range_label="index+worktree"
  changes="$(printf '%s\n%s\n' "$staged_changes" "$unstaged_changes" | sed '/^$/d' | sort -u)"
fi

zh_changed=0
en_changed=0

if [[ -n "$changes" ]]; then
  while IFS= read -r file; do
    [[ -z "$file" ]] && continue
    if [[ "$file" == "$ZH_FILE" ]]; then
      zh_changed=1
    fi
    if [[ "$file" == "$EN_FILE" ]]; then
      en_changed=1
    fi
  done <<< "$changes"
fi

if [[ "$zh_changed" -eq 0 && "$en_changed" -eq 0 ]]; then
  printf '[locale-sync] no locale file changes detected in %s\n' "$range_label"
  exit 0
fi

if [[ "$zh_changed" -eq 1 && "$en_changed" -eq 1 ]]; then
  printf '[locale-sync] locale files changed together in %s\n' "$range_label"
  exit 0
fi

changed_file="$ZH_FILE"
missing_file="$EN_FILE"
if [[ "$en_changed" -eq 1 ]]; then
  changed_file="$EN_FILE"
  missing_file="$ZH_FILE"
fi

printf '[locale-sync] expected both locale files to change together\n' >&2
printf '[locale-sync] changed: %s\n' "$changed_file" >&2
printf '[locale-sync] missing: %s\n' "$missing_file" >&2
printf '[locale-sync] diff range: %s\n' "$range_label" >&2
exit 1
