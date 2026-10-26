"""Existing CLI error renderer must preserve the shared recovery field."""
import json
import os
from pathlib import Path
import subprocess
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

ROOT = Path(__file__).resolve().parents[1]

class SearchRecoveryCLI(unittest.TestCase):
    def test_error_renderer_keeps_safe_next_action(self):
        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *_args):
                return None  # Owned fixture logs contain no useful evidence.

            def do_GET(self):
                body = json.dumps({
                    "error": "search_provider_unavailable",
                    "message": "Existing reference missing",
                    "suggested_next_action": "connect_existing_credential_reference: search.providers.brave.api_key_env; GET /api/search/providers",
                    "private_extra": "synthetic-private-extra-never-echo",
                }).encode()
                self.send_response(503)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)

        server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        thread = threading.Thread(target=server.serve_forever)
        thread.start()
        try:
            env = dict(os.environ, UIAI_ENGINE_URL=f"http://127.0.0.1:{server.server_port}",
                       UIAI_API_KEY="", UIAI_BEARER_TOKEN="", UIAI_EVIDENCE_SCOPE_JSON="")
            result = subprocess.run(["bash", str(ROOT / "scripts/uiai"), "--json", "tools", "agent-card"],
                                    env=env, capture_output=True, text=True, timeout=10)
            self.assertEqual(result.returncode, 1)
            self.assertIn("next: connect_existing_credential_reference", result.stderr)
            self.assertIn("search.providers.brave.api_key_env", result.stderr)
            self.assertNotIn("synthetic-private-extra-never-echo", result.stdout + result.stderr)
        finally:
            server.shutdown()
            server.server_close()
            thread.join(timeout=3)

if __name__ == "__main__":
    unittest.main()
