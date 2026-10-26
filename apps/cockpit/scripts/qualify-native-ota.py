#!/usr/bin/env python3
"""Native, isolated Cockpit OTA canary; never installs on a customer device.

Runs the real app/updater in a fresh hosted macOS runner. Baseline is a declared
source/version canary, not a claim about an unidentified historical public binary.
Only the baseline uses a private loopback feed; the candidate retains production
configuration and the existing verification key. No signature bypass exists.
"""
import argparse
import base64
import hashlib
import http.server
import json
import os
from pathlib import Path
import platform
import shutil
import signal
import subprocess
import tarfile
import threading
import time


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def corrupt_signature(value):
    lines = base64.b64decode(value, validate=True).decode().splitlines()
    signature = bytearray(base64.b64decode(lines[1], validate=True))
    signature[-1] ^= 1
    lines[1] = base64.b64encode(signature).decode()
    return base64.b64encode(('\n'.join(lines) + '\n').encode()).decode()


def plist_value(app, key):
    return subprocess.check_output(['/usr/libexec/PlistBuddy', '-c',
        'Print :' + key, str(app/'Contents/Info.plist')], text=True).strip()


def version(app):
    return plist_value(app, 'CFBundleShortVersionString')


def pids(executable):
    output = subprocess.check_output(['ps', '-axww', '-o', 'pid=,command='], text=True)
    return [int(row.split(None, 1)[0]) for row in output.splitlines()
        if len(row.split(None, 1)) == 2 and
        (row.split(None, 1)[1] == str(executable) or
         row.split(None, 1)[1].startswith(str(executable) + ' '))]


def stop(executable):
    for pid in pids(executable):
        os.kill(pid, signal.SIGTERM)
    deadline = time.monotonic() + 10
    while pids(executable) and time.monotonic() < deadline:
        time.sleep(.2)
    if pids(executable):
        raise RuntimeError('Owned native fixture did not stop cleanly')


