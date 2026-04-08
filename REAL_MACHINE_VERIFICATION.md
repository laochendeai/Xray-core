# Real Machine Verification

This checklist is for the phase-one stability control plane already implemented in:

- [subscription_manager.go](/home/leo-cy/Xray-core-laochendeai/app/webpanel/subscription_manager.go)
- [webpanel_control_plane.go](/home/leo-cy/Xray-core-laochendeai/app/webpanel/webpanel_control_plane.go)
- [tun_manager.go](/home/leo-cy/Xray-core-laochendeai/app/webpanel/tun_manager.go)
- [NodePool.vue](/home/leo-cy/Xray-core-laochendeai/web/src/views/NodePool.vue)

Use this on a Linux machine that will actually run the product.

## Goal

Prove the real-machine contract:

- transparent mode only uses active nodes
- pool becoming unsafe forces automatic clean fallback
- operator can see the reason in `NodePool`
- reboot returns to a clean machine state

## Prerequisites

- built `xray` binary for this repo
- config file with `webpanel` enabled
- TUN helper installed and sudoers configured
- WebPanel reachable from localhost
- enough active nodes in the pool to meet `minActiveNodes`

Recommended installation flow:

1. Install helper and sudoers:
   ```bash
   sudo ./scripts/install-webpanel-tun-sudoers.sh --config /path/to/config.json
   ```
2. Install the service so startup clean-state enforcement runs every boot:
   ```bash
   sudo ./scripts/install-webpanel-systemd-service.sh --config /path/to/config.json
   ```

## Transparent TUN Baseline (#38)

Use this when you need a repeatable correctness baseline after rebuilding or restarting the local service.

1. Rebuild and restart the local binary:
   ```bash
   make build
   ./xray run -c /path/to/config.json
   ```
2. Run the standard local validation gate:
   ```bash
   bash scripts/check.sh
   ```
3. Run the real-machine baseline flow:
   ```bash
   ./scripts/verify-webpanel-tun-baseline.sh --config /path/to/config.json
   ```

If the protected direct site is not derivable from `webpanel.tun.protectDomains`, pass one explicitly:

```bash
./scripts/verify-webpanel-tun-baseline.sh \
  --config /path/to/config.json \
  --protected-url https://hifly.cc/study/feature/index.html
```

What this flow verifies:

- `directEgress` before and after TUN enablement
- same-user public IP changes once TUN is running, while the independent direct egress record stays stable
- runtime DNS/routing markers in `runtime/tun/config.json`
- one protected direct URL under transparent mode
- one forced proxy-path probe via an active outbound
- one HTTP/3 probe when local `curl` supports `--http3`; otherwise it records the UDP/443 and proxy-path prerequisites instead

Artifacts:

- `preflight.txt`
- `baseline-summary.txt`
- `baseline-summary.json`
- per-stage snapshot directories with:
  - `tun-status.json`
  - `tun-settings.json`
  - `runtime-config.json`
  - `route-probe-cache.json`
  - `egress-probe-cache.json`
  - route / rule / `resolvectl` captures

How to read the IP fields reported by `baseline-summary.txt` and `baseline-summary.json`:

- `direct_egress_ip` and `ipSemantics.directEgressIp*` are the machine's own direct public egress from `directEgress`. This should stay stable before and with TUN unless the real WAN egress actually changed.
- `same_user_path_public_ip` and `sameUserPath*` come from the same user path calling an IP-echo endpoint. With TUN enabled this can become the proxy exit IP, which does not mean the machine's direct public egress changed.
- `target_remote_ip` and `protectedTargetProbe.targetRemoteIp` are the remote address that `curl` connected to for the protected URL. This is usually the site origin or CDN edge IP, not your own public egress.

The fallback rehearsal remains a separate, extended verification and is not part of this minimum baseline.

## Verification Scripts

### 1. Read-Only Preflight

This checks config, helper/runtime paths, login, TUN status, node-pool summary, and current machine state.

```bash
./scripts/verify-webpanel-control-plane.sh preflight --config /path/to/config.json
```

Expected baseline before the destructive drill:

- `helperExists=true`
- `elevationReady=true` when `useSudo=true`
- `allowRemote=false`
- `machineState=clean`
- `running=false`

### 2. Evidence Snapshot

This captures:

- `/api/v1/tun/status`
- `/api/v1/node-pool`
- `control_plane_state.json`
- `node_pool_state.json`
- `runtime/tun/config.json`
- `runtime/tun/xray-tun.log`
- `ip link` and `ip route`

```bash
./scripts/verify-webpanel-control-plane.sh snapshot --config /path/to/config.json --output-dir /tmp/webpanel-proof
```

### 3. Fallback Rehearsal

This is the real-machine critical-path drill. It:

1. starts transparent mode if needed
2. quarantines enough active nodes to drop below `minActiveNodes`
3. waits for automatic fallback
4. restores quarantined nodes back to active unless `--keep-quarantine` is set

