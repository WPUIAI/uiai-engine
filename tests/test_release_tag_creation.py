"""Exercise the real tag creator against owned local Git fixtures; never publish."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]


class ReleaseTagCreationTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name) / 'source'
        self.root.mkdir()
        self.remote = Path(self.tmp.name) / 'remote.git'
        self.git('init', '-q')
        self.git('config', 'user.name', 'Owned fixture')
        self.git('config', 'user.email', 'fixture@example.invalid')
        subprocess.run(['git', 'init', '--bare', '-q', str(self.remote)], check=True)
        self.git('remote', 'add', 'origin', str(self.remote))
        scripts = self.root / 'scripts'
        scripts.mkdir()
        shutil.copyfile(ROOT / 'scripts/create-cockpit-dev-release-tag.sh', scripts / 'create-cockpit-dev-release-tag.sh')
        (scripts / 'stamp-uiai-version.py').write_text("from pathlib import Path\nPath('.stamp-called').write_text('called')\n")
        (scripts / 'verify-uiai-version-surfaces.py').write_text('pass\n')
        (scripts / 'local-uiai-preflight.sh').write_text('#!/bin/bash\nprintf "%s" "$1" > .strict-called\nexit "${PREFLIGHT_RESULT:-0}"\n')
        (self.root / 'sentinel.txt').write_text('preserve original work')
        self.git('add', '.')
        self.git('commit', '-qm', 'fixture: initial')
        self.head = self.git('rev-parse', 'HEAD').stdout.strip()
        binary = self.root / 'bin'
        binary.mkdir()
        gh = binary / 'gh'
        gh.write_text('#!/usr/bin/env python3\nimport json,os,sys\nif sys.argv[1:3]==["run","list"]:print(json.dumps([{"databaseId":17,"headSha":os.environ["FIXTURE_SHA"]}]))\n')
        with gh.open('a') as stream:
            stream.write('if sys.argv[1:3]==["run","watch"]:sys.exit(int(os.environ.get("CI_RESULT","0")))\n')
        gh.chmod(0o755)
        self.env = dict(os.environ, PATH=str(binary)+os.pathsep+os.environ['PATH'], FIXTURE_SHA=self.head, BASH_ENV='/dev/null')

    def git(self, *args):
        return subprocess.run(['git', *args], cwd=self.root, check=True, capture_output=True, text=True)

    def run_creator(self, *args, **env):
        return subprocess.run(['bash', 'scripts/create-cockpit-dev-release-tag.sh', *args], cwd=self.root,
            env=dict(self.env, **env), capture_output=True, text=True, timeout=15)

    def test_dry_run_does_not_execute_writer_or_touch_existing_work(self):
        sentinel = self.root / 'sentinel.txt'
        sentinel.write_text('other actor edits')
        before = self.git('diff').stdout
        result = self.run_creator('--tag', 'cockpit-v0.3.1', '--dry-run')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse((self.root / '.stamp-called').exists())
        self.assertEqual(self.git('diff').stdout, before)
        self.assertEqual(self.git('diff', '--cached').stdout, '')

    def test_prepared_parent_and_child_tags_share_one_verified_source(self):
        result = self.run_creator('--tag', 'engine-v2.0.24', '--tag', 'cockpit-v0.3.1', '--tag-only', '--push')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse((self.root / '.stamp-called').exists())
        self.assertEqual((self.root / '.strict-called').read_text(), '--strict')
        for tag in ('engine-v2.0.24', 'cockpit-v0.3.1'):
            actual = subprocess.check_output(['git', '--git-dir='+str(self.remote), 'rev-parse', tag], text=True).strip()
            self.assertEqual(actual, self.head)

    def test_failed_strict_preflight_creates_no_tags(self):
        result = self.run_creator('--tag', 'engine-v2.0.24', '--tag-only', '--push', PREFLIGHT_RESULT='31')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.git('tag', '--list').stdout, '')

    def test_failed_exact_source_ci_creates_no_tags(self):
        result = self.run_creator('--tag', 'engine-v2.0.24', '--tag-only', '--push', CI_RESULT='32')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.git('tag', '--list').stdout, '')

    def test_existing_immutable_tag_is_never_replaced(self):
        self.git('tag', 'cockpit-v0.3.1')
        result = self.run_creator('--tag', 'cockpit-v0.3.1', '--tag-only', '--push')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.git('rev-parse', 'cockpit-v0.3.1').stdout.strip(), self.head)


if __name__ == '__main__':
    unittest.main()
