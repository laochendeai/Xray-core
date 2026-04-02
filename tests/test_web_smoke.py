#!/usr/bin/env python3
from __future__ import annotations

import json
import os
import socket
import subprocess
import tempfile
import time
import unittest
import urllib.error
import urllib.request
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
BASE_CONFIG = REPO_ROOT / "test-config.json"


def reserve_port() -> int:
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.bind(("127.0.0.1", 0))
    port = sock.getsockname()[1]
    sock.close()
    return port


def http_json(method: str, url: str, payload: dict | None = None, token: str | None = None) -> dict:
    headers = {}
    data = None
    if payload is not None:
        headers["Content-Type"] = "application/json"
        data = json.dumps(payload).encode("utf-8")
    if token:
        headers["Authorization"] = f"Bearer {token}"

    request = urllib.request.Request(url, method=method, data=data, headers=headers)
    with urllib.request.urlopen(request, timeout=5) as response:
        body = response.read().decode("utf-8")
        return json.loads(body)


class WebSmokeTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.temp_dir = tempfile.TemporaryDirectory(prefix="xray-web-smoke-")
        cls.work_dir = Path(cls.temp_dir.name)
        cls.binary_path = cls.work_dir / "xray-smoke"
        cls.log_path = cls.work_dir / "xray-smoke.log"
        cls.config_path = cls.work_dir / "test-config.json"

        api_port = reserve_port()
        web_port = reserve_port()
        cls.base_url = f"http://127.0.0.1:{web_port}"

        config = json.loads(BASE_CONFIG.read_text(encoding="utf-8"))
        config["api"]["listen"] = f"127.0.0.1:{api_port}"
        config["webpanel"]["listen"] = f"127.0.0.1:{web_port}"
        config["webpanel"]["apiEndpoint"] = f"127.0.0.1:{api_port}"
        config["webpanel"]["configPath"] = str(cls.config_path)
        cls.config_path.write_text(json.dumps(config, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

        override_binary = os.environ.get("XRAY_SMOKE_BINARY")
        if override_binary:
            cls.binary_path = Path(override_binary)
        else:
            subprocess.run(
                ["go", "build", "-o", str(cls.binary_path), "./main"],
                cwd=REPO_ROOT,
                check=True,
            )

        cls.log_file = cls.log_path.open("w", encoding="utf-8")
        cls.proc = subprocess.Popen(
            [str(cls.binary_path), "run", "-c", str(cls.config_path)],
            cwd=REPO_ROOT,
            stdout=cls.log_file,
            stderr=subprocess.STDOUT,
        )

        try:
            cls.wait_for_ready()
            login = http_json(
                "POST",
                f"{cls.base_url}/api/v1/auth/login",
                {"username": "admin", "password": "admin123"},
            )
            cls.token = login["token"]
        except Exception:
            cls.tearDownClass()
            raise

    @classmethod
    def tearDownClass(cls) -> None:
        proc = getattr(cls, "proc", None)
        if proc is not None and proc.poll() is None:
            proc.terminate()
            try:
                proc.wait(timeout=10)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait(timeout=5)

        log_file = getattr(cls, "log_file", None)
        if log_file is not None and not log_file.closed:
            log_file.close()

        temp_dir = getattr(cls, "temp_dir", None)
        if temp_dir is not None:
            temp_dir.cleanup()

    @classmethod
    def wait_for_ready(cls) -> None:
        deadline = time.time() + 30
        last_error = None
        while time.time() < deadline:
            if cls.proc.poll() is not None:
                raise AssertionError(
                    f"xray exited early with code {cls.proc.returncode}\n{cls.read_log_tail()}"
                )
            try:
                request = urllib.request.Request(cls.base_url, method="GET")
                with urllib.request.urlopen(request, timeout=2) as response:
                    if response.status == 200:
                        return
            except Exception as exc:  # pragma: no cover - best effort retries
                last_error = exc
            time.sleep(0.5)

        raise AssertionError(f"web panel did not become ready: {last_error}\n{cls.read_log_tail()}")

    @classmethod
    def read_log_tail(cls) -> str:
        if not cls.log_path.exists():
            return "<no log captured>"
        content = cls.log_path.read_text(encoding="utf-8", errors="replace")
        return content[-4000:]

    def api_get(self, path: str) -> dict:
        try:
            return http_json("GET", f"{self.base_url}{path}", token=self.token)
        except urllib.error.HTTPError as exc:  # pragma: no cover - easier diagnostics
            body = exc.read().decode("utf-8", errors="replace")
            self.fail(f"{path} returned {exc.code}: {body}\n{self.read_log_tail()}")

    def test_root_serves_app_shell(self) -> None:
        request = urllib.request.Request(self.base_url, method="GET")
        with urllib.request.urlopen(request, timeout=5) as response:
            html = response.read().decode("utf-8")
        self.assertIn("Xray Web Panel", html)
        self.assertIn("/assets/index-", html)

    def test_login_and_authenticated_api_flow(self) -> None:
        subscriptions = self.api_get("/api/v1/subscriptions")
        self.assertIn("subscriptions", subscriptions)
        self.assertIsInstance(subscriptions["subscriptions"], list)

        node_pool = self.api_get("/api/v1/node-pool")
        self.assertIn("nodes", node_pool)
        self.assertIn("summary", node_pool)
        self.assertIn("recentEvents", node_pool)

        tun_settings = self.api_get("/api/v1/tun/settings")
        self.assertIn("selectionPolicy", tun_settings)
        self.assertIn("routeMode", tun_settings)
        self.assertIn("remoteDns", tun_settings)
        self.assertIsInstance(tun_settings["remoteDns"], list)


if __name__ == "__main__":
    unittest.main(verbosity=2)
