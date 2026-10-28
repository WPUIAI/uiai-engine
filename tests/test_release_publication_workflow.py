"""Execute checked-in release shell steps against isolated, value-free GitHub fixtures.

These tests do not publish releases, verify signing keys, or prove native OTA.
"""
from pathlib import Path
import hashlib
import json
import os
import re
import subprocess
import shutil
import tempfile
import textwrap
import unittest

ROOT = Path(__file__).resolve().parents[1]
ENGINE = ROOT / '.github/workflows/engine-release.yml'
COCKPIT = ROOT / '.github/workflows/cockpit-release.yml'


def command(workflow, name):
    source = workflow.read_text()
    block = source.split('- name: ' + name + '\n', 1)[1]
    block = re.split(r'\n      - (?:name:|uses:)', block, maxsplit=1)[0]
    run = block.split('        run: |\n', 1)[1]
    lines = []
    for line in run.splitlines():
        if line and not line.startswith('          '):
            break
        lines.append(line)
    return textwrap.dedent('\n'.join(lines))


FAKE_GH = r'''#!/usr/bin/env python3
import json, os, pathlib, sys, shutil
args = sys.argv[1:]
with open(os.environ['GH_CALLS'], 'a') as log: log.write(json.dumps(args)+'\n')
operation = args[1] if len(args)>1 else ''
if operation in ('list','view'):
    if os.environ.get('READ_FAILURE') == '1': sys.exit(78)
    state=os.environ.get('DRAFT_STATE', 'true')
    if operation == 'list':
        values=[{'tagName':'cockpit-latest','isDraft':False}]
        if state: values.append({'tagName':os.environ['RELEASE_TAG'],'isDraft':state=='true'})
        print(json.dumps(values))
    else: print(state)
elif operation == 'download':
    target=pathlib.Path(args[args.index('--dir')+1]); target.mkdir(parents=True,exist_ok=True)
    if os.environ.get('OTA_CANDIDATE_DIR'):
        for file in pathlib.Path(os.environ['OTA_CANDIDATE_DIR']).glob('*.app.tar.gz*'): shutil.copyfile(file,target/file.name)
    else: (target/os.environ['ENGINE_ASSET_NAME']).write_bytes(os.environ.get('DOWNLOAD_BYTES','candidate').encode())
elif operation == 'upload':
    if os.environ.get('UPLOAD_FAILURE') == '1' or (args[2]=='cockpit-latest' and os.environ.get('FAIL_CHANNEL_UPLOAD')=='1'): sys.exit(79)
    skip=False
    files=[]
    for item in args[3:]:
        if skip: skip=False;continue
        if item=='--repo': skip=True;continue
        if item.startswith('--'):continue
        files.append(item)
    if not files or any(not pathlib.Path(item).is_file() for item in files): sys.exit(80)
'''


class ReleasePublicationTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        (self.root/'bin').mkdir()
        gh = self.root/'bin/gh'
        gh.write_text(FAKE_GH)
        gh.chmod(0o755)
        (self.root/'dist').mkdir()
        for name in ('uiai-engine-linux-amd64', 'uiai-engine-linux-amd64.sha256', 'release-notes.md'):
            (self.root/'dist'/name).write_text('candidate')
        self.env = dict(os.environ, PATH=str(self.root/'bin')+os.pathsep+os.environ['PATH'],
            BASH_ENV='/dev/null', ENV='/dev/null', GH_CALLS=str(self.root/'calls.jsonl'), RELEASE_TAG='engine-v2.0.24',
            ENGINE_ASSET_NAME='uiai-engine-linux-amd64', DRAFT_STATE='true',
            EXPECTED_SHA256=hashlib.sha256(b'candidate').hexdigest(),
            TAG='cockpit-v0.3.1', GITHUB_REPOSITORY='fixture/fixture')

    def calls(self):
        path = self.root/'calls.jsonl'
        return [json.loads(line) for line in path.read_text().splitlines()] if path.exists() else []

    def run_step(self, script, **env):
        return subprocess.run(['bash','-euo','pipefail','-c',script], cwd=self.root,
            env=dict(self.env, **env), capture_output=True, text=True, timeout=10)

    def stage(self):
        return command(ENGINE,'Stage draft candidate (never publish before verification)')

    def test_new_candidate_is_draft_not_public(self):
        result = self.run_step(self.stage(), DRAFT_STATE='')
        self.assertEqual(result.returncode,0,result.stderr)
        created = [x for x in self.calls() if x[:2]==['release','create']]
        self.assertEqual(len(created),1)
        self.assertIn('--draft',created[0])
        self.assertIn('--verify-tag',created[0])
        self.assertFalse(any('--draft=false' in x for x in self.calls()))

    def test_published_candidate_is_not_overwritten(self):
        result = self.run_step(self.stage(), DRAFT_STATE='false')
        self.assertNotEqual(result.returncode,0)
        self.assertTrue(all(x[1]=='list' for x in self.calls()))

    def test_failed_metadata_read_never_becomes_create(self):
        result = self.run_step(self.stage(), READ_FAILURE='1')
        self.assertNotEqual(result.returncode,0)
        self.assertEqual(len(self.calls()),1)

    def test_existing_draft_may_resume_without_publication(self):
        result = self.run_step(self.stage())
        self.assertEqual(result.returncode,0,result.stderr)
        self.assertTrue(any(x[1]=='upload' for x in self.calls()))
        self.assertFalse(any('--draft=false' in x for x in self.calls()))

    def test_publish_requires_exact_tested_binary(self):
        script=command(ENGINE,'Verify draft bytes and publish')
        result=self.run_step(script, DOWNLOAD_BYTES='different-candidate')
        self.assertNotEqual(result.returncode,0)
        self.assertFalse(any('--draft=false' in x for x in self.calls()))

    def test_matching_deployed_binary_can_be_promoted(self):
        result=self.run_step(command(ENGINE,'Verify draft bytes and publish'))
        self.assertEqual(result.returncode,0,result.stderr)
        edits=[x for x in self.calls() if '--draft=false' in x]
        self.assertEqual(len(edits),1)
        self.assertIn('--latest',edits[0])
        self.assertIn('--prerelease=false',edits[0])

    def test_missing_digest_cannot_promote(self):
        result=self.run_step(command(ENGINE,'Verify draft bytes and publish'),EXPECTED_SHA256='')
        self.assertNotEqual(result.returncode,0)
        self.assertFalse(any('--draft=false' in x for x in self.calls()))

    def test_publish_depends_on_successful_real_deployment(self):
        source=ENGINE.read_text()
        block=source.split('  publish-release:\n',1)[1]
        self.assertIn('needs: [build-release, deploy-ovh]',block)
        self.assertIn("needs.deploy-ovh.result == 'success'",block)
        self.assertIn('inputs.deploy_to_ovh && !inputs.dry_run',block)
        self.assertIn("if: ${{ github.event_name == 'push' || !inputs.dry_run }}",source)
        self.assertNotIn('scripts/next-version.py',source)
        self.assertIn('python3 scripts/verify-uiai-version-surfaces.py "$RELEASE_TAG"',source)

    def test_deploy_caller_supplies_an_explicit_idle_target_for_each_named_worker(self):
        scripts=self.root/'scripts';scripts.mkdir()
        stub=scripts/'deploy-engine-ovh.sh'
        stub.write_text('#!/bin/bash\nprintf \"%s\\n\" \"$REMOTE_IDLE_BASE_URLS\"\n')
        stub.chmod(0o755)
        env={key:'' for key in ('REMOTE_INSTALL_ROOT','REMOTE_SERVICE_NAME','REMOTE_HEALTH_URL','REMOTE_EXTRA_SERVICES','REMOTE_SMOKE_BASE_URLS','REMOTE_IDLE_BASE_URLS')}
        result=self.run_step(command(ENGINE,'Deploy to OVH worker'),**env)
        self.assertEqual(result.returncode,0,result.stderr)
        self.assertEqual(result.stdout.strip().split(),['http://127.0.0.1:7456','http://127.0.0.1:7460'])

    def ota_tree(self, arm_version='0.3.1'):
        for arch,version in [('x86','0.3.1'),('arm64',arm_version)]:
            base=self.root/('cockpit-ota-'+arch)
            updater=base/'release-proof/cockpit/updater'; updater.mkdir(parents=True)
            platform='darwin-x86_64' if arch=='x86' else 'darwin-aarch64'
            name='uaiengine-cockpit_0.3.1_'+arch+'.app.tar.gz'
            (updater/name).write_text('fixture NOT a signed native bundle')
            (updater/(name+'.sig')).write_text('fixture NOT a signature proof')
            (updater/'latest.json').write_text(json.dumps({'version':version,'platforms':{
                platform:{'signature':'fixture-only','url':'https://example.invalid/'+name}}}))
            (base/'release-proof/cockpit/proof.json').write_text('{}')
            dmg=base/'src-tauri/target/release/bundle/dmg';dmg.mkdir(parents=True)
            (dmg/(arch+'.dmg')).write_text('fixture')

    def ota_command(self,name):
        # Rewrite only the isolated runner output root; execute the actual shell.
        return command(COCKPIT,name).replace('/tmp/cockpit-ota',str(self.root/'cockpit-ota'))

    def test_both_platforms_are_merged_and_the_actual_download_directory_is_published(self):
        self.ota_tree()
        result=self.run_step(self.ota_command('Merge updater latest.json (x86_64 + aarch64)'))
        self.assertEqual(result.returncode,0,result.stderr)
        merged=json.loads((self.root/'cockpit-ota-x86/release-proof/cockpit/updater/latest.json').read_text())
        self.assertEqual(set(merged['platforms']),{'darwin-x86_64','darwin-aarch64'})
        result=self.run_step(self.ota_command('Stage immutable proof (channel unchanged)'))
        self.assertEqual(result.returncode,0,result.stderr)
        self.assertFalse(any(x[:3]==['release','upload','cockpit-latest'] for x in self.calls()))
        updater=self.root/'cockpit-ota-x86/release-proof/cockpit/updater'
        shutil.copytree(updater,self.root/'cockpit-ota-verified')
        result=self.run_step(self.ota_command('Promote immutable release and mutable channel'),OTA_CANDIDATE_DIR=str(updater))
        self.assertEqual(result.returncode,0,result.stderr)
        uploads=[x for x in self.calls() if x[:3]==['release','upload','cockpit-latest']]
        self.assertEqual(len(uploads),2)
        self.assertTrue(all(not x.endswith('latest.json') for x in uploads[0]))
        self.assertTrue(any(x.endswith('latest.json') for x in uploads[-1]))

    def test_preflight_and_publisher_share_the_strict_rust_gate(self):
        invocation='bash scripts/check-cockpit-rust.sh'
        self.assertIn(invocation, (ROOT/'scripts/local-uiai-preflight.sh').read_text())
        self.assertIn(invocation, command(COCKPIT,'Rust checks'))
        shared=(ROOT/'scripts/check-cockpit-rust.sh').read_text()
        self.assertIn('cargo test',shared)
        self.assertIn('cargo clippy -- -D warnings',shared)

    def test_cockpit_stays_draft_until_all_assets_and_checks_complete(self):
        source=COCKPIT.read_text()
        create=source.split('  create-release:\n',1)[1].split('  publish-macos-ota:\n',1)[0]
        self.assertIn('draft: true',create)
        self.assertIn('qualified-macos',create)
        promote=source.split('  promote-release:\n',1)[1]
        self.assertIn('needs: [create-release, publish-macos-ota, linux-binary]',promote)
        stage=command(COCKPIT,'Stage immutable proof (channel unchanged)')
        self.assertNotIn('cockpit-latest',stage)

    def test_stable_cockpit_publication_is_not_a_prerelease(self):
        self.ota_tree()
        result=self.run_step(self.ota_command('Merge updater latest.json (x86_64 + aarch64)'))
        self.assertEqual(result.returncode,0,result.stderr)
        updater=self.root/'cockpit-ota-x86/release-proof/cockpit/updater'
        shutil.copytree(updater,self.root/'cockpit-ota-verified')
        result=self.run_step(self.ota_command('Promote immutable release and mutable channel'),OTA_CANDIDATE_DIR=str(updater))
        self.assertEqual(result.returncode,0,result.stderr)
        edits=[x for x in self.calls() if x[:3]==['release','edit','cockpit-v0.3.1']]
        self.assertEqual(len(edits),1)
        self.assertIn('--prerelease=false',edits[0])
        self.assertIn('--latest=false',edits[0], 'Cockpit must not displace the parent Engine installer Latest')

    def test_dev_cockpit_publication_remains_a_prerelease(self):
        self.ota_tree()
        result=self.run_step(self.ota_command('Merge updater latest.json (x86_64 + aarch64)'))
        self.assertEqual(result.returncode,0,result.stderr)
        updater=self.root/'cockpit-ota-x86/release-proof/cockpit/updater'
        shutil.copytree(updater,self.root/'cockpit-ota-verified')
        result=self.run_step(self.ota_command('Promote immutable release and mutable channel'),OTA_CANDIDATE_DIR=str(updater),TAG='cockpit-v0.3.1-dev')
        self.assertEqual(result.returncode,0,result.stderr)
        edits=[x for x in self.calls() if x[:3]==['release','edit','cockpit-v0.3.1-dev']]
        self.assertEqual(len(edits),1)
        self.assertIn('--prerelease',edits[0])
        self.assertIn('--latest=false',edits[0])

    def test_failed_bundle_upload_does_not_advance_feed(self):
        self.ota_tree()
        result=self.run_step(self.ota_command('Merge updater latest.json (x86_64 + aarch64)'))
        self.assertEqual(result.returncode,0,result.stderr)
        updater=self.root/'cockpit-ota-x86/release-proof/cockpit/updater'
        shutil.copytree(updater,self.root/'cockpit-ota-verified')
        result=self.run_step(self.ota_command('Promote immutable release and mutable channel'),OTA_CANDIDATE_DIR=str(updater),FAIL_CHANNEL_UPLOAD='1')
        self.assertNotEqual(result.returncode,0)
        self.assertFalse(any(x[:3]==['release','upload','cockpit-latest'] and any(f.endswith('latest.json') for f in x) for x in self.calls()))

    def test_cross_version_platform_merge_fails(self):
        self.ota_tree(arm_version='0.3.2')
        result=self.run_step(self.ota_command('Merge updater latest.json (x86_64 + aarch64)'))
        self.assertNotEqual(result.returncode,0)
        self.assertEqual(self.calls(),[])

    def test_missing_arm_signature_cannot_silently_publish(self):
        self.ota_tree()
        next((self.root/'cockpit-ota-arm64/release-proof/cockpit/updater').glob('*.sig')).unlink()
        result=self.run_step(self.ota_command('Merge updater latest.json (x86_64 + aarch64)'))
        self.assertNotEqual(result.returncode,0)
        self.assertEqual(self.calls(),[])


if __name__=='__main__': unittest.main()
