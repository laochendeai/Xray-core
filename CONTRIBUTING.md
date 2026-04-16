# Contributing

This repository uses a hard-gated change workflow for code, docs, WebPanel, and release-related changes.

## Rule entrypoints

Use these files together:

- [`CLAUDE.md`](CLAUDE.md): project-specific working rules for planning, regression checks, risky actions, worktree isolation, repo closeout, runtime safety, and local-state handling
- [`README.md`](README.md): repository overview, release matrix, build entry points, and contributor workflow summary
- [`REAL_MACHINE_VERIFICATION.md`](REAL_MACHINE_VERIFICATION.md): required runtime-sensitive verification for TUN, control-plane, routing, and reboot-sensitive changes
- [`.github/ISSUE_TEMPLATE/change-request.md`](.github/ISSUE_TEMPLATE/change-request.md): issue template for planned changes
- [`.github/pull_request_template.md`](.github/pull_request_template.md): PR checklist for verification, locale parity, and runtime-sensitive notes
- [`.githooks/pre-push`](.githooks/pre-push): local push gate
- [`scripts/check.sh`](scripts/check.sh): unified local verification gate mirrored by CI
- [`.github/CODEOWNERS`](.github/CODEOWNERS): review ownership for governance-sensitive paths
- [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md): contributor conduct expectations
- [`SECURITY.md`](SECURITY.md): private security reporting path

## Required workflow

1. Open or refine an issue for planned work.
2. Create a branch for that issue.
3. Make the smallest change that satisfies the requested outcome.
4. Run `bash scripts/check.sh` before push or PR.
5. Keep the shared hook path enabled:
   ```bash
   git config core.hooksPath .githooks
   ```
6. Open a PR using [`.github/pull_request_template.md`](.github/pull_request_template.md).
7. Include a closing issue reference such as `Closes #123` in the PR body.

## Additional repository requirements

- Any user-facing UI text change must update both `web/src/i18n/locales/zh-CN.json` and `web/src/i18n/locales/en.json` in the same change.
- Runtime-sensitive changes must include verification notes from [`REAL_MACHINE_VERIFICATION.md`](REAL_MACHINE_VERIFICATION.md).
- Do not bypass hooks, skip the unified verification gate, or mix unrelated changes in one branch.
- The unified verification gate rejects local runtime state, backups, debug output, and built binaries in the evaluated change set before push/PR.
- Do not include local runtime state, debug output, generated artifacts, or built binaries in a PR unless the change explicitly requires them.

## High-risk changes

Changes touching these areas need extra care and usually need an implementation plan before editing:

- TUN, DNS, routing, transparent proxy behavior, startup, clean-state, or runtime helpers
- node-pool lifecycle, quarantine logic, subscription import, or validation flow
- release packaging, installer flows, and GitHub Actions release behavior
- machine-level helper scripts, local service setup, or other runtime-sensitive flows

Use [`CLAUDE.md`](CLAUDE.md) as the canonical rule layer for those cases.