Dry-run first:

```bash
./scripts/rehearse-webpanel-fallback.sh --config /path/to/config.json
```

Execute for real:

```bash
./scripts/rehearse-webpanel-fallback.sh --config /path/to/config.json --apply --output-dir /tmp/webpanel-proof
```

Pass condition:

- terminal `machineState=clean`
- terminal `running=false`
- terminal `lastStateReason=automatic_fallback_min_active_not_met`

Fail condition:

- terminal `machineState=degraded`
- helper stop failure
- timeout waiting for fallback

### 4. Single-Node Forced Probe

This temporarily pins balancer `auto` to one outbound tag, sends a probe through the local
SOCKS inbound, then restores the previous balancer target.

Use it to answer whether a specific `pool_*` node is actually usable on the machine:

```bash
./scripts/probe-webpanel-outbound.sh --target pool_db7e14360e59
```

Known-good baseline:

```bash
./scripts/probe-webpanel-outbound.sh --target proxy-01
```

Interpretation:

- `curl exit=0` means the forced outbound can relay the probe URL end to end.
- `curl exit=28` usually means timeout to the node or upstream path.
- `curl exit=35` usually means the TLS/WebSocket/REALITY handshake failed after the route was forced.

Operational note:

- this does not enable TUN
- this only affects traffic that is explicitly sent to the local proxy while the balancer override is active

## Manual UI Checks

Run these while the drill is happening.

### NodePool

- top strip shows `clean / proxied / degraded / recovering`
- active count and minimum required count are visible
- last fallback reason is visible without opening logs
- quarantined nodes move into the quarantine group
- recent events show the trust-affecting transitions

### Settings > TUN

- helper availability is visible
- privilege readiness is visible
- runtime config path is visible
- raw helper output is visible on failures
- no primary transparent-mode controls are duplicated here

## Reboot Check

After the fallback drill passes, reboot the machine and run:

```bash
./scripts/verify-webpanel-control-plane.sh post-reboot --config /path/to/config.json --output-dir /tmp/webpanel-proof
```

Pass condition:

- `machineState=clean`
- `running=false`
- `lastStateReason=startup_default_clean`
- no TCP or UDP/53 capture policy rules remain
- the capture route table no longer points at the TUN interface
- the TUN interface itself is absent

## Evidence To Keep

- the snapshot directory created by the scripts
- screenshots of `NodePool` before start, during proxied state, and after fallback
- `runtime/tun/xray-tun.log`
- the final `control_plane_state.json`

## Protocol Support Snapshot

Snapshot time: `2026-04-09 01:45 +08:00`

This repo's top-level GitHub `README` is not a protocol support matrix. It does not
spell out whether subscription share links like `tuic://`, `hysteria2://`, or
`ss://` with 2022 methods are supported. For this branch, support must be judged
from the code and from real-machine probes.

Code-level support in this worktree:

- `anytls` client config exists in
  [infra/conf/anytls.go](/home/leo-cy/Xray-core-laochendeai/infra/conf/anytls.go),
  and the minimal outbound TCP relay path exists in
  [outbound.go](/home/leo-cy/Xray-core-laochendeai/proxy/anytls/outbound.go) and
  [client.go](/home/leo-cy/Xray-core-laochendeai/proxy/anytls/client.go).
- `hysteria` client/server config exists in
  [infra/conf/hysteria.go](/home/leo-cy/Xray-core-laochendeai/infra/conf/hysteria.go)
  and the client path explicitly requires `version == 2`.
- `shadowsocks_2022` client/server config exists in
  [infra/conf/shadowsocks.go](/home/leo-cy/Xray-core-laochendeai/infra/conf/shadowsocks.go)
  and the underlying core package exists under
  [proxy/shadowsocks_2022](/home/leo-cy/Xray-core-laochendeai/proxy/shadowsocks_2022).
- webpanel import/build support for `anytls://`, `hysteria2://`, and `ss://` 2022 methods exists in
  [share_link_parser.go](/home/leo-cy/Xray-core-laochendeai/app/webpanel/share_link_parser.go),
  [share_link.go](/home/leo-cy/Xray-core-laochendeai/app/webpanel/share_link.go), and
  [outbound_config.go](/home/leo-cy/Xray-core-laochendeai/app/webpanel/outbound_config.go).
- `tuic` does not appear anywhere in this repo tree. A repo-wide search for `tuic`
  returned no matches, so this branch should be treated as `TUIC unsupported`.

Machine-level evidence on this Linux host:

- local verification for the new AnyTLS slice passed on this host:
  `go test ./proxy/anytls ./app/webpanel ./infra/conf -run AnyTLS`
  returned `ok`, including a TLS-backed AnyTLS relay fixture in
  [outbound_test.go](/home/leo-cy/Xray-core-laochendeai/proxy/anytls/outbound_test.go).
