#!/usr/bin/env python3
"""Render escaped sudoers entries for WebPanel TUN helper actions."""

from __future__ import annotations

import sys


def escape_sudoers_token(value: str) -> str:
    escaped: list[str] = []
    for char in value:
        if char in {"\\", ",", ":", "="}:
            escaped.append("\\")
        escaped.append(char)
    return "".join(escaped)


def render_line(
    target_user: str,
    helper_dst: str,
    action: str,
    xray_dst: str,
    runtime_config_path: str,
    state_dir: str,
    interface_name: str,
    remote_dns: list[str],
) -> str:
    args = [
        helper_dst,
        action,
        xray_dst,
        runtime_config_path,
        state_dir,
        interface_name,
        *remote_dns,
    ]
    escaped_args = " ".join(escape_sudoers_token(value) for value in args)
    return f"{target_user} ALL=(root) NOPASSWD: {escaped_args}"


def main(argv: list[str]) -> int:
    if len(argv) < 8:
        print(
            "usage: render-webpanel-tun-sudoers.py TARGET_USER HELPER_DST XRAY_DST "
            "RUNTIME_CONFIG_PATH STATE_DIR INTERFACE_NAME REMOTE_DNS...",
            file=sys.stderr,
        )
        return 2

    target_user = argv[1]
    helper_dst = argv[2]
    xray_dst = argv[3]
    runtime_config_path = argv[4]
    state_dir = argv[5]
    interface_name = argv[6]
    remote_dns = argv[7:]

    lines = ["# Managed by install-webpanel-tun-sudoers.sh"]
    for action in ("start", "stop", "toggle"):
        lines.append(
            render_line(
                target_user,
                helper_dst,
                action,
                xray_dst,
                runtime_config_path,
                state_dir,
                interface_name,
                remote_dns,
            )
        )
    sys.stdout.write("\n".join(lines) + "\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
