#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PI_EXT="$ROOT_DIR/.pi/extensions/uiai-engine.ts"

python3 - <<'PY' "$PI_EXT"
import json
import re
import sys
from pathlib import Path

src = Path(sys.argv[1]).read_text()

required_snippets = [
    'renderResult: definition.renderResult || compactRenderResult',
    'keyHint("app.tools.expand", "expand")',
    'theme.fg("error", compactSummary(data, details))',
    'theme.fg("success", compactSummary(data, details))',
    'JSON.stringify(data, null, 2)',
    'return new Text(`${line}\\n${theme.fg("toolOutput", body)}`, 0, 0);',
]
missing = [snippet for snippet in required_snippets if snippet not in src]
if missing:
    raise SystemExit('Pi render helper missing expected compact/expanded behavior: ' + '; '.join(missing))

for tool in ["uiai_status", "uiai_errors", "uiai_browser_open", "uiai_browser_diagnostics", "uiai_frame_catalog"]:
    pattern = rf'name:\s*"{re.escape(tool)}"[\s\S]*?(?=\n\tpi\.registerTool|\n\tpi\.registerCommand|\n\}})'
    block = re.search(pattern, src)
    if not block:
        raise SystemExit(f'Pi tool registration block not found: {tool}')
    if 'renderResult:' in block.group(0):
        raise SystemExit(f'Pi tool unexpectedly overrides compact default renderResult: {tool}')

# Mirror compactSummary for representative success/error fixtures so this smoke proves the expected operator-visible text contract.
def compact_summary(data, details=None):
    details = details or {}
    endpoint = str(details.get('endpoint') or 'UIAI')
    if data.get('error') or data.get('error_class'):
        ident = f" id={data.get('error_id')}" if data.get('error_id') else ''
        next_action = f" → {data.get('suggested_next_action')}" if data.get('suggested_next_action') else ''
        return f"{endpoint} error{':' + data.get('error_class') if data.get('error_class') else ''}{ident} {data.get('message') or data.get('error') or ''}{next_action}".strip()
    if isinstance(data.get('events'), list):
        return f"{endpoint} {data.get('count', len(data['events']))} error events stored={data.get('stored_count', '?')}"
    if data.get('session_id') or data.get('id'):
        ident = data.get('session_id') or data.get('id')
        url = f" {data.get('url')}" if data.get('url') else ''
        return f"{endpoint} {ident}{url}"
    if data.get('summary') is not None:
        return f"{endpoint} summary {json.dumps(data.get('summary'), separators=(',', ':'))}"
    if isinstance(data.get('count'), int):
        return f"{endpoint} count={data.get('count')}"
    if data.get('status'):
        return f"{endpoint} status={data.get('status')}"
    return f"{endpoint} ok"

fixtures = [
    ({'status': 'healthy'}, {'endpoint': '/api/status'}, '/api/status status=healthy'),
    ({'events': [{"id": "e1"}], 'count': 1, 'stored_count': 9}, {'endpoint': '/api/errors'}, '/api/errors 1 error events stored=9'),
    ({'session_id': 'sid1', 'url': 'https://example.com'}, {'endpoint': '/api/session'}, '/api/session sid1 https://example.com'),
    ({'error_class': 'auth_failed', 'error_id': 'uiai-error-1', 'message': 'Unauthorized', 'suggested_next_action': 'Set UIAI_API_KEY'}, {'endpoint': '/api/status'}, '/api/status error:auth_failed id=uiai-error-1 Unauthorized → Set UIAI_API_KEY'),
]
for data, details, expected in fixtures:
    actual = compact_summary(data, details)
    if actual != expected:
        raise SystemExit(f'compact summary mismatch: {actual!r} != {expected!r}')
    expanded_body = json.dumps(data, indent=2)
    if '\n' not in expanded_body or not expanded_body.startswith('{'):
        raise SystemExit('expanded JSON body proof failed')

print(f"pi render smoke ok: fixtures={len(fixtures)} default_render_tools=5 compact_and_expanded=true")
PY