- `GET /api/v1/tun/status` returned `running=false` and `machineState=clean`.
  Starting the webpanel service did not globally hijack other application traffic.
- `GET /api/v1/node-pool` returned `activeCount=42`, `stagingCount=159`,
  `quarantineCount=24`, `candidateCount=7`, and `healthy=true`.
- `ss-2022` is verified end to end on this machine:
  `./scripts/probe-webpanel-outbound.sh --target pool_8367edd140af`
  returned `HTTP/2 204` with `curl exit: 0`.
- one `hysteria2` sample still failed end to end:
  `./scripts/probe-webpanel-outbound.sh --target pool_3dee073a0dbb`
  returned `curl exit: 28` after timeout.
- one `ss + v2ray-plugin` sample still failed end to end:
  `./scripts/probe-webpanel-outbound.sh --target pool_6ce208a0e924`
  returned `curl exit: 35` with `SSL_ERROR_SYSCALL`.

Current interpretation:

- `anytls` is now supported for the first outbound-only slice:
  config build, WebPanel import/build/decode, and basic TCP relay are all wired in.
- `anytls` is not yet a full protocol parity implementation:
  no inbound path, no UDP, no session pool, and no padding-scheme updates yet.
- `ss-2022` support is real, not just parser-level.
- `hysteria2` import/build wiring is present, but the current supplied nodes are not
  yet validated as usable on this machine.
- `ss + v2ray-plugin` import/build wiring is present, but the current supplied
  plugin nodes are not yet validated as usable on this machine.

### AnyTLS Local Smoke (#53)

Use this flow when you want a teammate-runnable local proof that the first AnyTLS
slice works outside of unit tests.

1. Start a local reference AnyTLS server:
   ```bash
   go run github.com/anytls/anytls-go/cmd/server@v0.0.12 \
     -l 127.0.0.1:18443 \
     -p test-pass
   ```
2. Import this share link through the WebPanel subscription/manual import flow:
   ```text
   anytls://test-pass@127.0.0.1:18443/?sni=127.0.0.1&insecure=1#anytls-local
   ```
3. Verify the generated outbound keeps AnyTLS settings at the protocol layer and TLS
   at the stream layer. The effective shape should be equivalent to:
   ```json
   {
     "protocol": "anytls",
     "settings": {
       "address": "127.0.0.1",
       "port": 18443,
       "password": "test-pass"
     },
     "streamSettings": {
       "network": "tcp",
       "security": "tls",
       "tlsSettings": {
         "serverName": "127.0.0.1",
         "allowInsecure": true
       }
     }
   }
   ```
4. Promote that imported node to active, then force a probe through it:
   ```bash
   ./scripts/probe-webpanel-outbound.sh --target pool_<node-id>
   ```

Pass condition:

- the probe returns `curl exit: 0`
- the local AnyTLS server logs an accepted TCP connection
- WebPanel shows the imported node as `protocol=anytls`

Expected current limits of this smoke:

- only TCP is covered
- this does not prove UDP or transparent-TUN specific behavior
- this does not validate session reuse or padding-scheme updates

### Hysteria2 Root-Cause Notes

The current `hysteria2` sample file only uses these query params:

- `insecure=1`
- `sni=<host>`

No sample link currently uses:

- `obfs`
- `obfs-password`
- `pinSHA256`

That matters because the current failure cannot be explained by an unsupported URI
field in the supplied sample set. The parser already covers the exact fields present
in `/home/leo-cy/share/hysteria2.txt`.

Additional runtime evidence from the live Xray process:

- during a forced probe of `pool_3dee073a0dbb`, the core logged:
  `proxy/hysteria: failed to find an available destination > transport/internet/hysteria: RoundTrip err > timeout: no recent network activity`
- the node-pool counters for all 9 imported `hysteria2` nodes converged to:
  `failedPings == totalPings`, `avgDelayMs == 0`, and very high `consecutiveFails`

Working hypothesis as of `2026-03-26`:

- the current `hysteria2` implementation path is wired in correctly enough to create
  runtime outbounds and attempt the QUIC/HTTP3 auth round trip
- the supplied `iptk123.com` nodes are currently silent or unusable from this machine
  rather than being blocked by a parser mismatch in the imported URI fields
- webpanel support has been extended to cover additional official `hy2` sharing fields
  needed by other providers:
  `pinSHA256`, `obfs=salamander`, and `obfs-password`

Follow-up work if `hysteria2` remains important:

- cross-check one of the same nodes with an external client that is independent of this
  repo
- extend webpanel `hysteria2` URI support for optional official fields like
  `pinSHA256` and `obfs=salamander` even though the current sample set does not use them

## Notes

- `rehearse-webpanel-fallback.sh` is intentionally destructive when `--apply` is used. Run it only on the intended validation machine.
- The rehearsal script tries to promote quarantined nodes back to active on exit.
- Missing frontend unit tests are still a known repo constraint; do not confuse that with real-machine acceptance.
