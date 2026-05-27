#!/usr/bin/env bash
# Stress UIAI browser diagnostics on an isolated local engine port.
# Starts a temp static site + temp uiai-engine, opens sessions, verifies console/exception/network capture.

set -euo pipefail

ROOT_DIR="${UIAI_ROOT:-/home/wpuiai/uiai-engine}"
ENGINE_PORT="${ENGINE_PORT:-7468}"
SITE_PORT="${SITE_PORT:-7469}"
SESSIONS="${SESSIONS:-4}"
ROUNDS="${ROUNDS:-3}"
WIDTH="${WIDTH:-800}"
HEIGHT="${HEIGHT:-600}"
OUT="${OUT:-/tmp/uiai-browser-diagnostics-stress.json}"
ENGINE_BIN="${ENGINE_BIN:-/tmp/uiai-engine-diag-stress}"

cd "$ROOT_DIR"

go build -o "$ENGINE_BIN" ./cmd/uiai-engine

TMPDIR=$(mktemp -d)
cleanup() {
  local code=$?
  if [[ -n "${ENGINE_PID:-}" ]]; then
    kill "$ENGINE_PID" 2>/dev/null || true
    wait "$ENGINE_PID" 2>/dev/null || true
  fi
  if [[ -n "${SITE_PID:-}" ]]; then
    kill "$SITE_PID" 2>/dev/null || true
    wait "$SITE_PID" 2>/dev/null || true
  fi
  rm -rf "$TMPDIR" "$ENGINE_BIN"
  exit "$code"
}
trap cleanup EXIT

cp config.yaml "$TMPDIR/config.yaml"
python3 - "$TMPDIR/config.yaml" "$ENGINE_PORT" <<'PY'
from pathlib import Path
import sys
p = Path(sys.argv[1])
port = sys.argv[2]
s = p.read_text()
s = s.replace('port: 7456', f'port: {port}', 1)
s = s.replace('vision_pool_size: 2', 'vision_pool_size: 2', 1)
s = s.replace('pool_size: 2', 'pool_size: 2', 1)
s = s.replace('max_pool: 2', 'max_pool: 2', 1)
p.write_text(s)
PY

mkdir -p "$TMPDIR/site"
cat > "$TMPDIR/site/index.html" <<'HTML'
<!doctype html>
<title>diagnostics stress</title>
<script>
const u = new URL(location.href);
const id = u.searchParams.get('id') || 'none';
const round = u.searchParams.get('round') || '0';
console.error('diag-console-error', {id, round});
console.warn('diag-console-warn', id);
fetch('/missing-api?id=' + encodeURIComponent(id) + '&round=' + encodeURIComponent(round));
setTimeout(() => { throw new Error('diag-exception-' + id + '-' + round); }, 25);
</script>
<h1>Diagnostics Stress</h1>
HTML

python3 -m http.server "$SITE_PORT" -d "$TMPDIR/site" >/tmp/uiai-diag-stress-site.log 2>&1 & SITE_PID=$!
"$ENGINE_BIN" -config "$TMPDIR/config.yaml" >/tmp/uiai-diag-stress-engine.log 2>&1 & ENGINE_PID=$!

for _ in $(seq 1 60); do
  if curl -fsS "http://127.0.0.1:$ENGINE_PORT/health" >/dev/null 2>&1; then break; fi
  sleep 1
done
curl -fsS "http://127.0.0.1:$ENGINE_PORT/health" >/dev/null

export ENGINE_PORT SITE_PORT WIDTH HEIGHT FOCUSA_WORKPOINT_ID="${FOCUSA_WORKPOINT_ID:-}" FOCUSA_CONTINUITY_ID="${FOCUSA_CONTINUITY_ID:-}" FOCUSA_PROJECT_ROOT="${FOCUSA_PROJECT_ROOT:-}" FOCUSA_EVIDENCE_REF="${FOCUSA_EVIDENCE_REF:-uiai-browser-diagnostics-stress:$OUT}"
python3 - "$SESSIONS" "$ROUNDS" "$OUT" <<'PY'
import concurrent.futures, json, os, subprocess, sys, time, urllib.request

