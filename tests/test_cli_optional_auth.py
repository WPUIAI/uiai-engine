"""Optional-auth transport regressions; UIAI_TEST_BASH selects a native shell."""
import http.server
import json
import os
from pathlib import Path
import subprocess
import threading
import unittest

ROOT = Path(__file__).resolve().parents[1]


class CLIAuthTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.requests = []
        class Handler(http.server.BaseHTTPRequestHandler):
            def do_GET(self):
                cls.requests.append((self.path, dict(self.headers)))
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(b'{"status":"ok"}')
            def log_message(self, *args):
                pass
        cls.server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()
        cls.url = f"http://127.0.0.1:{cls.server.server_port}"

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.server.server_close()
        cls.thread.join()

    def run_request(self, credentials=None, scope=None, shell=None):
        env = os.environ.copy()
        for name in ("UIAI_API_KEY", "UIAI_BEARER_TOKEN", "UIAI_EVIDENCE_SCOPE_JSON"):
            env.pop(name, None)
        env.update(credentials or {})
        env["UIAI_ENGINE_URL"] = self.url
        if scope is not None:
            env["UIAI_EVIDENCE_SCOPE_JSON"] = json.dumps(scope)
        if shell:
            cmd = [shell, "-uc", '. "$1"; uiai_curl_auth -fsS "$2"', "test", str(ROOT / "scripts/uiai-http-auth.sh"), self.url]
        else:
            cmd = [os.environ.get("UIAI_TEST_BASH", "bash"), str(ROOT / "scripts/uiai"), "--json", "health"]
        result = subprocess.run(cmd, env=env, text=True, capture_output=True, timeout=10)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(json.loads(result.stdout)["status"], "ok")
        return self.requests[-1][1]

    def test_unset_and_empty_auth(self):
        for values in ({}, {"UIAI_API_KEY": "", "UIAI_BEARER_TOKEN": ""}):
            headers = self.run_request(values)
            self.assertNotIn("X-API-Key", headers)
            self.assertNotIn("Authorization", headers)

    def test_authenticated_routes_and_precedence(self):
        for values, expected, absent in [
            ({"UIAI_API_KEY": "synthetic key with spaces"}, ("X-API-Key", "synthetic key with spaces"), "Authorization"),
            ({"UIAI_BEARER_TOKEN": "synthetic-bearer"}, ("Authorization", "Bearer synthetic-bearer"), "X-API-Key"),
            ({"UIAI_API_KEY": "synthetic-key", "UIAI_BEARER_TOKEN": "ignored"}, ("X-API-Key", "synthetic-key"), "Authorization"),
        ]:
            headers = self.run_request(values)
            self.assertEqual(headers.get(expected[0]), expected[1])
            self.assertNotIn(absent, headers)

    def test_empty_and_populated_scope(self):
        self.run_request(scope={})
        headers = self.run_request(scope={"project_ref": "synthetic project", "continuity_id": "synthetic-stream", "work_items": ["synthetic-item"]})
        self.assertEqual(headers["X-UIAI-Project-Ref"], "synthetic project")
        self.assertEqual(headers["X-UIAI-Continuity-Ref"], "synthetic-stream")
        self.assertEqual(json.loads(headers["X-UIAI-Work-Items"]), ["synthetic-item"])

    def test_control_characters_rejected_before_transport(self):
        for character in ("\n", "\r", "\t", "\x1f", "\x7f"):
            env = os.environ.copy()
            for name in ("UIAI_API_KEY", "UIAI_BEARER_TOKEN"):
                env.pop(name, None)
            env["UIAI_ENGINE_URL"] = self.url
            env["UIAI_EVIDENCE_SCOPE_JSON"] = json.dumps({"project_ref": "synthetic" + character + "injected"})
            before = len(self.requests)
            result = subprocess.run([os.environ.get("UIAI_TEST_BASH", "bash"), str(ROOT / "scripts/uiai"), "--json", "health"], env=env, capture_output=True, text=True, timeout=10)
            self.assertEqual(result.returncode, 4)
            self.assertIn("must not contain control characters", result.stderr)
            self.assertEqual(len(self.requests), before)

    def test_helper_is_posix_and_nounset_safe(self):
        for shell in ("sh", "bash"):
            for values in ({}, {"UIAI_API_KEY": "synthetic-key"}, {"UIAI_BEARER_TOKEN": "synthetic-bearer"}):
                self.run_request(values, shell=shell)

    def test_consumers_share_one_optional_auth_owner(self):
        names = ["uiai", "check-tool-parity.sh", "smoke-agent-integrations.sh", "smoke-mcp-tool-routes.sh", "smoke-pi-extension-registration.sh", "smoke-mcp-structured-failure.sh", "smoke-focusa-error-evidence.sh", "smoke-failed-network-diagnostics.sh"]
        for name in names:
            source = (ROOT / "scripts" / name).read_text()
            self.assertIn("uiai-http-auth.sh", source, name)
            self.assertNotIn("AUTH_ARGS", source, name)
            result = subprocess.run([os.environ.get("UIAI_TEST_BASH", "bash"), "-n", str(ROOT / "scripts" / name)], capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)


if __name__ == "__main__":
    unittest.main()
