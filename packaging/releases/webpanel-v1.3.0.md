Webpanel v1.3.0

- Added Node Pool intelligence so active nodes expose exit IP, cleanliness, and network-type hints directly in the WebPanel.
- Added transparent destination bindings so OpenAI, ChatGPT, and custom domains can be pinned to a chosen active node.
- Added first-slice AnyTLS support plus share-link canonicalization, and preserved VLESS encryption values during link import.
- Expanded transparent-mode diagnostics with direct/proxy egress reporting, DNS path visibility, baseline verification, and the readiness center.
- Added experimental UDP/QUIC aggregation scaffolding with local prototype and relay benchmark diagnostics so aggregation work can be inspected without replacing the stable single-path runtime.
- Based on Xray core 26.2.6; this WebPanel feature release does not change the upstream core version number.
