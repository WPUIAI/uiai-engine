"""Isolated upgrade fixtures: all credential strings are synthetic, never provider credentials."""
from pathlib import Path
import json
import os
import stat
import subprocess
import tempfile
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import unittest

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / 'scripts/install-agent-integrations.sh'

class IntegrationAuthenticationUpgradeTests(unittest.TestCase):
    def run_install(self, initial=None, overrides=None):
        temp = tempfile.TemporaryDirectory(prefix='uiai-owned-auth-install-')
        self.addCleanup(temp.cleanup)
        root = Path(temp.name)
        config = root / 'mcp.json'
        if initial is not None:
            config.write_text(json.dumps(initial))
            config.chmod(0o600)
        env = {key: value for key, value in os.environ.items() if not key.startswith('UIAI_')}
        env.update(HOME=str(root), UIAI_INSTALL_PI='0', UIAI_MCP_CONFIG_DEST=str(config), UIAI_MCP_BUNDLE_ROOT=str(root / 'runtime'))
        env.update(overrides or {})
        result = subprocess.run(['bash', '-c', 'umask 022; exec bash "$1"', 'owned-fixture', str(SCRIPT)], env=env, capture_output=True, text=True, timeout=10)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        return config, json.loads(config.read_text()), result

    def existing(self):
        return {'mcpServers': {'uiai-browser': {'command': 'node', 'args': ['/old/owned-fixture.mjs'], 'env': {
            'UIAI_ENGINE_URL': 'https://existing-engine.fixture.invalid',
            'UIAI_API_KEY': 'synthetic-old-api-key',
            'UIAI_BEARER_TOKEN': 'synthetic-old-bearer',
            'NO_PROXY': 'owned-fixture.invalid'}}, 'unrelated': {'command': 'synthetic-other-server'}}}

    def test_upgrade_preserves_saved_authentication_and_endpoint_without_environment_overrides(self):
        initial = self.existing()
        _, actual, _ = self.run_install(initial)
        self.assertEqual(actual['mcpServers']['uiai-browser']['env'], initial['mcpServers']['uiai-browser']['env'])
        self.assertEqual(actual['mcpServers']['unrelated'], initial['mcpServers']['unrelated'])

    def test_explicit_credential_and_endpoint_updates_do_not_discard_unrelated_environment(self):
        changes = {'UIAI_ENGINE_URL': 'https://new-engine.fixture.invalid', 'UIAI_API_KEY': 'synthetic-new-api-key', 'UIAI_BEARER_TOKEN': 'synthetic-new-bearer'}
        _, actual, result = self.run_install(self.existing(), changes)
        expected = {**self.existing()['mcpServers']['uiai-browser']['env'], **changes}
        self.assertEqual(actual['mcpServers']['uiai-browser']['env'], expected)
        for value in ('synthetic-new-api-key', 'synthetic-new-bearer'):
            self.assertNotIn(value, result.stdout + result.stderr)

    def test_upgrade_does_not_make_a_private_authentication_config_world_readable(self):
        config, _, _ = self.run_install(self.existing())
        self.assertEqual(stat.S_IMODE(config.stat().st_mode), 0o600)

    def test_upgraded_installed_mcp_consumer_uses_preserved_authentication(self):
        captured = []
        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                captured.append((self.headers.get('X-API-Key'), self.headers.get('Authorization')))
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(b'{"status":"owned-authentication-fixture"}')
            do_POST = do_GET
            def log_message(self, *_):
                pass  # Synthetic fixture only; no credential logging.
        server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        try:
            initial = self.existing()
            initial['mcpServers']['uiai-browser']['env']['UIAI_ENGINE_URL'] = f'http://127.0.0.1:{server.server_port}'
            _, installed, _ = self.run_install(initial)
            entry = installed['mcpServers']['uiai-browser']
            env = {key: value for key, value in os.environ.items() if not key.startswith('UIAI_')}
            env.update(entry['env'])
            request = {'jsonrpc': '2.0', 'id': 1, 'method': 'tools/call', 'params': {'name': 'browser_read', 'arguments': {'session_id': 'owned-fixture'}}}
            child = subprocess.Popen([entry['command'], *entry['args']], env=env, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
            try:
                stdout, stderr = child.communicate(json.dumps(request)+'\n', timeout=10)
            except subprocess.TimeoutExpired:
                child.kill()
                child.communicate()
                self.fail('Installed MCP authentication fixture timed out')
            self.assertEqual(child.returncode, 0, stderr)
            reply = json.loads(stdout.strip())
            self.assertFalse(reply['result'].get('isError', False), stdout)
            self.assertEqual(captured, [('synthetic-old-api-key', 'Bearer synthetic-old-bearer')])
            self.assertNotIn('synthetic-old-api-key', stdout + stderr)
            self.assertNotIn('synthetic-old-bearer', stdout + stderr)
        finally:
            server.shutdown()
            server.server_close()
            thread.join(timeout=2)

    def test_fresh_authentication_config_is_private_even_under_a_permissive_umask(self):
        config, actual, result = self.run_install(overrides={'UIAI_API_KEY': 'synthetic-new-api-key'})
        self.assertEqual(actual['mcpServers']['uiai-browser']['env']['UIAI_API_KEY'], 'synthetic-new-api-key')
        self.assertEqual(stat.S_IMODE(config.stat().st_mode), 0o600)
        self.assertNotIn('synthetic-new-api-key', result.stdout + result.stderr)

if __name__ == '__main__':
    unittest.main()
