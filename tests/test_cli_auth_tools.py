import json
import os
import subprocess
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class AuthHandler(BaseHTTPRequestHandler):
    calls = []

    def do_POST(self):
        length = int(self.headers.get("content-length", "0"))
        body = json.loads(self.rfile.read(length) or b"{}")
        self.__class__.calls.append({"method": "POST", "path": self.path, "body": body})
        self.send_response(409 if body.get("resume_token") == "stale-private-token" else 200)
        self.send_header("content-type", "application/json")
        self.end_headers()
        if body.get("resume_token") == "stale-private-token":
            payload = {
                "code": "auth_challenge_pending",
                "error": "invalid auth challenge resume token",
                "suggested_next_action": "Call browser_auth_cancel or obtain the current private token before browser_auth_resolve.",
                "resume_token": "resume_server_must_not_leak",
                "private_secret": "server-secret-must-not-leak",
            }
        elif self.path.endswith("/auth/resolve"):
            payload = {"resolved": True, "session": {"id": "cli/auth fixture"}, "resume_token": "resume_success_must_not_leak", "credential": "cli-secret"}
        else:
            payload = {"cancelled": True, "resume_token": "resume_cancel_must_not_leak", "password": "cli-password"}
        self.wfile.write(json.dumps(payload).encode())

    def log_message(self, *_args):
        pass


class CLIAuthToolsTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        AuthHandler.calls = []
        cls.server = ThreadingHTTPServer(("127.0.0.1", 0), AuthHandler)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()
        cls.env = {**os.environ, "UIAI_ENGINE_URL": "http://127.0.0.1:" + str(cls.server.server_port)}

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.server.server_close()
        cls.thread.join(timeout=2)

    def run_cli(self, *args, input_text=None, env=None):
        return subprocess.run(
            [os.environ.get("UIAI_TEST_BASH", "bash"), str(ROOT / "scripts/uiai"), "--json", *args],
            cwd=ROOT,
            env=env or self.env,
            input=input_text,
            text=True,
            capture_output=True,
            timeout=10,
        )

    def test_resolve_and_cancel_route_and_redact(self):
        resolved = self.run_cli("session", "auth", "resolve", "cli/auth fixture", input_text="current-private-token\n")
        self.assertEqual(resolved.returncode, 0, resolved.stderr)
        self.assertEqual(AuthHandler.calls[-1], {
            "method": "POST",
            "path": "/api/session/cli%2Fauth%20fixture/auth/resolve",
            "body": {"resume_token": "current-private-token"},
        })
        self.assertIn('"resolved":true', resolved.stdout.replace(" ", "").replace("\n", ""))
        for secret in ("resume_success", "cli-secret", "resume_token", "credential"):
            self.assertNotIn(secret, resolved.stdout)

        cancelled = self.run_cli("session", "auth", "cancel", "cli/auth fixture")
        self.assertEqual(cancelled.returncode, 0, cancelled.stderr)
        self.assertEqual(AuthHandler.calls[-1], {
            "method": "POST",
            "path": "/api/session/cli%2Fauth%20fixture/auth/cancel",
            "body": {},
        })
        self.assertIn('"cancelled":true', cancelled.stdout.replace(" ", "").replace("\n", ""))
        for secret in ("resume_cancel", "cli-password", "resume_token", "password"):
            self.assertNotIn(secret, cancelled.stdout)

    def test_stale_token_fails_closed_with_actionable_guidance(self):
        stale = self.run_cli("session", "auth", "resolve", "cli/auth fixture", input_text="stale-private-token\n")
        self.assertNotEqual(stale.returncode, 0)
        self.assertIn("browser_auth_cancel", stale.stderr)
        for secret in ("stale-private-token", "resume_server", "server-secret", "private_secret"):
            self.assertNotIn(secret, stale.stdout + stale.stderr)

    def test_stdin_token_takes_precedence_over_environment_fallback(self):
        env = {**self.env, "UIAI_AUTH_RESUME_TOKEN": "environment-token-must-not-win"}
        resolved = self.run_cli(
            "session", "auth", "resolve", "cli/auth fixture",
            input_text="stdin-private-token\n", env=env,
        )
        self.assertEqual(resolved.returncode, 0, resolved.stderr)
        self.assertEqual(AuthHandler.calls[-1]["body"], {"resume_token": "stdin-private-token"})
        self.assertNotIn("environment-token-must-not-win", resolved.stdout + resolved.stderr)


if __name__ == "__main__":
    unittest.main()
