from pathlib import Path
import base64
import importlib.util
import json
import os
import subprocess
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT/'apps/cockpit/scripts/qualify-native-ota.py'
spec = importlib.util.spec_from_file_location('native_ota', SCRIPT)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class NativeQualificationBoundaryTests(unittest.TestCase):
    def test_signature_negative_control_changes_signature_not_trust_key(self):
        signature = bytes(range(74))
        record = 'untrusted comment: synthetic signature fixture\n'+base64.b64encode(signature).decode()+'\ntrusted comment: fixture\nunchanged-global-signature\n'
        value = base64.b64encode(record.encode()).decode()
        changed = base64.b64decode(module.corrupt_signature(value)).decode().splitlines()
        original = record.splitlines()
        self.assertEqual(changed[0], original[0])
        self.assertEqual(changed[2:], original[2:])
        actual = base64.b64decode(changed[1])
        self.assertEqual(actual[:-1], signature[:-1])
        self.assertEqual(actual[-1], signature[-1] ^ 1)

    def test_exact_owned_pid_match_supports_spaces_without_matching_another_app(self):
        executable = Path('/tmp/native fixture/UIAI Engine Cockpit.app/Contents/MacOS/uaiengine-cockpit')
        rows = f' 12 {executable}\n 13 {executable} --fixture\n 14 {executable}-other\n 15 /unrelated/app\n'
        with patch.object(module.subprocess, 'check_output', return_value=rows):
            self.assertEqual(module.pids(executable), [12,13])

    def test_non_mac_execution_fails_before_app_or_credential_use(self):
        args = ['python3', str(SCRIPT), '--app','/unrelated.app','--candidate','/absent',
            '--signature','/absent.sig','--baseline-version','0.3.0',
            '--candidate-version','0.3.1','--platform','darwin-aarch64',
            '--identifier','fixture','--output','/absent-output']
        if module.platform.system() != 'Darwin':
            result = subprocess.run(args, capture_output=True, text=True, timeout=5)
            self.assertNotEqual(result.returncode,0)
            self.assertIn('isolated hosted macOS runner',result.stderr)

    def test_customer_app_path_cannot_be_used_as_a_fixture(self):
        with tempfile.TemporaryDirectory() as tmp:
            args = [str(SCRIPT),'--app','/Applications/Customer.app','--candidate','/absent',
                '--signature','/absent.sig','--baseline-version','0.3.0',
                '--candidate-version','0.3.1','--platform','darwin-aarch64',
                '--identifier','fixture','--output','/absent-output']
            with patch.dict(os.environ, {'GITHUB_ACTIONS':'true','RUNNER_TEMP':tmp}), \
                 patch.object(module.platform,'system',return_value='Darwin'), \
                 patch('sys.argv',args):
                with self.assertRaisesRegex(SystemExit,'exact owned fixture'):
                    module.main()

    def test_qualification_has_no_publish_permission_or_duplicate_checkout(self):
        workflow=(ROOT/'.github/workflows/cockpit.yml').read_text()
        job=workflow.split('  native-ota-qualification:\n',1)[1]
        self.assertIn('contents: read',job)
        self.assertNotIn('contents: write',job)
        self.assertNotIn('gh release create',job)
        self.assertNotIn('gh release upload',job)
        self.assertNotIn('git clone',job)
        self.assertIn('macos-15-intel',job)
        self.assertIn('macos-14',job)
        self.assertIn('TAURI_SIGNING_PRIVATE_KEY: ${{ secrets.TAURI_SIGNING_PRIVATE_KEY }}',job)
        consumer=job.split('- name: Real native invalid-signature',1)[1]
        self.assertNotIn('TAURI_SIGNING_PRIVATE_KEY: ${{',consumer)
        self.assertIn('Signing credentials are deliberately NOT',consumer)
        self.assertIn('Existing updater trust key changed',job)
        self.assertIn('baseline_source_sha',SCRIPT.read_text())
        self.assertIn("'historical_public_binary_upgrade': 'not claimed'",SCRIPT.read_text())


    def test_canonical_stamp_aligns_native_ui_lock_and_manifest_without_changing_engine(self):
        owner_spec=importlib.util.spec_from_file_location('stamp_owner',ROOT/'scripts/stamp-uiai-version.py')
        owner=importlib.util.module_from_spec(owner_spec);owner_spec.loader.exec_module(owner)
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            values={
                'apps/cockpit/package.json':{'name':'fixture','version':'0.1.0-dev'},
                'apps/cockpit/package-lock.json':{'version':'0.1.0-dev','packages':{'':{'version':'0.1.0-dev'}},'lockfileVersion':3},
                'apps/cockpit/src-tauri/tauri.conf.json':{'version':'0.1.0-dev','plugins':{'updater':{'pubkey':'unchanged-fixture-public-key'}}},
            }
            for path,value in values.items():
                file=root/path;file.parent.mkdir(parents=True,exist_ok=True);file.write_text(json.dumps(value))
            (root/'apps/cockpit/src-tauri/Cargo.toml').write_text('[package]\nname = "uaiengine-cockpit"\nversion = "0.1.0-dev"\n')
            (root/'cmd/uiai-engine').mkdir(parents=True)
            (root/'cmd/uiai-engine/main.go').write_text('package main\nvar version = "2.0.24"\n')
            with patch.object(owner,'ROOT',root),patch.object(owner.subprocess,'check_output',return_value=b'fixture-sha'),patch('sys.argv',['stamp','cockpit-v0.3.1']):
                self.assertEqual(owner.main(),0)
            self.assertEqual(json.loads((root/'apps/cockpit/package.json').read_text())['version'],'0.3.1')
            lock=json.loads((root/'apps/cockpit/package-lock.json').read_text())
            self.assertEqual(lock['version'],'0.3.1');self.assertEqual(lock['packages']['']['version'],'0.3.1')
            config=json.loads((root/'apps/cockpit/src-tauri/tauri.conf.json').read_text())
            self.assertEqual(config['version'],'0.3.1')
            self.assertEqual(config['plugins']['updater']['pubkey'],'unchanged-fixture-public-key')
            self.assertIn('version = "0.3.1"',(root/'apps/cockpit/src-tauri/Cargo.toml').read_text())
            ui=(root/'apps/cockpit/src/lib/version.ts').read_text()
            self.assertIn('APP_VERSION = "0.3.1"',ui);self.assertIn('APP_CHANNEL = "stable"',ui)
            self.assertIn('"2.0.24"',(root/'cmd/uiai-engine/main.go').read_text())
            manifest=json.loads((root/'docs/distribution-manifest.json').read_text())
            self.assertEqual(manifest['cockpit_version'],'0.3.1');self.assertEqual(manifest['engine_version'],'2.0.24')

    def test_legacy_release_entry_uses_one_version_owner_and_portable_path_resolution(self):
        adapter=(ROOT/'apps/cockpit/scripts/stamp-cockpit-version/stamp-cockpit-version').read_text()
        self.assertIn('scripts/stamp-uiai-version.py',adapter)
        self.assertNotIn('cat >',adapter)
        release=(ROOT/'apps/cockpit/scripts/cockpit-release.sh').read_text()
        self.assertNotIn('readlink -f',release)
        self.assertIn('BASH_SOURCE[0]',release)