def wait(predicate, label, timeout=90):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if predicate():
            return
        time.sleep(.25)
    raise RuntimeError('Timed out: ' + label)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--app', type=Path, required=True)
    parser.add_argument('--candidate', type=Path, required=True)
    parser.add_argument('--signature', type=Path, required=True)
    parser.add_argument('--baseline-version', required=True)
    parser.add_argument('--candidate-version', required=True)
    parser.add_argument('--platform', required=True)
    parser.add_argument('--identifier', required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if platform.system() != 'Darwin' or os.environ.get('GITHUB_ACTIONS') != 'true':
        raise SystemExit('Native qualification requires an isolated hosted macOS runner')
    root = Path(os.environ['RUNNER_TEMP']).resolve()
    app = args.app.resolve()
    fixture = root/'native-ota/install/UIAI Engine Cockpit.app'
    if app != fixture or root not in app.parents:
        raise SystemExit('App must be the exact owned fixture under RUNNER_TEMP')
    if args.identifier != 'com.wpuiai.uiaiengine.cockpit':
        raise SystemExit('Unexpected fixture application identity')
    actual_platform = 'darwin-aarch64' if platform.machine() == 'arm64' else 'darwin-x86_64'
    if actual_platform != args.platform:
        raise SystemExit('Runner is not native to the declared target')
    name = plist_value(app, 'CFBundleExecutable')
    if not name or Path(name).name != name or name in ('.', '..'):
        raise SystemExit('Unsafe native executable metadata')
    executable = app/'Contents/MacOS'/name
    if version(app) != args.baseline_version or pids(executable):
        raise SystemExit('Baseline identity or exclusive fixture precondition failed')
    with tarfile.open(args.candidate) as archive:
        members = [m for m in archive.getmembers() if m.isfile() and
            m.name.endswith('.app/Contents/MacOS/' + name)]
        if len(members) != 1:
            raise SystemExit('Candidate must have exactly one expected native executable')
        expected = hashlib.sha256(archive.extractfile(members[0]).read()).hexdigest()
    baseline_sha = sha(executable)
    if expected == baseline_sha:
        raise SystemExit('Baseline and candidate executables are identical')
    signature = args.signature.read_text().strip()
    bad_signature = corrupt_signature(signature)
    state = {'phase': 'bad_signature', 'downloads': 0, 'manifest_requests': 0}

    class Handler(http.server.BaseHTTPRequestHandler):
        def do_GET(self):
            if self.path == '/latest.json':
                state['manifest_requests'] += 1
                if state['phase'] == 'no_update':
                    self.send_response(204); self.end_headers(); return
                body = json.dumps({'version': args.candidate_version,
                    'notes': 'Private native qualification, not a public release',
                    'pub_date': '2026-10-26T00:00:00Z', 'platforms': {args.platform: {
                        'signature': signature if state['phase'] == 'valid' else bad_signature,
                        'url': 'http://127.0.0.1:48763/candidate.app.tar.gz'}}}).encode()
            elif self.path == '/candidate.app.tar.gz':
                state['downloads'] += 1
                body = args.candidate.read_bytes()
            else:
                self.send_error(404); return
            self.send_response(200)
            self.send_header('Content-Type', 'application/json' if self.path.endswith('.json') else 'application/octet-stream')
            self.send_header('Content-Length', str(len(body)))
            self.end_headers(); self.wfile.write(body)

        def log_message(self, *_):
            pass  # No request/body/credential payloads in receipts.

    server = http.server.ThreadingHTTPServer(('127.0.0.1', 48763), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    backup = root/'native-ota-baseline-backup.app'
    shutil.copytree(app, backup, symlinks=True)
    data = Path.home()/'Library/Application Support'/args.identifier
    data.mkdir(parents=True, exist_ok=True)
    marker = data/'ota-qualification-preservation-marker'
    if marker.exists():
        raise RuntimeError('Unexpected prior fixture marker')
    marker.write_text('value-free native OTA preservation canary\n')
    marker_sha = sha(marker)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    # Do not give the browser process signing/provider/Actions credentials.
    app_env = {key: value for key, value in os.environ.items() if key in {
        'HOME', 'PATH', 'TMPDIR', 'USER', 'LOGNAME', 'SHELL', 'LANG', 'LC_ALL',
        '__CF_USER_TEXT_ENCODING'}}
    with (args.output.parent/'native-app.log').open('wb') as log:
        try:
            baseline = subprocess.Popen([str(executable)], stdout=log, stderr=log, env=app_env)
            wait(lambda: state['downloads'] > 0, 'native updater attempted signed artifact download')
            time.sleep(3)
            if baseline.poll() is not None or sha(executable) != baseline_sha:
                raise RuntimeError('Invalid signature changed the baseline')
            state['phase'] = 'valid'
            wait(lambda: baseline.poll() is not None, 'baseline process exits for native relaunch')
            wait(lambda: pids(executable) and baseline.pid not in pids(executable), 'new native process starts')
            if version(app) != args.candidate_version or sha(executable) != expected or sha(marker) != marker_sha:
                raise RuntimeError('Running candidate identity/data preservation mismatch')
            upgraded_pids = pids(executable)
            state['phase'] = 'no_update'
            stop(executable)
            # Restore only this exact, freshly-created test installation.
            shutil.rmtree(app)
            shutil.copytree(backup, app, symlinks=True)
            rollback = subprocess.Popen([str(executable)], stdout=log, stderr=log, env=app_env)
            wait(lambda: rollback.poll() is None and rollback.pid in pids(executable), 'restored native baseline starts', 15)
            time.sleep(2)
            if version(app) != args.baseline_version or sha(executable) != baseline_sha or sha(marker) != marker_sha:
                raise RuntimeError('Owned-fixture rollback/data preservation mismatch')
            result = {'schema': 'uiai.cockpit_native_ota_qualification.v1', 'status': 'passed',
                'platform': actual_platform, 'baseline_version': args.baseline_version,
                'candidate_version': args.candidate_version,
                'baseline_source_sha': os.environ['BASELINE_SOURCE_SHA'],
                'candidate_source_sha': os.environ['CANDIDATE_SOURCE_SHA'],
                'baseline_build_adapter_source_sha': os.environ['CANDIDATE_SOURCE_SHA'],
                'observed_at': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()),
                'candidate_archive_sha256': sha(args.candidate),
                'candidate_executable_sha256': expected, 'upgraded_process_ids': upgraded_pids,
                'invalid_signature_rejected': True, 'automatic_retry_install_relaunch': True,
                'preservation_marker_unchanged': True, 'owned_fixture_rollback': True,
                'running_version_evidence': 'new process plus native bundle version and exact candidate executable hash',
                'baseline_private_feed_override': True, 'candidate_production_config_preserved': True,
                'historical_public_binary_upgrade': 'not claimed', 'public_channel_changed': False}
            args.output.write_text(json.dumps(result, indent=2)+'\n')
            print(json.dumps(result))
        finally:
            stop(executable)
            server.shutdown(); server.server_close()


if __name__ == '__main__':
    main()
