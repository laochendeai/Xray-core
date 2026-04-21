#!/usr/bin/env python3
"""Normalize WebPanel TUN installer config fields.

The WebPanel runtime normalizes remote DNS entries before building the TUN
helper command. The sudoers installer must write the same exact arguments, or
`sudo -n -l` readiness checks will reject otherwise valid entries.
"""

from __future__ import annotations

import ipaddress
import json
import sys
from pathlib import Path


KNOWN_DOH = {
    "1.1.1.1": "https://cloudflare-dns.com/dns-query",
    "1.0.0.1": "https://cloudflare-dns.com/dns-query",
    "8.8.8.8": "https://dns.google/dns-query",
    "8.8.4.4": "https://dns.google/dns-query",
    "9.9.9.9": "https://dns.quad9.net/dns-query",
    "149.112.112.112": "https://dns.quad9.net/dns-query",
    "223.5.5.5": "https://dns.alidns.com/dns-query",
    "223.6.6.6": "https://dns.alidns.com/dns-query",
    "119.29.29.29": "https://doh.pub/dns-query",
}

PASSTHROUGH_PREFIXES = ("https://", "https+local://", "quic+local://")
STRIP_PREFIXES = ("tcp://", "tcp+local://", "udp://", "udp+local://")


def normalize_resolver_host(value: str) -> str:
    host = value.strip()
    if host.startswith("//"):
        host = host[2:]

    for separator in ("/", "?", "#"):
        if separator in host:
            host = host.split(separator, 1)[0]

    if host.startswith("["):
        end = host.find("]")
        if end >= 0:
            bracketed_host = host[1:end]
            remainder = host[end + 1 :]
            if not remainder or remainder.startswith(":"):
                host = bracketed_host
    elif host.count(":") == 1:
        split_host, port = host.rsplit(":", 1)
        if split_host and port.isdigit():
            host = split_host

    return host.strip("[]")


def normalize_resolver(value: object) -> str:
    trimmed = str(value or "").strip()
    if not trimmed:
        return ""

    lower = trimmed.lower()
    if lower.startswith(PASSTHROUGH_PREFIXES):
        return trimmed

    for prefix in STRIP_PREFIXES:
        if lower.startswith(prefix):
            trimmed = trimmed[len(prefix) :]
            break
    else:
        if "://" in lower:
            trimmed = trimmed.split("://", 1)[1]

    host = normalize_resolver_host(trimmed)
    if not host:
        return ""

    if host in KNOWN_DOH:
        return KNOWN_DOH[host]

    try:
        parsed_ip = ipaddress.ip_address(host)
    except ValueError:
        parsed_ip = None
    if parsed_ip and parsed_ip.version == 6:
        return f"https://[{parsed_ip.compressed}]/dns-query"

    return f"https://{host}/dns-query"


def normalize_remote_dns(values: list[object]) -> list[str]:
    normalized: list[str] = []
    seen: set[str] = set()
    for value in values:
        next_value = normalize_resolver(value)
        if not next_value or next_value in seen:
            continue
        normalized.append(next_value)
        seen.add(next_value)
    return normalized


def update_config(config_path: Path, helper_dst: str, xray_dst: str) -> None:
    data = json.loads(config_path.read_text(encoding="utf-8"))
    webpanel = data.setdefault("webpanel", {})
    tun = webpanel.setdefault("tun", {})
    tun["helperPath"] = helper_dst
    tun["binaryPath"] = xray_dst
    tun["useSudo"] = True

    remote_dns = normalize_remote_dns(
        tun.get("remoteDns")
        or [
            "https://cloudflare-dns.com/dns-query",
            "https://dns.google/dns-query",
        ]
    )
    if remote_dns:
        tun["remoteDns"] = remote_dns

    config_path.write_text(
        json.dumps(data, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )


def main(argv: list[str]) -> int:
    if len(argv) != 4:
        print(
            "usage: normalize-webpanel-tun-config.py CONFIG_PATH HELPER_DST XRAY_DST",
            file=sys.stderr,
        )
        return 2

    update_config(Path(argv[1]), argv[2], argv[3])
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
