#!/usr/bin/env python3
from __future__ import annotations

import contextlib
import json
import os
import socket
import subprocess
import tempfile
import threading
import time
import unittest
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, HTTPServer
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


@contextlib.contextmanager
def serve_subscription_content(body: str):
    state = {"body": body}

    class SubscriptionHandler(BaseHTTPRequestHandler):
        def do_GET(self) -> None:  # pragma: no cover - exercised via smoke test
            payload = state["body"].encode("utf-8")
            self.send_response(200)
            self.send_header("Content-Type", "text/plain; charset=utf-8")
            self.send_header("Content-Length", str(len(payload)))
            self.end_headers()
            self.wfile.write(payload)

        def log_message(self, format: str, *args) -> None:  # pragma: no cover - keep CI logs quiet
            return

    server = HTTPServer(("127.0.0.1", reserve_port()), SubscriptionHandler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        def update_body(value: str) -> None:
            state["body"] = value

        yield f"http://127.0.0.1:{server.server_port}/subscription", update_body
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)


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
        inbound_port = reserve_port()
        user_inbound_port = reserve_port()
        cls.base_url = f"http://127.0.0.1:{web_port}"

        config = json.loads(BASE_CONFIG.read_text(encoding="utf-8"))
        config["api"]["listen"] = f"127.0.0.1:{api_port}"
        config["webpanel"]["listen"] = f"127.0.0.1:{web_port}"
        config["webpanel"]["apiEndpoint"] = f"127.0.0.1:{api_port}"
        config["webpanel"]["configPath"] = str(cls.config_path)
        config["inbounds"] = [
            {
                "tag": "smoke-socks",
                "listen": "127.0.0.1",
                "port": inbound_port,
                "protocol": "socks",
                "settings": {
                    "auth": "noauth",
                    "udp": True,
                },
            },
            {
                "tag": "smoke-vless",
                "listen": "127.0.0.1",
                "port": user_inbound_port,
                "protocol": "vless",
                "settings": {
                    "clients": [
                        {
                            "id": "11111111-1111-1111-1111-111111111111",
                            "email": "smoke-user@example.com",
                        }
                    ],
                    "decryption": "none",
                },
            },
        ]
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

    def api_json(self, method: str, path: str, payload: dict | None = None) -> dict:
        try:
            return http_json(method, f"{self.base_url}{path}", payload=payload, token=self.token)
        except urllib.error.HTTPError as exc:  # pragma: no cover - easier diagnostics
            body = exc.read().decode("utf-8", errors="replace")
            self.fail(f"{method} {path} returned {exc.code}: {body}\n{self.read_log_tail()}")

    def wait_for_api(self, path: str, predicate, timeout: float = 10.0) -> dict:
        deadline = time.time() + timeout
        last_response = None

        while time.time() < deadline:
            response = self.api_get(path)
            last_response = response
            if predicate(response):
                return response
            time.sleep(0.25)

        self.fail(f"condition not met for {path}: {last_response}\n{self.read_log_tail()}")

    def test_root_serves_app_shell(self) -> None:
        request = urllib.request.Request(self.base_url, method="GET")
        with urllib.request.urlopen(request, timeout=5) as response:
            html = response.read().decode("utf-8")
        self.assertIn("Xray Web Panel", html)
        self.assertIn("/assets/index-", html)

    def test_login_and_authenticated_api_flow(self) -> None:
        readiness = self.api_get("/api/v1/readiness")
        self.assertIn("healthy", readiness)
        self.assertIn("blockingCount", readiness)
        self.assertIn("warningCount", readiness)
        self.assertIn("checks", readiness)
        self.assertIsInstance(readiness["checks"], list)

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

    def test_edit_and_clone_inbounds_and_outbounds(self) -> None:
        clone_inbound_tag = "smoke-socks-copy"
        clone_outbound_tag = "direct-copy"
        edited_inbound_port = reserve_port()
        cloned_inbound_port = reserve_port()

        try:
            inbound_detail = self.api_get("/api/v1/inbounds/smoke-socks")["inbound"]
            edited_inbound = json.loads(json.dumps(inbound_detail))
            edited_range = edited_inbound["receiverSettings"]["portList"]["range"][0]
            from_key = "From" if "From" in edited_range else "from"
            to_key = "To" if "To" in edited_range else "to"
            edited_range[from_key] = edited_inbound_port
            edited_range[to_key] = edited_inbound_port
            self.api_json(
                "PUT",
                "/api/v1/inbounds/smoke-socks",
                {
                    "inbound": edited_inbound,
                },
            )

            updated_inbound = self.api_get("/api/v1/inbounds/smoke-socks")["inbound"]
            updated_port = updated_inbound["receiverSettings"]["portList"]["range"][0]
            self.assertEqual(updated_port[from_key], edited_inbound_port)
            self.assertEqual(updated_port[to_key], edited_inbound_port)

            cloned_inbound = json.loads(json.dumps(updated_inbound))
            cloned_inbound["tag"] = clone_inbound_tag
            cloned_range = cloned_inbound["receiverSettings"]["portList"]["range"][0]
            cloned_range[from_key] = cloned_inbound_port
            cloned_range[to_key] = cloned_inbound_port
            self.api_json(
                "POST",
                "/api/v1/inbounds",
                {
                    "inbound": cloned_inbound,
                },
            )

            inbounds = self.api_get("/api/v1/inbounds")["inbounds"]
            self.assertTrue(any(item["tag"] == clone_inbound_tag for item in inbounds))

            outbound_detail = self.api_get("/api/v1/outbounds/direct")["outbound"]
            edited_outbound = json.loads(json.dumps(outbound_detail))
            edited_outbound.setdefault("proxySettings", {})
            edited_outbound["proxySettings"]["domainStrategy"] = "USE_IP4"
            self.api_json(
                "PUT",
                "/api/v1/outbounds/direct",
                {
                    "outbound": edited_outbound,
                },
            )

            updated_outbound = self.api_get("/api/v1/outbounds/direct")["outbound"]
            self.assertEqual(updated_outbound["proxySettings"]["domainStrategy"], "USE_IP4")

            cloned_outbound = json.loads(json.dumps(updated_outbound))
            cloned_outbound["tag"] = clone_outbound_tag
            self.api_json(
                "POST",
                "/api/v1/outbounds",
                {
                    "outbound": cloned_outbound,
                },
            )

            outbounds = self.api_get("/api/v1/outbounds")["outbounds"]
            self.assertTrue(any(item["tag"] == clone_outbound_tag for item in outbounds))
        finally:
            with contextlib.suppress(Exception):
                self.api_json("DELETE", f"/api/v1/inbounds/{clone_inbound_tag}")
            with contextlib.suppress(Exception):
                self.api_json("DELETE", f"/api/v1/outbounds/{clone_outbound_tag}")

    def test_edit_user_reset_traffic_and_generate_share_link(self) -> None:
        updated_email = "smoke-user-updated@example.com"
        updated_uuid = "22222222-2222-2222-2222-222222222222"

        users = self.api_get("/api/v1/users/")["users"]
        user = next(item for item in users if item["email"] == "smoke-user@example.com")
        self.assertEqual(user["inboundTag"], "smoke-vless")

        updated_account = json.loads(json.dumps(user["account"]))
        updated_account["id"] = updated_uuid
        self.api_json(
            "PUT",
            "/api/v1/inbounds/smoke-vless/users/smoke-user%40example.com",
            {
                "email": updated_email,
                "level": 1,
                "accountType": user["accountType"],
                "account": updated_account,
            },
        )

        updated_users = self.wait_for_api(
            "/api/v1/users/",
            lambda data: any(item["email"] == updated_email and item["level"] == 1 for item in data["users"]),
        )
        updated_user = next(item for item in updated_users["users"] if item["email"] == updated_email)
        self.assertEqual(updated_user["account"]["id"], updated_uuid)

        reset_result = self.api_json(
            "POST",
            "/api/v1/users/smoke-user-updated%40example.com/reset-traffic",
        )
        self.assertEqual(reset_result["message"], "User traffic reset successfully")

        share_result = self.api_json(
            "POST",
            "/api/v1/share/generate",
            {
                "protocol": "vless",
                "address": "example.com",
                "port": 443,
                "uuid": updated_uuid,
                "type": "tcp",
                "tls": "tls",
                "sni": "example.com",
            },
        )
        self.assertIn(updated_uuid, share_result["link"])

    def test_add_subscription_refresh_and_remove_node_observe_api_state_change(self) -> None:
        initial_link = (
            "vless://11111111-1111-1111-1111-111111111111@smoke-a.example.com:443"
            "?encryption=none&security=tls&sni=smoke-a.example.com&type=tcp#smoke-a"
        )
        refreshed_link = (
            "vless://22222222-2222-2222-2222-222222222222@smoke-b.example.com:443"
            "?encryption=none&security=tls&sni=smoke-b.example.com&type=tcp#smoke-b"
        )
        subscription_id = None
        node_id = None

        with serve_subscription_content(f"{initial_link}\n") as (subscription_url, set_subscription_body):
            try:
                created = self.api_json(
                    "POST",
                    "/api/v1/subscriptions",
                    {
                        "url": subscription_url,
                        "sourceType": "url",
                        "remark": "smoke-subscription",
                        "autoRefresh": False,
                        "refreshIntervalMin": 60,
                    },
                )
                subscription = created["subscription"]
                subscription_id = subscription["id"]
                self.assertEqual(subscription["sourceType"], "url")
                self.assertEqual(subscription["nodeCount"], 1)

                initial_pool = self.wait_for_api(
                    "/api/v1/node-pool",
                    lambda data: any(item["subscriptionId"] == subscription_id for item in data["nodes"]),
                )
                node = next(
                    (item for item in initial_pool["nodes"] if item["subscriptionId"] == subscription_id),
                    None,
                )
                self.assertIsNotNone(node, "new subscription node was not visible in node pool")

                node_id = node["id"]
                self.assertEqual(node["remark"], "smoke-a")
                self.assertEqual(node["address"], "smoke-a.example.com")
                self.assertEqual(node["status"], "staging")
                self.assertEqual(node["statusReason"], "subscription_node_discovered")
                self.assertEqual(initial_pool["summary"]["stagingCount"], 1)

                set_subscription_body(f"{refreshed_link}\n")
                refresh_result = self.api_json("POST", f"/api/v1/subscriptions/{subscription_id}/refresh")
                self.assertEqual(refresh_result["message"], "Subscription refreshed successfully")

                refreshed_pool = self.wait_for_api(
                    "/api/v1/node-pool",
                    lambda data: any(
                        item["id"] == node_id
                        and item["status"] == "staging"
                        and item.get("subscriptionMissing") is True
                        for item in data["nodes"]
                    )
                    and any(item["address"] == "smoke-b.example.com" for item in data["nodes"]),
                )
                missing_node = next(item for item in refreshed_pool["nodes"] if item["id"] == node_id)
                replacement_node = next(
                    item for item in refreshed_pool["nodes"] if item["address"] == "smoke-b.example.com"
                )
                self.assertEqual(missing_node["status"], "staging")
                self.assertEqual(missing_node["statusReason"], "subscription_node_discovered")
                self.assertTrue(missing_node["subscriptionMissing"])
                self.assertEqual(replacement_node["status"], "staging")
                self.assertEqual(replacement_node["statusReason"], "subscription_node_discovered")
                self.assertEqual(refreshed_pool["summary"]["candidateCount"], 0)
                self.assertEqual(refreshed_pool["summary"]["stagingCount"], 2)

                remove_result = self.api_json("POST", f"/api/v1/node-pool/{node_id}/remove")
                self.assertEqual(remove_result["message"], "Node removed successfully")

                removed_pool = self.wait_for_api(
                    "/api/v1/node-pool",
                    lambda data: any(
                        item["id"] == node_id
                        and item["status"] == "removed"
                        and item["statusReason"] == "manual_remove"
                        for item in data["nodes"]
                    ),
                )
                removed_node = next(item for item in removed_pool["nodes"] if item["id"] == node_id)
                self.assertEqual(removed_node["status"], "removed")
                self.assertEqual(removed_node["statusReason"], "manual_remove")
                self.assertEqual(removed_pool["summary"]["removedCount"], 1)
                self.assertTrue(
                    any(
                        event["nodeId"] == node_id
                        and event["reason"] == "manual_remove"
                        and event["status"] == "removed"
                        for event in removed_pool["recentEvents"]
                    )
                )
            finally:
                if subscription_id is not None:
                    with contextlib.suppress(Exception):
                        self.api_json("DELETE", f"/api/v1/subscriptions/{subscription_id}")
                if node_id is not None:
                    with contextlib.suppress(Exception):
                        self.api_json(
                            "POST",
                            "/api/v1/node-pool/bulk-purge-removed",
                            {"ids": [node_id]},
                        )

    def test_update_subscription_and_toggle_auto_refresh(self) -> None:
        initial_link = (
            "vless://66666666-6666-6666-6666-666666666666@edit-a.example.com:443"
            "?encryption=none&security=tls&sni=edit-a.example.com&type=tcp#edit-a"
        )
        updated_link = (
            "vless://77777777-7777-7777-7777-777777777777@edit-b.example.com:443"
            "?encryption=none&security=tls&sni=edit-b.example.com&type=tcp#edit-b"
        )
        subscription_id = None
        original_node_id = None
        replacement_node_id = None

        with serve_subscription_content(f"{initial_link}\n") as (subscription_url_a, _):
            with serve_subscription_content(f"{updated_link}\n") as (subscription_url_b, _):
                try:
                    created = self.api_json(
                        "POST",
                        "/api/v1/subscriptions",
                        {
                            "url": subscription_url_a,
                            "sourceType": "url",
                            "remark": "editable-subscription",
                            "autoRefresh": True,
                            "refreshIntervalMin": 60,
                        },
                    )
                    subscription = created["subscription"]
                    subscription_id = subscription["id"]

                    initial_pool = self.wait_for_api(
                        "/api/v1/node-pool",
                        lambda data: any(item["subscriptionId"] == subscription_id for item in data["nodes"]),
                    )
                    original_node = next(
                        item for item in initial_pool["nodes"] if item["subscriptionId"] == subscription_id
                    )
                    original_node_id = original_node["id"]
                    self.assertEqual(original_node["address"], "edit-a.example.com")

                    updated = self.api_json(
                        "PUT",
                        f"/api/v1/subscriptions/{subscription_id}",
                        {
                            "sourceType": "url",
                            "url": subscription_url_b,
                            "remark": "editable-subscription-updated",
                            "autoRefresh": False,
                            "refreshIntervalMin": 120,
                        },
                    )
                    updated_subscription = updated["subscription"]
                    self.assertEqual(updated_subscription["id"], subscription_id)
                    self.assertEqual(updated_subscription["url"], subscription_url_b)
                    self.assertEqual(updated_subscription["remark"], "editable-subscription-updated")
                    self.assertFalse(updated_subscription["autoRefresh"])
                    self.assertEqual(updated_subscription["refreshIntervalMin"], 120)

                    reconciled_pool = self.wait_for_api(
                        "/api/v1/node-pool",
                        lambda data: any(
                            item["id"] == original_node_id
                            and item["status"] == "staging"
                            and item.get("subscriptionMissing") is True
                            for item in data["nodes"]
                        )
                        and any(
                            item["subscriptionId"] == subscription_id and item["address"] == "edit-b.example.com"
                            for item in data["nodes"]
                        ),
                    )
                    replacement_node = next(
                        item
                        for item in reconciled_pool["nodes"]
                        if item["subscriptionId"] == subscription_id and item["address"] == "edit-b.example.com"
                    )
                    replacement_node_id = replacement_node["id"]
                    self.assertEqual(replacement_node["status"], "staging")

                    listed = self.api_get("/api/v1/subscriptions")
                    listed_subscription = next(
                        item for item in listed["subscriptions"] if item["id"] == subscription_id
                    )
                    self.assertEqual(listed_subscription["url"], subscription_url_b)
                    self.assertFalse(listed_subscription["autoRefresh"])
                    self.assertEqual(listed_subscription["refreshIntervalMin"], 120)

                    resumed = self.api_json(
                        "PUT",
                        f"/api/v1/subscriptions/{subscription_id}",
                        {
                            "sourceType": "url",
                            "autoRefresh": True,
                        },
                    )
                    resumed_subscription = resumed["subscription"]
                    self.assertEqual(resumed_subscription["id"], subscription_id)
                    self.assertTrue(resumed_subscription["autoRefresh"])
                    self.assertEqual(resumed_subscription["refreshIntervalMin"], 120)

                    listed = self.api_get("/api/v1/subscriptions")
                    listed_subscription = next(
                        item for item in listed["subscriptions"] if item["id"] == subscription_id
                    )
                    self.assertTrue(listed_subscription["autoRefresh"])
                    self.assertEqual(listed_subscription["refreshIntervalMin"], 120)
                finally:
                    if subscription_id is not None:
                        with contextlib.suppress(Exception):
                            self.api_json("DELETE", f"/api/v1/subscriptions/{subscription_id}")
                    removable_ids = [node_id for node_id in (original_node_id, replacement_node_id) if node_id]
                    if removable_ids:
                        with contextlib.suppress(Exception):
                            self.api_json(
                                "POST",
                                "/api/v1/node-pool/bulk-purge-removed",
                                {"ids": removable_ids},
                            )


if __name__ == "__main__":
    unittest.main(verbosity=2)
