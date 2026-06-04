#!/usr/bin/env bash
set -euo pipefail

ENGINE_URL="${UIAI_ENGINE_URL:-http://localhost:7456}"
TIMEOUT_SECONDS="${UIAI_SMOKE_TIMEOUT_SECONDS:-20}"

need(){ command -v "$1" >/dev/null 2>&1 || { echo "missing required command: $1" >&2; exit 1; }; }
need curl
need jq

say(){ printf "%s\n" "$*"; }
fetch(){ curl -fsS --max-time "$TIMEOUT_SECONDS" "$@"; }

say "UIAI agent integration smoke"
say "engine_url=$ENGINE_URL"

fetch "$ENGINE_URL/api/health" | jq -e '.status == "healthy" or .status == "ok"' >/dev/null
fetch "$ENGINE_URL/api/tools/agent-card" | jq -e '.service == "uiai-engine"' >/dev/null
fetch "$ENGINE_URL/api/tools/graph" | jq -e '.focusa_integration.preferred_focusa_tools | index("focusa_browser_diagnostics_intake")' >/dev/null
fetch "$ENGINE_URL/api/tools/search?q=read" | jq -e '.tools[] | select(.name == "browser_read")' >/dev/null
fetch "$ENGINE_URL/api/tools/mcp" | jq -e '.tools[] | select(.name == "browser_open") | .related_tools | index("focusa_browser_diagnostics_intake")' >/dev/null
node --check "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/mcp/browser-session-mcp.mjs" >/dev/null

PI_EXT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/.pi/extensions/uiai-engine.ts"
for tool in \
  pi_uiai_agent_card \
  pi_uiai_tool_search \
  pi_uiai_tool_graph \
  uiai_health \
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

python3 - <<'PY' "$PI_EXT" "$ENGINE_URL"
import json, re, sys, urllib.request
pi_ext, engine = sys.argv[1], sys.argv[2].rstrip('/')
src = open(pi_ext).read()
pi_tools = set(re.findall(r'name:\s*"([^"]+)"', src))
mcp = json.load(urllib.request.urlopen(engine + '/api/tools/mcp'))['tools']
missing = []
for tool in mcp:
    name = tool['name']
    candidates = {name, 'uiai_' + name}
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
