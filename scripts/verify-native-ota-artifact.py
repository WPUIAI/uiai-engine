#!/usr/bin/env python3
"""Verify that publication consumes the exact bytes exercised by native OTA CI.

This checks a trusted CI receipt; it does not mint native-test or signing authority.
"""
import hashlib
import json
from pathlib import Path, PurePosixPath
import plistlib
import re
import sys
import tarfile


def sha256(path):
    digest = hashlib.sha256()
    with path.open('rb') as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            digest.update(chunk)
    return digest.hexdigest()


def verify(root, platform, source_sha, version, repository):
    if platform not in ('darwin-x86_64', 'darwin-aarch64'):
        raise ValueError('Unsupported native platform')
    if not re.fullmatch(r'[0-9a-f]{40}', source_sha) or not re.fullmatch(r'\d+\.\d+\.\d+', version):
        raise ValueError('Expected immutable source SHA and stable native version')
    if not re.fullmatch(r'[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+', repository):
        raise ValueError('Invalid repository identity')
    root = Path(root)
    receipt = json.loads((root / 'native-ota-receipt.json').read_text())
    expected = dict(schema='uiai.cockpit_native_ota_qualification.v1', status='passed',
                    platform=platform, candidate_source_sha=source_sha, candidate_version=version)
    for key, value in expected.items():
        if receipt.get(key) != value:
            raise ValueError('Native qualification binding mismatch: ' + key)
    for key in ('invalid_signature_rejected', 'automatic_retry_install_relaunch',
                'preservation_marker_unchanged', 'owned_fixture_rollback',
                'candidate_production_config_preserved'):
        if receipt.get(key) is not True:
            raise ValueError('Native qualification predicate missing: ' + key)
    if receipt.get('public_channel_changed') is not False:
        raise ValueError('Qualification must not mutate the public channel')
    updater = root / 'release-proof/cockpit/updater'
    manifest = json.loads((updater / 'latest.json').read_text())
    if manifest['version'] != version or set(manifest['platforms']) != {platform}:
        raise ValueError('Candidate manifest version/platform mismatch')
    arch = platform.removeprefix('darwin-')
    name = f'uaiengine-cockpit_{version}_{arch}.app.tar.gz'
    archive = updater / name
    signature = archive.with_name(name + '.sig').read_text().strip()
    entry = manifest['platforms'][platform]
    if not signature or entry['signature'] != signature:
        raise ValueError('Manifest signature differs from qualified bundle signature')
    if entry['url'] != f'https://github.com/{repository}/releases/download/cockpit-latest/{name}':
        raise ValueError('Candidate manifest points outside the approved production feed')
    if sha256(archive) != receipt['candidate_archive_sha256']:
        raise ValueError('Publication archive differs from native-tested bytes')
    with tarfile.open(archive, 'r:gz') as tar:
        members = tar.getmembers()
        if any(PurePosixPath(m.name).is_absolute() or '..' in PurePosixPath(m.name).parts for m in members):
            raise ValueError('Unsafe candidate archive path')
        plists = [m for m in members if m.isfile() and m.name.endswith('.app/Contents/Info.plist')]
        if len(plists) != 1:
            raise ValueError('Expected exactly one native app bundle')
        info = plistlib.loads(tar.extractfile(plists[0]).read())
        executable = info['CFBundleExecutable']
        if not executable or PurePosixPath(executable).name != executable:
            raise ValueError('Invalid bundle executable')
        if info['CFBundleShortVersionString'] != version:
            raise ValueError('Native bundle version differs from tested candidate')
        binary = plists[0].name.removesuffix('Info.plist') + 'MacOS/' + executable
        digest = hashlib.sha256()
        with tar.extractfile(binary) as stream:
            for chunk in iter(lambda: stream.read(1024 * 1024), b''):
                digest.update(chunk)
        if digest.hexdigest() != receipt['candidate_executable_sha256']:
            raise ValueError('Publication executable differs from native-running bytes')
    dmgs = list((root / 'src-tauri').rglob('*.dmg'))
    if len(dmgs) != 1:
        raise ValueError('Expected exactly one candidate DMG')
    metadata_files = list((root / 'release-proof/cockpit').glob('*-metadata.json'))
    if len(metadata_files) != 1:
        raise ValueError('Expected exactly one candidate release metadata file')
    metadata = json.loads(metadata_files[0].read_text())
    matches = [item for item in metadata['artifacts'] if item['name'] == dmgs[0].name and item['platform'] == platform]
    if metadata['version'] != version or len(matches) != 1 or matches[0]['sha256'] != sha256(dmgs[0]):
        raise ValueError('Candidate DMG differs from the frozen release build')
    return receipt


def main():
    if len(sys.argv) != 6:
        raise SystemExit('usage: verify-native-ota-artifact.py ROOT PLATFORM SHA VERSION OWNER/REPO')
    try:
        verify(*sys.argv[1:])
    except (ValueError, KeyError, OSError, tarfile.TarError) as exc:
        raise SystemExit('Native-qualified publication rejected: ' + str(exc)) from exc
    print('native_qualified_publication=passed')


if __name__ == '__main__':
    main()
