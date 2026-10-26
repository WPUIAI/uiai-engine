from pathlib import Path
import copy
import hashlib
import importlib.util
import io
import json
import plistlib
import tarfile
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
SPEC = importlib.util.spec_from_file_location('qualified_ota', ROOT/'scripts/verify-native-ota-artifact.py')
module = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(module)


class QualifiedArtifactTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.sha = 'a'*40
        self.platform = 'darwin-aarch64'
        self.version = '0.3.1'
        self.repo = 'fixture/fixture'
        self.updater = self.root/'release-proof/cockpit/updater'
        self.updater.mkdir(parents=True)
        self.archive = self.updater/'uaiengine-cockpit_0.3.1_aarch64.app.tar.gz'
        self.binary = b'SYNTHETIC BOUNDARY FIXTURE, NOT A NATIVE EXECUTION PROOF'
        self.build_archive()
        self.archive.with_name(self.archive.name+'.sig').write_text('fixture-signature')
        self.manifest = {'version':self.version,'platforms':{self.platform:{
            'signature':'fixture-signature', 'url':f'https://github.com/{self.repo}/releases/download/cockpit-latest/{self.archive.name}'}}}
        self.receipt = dict(schema='uiai.cockpit_native_ota_qualification.v1', status='passed',
            platform=self.platform,candidate_source_sha=self.sha,candidate_version=self.version,
            invalid_signature_rejected=True,automatic_retry_install_relaunch=True,
            preservation_marker_unchanged=True,owned_fixture_rollback=True,
            candidate_production_config_preserved=True,public_channel_changed=False,
            candidate_archive_sha256=module.sha256(self.archive),
            candidate_executable_sha256=hashlib.sha256(self.binary).hexdigest())
        dmg=self.root/'src-tauri/target/aarch64-apple-darwin/release/bundle/dmg/fixture.dmg'
        dmg.parent.mkdir(parents=True);dmg.write_bytes(b'fixture DMG')
        metadata={'version':self.version,'artifacts':[{'name':dmg.name,'platform':self.platform,'sha256':module.sha256(dmg)}]}
        (self.root/'release-proof/cockpit/cockpit-v0.3.1-metadata.json').write_text(json.dumps(metadata))
        self.save()

    def build_archive(self, version='0.3.1'):
        with tarfile.open(self.archive,'w:gz') as tar:
            entries={'Fixture.app/Contents/Info.plist':plistlib.dumps({'CFBundleExecutable':'fixture','CFBundleShortVersionString':version}),
                'Fixture.app/Contents/MacOS/fixture':self.binary}
            for name,raw in entries.items():
                member=tarfile.TarInfo(name);member.size=len(raw)
                tar.addfile(member,io.BytesIO(raw))

    def save(self):
        (self.root/'native-ota-receipt.json').write_text(json.dumps(self.receipt))
        (self.updater/'latest.json').write_text(json.dumps(self.manifest))

    def verify(self):
        return module.verify(self.root,self.platform,self.sha,self.version,self.repo)

    def test_exact_bound_artifacts_pass(self):
        self.assertEqual(self.verify(),self.receipt)

    def test_changed_archive_cannot_pass_even_when_running_executable_is_identical(self):
        with tarfile.open(self.archive, 'w:gz') as tar:
            entries = {'Fixture.app/Contents/Info.plist': plistlib.dumps({'CFBundleExecutable':'fixture','CFBundleShortVersionString':self.version}),
                       'Fixture.app/Contents/MacOS/fixture':self.binary,
                       'Fixture.app/Contents/Resources/changed.txt':b'changed resource'}
            for name,raw in entries.items():
                member=tarfile.TarInfo(name);member.size=len(raw)
                tar.addfile(member,io.BytesIO(raw))
        with self.assertRaisesRegex(ValueError, 'archive differs'):
            self.verify()

    def test_foreign_missing_or_false_qualification_is_rejected(self):
        original=copy.deepcopy(self.receipt)
        changes={'status':'failed','platform':'darwin-x86_64','candidate_source_sha':'b'*40,
            'candidate_version':'9.0.0','candidate_archive_sha256':'0'*64,
            'candidate_executable_sha256':'0'*64,'schema':'foreign'}
        for key in ('invalid_signature_rejected','automatic_retry_install_relaunch',
                    'preservation_marker_unchanged','owned_fixture_rollback','candidate_production_config_preserved'):
            changes[key]=False
        changes['public_channel_changed']=True
        for key,value in changes.items():
            with self.subTest(key=key):
                self.receipt=dict(original,**{key:value});self.save()
                with self.assertRaises(ValueError):self.verify()
        self.receipt=original
        del self.receipt['preservation_marker_unchanged'];self.save()
        with self.assertRaises(ValueError):self.verify()

    def test_replaced_bundle_signature_feed_and_native_version_are_rejected(self):
        original=copy.deepcopy(self.manifest)
        for changes in ({'signature':'foreign'},{'url':'https://foreign.invalid/bundle'}):
            self.manifest=copy.deepcopy(original);self.manifest['platforms'][self.platform].update(changes);self.save()
            with self.assertRaises(ValueError):self.verify()
        self.manifest=original;self.save()
        self.archive.write_bytes(b'rebuilt different bytes')
        with self.assertRaises(ValueError):self.verify()
        self.build_archive(version='0.3.0')
        self.receipt['candidate_archive_sha256']=module.sha256(self.archive);self.save()
        with self.assertRaises(ValueError):self.verify()

    def test_missing_receipt_or_dmg_is_rejected(self):
        (self.root/'native-ota-receipt.json').unlink()
        with self.assertRaises(OSError):self.verify()
        self.save()
        next((self.root/'src-tauri').rglob('*.dmg')).write_bytes(b'foreign DMG')
        with self.assertRaises(ValueError):self.verify()

    def test_publication_reuses_the_native_owner_without_rebuilding_mac_packages(self):
        release=(ROOT/'.github/workflows/cockpit-release.yml').read_text()
        reusable=(ROOT/'.github/workflows/cockpit.yml').read_text()
        self.assertIn('  workflow_call:',reusable)
        self.assertIn('uses: ./.github/workflows/cockpit.yml',release)
        self.assertIn('candidate_ref: ${{ needs.checks.outputs.candidate_sha }}',release)
        self.assertNotIn('  signed-macos-build:',release)
        self.assertNotIn('  signed-macos-build-arm64:',release)
        self.assertIn('needs: [checks, qualified-macos]',release)
        self.assertIn('verify-native-ota-artifact.py /tmp/cockpit-ota-x86',release)
        self.assertIn('verify-native-ota-artifact.py /tmp/cockpit-ota-arm64',release)
        self.assertLess(reusable.index('Freeze the complete candidate'),reusable.index('ref: ${{ inputs.baseline_ref }}'))
        qualified=reusable.split('- name: Preserve only qualified release packages',1)[1]
        self.assertNotIn('if: always()',qualified)
        self.assertIn('inputs.artifact_suffix || env.CANDIDATE_SOURCE_SHA',qualified)


if __name__=='__main__':unittest.main()
