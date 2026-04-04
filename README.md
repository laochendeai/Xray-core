# Xray-core (laochendeai fork)

This repository is a maintained fork of [XTLS/Xray-core](https://github.com/XTLS/Xray-core) with an embedded WebPanel focused on local operation and transparent-proxy management. It keeps the upstream Xray-core runtime, and adds a single-binary control plane for subscriptions, node-pool lifecycle management, TUN orchestration, DNS split-policy, and update discovery.

The upstream project remains [Project X](https://github.com/XTLS). Upstream README and ecosystem links are still useful reference material, but the fork-specific behavior in this repository is documented here first.

## Fork Highlights

- Embedded WebPanel served from the Xray binary through `embed.FS`
- Subscription import from remote URL, manual paste, and local file upload
- Node pool lifecycle with candidate, validation, active, quarantine, and removed states
- Transparent proxy / TUN controls with direct-vs-proxy policy, remote DNS list, and clean restore
- Dashboard update discovery against upstream releases

## Documentation Map

- [`web/README.md`](web/README.md): WebPanel pages, APIs, transparent-mode behavior, and development notes
- [`README.md`](README.md): fork overview, build entry points, and repository workflow
- [`scripts/check.sh`](scripts/check.sh): single local verification entry point before push

## Upstream Reference

[Project X](https://github.com/XTLS) originates from XTLS protocol, providing a set of network tools such as [Xray-core](https://github.com/XTLS/Xray-core) and [REALITY](https://github.com/XTLS/REALITY).

The upstream [README](https://github.com/XTLS/Xray-core#readme) is open, so feel free to submit your project [here](https://github.com/XTLS/Xray-core/pulls).

## Sponsors

[![Remnawave](https://github.com/user-attachments/assets/a22d34ae-01ee-441c-843a-85356748ed1e)](https://docs.rw)

[![Happ](https://github.com/user-attachments/assets/14055dab-e8bb-48bd-89e8-962709e4098e)](https://happ.su)

[**Sponsor Xray-core**](https://github.com/XTLS/Xray-core/issues/3668)

## Donation & NFTs

### [Collect a Project X NFT to support the development of Project X!](https://opensea.io/item/ethereum/0x5ee362866001613093361eb8569d59c4141b76d1/1)

[<img alt="Project X NFT" width="150px" src="https://raw2.seadn.io/ethereum/0x5ee362866001613093361eb8569d59c4141b76d1/7fa9ce900fb39b44226348db330e32/8b7fa9ce900fb39b44226348db330e32.svg" />](https://opensea.io/item/ethereum/0x5ee362866001613093361eb8569d59c4141b76d1/1)

- **TRX(Tron)/USDT/USDC: `TNrDh5VSfwd4RPrwsohr6poyNTfFefNYan`**
- **TON: `UQApeV-u2gm43aC1uP76xAC1m6vCylstaN1gpfBmre_5IyTH`**
- **BTC: `1JpqcziZZuqv3QQJhZGNGBVdCBrGgkL6cT`**
- **XMR: `4ABHQZ3yJZkBnLoqiKvb3f8eqUnX4iMPb6wdant5ZLGQELctcerceSGEfJnoCk6nnyRZm73wrwSgvZ2WmjYLng6R7sR67nq`**
- **SOL/USDT/USDC: `3x5NuXHzB5APG6vRinPZcsUv5ukWUY1tBGRSJiEJWtZa`**
- **ETH/USDT/USDC: `0xDc3Fe44F0f25D13CACb1C4896CD0D321df3146Ee`**
- **Project X NFT: https://opensea.io/item/ethereum/0x5ee362866001613093361eb8569d59c4141b76d1/1**
- **VLESS NFT: https://opensea.io/collection/vless**
- **REALITY NFT: https://opensea.io/item/ethereum/0x5ee362866001613093361eb8569d59c4141b76d1/2**
- **Related links: [VLESS Post-Quantum Encryption](https://github.com/XTLS/Xray-core/pull/5067), [XHTTP: Beyond REALITY](https://github.com/XTLS/Xray-core/discussions/4113), [Announcement of NFTs by Project X](https://github.com/XTLS/Xray-core/discussions/3633)**

## License

[Mozilla Public License Version 2.0](https://github.com/XTLS/Xray-core/blob/main/LICENSE)

## Documentation

[Project X Official Website](https://xtls.github.io)

## Telegram

[Project X](https://t.me/projectXray)

[Project X Channel](https://t.me/projectXtls)

[Project VLESS](https://t.me/projectVless) (Русский)

[Project XHTTP](https://t.me/projectXhttp) (Persian)

## Installation

- Linux Script
  - [XTLS/Xray-install](https://github.com/XTLS/Xray-install) (**Official**)
  - [tempest](https://github.com/team-cloudchaser/tempest) (supports [`systemd`](https://systemd.io) and [OpenRC](https://github.com/OpenRC/openrc); Linux-only)
- Docker
  - [ghcr.io/xtls/xray-core](https://ghcr.io/xtls/xray-core) (**Official**)
  - [teddysun/xray](https://hub.docker.com/r/teddysun/xray)
  - [wulabing/xray_docker](https://github.com/wulabing/xray_docker)
- Web Panel
  - [Remnawave](https://github.com/remnawave/panel)
  - [3X-UI](https://github.com/MHSanaei/3x-ui)
  - [PasarGuard](https://github.com/PasarGuard/panel)
  - [Xray-UI](https://github.com/qist/xray-ui)
  - [X-Panel](https://github.com/xeefei/X-Panel)
  - [Marzban](https://github.com/Gozargah/Marzban)
  - [Hiddify](https://github.com/hiddify/Hiddify-Manager)
  - [TX-UI](https://github.com/AghayeCoder/tx-ui)
- One Click
  - [Xray-REALITY](https://github.com/zxcvos/Xray-script), [xray-reality](https://github.com/sajjaddg/xray-reality), [reality-ezpz](https://github.com/aleskxyz/reality-ezpz)
  - [Xray_bash_onekey](https://github.com/hello-yunshu/Xray_bash_onekey), [XTool](https://github.com/LordPenguin666/XTool), [VPainLess](https://github.com/vpainless/vpainless)
  - [v2ray-agent](https://github.com/mack-a/v2ray-agent), [Xray_onekey](https://github.com/wulabing/Xray_onekey), [ProxySU](https://github.com/proxysu/ProxySU)
- Magisk
  - [NetProxy-Magisk](https://github.com/Fanju6/NetProxy-Magisk)
  - [Xray4Magisk](https://github.com/Asterisk4Magisk/Xray4Magisk)
  - [Xray_For_Magisk](https://github.com/E7KMbb/Xray_For_Magisk)
- Homebrew
  - `brew install xray`

## Usage

- Example
  - [VLESS-XTLS-uTLS-REALITY](https://github.com/XTLS/REALITY#readme)
  - [VLESS-TCP-XTLS-Vision](https://github.com/XTLS/Xray-examples/tree/main/VLESS-TCP-XTLS-Vision)
  - [All-in-One-fallbacks-Nginx](https://github.com/XTLS/Xray-examples/tree/main/All-in-One-fallbacks-Nginx)
- Xray-examples
  - [XTLS/Xray-examples](https://github.com/XTLS/Xray-examples)
  - [chika0801/Xray-examples](https://github.com/chika0801/Xray-examples)
  - [lxhao61/integrated-examples](https://github.com/lxhao61/integrated-examples)
- Tutorial
  - [XTLS Vision](https://github.com/chika0801/Xray-install)
  - [REALITY (English)](https://cscot.pages.dev/2023/03/02/Xray-REALITY-tutorial/)
  - [XTLS-Iran-Reality (English)](https://github.com/SasukeFreestyle/XTLS-Iran-Reality)
  - [Xray REALITY with 'steal oneself' (English)](https://computerscot.github.io/vless-xtls-utls-reality-steal-oneself.html)
  - [Xray with WireGuard inbound (English)](https://g800.pages.dev/wireguard)

## GUI Clients

- OpenWrt
  - [PassWall](https://github.com/Openwrt-Passwall/openwrt-passwall), [PassWall 2](https://github.com/Openwrt-Passwall/openwrt-passwall2)
  - [ShadowSocksR Plus+](https://github.com/fw876/helloworld)
  - [luci-app-xray](https://github.com/yichya/luci-app-xray) ([openwrt-xray](https://github.com/yichya/openwrt-xray))
- Asuswrt-Merlin
  - [XRAYUI](https://github.com/DanielLavrushin/asuswrt-merlin-xrayui)
  - [fancyss](https://github.com/hq450/fancyss)
- Windows
  - [v2rayN](https://github.com/2dust/v2rayN)
  - [Furious](https://github.com/LorenEteval/Furious)
  - [Invisible Man - Xray](https://github.com/InvisibleManVPN/InvisibleMan-XRayClient)
  - [AnyPortal](https://github.com/AnyPortal/AnyPortal)
  - [GenyConnect](https://github.com/genyleap/GenyConnect)
- Android
  - [v2rayNG](https://github.com/2dust/v2rayNG)
  - [X-flutter](https://github.com/XTLS/X-flutter)
  - [SaeedDev94/Xray](https://github.com/SaeedDev94/Xray)
  - [SimpleXray](https://github.com/lhear/SimpleXray)
  - [XrayFA](https://github.com/Q7DF1/XrayFA)
  - [AnyPortal](https://github.com/AnyPortal/AnyPortal)
  - [NetProxy-Magisk](https://github.com/Fanju6/NetProxy-Magisk)
- iOS & macOS arm64 & tvOS
  - [Happ](https://apps.apple.com/app/happ-proxy-utility/id6504287215) | [Happ RU](https://apps.apple.com/ru/app/happ-proxy-utility-plus/id6746188973) | [Happ tvOS](https://apps.apple.com/us/app/happ-proxy-utility-for-tv/id6748297274)
  - [Streisand](https://apps.apple.com/app/streisand/id6450534064)
  - [OneXray](https://github.com/OneXray/OneXray)
- macOS arm64 & x64
  - [Happ](https://apps.apple.com/app/happ-proxy-utility/id6504287215) | [Happ RU](https://apps.apple.com/ru/app/happ-proxy-utility-plus/id6746188973)
  - [V2rayU](https://github.com/yanue/V2rayU)
  - [V2RayXS](https://github.com/tzmax/V2RayXS)
  - [Furious](https://github.com/LorenEteval/Furious)
  - [OneXray](https://github.com/OneXray/OneXray)
  - [GoXRay](https://github.com/goxray/desktop)
  - [AnyPortal](https://github.com/AnyPortal/AnyPortal)
  - [v2rayN](https://github.com/2dust/v2rayN)
  - [GenyConnect](https://github.com/genyleap/GenyConnect)
- Linux
  - [v2rayA](https://github.com/v2rayA/v2rayA)
  - [Furious](https://github.com/LorenEteval/Furious)
  - [GorzRay](https://github.com/ketetefid/GorzRay)
  - [GoXRay](https://github.com/goxray/desktop)
  - [AnyPortal](https://github.com/AnyPortal/AnyPortal)
  - [v2rayN](https://github.com/2dust/v2rayN)
  - [GenyConnect](https://github.com/genyleap/GenyConnect)

## Others that support VLESS, XTLS, REALITY, XUDP, PLUX...

- iOS & macOS arm64 & tvOS
  - [Shadowrocket](https://apps.apple.com/app/shadowrocket/id932747118)
  - [Loon](https://apps.apple.com/us/app/loon/id1373567447)
  - [Egern](https://apps.apple.com/us/app/egern/id1616105820)
  - [Quantumult X](https://apps.apple.com/us/app/quantumult-x/id1443988620)
- Xray Tools
  - [xray-knife](https://github.com/lilendian0x00/xray-knife)
  - [xray-checker](https://github.com/kutovoys/xray-checker)
- Xray Wrapper
  - [XTLS/libXray](https://github.com/XTLS/libXray)
  - [xtls-sdk](https://github.com/remnawave/xtls-sdk)
  - [xtlsapi](https://github.com/hiddify/xtlsapi)
  - [AndroidLibXrayLite](https://github.com/2dust/AndroidLibXrayLite)
  - [Xray-core-python](https://github.com/LorenEteval/Xray-core-python)
  - [xray-api](https://github.com/XVGuardian/xray-api)
- [XrayR](https://github.com/XrayR-project/XrayR)
  - [XrayR-release](https://github.com/XrayR-project/XrayR-release)
  - [XrayR-V2Board](https://github.com/missuo/XrayR-V2Board)
- Cores
  - [Amnezia VPN](https://github.com/amnezia-vpn)
  - [mihomo](https://github.com/MetaCubeX/mihomo)
  - [sing-box](https://github.com/SagerNet/sing-box)

## Contributing

[Code of Conduct](https://github.com/XTLS/Xray-core/blob/main/CODE_OF_CONDUCT.md)

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/XTLS/Xray-core)

### Change Workflow

This repository uses a hard-gated change workflow for local work and PRs.

1. Open an issue. Use [`.github/ISSUE_TEMPLATE/change-request.md`](.github/ISSUE_TEMPLATE/change-request.md) for planned changes.
2. Create a branch for that issue.
3. Make the change.
4. Run `bash scripts/check.sh`.
5. Push. The local pre-push hook in [`.githooks/pre-push`](.githooks/pre-push) runs the same check again.
6. Open a PR with [`.github/pull_request_template.md`](.github/pull_request_template.md), include `Closes #<issue-number>` in the PR body, and wait for the required GitHub Actions checks to pass.

### UI Text / i18n Rule

- Any change to user-facing UI text must update both [`web/src/i18n/locales/zh-CN.json`](web/src/i18n/locales/zh-CN.json) and [`web/src/i18n/locales/en.json`](web/src/i18n/locales/en.json) in the same PR.
- Do not merge a change that ships one locale ahead of the other.

`scripts/check.sh` is the single local entry point. It runs the WebPanel Go tests, the frontend Vitest suite, the frontend production build, and the key-path regression smoke test in [`tests/test_web_smoke.py`](tests/test_web_smoke.py).

Release packaging workflows in [`.github/workflows/release.yml`](.github/workflows/release.yml) and [`.github/workflows/release-win7.yml`](.github/workflows/release-win7.yml) are reserved for version tags, published releases, and manual dispatch, so normal PRs stay on the fast validation path.

### Release Matrix

The matrix below describes what the current release workflows produce. It reflects the workflow definitions in [`.github/workflows/release.yml`](.github/workflows/release.yml), [`.github/workflows/release-win7.yml`](.github/workflows/release-win7.yml), and [`.github/workflows/docker.yml`](.github/workflows/docker.yml). Existing zip examples are based on release `webpanel-v1.2.2`; the Windows installer entry reflects the workflow added on this branch.

#### Binary Packages

| Target | Architectures | Artifact Type | Workflow | Example Asset |
| --- | --- | --- | --- | --- |
| Android | `amd64`, `arm64` | `zip` with CLI binary, not an APK | `Build and Release` | `Xray-android-arm64-v8a.zip` |
| macOS | `amd64`, `arm64` | `zip` with CLI binary | `Build and Release` | `Xray-macos-arm64-v8a.zip` |
| Windows | `386`, `amd64`, `arm64` | `zip` with CLI binary; `amd64` also ships a direct `exe` installer | `Build and Release` | `Xray-windows-64.zip`, `Xray-windows-64-setup.exe` |
| Windows 7 | `386`, `amd64` | `zip` with CLI binary | `Build and Release for Windows 7` | `Xray-win7-64.zip` |
| Linux | `386`, `amd64`, `armv5`, `armv6`, `armv7`, `arm64`, `riscv64`, `loong64`, `mips`, `mipsle`, `mips64`, `mips64le`, `ppc64`, `ppc64le`, `s390x` | `zip` with CLI binary | `Build and Release` | `Xray-linux-64.zip` |
| FreeBSD | `386`, `amd64`, `armv7`, `arm64` | `zip` with CLI binary | `Build and Release` | `Xray-freebsd-64.zip` |
| OpenBSD | `386`, `amd64`, `armv7`, `arm64` | `zip` with CLI binary | `Build and Release` | `Xray-openbsd-64.zip` |

Notes:
- The Linux `mips` and `mipsle` packages also include soft-float binaries inside the same archive.
- Release archives include checksum sidecars as `.dgst` files.
- The Windows `exe` installer currently covers the modern `amd64` build only. Windows `386`, Windows `arm64`, and Windows 7 remain zip-only.

#### Container Images

| Target | Architectures | Artifact Type | Workflow |
| --- | --- | --- | --- |
| Docker / GHCR | `linux/amd64`, `linux/386`, `linux/arm/v6`, `linux/arm/v7`, `linux/arm64/v8`, `linux/ppc64le`, `linux/s390x`, `linux/riscv64`, `linux/loong64` | Multi-arch container image | `Build and Push Docker Image` |

#### Not Covered Yet

| Area | Current Status |
| --- | --- |
| Android app delivery | No APK is published. Current Android artifacts are raw binaries in zip archives. |
| Apple mobile platforms | No iOS, iPadOS, tvOS, or watchOS release artifacts are produced here. |
| Desktop installers beyond Windows amd64 | No `msi`, `dmg`, `pkg`, `AppImage`, `deb`, or `rpm` package is published. |
| Windows installer coverage | Only the modern Windows `amd64` build gets a direct `.exe` installer. Windows `386`, Windows `arm64`, and Windows 7 remain zip-only. |
| Code signing / notarization | No release workflow in this fork currently performs Windows signing or macOS notarization. |
| Windows container images | Docker publishing currently targets Linux container architectures only. |
| Extra Android 32-bit builds | No `armeabi-v7a` or `x86` Android release artifact is produced. |

If you need a user-installable desktop or mobile client, treat these artifacts as core binaries rather than finished application packages.

Fresh clones should install the shared hook path once:

```bash
git config core.hooksPath .githooks
```

## Credits

- [Xray-core v1.0.0](https://github.com/XTLS/Xray-core/releases/tag/v1.0.0) was forked from [v2fly-core 9a03cc5](https://github.com/v2fly/v2ray-core/commit/9a03cc5c98d04cc28320fcee26dbc236b3291256), and we have made & accumulated a huge number of enhancements over time, check [the release notes for each version](https://github.com/XTLS/Xray-core/releases).
- For third-party projects used in [Xray-core](https://github.com/XTLS/Xray-core), check your local or [the latest go.mod](https://github.com/XTLS/Xray-core/blob/main/go.mod).

## One-line Compilation

### Windows (PowerShell)

```powershell
$env:CGO_ENABLED=0
go build -o xray.exe -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -v ./main
```

### Linux / macOS

```bash
CGO_ENABLED=0 go build -o xray -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -v ./main
```

### Reproducible Releases

Make sure that you are using the same Go version, and remember to set the git commit id (7 bytes):

```bash
CGO_ENABLED=0 go build -o xray -trimpath -buildvcs=false -gcflags="all=-l=4" -ldflags="-X github.com/xtls/xray-core/core.build=REPLACE -s -w -buildid=" -v ./main
```

If you are compiling a 32-bit MIPS/MIPSLE target, use this command instead:

```bash
CGO_ENABLED=0 go build -o xray -trimpath -buildvcs=false -gcflags="-l=4" -ldflags="-X github.com/xtls/xray-core/core.build=REPLACE -s -w -buildid=" -v ./main
```

## Stargazers over time

[![Stargazers over time](https://starchart.cc/XTLS/Xray-core.svg)](https://starchart.cc/XTLS/Xray-core)