sessions = int(sys.argv[1])
rounds = int(sys.argv[2])
out_path = sys.argv[3]
engine = f"http://127.0.0.1:{os.environ['ENGINE_PORT']}"
site = f"http://127.0.0.1:{os.environ['SITE_PORT']}"
width = int(os.environ['WIDTH'])
height = int(os.environ['HEIGHT'])
focusa_scope = {k: v for k, v in {
    'workpoint_id': os.environ.get('FOCUSA_WORKPOINT_ID', ''),
    'continuity_id': os.environ.get('FOCUSA_CONTINUITY_ID', ''),
    'project_root': os.environ.get('FOCUSA_PROJECT_ROOT', ''),
    'evidence_ref': os.environ.get('FOCUSA_EVIDENCE_REF', ''),
}.items() if v}

def http_json(method, url, body=None):
    data = None if body is None else json.dumps(body).encode()
    req = urllib.request.Request(url, data=data, method=method, headers={'Content-Type': 'application/json'})
    with urllib.request.urlopen(req, timeout=45) as res:
        return json.loads(res.read().decode())

def run_one(round_idx, idx):
    label = f"r{round_idx}s{idx}"
    target = f"{site}/?id={label}&round={round_idx}"
    start = time.time()
    open_body = {'url': target, 'width': width, 'height': height}
    if focusa_scope:
        open_body['focusa_scope'] = focusa_scope
    opened = http_json('POST', f"{engine}/api/session", open_body)
    sid = opened['session']['id']
    time.sleep(0.35)
    diag = http_json('GET', f"{engine}/api/session/{sid}/diagnostics?limit=50")
    http_json('POST', f"{engine}/api/session/{sid}/diagnostics/clear")
    http_json('DELETE', f"{engine}/api/session/{sid}")
    elapsed_ms = round((time.time() - start) * 1000)
    summary = diag.get('summary', {})
    ok = (
        summary.get('console_errors', 0) >= 1 and
        summary.get('exceptions', 0) >= 1 and
        summary.get('failed_requests', 0) >= 1
    )
    return {
        'label': label,
        'session_id': sid,
        'ok': ok,
        'elapsed_ms': elapsed_ms,
        'summary': summary,
        'console_texts': [e.get('text') for e in diag.get('console', [])[:3]],
        'exception_texts': [e.get('text') for e in diag.get('exceptions', [])[:3]],
        'failed_urls': [e.get('url') for e in diag.get('failed_requests', [])[:5]],
    }

results = []
started = time.time()
for r in range(rounds):
    with concurrent.futures.ThreadPoolExecutor(max_workers=sessions) as ex:
        futs = [ex.submit(run_one, r, i) for i in range(sessions)]
        for fut in concurrent.futures.as_completed(futs):
            results.append(fut.result())

total_ms = round((time.time() - started) * 1000)
passed = sum(1 for r in results if r['ok'])
evidence_ref = os.environ.get('FOCUSA_EVIDENCE_REF', f'uiai-browser-diagnostics-stress:{out_path}')
report = {
    'ok': passed == len(results),
    'sessions': sessions,
    'rounds': rounds,
    'total_runs': len(results),
    'passed': passed,
    'failed': len(results) - passed,
    'total_ms': total_ms,
    'avg_elapsed_ms': round(sum(r['elapsed_ms'] for r in results) / len(results), 1) if results else 0,
    'max_elapsed_ms': max((r['elapsed_ms'] for r in results), default=0),
    'focusa_evidence': {
        'target_ref': 'WPUIAI browser diagnostics stress',
        'result': f"diagnostics stress ok={passed == len(results)} passed={passed}/{len(results)} avg_ms={round(sum(r['elapsed_ms'] for r in results) / len(results), 1) if results else 0}",
        'evidence_ref': evidence_ref,
        'diagnostics_ref': out_path,
        'focusa_scope': focusa_scope,
        'intake_tool': 'focusa_evidence_capture',
    },
    'results': results,
}
with open(out_path, 'w') as f:
    json.dump(report, f, indent=2)
print(json.dumps({k: report[k] for k in ['ok','sessions','rounds','total_runs','passed','failed','total_ms','avg_elapsed_ms','max_elapsed_ms']}, indent=2))
if not report['ok']:
    raise SystemExit(1)
PY

printf 'stress_report=%s\n' "$OUT"
