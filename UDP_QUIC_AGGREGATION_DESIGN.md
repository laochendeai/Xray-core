# UDP/QUIC Aggregation Architecture And Rollout Design

## Source Of Truth

- Parent issue: `#35`
- This design issue: `#39`
- Current transparent-TUN baseline:
  - Runtime generation and balancer insertion in `app/webpanel/tun_manager.go`
  - Stable-mode diagnostics in `app/webpanel/webpanel_control_plane.go`
  - Root helper install and exact-argument sudoers in `scripts/install-webpanel-tun-sudoers.sh`

This document defines the staged architecture for experimental UDP/QUIC aggregation. It does not replace stable mode and does not change DNS-affinity correctness scope from `#34`.

## Problem Statement

The current transparent TUN path can proxy QUIC (`UDP/443`) but still routes each flow through the existing single-path `node-pool-active` selection behavior. That means QUIC compatibility is present, but multi-path aggregation is not.

We need an explicit architecture that allows experimentation behind a feature flag, with clean rollback to the current path.

## Goals

- Add an experimental path for UDP/QUIC aggregation that can use multiple candidate paths for one logical flow.
- Keep the current transparent stable path as default behavior.
- Keep rollback to single-path mode immediate and deterministic.
- Add enough observability to decide whether aggregation improves real outcomes.

## Non-Goals

- Default-on production rollout in this phase.
- TCP aggregation.
- Replacing or weakening stable-mode behavior.
- Reworking DNS-affinity correctness (handled by `#34`).
- Building a full remote relay fleet.

## Current Baseline Constraints

Current runtime generation writes:

- `tun-in` inbound with QUIC sniffing.
- DNS split flow and protected direct-domain rules.
- Catch-all TUN routing to `balancerTag: "node-pool-active"`.

Current control plane and helper flow assumes:

- `start/stop/toggle` operations are executed by the root-owned helper.
- Safety and fallback behavior are keyed to the current stable-mode state machine.

The aggregation path must fit these constraints and remain switchable off at runtime.

## High-Level Architecture

### Control Plane Layer

Add an aggregation mode switch in WebPanel TUN settings (disabled by default). Control plane chooses one of two data paths:

- `stable_single_path`: existing path (default).
- `experimental_udp_quic_aggregation`: new path, feature-gated.

Control-plane responsibilities:

- Gate entry by feature flag and readiness checks.
- Surface mode, health summary, and rollback state.
- Force fallback to stable mode on aggregation health failures.

### Local Data Plane Modules

When aggregation mode is enabled, introduce the following logical modules in the local side:

1. `FlowClassifier`
- Detects candidate UDP/QUIC traffic from TUN.
- Builds flow keys (5-tuple plus protocol hints).
- Sends non-candidate traffic to stable single-path behavior.

2. `SessionMap`
- Maintains per-flow session state.
- Stores selected candidate paths, sequence context, timers, and failure counters.
- Handles session lifecycle (create/update/expire).

3. `PathQualityCollector`
- Maintains per-path RTT/jitter/loss/reorder estimates.
- Consumes active probes and passive flow telemetry.
- Produces scheduler-ready quality snapshots.

4. `PathScheduler`
- Chooses one or more paths per session based on policy.
- Supports at least:
  - `single_best` (compat mode)
  - `redundant_2` (duplicate key packets)
  - `weighted_split` (weighted distribution)
- Applies hard safety rules (path quarantine, max skew, min viable paths).

5. `PacketMux`
- Encapsulates local session packets toward remote relay channels.
- Adds session id, packet sequence, and optional redundancy metadata.
- Handles retransmit/redundancy policy when scheduler asks for it.

6. `AggregationFallbackGuard`
- Watches error budget and quality thresholds.
- Triggers rollback to stable mode if guardrails are violated.

### Remote Relay/Collector Module

Provide one minimal remote endpoint for experiments:

- `RelayIngress`: receives encapsulated packets from multiple local paths.
- `SessionAssembler`: reorders and deduplicates by session id + sequence.
- `RelayEgress`: emits restored UDP flow to destination and relays responses.
- `RelayTelemetry`: returns path/session stats to local side.

This remote component is prototype-only in this phase and does not imply production fleet architecture.

## Module Boundaries

### Local Ownership Boundary

- Runtime and control-plane integration.
- Session mapping, scheduler, local telemetry.
- Feature-flag behavior and rollback.

### Remote Ownership Boundary

- Relay protocol endpoint.
- Session reassembly and egress behavior.
- Relay-side metrics and benchmark harness integration.

### Shared Contract Boundary

- Encapsulation header format.
- Session id and packet sequence semantics.
- Metric schema and periodic report format.
- Error and backpressure signaling.

## Feature Flagging And Configuration

Add an explicit experimental config surface under `webpanel.tun`:

