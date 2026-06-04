#!/usr/bin/env bash
set -euo pipefail

ENGINE_URL="${UIAI_ENGINE_URL:-http://localhost:7456}"
TIMEOUT_SECONDS="${UIAI_SMOKE_TIMEOUT_SECONDS:-20}"
AUTH_ARGS=()
if [[ -n "${UIAI_API_KEY:-}" ]]; then
  AUTH_ARGS=(-H "X-API-Key: ${UIAI_API_KEY}")
elif [[ -n "${UIAI_BEARER_TOKEN:-}" ]]; then
  AUTH_ARGS=(-H "Authorization: Bearer ${UIAI_BEARER_TOKEN}")
fi

need(){ command -v "$1" >/dev/null 2>&1 || { echo "missing required command: $1" >&2; exit 1; }; }
need curl
need jq

say(){ printf "%s\n" "$*"; }
fetch(){ curl -fsS --max-time "$TIMEOUT_SECONDS" "$@"; }
fetch_auth(){ curl -fsS --max-time "$TIMEOUT_SECONDS" "${AUTH_ARGS[@]}" "$@"; }

say "UIAI agent integration smoke"
say "engine_url=$ENGINE_URL"

fetch "$ENGINE_URL/api/health" | jq -e '.status == "healthy" or .status == "ok"' >/dev/null
fetch "$ENGINE_URL/api/tools/agent-card" | jq -e '.service == "uiai-engine"' >/dev/null
fetch "$ENGINE_URL/api/tools/graph" | jq -e '.focusa_integration.preferred_focusa_tools | index("focusa_browser_diagnostics_intake")' >/dev/null
fetch "$ENGINE_URL/api/tools/search?q=read" | jq -e '.tools[] | select(.name == "browser_read")' >/dev/null
fetch "$ENGINE_URL/api/tools/search?q=search" | jq -e '.tools[] | select(.name == "browser_search")' >/dev/null
fetch "$ENGINE_URL/api/tools/mcp" | jq -e '.tools[] | select(.name == "browser_open") | .related_tools | index("focusa_browser_diagnostics_intake")' >/dev/null
fetch "$ENGINE_URL/api/tools/mcp" | jq -e '.tools[] | select(.name == "browser_search")' >/dev/null
fetch "$ENGINE_URL/api/tools/mcp" | jq -e '.tools[] | select(.name == "uiai_health")' >/dev/null
fetch "$ENGINE_URL/api/tools/mcp" | jq -e '.tools[] | select(.name == "uiai_status")' >/dev/null
fetch "$ENGINE_URL/api/tools/mcp" | jq -e '.tools[] | select(.name == "critique_models")' >/dev/null
fetch "$ENGINE_URL/api/tools/mcp" | jq -e '.tools[] | select(.name == "critique_dimensions")' >/dev/null
fetch "$ENGINE_URL/api/tools/mcp" | jq -e '.tools[] | select(.name == "frame_catalog")' >/dev/null
fetch "$ENGINE_URL/api/tools/mcp" | jq -e '.tools[] | select(.name == "uiai_errors")' >/dev/null
fetch "$ENGINE_URL/api/media/frame/catalog" | jq -e '.count > 0' >/dev/null
fetch "$ENGINE_URL/api/errors?limit=1" | jq -e 'has("events") and has("redaction")' >/dev/null
fetch_auth "$ENGINE_URL/api/search/providers" | jq -e '.providers[] | select(.id == "brave") | has("configured")' >/dev/null
fetch_auth -X POST "$ENGINE_URL/api/search" -H "Content-Type: application/json" -d '{"query":"UIAI Engine browser agents","limit":1}' | jq -e '.provider == "brave" and .count >= 1' >/dev/null
node --check "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/mcp/browser-session-mcp.mjs" >/dev/null

PI_EXT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/.pi/extensions/uiai-engine.ts"
for tool in \
  pi_uiai_agent_card \
  pi_uiai_tool_search \
  pi_uiai_tool_graph \
  uiai_search \
  uiai_health \
  uiai_status \
  uiai_critique_models \
  uiai_critique_dimensions \
  uiai_errors \
  uiai_browser_open \
  uiai_browser_screenshot \
  uiai_browser_scroll \
  uiai_browser_snapshot \
  uiai_browser_dom \
  uiai_browser_navigate \
  uiai_browser_click \
  uiai_browser_hover \
  uiai_browser_type \
  uiai_browser_fill \
  uiai_browser_select \
  uiai_browser_press \
  uiai_browser_back \
  uiai_browser_forward \
  uiai_browser_eval \
  uiai_browser_eval_async \
  uiai_browser_resize \
  uiai_browser_css \
  uiai_browser_wait \
  uiai_browser_text \
  uiai_browser_read \
  uiai_browser_cookies \
  uiai_browser_diagnostics \
  uiai_browser_diagnostics_clear \
  uiai_browser_close \
  uiai_screenshot \
  uiai_frame_catalog \
  uiai_frame_render
  do
  grep -q "name: \"$tool\"" "$PI_EXT"
done

python3 - <<'PY' "$PI_EXT" "$ENGINE_URL" "${UIAI_API_KEY:-}" "${UIAI_BEARER_TOKEN:-}"
import json, re, sys, urllib.request
pi_ext, engine, api_key, bearer = sys.argv[1], sys.argv[2].rstrip('/'), sys.argv[3], sys.argv[4]
src = open(pi_ext).read()
pi_tools = set(re.findall(r'name:\s*"([^"]+)"', src))
req = urllib.request.Request(engine + '/api/tools/mcp')
if api_key:
    req.add_header('X-API-Key', api_key)
elif bearer:
    req.add_header('Authorization', 'Bearer ' + bearer)
mcp = json.load(urllib.request.urlopen(req))['tools']
missing = []
for tool in mcp:
    name = tool['name']
    candidates = {name, 'uiai_' + name}
    if name == 'browser_search':
        candidates.add('uiai_search')
    if name.startswith('critique_'):
        candidates.add('uiai_' + name)
    if name.startswith('browser_'):
        candidates.add('uiai_' + name)
    elif name.startswith('frame_'):
        candidates.add('uiai_' + name)
    elif name.startswith('uiai_'):
        candidates.add('pi_' + name)
    if not (candidates & pi_tools):
        missing.append(name)
if missing:
    raise SystemExit('Pi extension missing MCP mirrors: ' + ', '.join(missing))
PY

say "agent integration smoke ok"
