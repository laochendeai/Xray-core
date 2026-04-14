# Xray-core (laochendeai fork) repository rules

This file is the project-specific rule layer for agent work in this repository. Use it together with the existing issue, PR, hook, and verification assets that already live in the repo.

## Repository identity

- This repository is a maintained fork of Xray-core with an embedded WebPanel and a local control plane for subscriptions, node-pool lifecycle management, TUN orchestration, DNS split-policy, and update discovery.
- Treat changes here as product changes to a local runtime and control plane, not as isolated library edits.
- Prefer the smallest change that satisfies the requested outcome. Do not widen scope without user approval.

## Required change workflow

- For planned work, start from an issue. Use `.github/ISSUE_TEMPLATE/change-request.md` when opening or refining a change request.
- Work on one issue per branch. Do not mix unrelated changes in the same branch.
- Before push or PR, run `bash scripts/check.sh`.
- Keep `.githooks/pre-push` enabled so the local push gate matches the repository workflow.
- PR bodies must include a closing issue reference such as `Closes #123`, matching `.github/pull_request_template.md` and `.github/workflows/ci.yml`.

## Planning gate for non-trivial work

Make a brief implementation plan before editing when the task is non-trivial.

Use a plan first when the change:
- touches TUN, DNS, routing, transparent proxy behavior, startup, clean-state enforcement, or runtime helpers
- changes node-pool lifecycle, quarantine logic, subscription import, or validation flow
- changes both Go backend behavior and WebPanel behavior
- changes release packaging, GitHub Actions, installer behavior, or other distribution logic
- renames, deletes, or migrates important config, state, or API fields

The plan should capture:
- the requested outcome
- the files or flows that will change
- the main regression surface
- the verification path, including whether `REAL_MACHINE_VERIFICATION.md` is required

## Verification and regression guard

- `bash scripts/check.sh` is the default local verification gate. Do not invent a second primary check path.
- Treat regression checking as broader than “the file I edited.” Verify adjacent user flows, API callers, config/state contracts, and docs that could drift because of the change.
- For frontend or WebPanel changes, verify the affected route and the main user path locally, not only via tests.
- For backend or API changes, verify the affected caller path in the WebPanel when applicable.
- If behavior, workflow, or release expectations changed, update the relevant documentation in the same change.

## Runtime and network safety

Runtime-sensitive changes require more than unit tests.

Use `REAL_MACHINE_VERIFICATION.md` when the change affects:
- TUN enable / disable / restore behavior
- control-plane fallback, clean-state, or reboot-sensitive logic
- routing, DNS, direct-vs-proxy policy, or startup state restoration
- scripts or helpers that interact with local machine networking or service startup

Do not claim runtime verification unless you actually performed it or clearly say why it was not run.

## Hook gate and CI parity

- `scripts/check.sh` is the single local gate and `.githooks/pre-push` is the local hook mirror of that gate.
- Keep local verification aligned with `.github/workflows/ci.yml` rather than creating a separate undocumented workflow.
- Do not bypass hooks unless the user explicitly asks for it.

## Worktree isolation

Prefer an isolated worktree when:
- the task is multi-step or risky
- the current working tree is already dirty
- multiple issues are being worked on in parallel
- the change affects runtime behavior, packaging, or other high-regression surfaces

Do not mix unrelated fixes into the same worktree or branch.

## Repo closeout

Before considering work complete:
- leave the intended branch/worktree in a reviewable state
- summarize what was verified
- call out any remaining risk or follow-up
- make sure temporary debug edits, generated artifacts, or local-only state files are not being proposed unintentionally
- update relevant docs when user-visible behavior, verification flow, or release behavior changed

## UI text and i18n

- Any change to user-facing UI text must update both `web/src/i18n/locales/zh-CN.json` and `web/src/i18n/locales/en.json` in the same change.
- Do not leave one locale ahead of the other.

## Risky actions and permission boundaries

Ask before taking actions that are destructive, hard to reverse, or affect shared state.

Always get explicit user confirmation before:
- deleting files, branches, worktrees, or generated state the user may still need
- changing release workflows, packaging outputs, installer behavior, or published artifact expectations
- changing `.githooks`, network-affecting helper scripts, service install scripts, or machine-level runtime configuration
- pushing, force-pushing, merging, tagging, or creating releases
- running commands that change live routing, DNS, TUN state, or local service installation outside the requested verification flow

Never use destructive git shortcuts such as `git reset --hard`, `git checkout --`, or `git clean -f` unless the user explicitly requested that exact action.

## MCP and external-tool governance

- Use repository and local tools first. Only use external web or MCP-backed tools when local files, hooks, scripts, or GitHub metadata are not enough.
- Treat MCP and web-reader style tools as potentially outbound. Do not send local configs, node data, state snapshots, secrets, or private runtime evidence to third-party tools unless the user explicitly asked for that workflow.
- For GitHub state related to this repository, prefer the repository's own files and `gh`/GitHub metadata before broader web search.

## Memory and persistence boundaries

- Do not treat runtime state, node-pool contents, subscription payloads, local IPs, tokens, or machine-specific evidence as durable memory.
- Only retain stable project rules, durable workflow decisions, and long-lived collaboration guidance.
- Temporary debugging notes belong in the current task, PR, or verification notes, not in durable memory.

## Output style for repository work

When reporting work in this repository:
- reference code and docs with `file:line` when practical
- state what changed, what was verified, and any remaining risk
- do not claim UI or runtime validation that was not actually performed
- keep summaries concise and operational rather than promotional

## Generated, local-state, and large-file governance

- Do not propose commits that accidentally include local runtime state, generated artifacts, binaries, or debug output unless the change explicitly requires them.
- Pay extra attention to files such as `control_plane_state.json`, `node_pool_state.json`, `dev-config.current-nodes.json`, `output/`, `backups/`, built binaries, and other local-only evidence.
- If a large file or generated artifact must change, explain why it belongs in the change and verify that the corresponding source-of-truth docs or workflow expectations stay accurate.
- Prefer updating source files, scripts, and docs over hand-editing generated outputs.

## Release and packaging sensitivity

- Changes to `.github/workflows/release*.yml`, `.github/workflows/docker.yml`, packaging assets, installer flows, or release-matrix documentation have a higher review bar than normal feature work.
- When release behavior changes, update the relevant README or release documentation in the same change and call out the artifact impact in verification notes.
- Do not silently change published artifact expectations, platform coverage, or naming conventions.

## Multi-agent and task coordination

- For multi-step work, keep tasks scoped so one branch/worktree tracks one coherent outcome.
- If parallel agent work is used, define ownership and avoid overlapping edits to the same flow without an explicit merge plan.
- Final closeout should name the verification that actually ran and any follow-up still needed.

## Rule maintenance

- Keep this file lean and repository-specific. Add rules here only when they are stable, repeatedly useful, and not already enforced better by code, hooks, or CI.
- If a rule becomes a repeated mechanical check, prefer moving enforcement into repository scripts, hooks, or CI rather than relying only on prose.
- If a rule is temporary or issue-specific, keep it in the issue, PR, or implementation plan instead of promoting it here.