```json
{
  "webpanel": {
    "tun": {
      "aggregation": {
        "enabled": false,
        "mode": "single_best",
        "maxPathsPerSession": 2,
        "schedulerPolicy": "weighted_split",
        "relayEndpoint": "",
        "health": {
          "maxSessionLossPct": 5,
          "maxPathJitterMs": 120,
          "rollbackOnConsecutiveFailures": 3
        }
      }
    }
  }
}
```

Rules:

- `enabled=false` must preserve exact stable-mode behavior.
- Missing or invalid aggregation config must be treated as disabled.
- Starting aggregation without relay endpoint readiness must fail closed to stable mode.

## Rollback Strategy

Rollback is first-class and immediate:

1. Hard fallback trigger
- Aggregation guard failures exceed thresholds.
- Relay unavailable.
- Session-map corruption/overrun checks fail.

2. Fallback action
- Stop new aggregation sessions.
- Drain or terminate active aggregation sessions by policy.
- Route all TUN traffic back to existing `node-pool-active` path.

3. Control-plane state
- Emit explicit fallback reason.
- Surface mode transition in status.
- Keep machine recoverable via existing clean restore flow.

No deployment stage in this document allows removing stable-mode fallback.

## Observability Plan

### Required Metrics

Local side:

- `agg_sessions_active`
- `agg_sessions_created_total`
- `agg_sessions_fallback_total`
- `agg_path_rtt_ms{path}`
- `agg_path_jitter_ms{path}`
- `agg_path_loss_pct{path}`
- `agg_path_reorder_pct{path}`
- `agg_scheduler_decision_total{policy}`

Relay side:

- `relay_sessions_active`
- `relay_dedupe_drops_total`
- `relay_reorder_buffer_depth`
- `relay_egress_errors_total`

User-impact metrics:

- startup-to-first-byte estimate for sample QUIC flows
- stall/rebuffer event count proxy
- throughput stability (variance and p95 swings)

### Required Status/Diagnostics Fields

- aggregation mode (`disabled`, `experimental`)
- effective scheduler policy
- active path count per representative session
- last fallback reason and timestamp

## Staged Rollout Plan

### Stage 0: Design Freeze

- Finish and approve this document.
- Freeze local/remote contract for prototype.

Exit criteria:

- Architecture review complete.
- Open questions reduced to explicitly tracked items.

### Stage 1: Feature-Gated Scaffolding

- Add config schema and runtime gating only.
- No data-plane aggregation logic yet.

Exit criteria:

- Disabled mode is behavior-identical to current stable path.
- Enabled mode can be rejected safely without side effects.

### Stage 2: Local Prototype

- Implement `FlowClassifier`, `SessionMap`, `PathQualityCollector`, `PathScheduler`, `PacketMux` prototype.

Exit criteria:

- Unit tests for scheduler/session lifecycle.
- No regression in stable mode when feature is off.

### Stage 3: Remote Prototype + Integration

- Implement minimal relay ingress/assembler/egress path.
- Integrate local-to-remote contract.

Exit criteria:

- End-to-end QUIC test flow through experimental path.
- Fallback path still deterministic.

### Stage 4: Controlled Benchmarking

- Compare stable single-path vs experimental aggregation in controlled conditions.

Exit criteria:

- Report includes latency/stall/throughput stability deltas.
- Known failure envelopes documented.

### Stage 5: Limited Dogfood

- Keep feature off by default.
- Enable on selected machines only.

Exit criteria:

- No unacceptable regression trend.
- Rollback drill validated.

## Testing Strategy

- Unit tests:
  - session lifecycle
  - scheduler decisions
  - guard-triggered fallback
  - packet dedupe/reorder behavior (local + relay)

- Integration tests:
  - feature off parity with stable path
  - feature on, relay healthy
  - relay down mid-session fallback

- Real-machine checks:
  - representative QUIC/video flows
  - controlled packet-loss and jitter scenarios
  - rollback verification

## Security And Trust Boundaries

- Experimental relay endpoint must be explicit and authenticated in later stages.
- Aggregation headers must not leak sensitive local metadata beyond what is needed for reassembly.
- Feature must fail closed to stable mode on auth/contract errors.
- Existing root helper privileges remain scoped to TUN mode switching and are not expanded for aggregation logic in this phase.

## Open Questions

1. Should phase-one relay transport run over existing outbound protocols or a dedicated relay channel type?
2. What is the minimum acceptable per-session overhead budget before aggregation is considered net-negative?
3. Which redundancy policy should be the initial default for experiments: `redundant_2` or `weighted_split`?
4. Do we need explicit QUIC connection-id affinity constraints in scheduler logic for better reorder control?
5. How should control-plane UX expose experimental mode warnings without confusing stable-mode users?

## Deliverables Mapped To Sub-Issues

- `#39`: this architecture and staged rollout design.
- `#40`: feature flag/config scaffolding and rollback hooks.
- `#41`: local scheduler/session prototype.
- `#42`: remote relay prototype and benchmark harness.
