import json
import subprocess
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

SCRIPT = Path(__file__).resolve().parent / 'deploy-engine-ovh.sh'

class DeploymentGuards(unittest.TestCase):
    def test_idle_checks_fail_closed(self):
        source = SCRIPT.read_text().split("<<'IDLE_PY'\n", 1)[1].split('\nIDLE_PY', 1)[0]
        fixture = {'count': 0, 'active_pages': 0}
        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                payload = {'count': fixture['count']} if self.path == '/api/session' else {'active_pages': fixture['active_pages']}
                self.send_response(200); self.end_headers(); self.wfile.write(json.dumps(payload).encode())
            def log_message(self, *args): pass
        server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
        thread = threading.Thread(target=server.serve_forever, daemon=True); thread.start()
        try:
            base = 'http://127.0.0.1:' + str(server.server_port)
            for sessions, pages, allowed in [(0, 0, True), (1, 0, False), (0, 1, False), (None, 0, False)]:
                fixture.update(count=sessions, active_pages=pages)
                result = subprocess.run(['python3', '-c', source, base], capture_output=True, timeout=10)
                self.assertEqual(result.returncode == 0, allowed)
        finally:
            server.shutdown(); server.server_close(); thread.join()

    def test_ssh_arguments_preserve_multiple_idle_targets(self):
        source = SCRIPT.read_text()
        self.assertIn('printf -v REMOTE_DEPLOY_COMMAND', source)
        value = 'http://127.0.0.1:7456 http://127.0.0.1:7460'
        shell = 'DEPLOY_ARGS=(printf "<%s>" "$1"); printf -v command "%q " "${DEPLOY_ARGS[@]}"; bash -c "$command"'
        result = subprocess.run(['bash', '-c', shell, 'fixture', value], capture_output=True, text=True, timeout=5)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout, '<' + value + '>')

    def test_home_writes_use_owner_adapter(self):
        source = SCRIPT.read_text()
        for command in ['owner_run install -d', 'owner_run cp -a', 'owner_run install -m']:
            self.assertIn(command, source)
        self.assertNotIn('install -o "$owner"', source)
        self.assertIn('assert_idle\nsystemctl restart "$service_name"', source)
        self.assertIn('assert_idle\n  systemctl restart "$extra"', source)

if __name__ == '__main__': unittest.main()