class NativeWindowCaptureBoundaryTests(unittest.TestCase):
    def fixture(self, root):
        executable = root/'native-ota/install/UIAI Engine Cockpit.app/Contents/MacOS/fixture'
        executable.parent.mkdir(parents=True)
        executable.write_bytes(b'owned fixture, not an executable')
        output = root/'candidate-window.png'
        return executable, output

    def test_wrong_scope_never_queries_or_captures_a_desktop(self):
        with tempfile.TemporaryDirectory() as tmp, patch.object(module.subprocess, 'check_output') as query, patch.object(module.subprocess, 'run') as capture:
            result = module.capture_owned_window(Path('/Applications/Unrelated.app/Contents/MacOS/app'), Path(tmp)/'out.png', Path(tmp))
            self.assertEqual(result['status'], 'blocked')
            query.assert_not_called(); capture.assert_not_called()

    def test_foreign_window_never_becomes_a_fallback_capture(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp); executable, output = self.fixture(root)
            with patch.object(module, 'pids', return_value=[42]), patch.object(module.time, 'sleep'), patch.object(module.subprocess, 'check_output', return_value='[{"id":11,"pid":43}]'), patch.object(module.subprocess, 'run') as capture:
                result = module.capture_owned_window(executable, output, root)
            self.assertEqual(result['status'], 'unavailable'); capture.assert_not_called()
            self.assertFalse(output.exists())

    def test_owned_capture_binds_pid_window_and_digest_without_claiming_pixels(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp); executable, output = self.fixture(root)
            def fake_query(command, **_):
                # CoreGraphics returns CFArrayRef, not an already-unwrapped JS
                # array; the native bridge conversion is required before filter.
                self.assertIn('ObjC.castRefToObject($.CGWindowListCopyWindowInfo', command[4])
                self.assertEqual(json.loads(command[5]), [42])
                return '[{"id":11,"pid":42}]'
            def fake_capture(command, **_):
                self.assertEqual(command[:5], ['/usr/sbin/screencapture', '-x', '-o', '-l', '11'])
                Path(command[-1]).write_bytes(b'\x89PNG\r\n\x1a\nsynthetic header fixture')
            with patch.object(module, 'pids', return_value=[42]), patch.object(module.time, 'sleep'), patch.object(module.subprocess, 'check_output', side_effect=fake_query), patch.object(module.subprocess, 'run', side_effect=fake_capture):
                result = module.capture_owned_window(executable, output, root)
            self.assertEqual(result['status'], 'captured'); self.assertEqual(result['owner_pid'], 42)
            self.assertEqual(result['sha256'], module.sha(output))
            self.assertEqual(result['visual_verification'], 'pending_pixel_inspection')

    def test_exited_fixture_and_existing_output_are_not_reused(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp); executable, output = self.fixture(root)
            with patch.object(module, 'pids', side_effect=[[42], []]), patch.object(module.time, 'sleep'), patch.object(module.subprocess, 'check_output', return_value='[{"id":11,"pid":42}]'), patch.object(module.subprocess, 'run') as capture:
                result = module.capture_owned_window(executable, output, root)
            self.assertEqual(result['status'], 'unavailable'); capture.assert_not_called()
            output.write_bytes(b'prior artifact')
            with patch.object(module, 'pids') as lookup:
                result = module.capture_owned_window(executable, output, root)
            self.assertEqual(result['status'], 'blocked'); lookup.assert_not_called()
            self.assertEqual(output.read_bytes(), b'prior artifact')

    def test_os_capture_failure_is_recorded_without_changing_permissions(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp); executable, output = self.fixture(root)
            error = subprocess.CalledProcessError(1, ['screencapture'], stderr=b'fixture permission unavailable')
            with patch.object(module, 'pids', return_value=[42]), patch.object(module.time, 'sleep'), patch.object(module.subprocess, 'check_output', return_value='[{"id":11,"pid":42}]'), patch.object(module.subprocess, 'run', side_effect=error):
                result = module.capture_owned_window(executable, output, root)
            self.assertEqual(result['status'], 'unavailable')
            self.assertIn('fixture permission unavailable', (root/result['diagnostics']).read_text())
            self.assertNotIn('sha256', result)


if __name__=='__main__': unittest.main()
