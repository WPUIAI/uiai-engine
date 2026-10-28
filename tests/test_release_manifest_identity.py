"""Run the canonical manifest gate against independent, owned Git fixtures."""
import datetime
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]


class ReleaseManifestIdentityTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.git('init', '-q')
        self.git('config', 'user.name', 'Owned fixture')
        self.git('config', 'user.email', 'fixture@example.invalid')
        for rel, body in [('cmd/uiai-engine/main.go', 'package main\nvar version = "2.0.24"\n'),
                          ('apps/cockpit/package.json', '{"version":"0.3.1"}'), ('sentinel', 'initial')]:
            p = self.root / rel
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_text(body)
        self.git('add', '.')
        self.git('commit', '-qm', 'fixture: parent')
        self.parent = self.git('rev-parse', 'HEAD').stdout.strip()

    def git(self, *args):
        return subprocess.run(['git', *args], cwd=self.root, check=True, capture_output=True, text=True)

    def manifest(self, source):
        p = self.root / 'docs/distribution-manifest.json'
        p.parent.mkdir(exist_ok=True)
        p.write_text(json.dumps({'engine_version':'2.0.24', 'cockpit_version':'0.3.1',
            'source_commit':source, 'generated_at':datetime.datetime.now(datetime.timezone.utc).isoformat()}))
        self.git('add', '.')
        self.git('commit', '-qm', 'fixture: manifest')

    def gate(self, fast=False):
        source = (ROOT / 'scripts/local-uiai-preflight.sh').read_text()
        code = source.split("python3 << 'PYFRESH'\n", 1)[1].split('\nPYFRESH', 1)[0]
        env = dict(os.environ)
        env.pop('PREFLIGHT_FAST', None)
        if fast:
            env['PREFLIGHT_FAST'] = '1'
        return subprocess.run([sys.executable, '-c', code], cwd=self.root, env=env,
            capture_output=True, text=True, timeout=10)

    def test_short_parent_identity_ignores_display_abbreviation(self):
        self.git('config', 'core.abbrev', '40')
        self.manifest(self.parent[:7])
        result = self.gate()
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_full_parent_identity_works_with_short_display(self):
        self.git('config', 'core.abbrev', '7')
        self.manifest(self.parent)
        result = self.gate()
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_strict_still_rejects_an_older_ancestor(self):
        self.manifest(self.parent[:7])
        (self.root / 'sentinel').write_text('later commit')
        self.git('add', '.')
        self.git('commit', '-qm', 'fixture: later')
        self.assertNotEqual(self.gate().returncode, 0)

    def test_fast_still_allows_a_verified_ancestor(self):
        self.manifest(self.parent[:7])
        (self.root / 'sentinel').write_text('later commit')
        self.git('add', '.')
        self.git('commit', '-qm', 'fixture: later')
        result = self.gate(fast=True)
        self.assertEqual(result.returncode, 0, result.stderr)


if __name__ == '__main__':
    unittest.main()
