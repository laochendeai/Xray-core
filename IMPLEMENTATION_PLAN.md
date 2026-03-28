# Stability Control Plane Implementation Plan

## Source Of Truth

- Approved product/design document:
  - `/home/leo-cy/.gstack/projects/laochendeai-Xray-core/leo-cy-main-design-20260326-003026.md`
- Existing implementation surface:
  - `app/webpanel`
  - `web/src`

This file turns the approved design into a team execution plan. It does not redefine scope.

## Scope Freeze

Phase one is the stability-first local control plane:

- Validate nodes before they can serve transparent traffic.
- Only the active pool may carry traffic.
- Automatically fall back to a clean machine state when the active pool becomes unsafe.
- Keep the primary operator workflow in `NodePool`.
- Keep `Settings > TUN` as diagnostics only.
- Reboot/startup must converge to a clean default.

Still out of scope:

- full bandwidth scoring
- strong cleanliness proof beyond `trusted / suspicious / unknown`
- process-routing UX
- cross-platform helper/runtime support
- monetization work

## Current Local Status

The current worktree already contains the first implementation pass for phase one:

- typed node lifecycle in `app/webpanel/subscription_manager.go`
- machine-state store in `app/webpanel/control_plane_state.go`
- control-plane orchestration in `app/webpanel/webpanel_control_plane.go`
- pool-aware runtime generation in `app/webpanel/tun_manager.go`
- NodePool cockpit in `web/src/views/NodePool.vue`
- Settings diagnostics split in `web/src/views/Settings.vue`
- backend tests for state migration and runtime config generation

Open work before merge:

- complete coordinator coverage for automatic fallback and startup-clean behavior
- run real helper dogfood against the actual machine/runtime
- run browser-level operator drill
- verify reboot returns to clean state on a real machine
- decide how to handle blocked frontend unit-test dependency installation

## Team Topology

One person may hold multiple roles, but ownership should stay explicit.

### Track 1: Control Plane Backend

Owner: backend/control-plane engineer

Write scope:

- `app/webpanel/subscription_manager.go`
- `app/webpanel/control_plane_state.go`
- `app/webpanel/webpanel.go`
- `app/webpanel/webpanel_control_plane.go`
- `app/webpanel/handler_node_pool.go`
- `app/webpanel/handler_tun.go`
- `app/webpanel/*_test.go`

Deliverables:

- typed lifecycle transitions remain the only source of truth
- pool-health callback drives machine-state fallback
- structured reasons/events are persisted, not only logged
- startup enforces clean state

Exit criteria:

- `go test ./app/webpanel/...` passes
- automatic fallback path is covered
- startup-clean path is covered
- illegal/manual transitions return explicit errors

### Track 2: Runtime And Helper Validation

Owner: runtime/ops engineer

Write scope:

- `app/webpanel/tun_manager.go`
- local helper script and runtime config behavior
- local machine verification notes/evidence

Deliverables:

- active pool injection works with real traffic
- fallback to clean succeeds with the real helper
- startup cleanup succeeds after reboot on the target machine class

Exit criteria:

- transparent mode starts only with a healthy pool
- pool drop below minimum triggers automatic clean fallback
- after reboot, `GET /api/v1/tun/status` reports `machineState=clean`

### Track 3: Operator Cockpit Frontend

Owner: frontend engineer

Write scope:

- `web/src/api/client.ts`
- `web/src/api/types.ts`
- `web/src/views/NodePool.vue`
- `web/src/views/Settings.vue`
- `web/src/i18n/locales/en.json`
- `web/src/i18n/locales/zh-CN.json`
- `web/dist`
- `app/webpanel/dist`

Deliverables:

- NodePool is the only primary control surface
- machine-state strip answers clean/proxied/degraded first
- grouped node sections expose status, cleanliness, reason, and actions
- Settings keeps only helper/runtime diagnostics

Exit criteria:

- `npm run build` passes
- NodePool works on desktop and mobile widths
- operator drill can be completed without visiting Settings for primary actions

### Track 4: QA, Release, And Rollback

Owner: tech lead or release owner

Responsibilities:

- keep landable PR boundaries intact
- collect manual verification evidence
- decide go/no-go based on real helper behavior, not compile success alone
- own rollback execution if runtime behavior regresses

## Landable Milestones

### M1: State Model Hardening

Status: mostly implemented locally

- node lifecycle and reason codes are frozen
- persistence format is stabilized
- backend tests cover migration, transitions, and pool summary behavior

Remaining:

- merge coordinator tests for fallback/startup flows

### M2: Machine-State Orchestration

Status: implemented locally, needs real-machine proof

- `WebPanel` coordinates pool health and TUN state
- `control_plane_state.json` persists machine summary
- TUN enablement is blocked when active pool is below minimum

Remaining:

- validate automatic clean fallback with the real helper
- validate degraded behavior when helper stop fails

### M3: Operator Cockpit

Status: implemented locally

- NodePool is the primary workspace
- Settings is diagnostics-only
- i18n strings exist for lifecycle and machine-state codes

Remaining:

- browser-level operator drill
- regression check on mobile/tablet layouts

### M4: Ship Candidate

Blocked on:

- real helper dogfood
- reboot clean-state verification
- decision on frontend test infra blockage

## PR And Rollback Boundaries

If this branch is split before merge, keep these boundaries.

### PR A: Lifecycle And Persistence

Files:

- `app/webpanel/subscription_manager.go`
- `app/webpanel/handler_node_pool.go`
- `web/src/api/client.ts`
- `web/src/api/types.ts`
- locale changes for status/reason codes

Rollback effect:

- revert new lifecycle semantics back to the old node-pool behavior
- leaves TUN diagnostics and helper behavior untouched

### PR B: Machine-State Coordination

Files:

- `app/webpanel/control_plane_state.go`
- `app/webpanel/webpanel.go`
- `app/webpanel/webpanel_control_plane.go`
- `app/webpanel/handler_tun.go`

Rollback effect:

- removes automatic fallback orchestration
- preserves the node lifecycle model

### PR C: Runtime Injection

Files:

- `app/webpanel/tun_manager.go`

Rollback effect:

- removes active-pool balancer/runtime injection
- highest-priority rollback if real traffic behavior is unstable

### PR D: Operator Cockpit UI

Files:

- `web/src/views/NodePool.vue`
- `web/src/views/Settings.vue`
- `web/dist`
- `app/webpanel/dist`

Rollback effect:

- restores the old operator workflow without touching backend state files

## Definition Of Done

Do not call phase one done until all of the following are true:

- mixed-quality subscription import leaves unqualified nodes out of live traffic
- active pool can be built manually and automatically
- transparent mode cannot start when the pool is below minimum
- active pool degradation forces automatic clean fallback
- the UI shows the fallback reason without opening logs
- a reboot returns the machine to clean state
- operator can verify node cleanliness status directly in NodePool

## Mandatory Verification Drill

Run in this order on a real machine:

1. Import a mixed subscription.
2. Wait for validation; confirm only active nodes are eligible for traffic.
3. Enable transparent mode from `NodePool`.
4. Force active pool below minimum.
5. Confirm the system returns to clean automatically.
6. Reboot the machine.
7. Confirm startup state is clean and reason is visible.

## Known Constraint

Frontend unit tests are still blocked by package registry access failures when installing `@vue/test-utils` and related dependencies. Do not treat missing frontend unit tests as acceptable forever, but do not destabilize the repo by forcing a broken package-install path into this branch.
