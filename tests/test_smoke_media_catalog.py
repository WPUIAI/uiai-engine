import http.server
import json
import os
from pathlib import Path
import subprocess
import threading
import unittest

ROOT = Path(__file__).resolve().parent.parent


class MediaCatalogSmokeTests(unittest.TestCase):
    def probe(self, status, body, authenticated=False):
        seen = []
        class Handler(http.server.BaseHTTPRequestHandler):
            def do_GET(self):
                seen.append((self.path, self.headers.get("Authorization")))
                self.send_response(status)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(body.encode())
            def log_message(self, *args):
                pass
        server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        try:
            env = dict(os.environ, UIAI_ENGINE_URL=f"http://127.0.0.1:{server.server_port}")
            env.pop("UIAI_API_KEY", None)
            env.pop("UIAI_BEARER_TOKEN", None)
            if authenticated:
                env["UIAI_BEARER_TOKEN"] = "synthetic-test-token"
            result = subprocess.run(["bash", str(ROOT / "scripts/smoke-media-catalog.sh")], env=env, capture_output=True, text=True, timeout=8)
            self.assertEqual(seen[0][0], "/api/media/frame/catalog")
            self.assertNotIn("synthetic-test-token", result.stdout + result.stderr)
            return result, seen
        finally:
            server.shutdown()
            server.server_close()
            thread.join()

    def test_entitled_catalog(self):
        result, seen = self.probe(200, '{"count":3}', authenticated=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("catalog content verified", result.stdout)
        self.assertEqual(seen[0][1], "Bearer synthetic-test-token")

    def test_unentitled_route_is_not_a_false_failure(self):
        result, _ = self.probe(402, '{"error":"license_required"}')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("catalog content not verified", result.stdout)
        self.assertIn("HTTP 402", result.stdout)

    def test_real_route_auth_and_content_failures_still_fail(self):
        for status, body in [(404,"{}"),(500,"{}"),(401,"{}"),(200,"not-json"),(200,'{"count":0}'),(200,'{"count":"3"}')]:
            with self.subTest(status=status, body=body):
                result, _ = self.probe(status, body)
                self.assertNotEqual(result.returncode, 0)


if __name__ == "__main__":
    unittest.main()
