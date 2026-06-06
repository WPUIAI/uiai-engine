#!/usr/bin/env bash
# Isolated browser/session failure regression smoke.
# Verifies stable error_class, suggested_next_action, and bounded /api/errors event for common failures.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="${UIAI_ROOT:-$(cd "$SCRIPT_DIR/.." && pwd)}"
ENGINE_PORT="${ENGINE_PORT:-7470}"
SITE_PORT="${SITE_PORT:-7471}"
OUT="${OUT:-/tmp/uiai-browser-error-regressions.json}"
ENGINE_BIN="${ENGINE_BIN:-/tmp/uiai-engine-error-regressions}"

cd "$ROOT_DIR"
go build -o "$ENGINE_BIN" ./cmd/uiai-engine

TMPDIR=$(mktemp -d)
cleanup() {
  local code=$?
  if [[ -n "${ENGINE_PID:-}" ]]; then kill "$ENGINE_PID" 2>/dev/null || true; wait "$ENGINE_PID" 2>/dev/null || true; fi
  if [[ -n "${SITE_PID:-}" ]]; then kill "$SITE_PID" 2>/dev/null || true; wait "$SITE_PID" 2>/dev/null || true; fi
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
s = s.replace('allow_private_urls: false', 'allow_private_urls: true', 1)
s = s.replace('data_dir: "/home/wpuiai/uiai-engine/data"', f'data_dir: "{p.parent / "data"}"', 1)
s = s.replace('share_dir: "/home/wpuiai/ai-api/shares"', f'share_dir: "{p.parent / "shares"}"', 1)
s = s.replace('script_dir: "/home/wpuiai/public_html/wp-content/plugins/wpuiai/assets/templates/devices"', f'script_dir: "{p.parent / "device-templates"}"', 1)
s = s.replace('health_file: "/var/log/uiai/ip-pool-health.json"', f'health_file: "{p.parent / "ip-pool-health.json"}"', 1)
s = s.replace('log_file: "/var/log/uiai/captcha-stats.jsonl"', f'log_file: "{p.parent / "captcha-stats.jsonl"}"', 1)
s = s.replace('file: "/var/log/uiai/engine.log"', f'file: "{p.parent / "engine.log"}"', 1)
p.write_text(s)
PY

mkdir -p "$TMPDIR/site"
cat > "$TMPDIR/site/index.html" <<'HTML'
<!doctype html>
<title>browser error regressions</title>
<script>
console.warn('browser-error-regressions-ready');
fetch('/missing-api-regression');
</script>
<h1 id="ready">Browser Error Regressions</h1>
<button id="present">Present Button</button>
HTML

python3 -m http.server "$SITE_PORT" -d "$TMPDIR/site" >/tmp/uiai-browser-errors-site.log 2>&1 & SITE_PID=$!
"$ENGINE_BIN" -config "$TMPDIR/config.yaml" >/tmp/uiai-browser-errors-engine.log 2>&1 & ENGINE_PID=$!

for _ in $(seq 1 60); do
  if curl -fsS "http://127.0.0.1:$ENGINE_PORT/health" >/dev/null 2>&1; then break; fi
  sleep 1
done
if ! curl -fsS "http://127.0.0.1:$ENGINE_PORT/health" >/dev/null; then
  echo "uiai browser error regression startup failed: engine health unavailable on port $ENGINE_PORT" >&2
  echo "--- engine log (/tmp/uiai-browser-errors-engine.log) ---" >&2
  sed -n '1,220p' /tmp/uiai-browser-errors-engine.log >&2 || true
  echo "--- site log (/tmp/uiai-browser-errors-site.log) ---" >&2
  sed -n '1,120p' /tmp/uiai-browser-errors-site.log >&2 || true
  exit 7
fi

export ENGINE_PORT SITE_PORT OUT
python3 - <<'PY'
import json, os, sys, time, urllib.error, urllib.request

engine = f"http://127.0.0.1:{os.environ['ENGINE_PORT']}"
site = f"http://127.0.0.1:{os.environ['SITE_PORT']}"
out_path = os.environ['OUT']


def request(method, path_or_url, body=None, timeout=30):
    url = path_or_url if path_or_url.startswith('http') else engine + path_or_url
    data = None if body is None else json.dumps(body).encode()
    req = urllib.request.Request(url, data=data, method=method, headers={'Content-Type': 'application/json'})
    try:
        with urllib.request.urlopen(req, timeout=timeout) as res:
            raw = res.read().decode()
            return res.status, json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        raw = e.read().decode()
        try:
            data = json.loads(raw) if raw else {}
        except Exception:
            data = {'raw': raw}
        return e.code, data


def open_session():
    status, data = request('POST', '/api/session', {'url': site + '/', 'width': 800, 'height': 600})
    if status not in (200, 201):
        raise SystemExit(f'open_session failed: {status} {data}')
    return data['session']['id']


def latest_event(cls):
    status, data = request('GET', f'/api/errors?source=browser_session&class={cls}&limit=5')
    if status != 200:
        return None
    events = data.get('events') or []
    return events[0] if events else None


def assert_envelope(name, status, data, expected_status, expected_class, action_substring):
    if status != expected_status:
        raise AssertionError(f'{name}: status={status} want={expected_status} body={data}')
    if data.get('error_class') != expected_class:
        raise AssertionError(f'{name}: error_class={data.get("error_class")} want={expected_class} body={data}')
    if not data.get('error_id'):
        raise AssertionError(f'{name}: missing error_id body={data}')
    action = data.get('suggested_next_action') or ''
    if action_substring not in action:
        raise AssertionError(f'{name}: action={action!r} missing {action_substring!r}')
    diag = data.get('diagnostics') or ''
    if '/api/errors' not in diag:
        raise AssertionError(f'{name}: missing diagnostics link body={data}')
    event = latest_event(expected_class)
    if not event or event.get('id') != data.get('error_id'):
        raise AssertionError(f'{name}: latest event mismatch event={event} envelope={data}')
    return {'name': name, 'ok': True, 'status': status, 'error_class': expected_class, 'error_id': data.get('error_id'), 'event_id': event.get('id'), 'suggested_next_action': action}

results = []
sid = open_session()
time.sleep(0.2)
try:
    status, data = request('POST', f'/api/session/{sid}/click', {'selector': '#definitely-missing-regression'})
    results.append(assert_envelope('missing_selector_click', status, data, 500, 'selector_not_found', 'snapshot or /diagnostics'))

    status, data = request('POST', f'/api/session/{sid}/wait', {'selector': '#never-appears-regression', 'timeout_ms': 150})
    results.append(assert_envelope('wait_timeout', status, data, 408, 'timeout', 'Read diagnostics'))

    status, data = request('POST', f'/api/session/{sid}/eval_async', {'js': 'throw new Error("eval-regression-failure")', 'timeout_ms': 1000})
    results.append(assert_envelope('eval_async_failure', status, data, 500, 'eval_failed', 'Read browser_diagnostics'))
finally:
    request('DELETE', f'/api/session/{sid}')

# Stale session currently returns a simple 404 rather than browser_session envelope; assert behavior and /api/errors HTTP event are bounded.
status, data = request('GET', f'/api/session/{sid}/diagnostics?limit=1')
if status != 404 or data.get('error') != 'session not found':
    raise AssertionError(f'stale_session_404: unexpected {status} {data}')
status_events, events_data = request('GET', '/api/errors?source=http&class=not_found&limit=5')
if status_events != 200 or not events_data.get('events'):
    raise AssertionError(f'stale_session_404: missing bounded http not_found event {status_events} {events_data}')
results.append({'name': 'stale_session_404', 'ok': True, 'status': status, 'error_class': 'not_found', 'event_id': events_data['events'][0].get('id'), 'suggested_next_action': 'reopen session'})

# URL policy failure happens during session open and is structured when private URLs are disallowed by config.
status, data = request('POST', '/api/session', {'url': 'file:///tmp/uiai-not-allowed.html', 'width': 800, 'height': 600})
if status not in (400, 500) or data.get('error_class') not in ('url_not_allowed', 'navigation_failed'):
    raise AssertionError(f'url_not_allowed: unexpected {status} {data}')
if data.get('error_class') == 'url_not_allowed':
    if not data.get('suggested_next_action') or 'URL' not in data.get('suggested_next_action'):
        raise AssertionError(f'url_not_allowed: missing action {data}')
    results.append({'name': 'url_not_allowed', 'ok': True, 'status': status, 'error_class': data.get('error_class'), 'error_id': data.get('error_id'), 'suggested_next_action': data.get('suggested_next_action')})
else:
    results.append({'name': 'url_not_allowed_feasible_as_navigation_failed', 'ok': True, 'status': status, 'error_class': data.get('error_class'), 'error_id': data.get('error_id'), 'suggested_next_action': data.get('suggested_next_action')})

report = {'ok': all(r.get('ok') for r in results), 'total': len(results), 'results': results}
with open(out_path, 'w') as f:
    json.dump(report, f, indent=2)
print(json.dumps({'ok': report['ok'], 'total': report['total'], 'classes': [r['error_class'] for r in results]}, indent=2))
if not report['ok']:
    raise SystemExit(1)
PY

printf 'browser_error_regression_report=%s\n' "$OUT"
