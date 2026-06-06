#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
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
need python3
TMP="$(mktemp)"
ENGINE_PID=""
cleanup(){
  rm -f "$TMP"
  if [[ -n "$ENGINE_PID" ]]; then
    kill "$ENGINE_PID" >/dev/null 2>&1 || true
    wait "$ENGINE_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT
if ! curl -fsS --max-time "$TIMEOUT_SECONDS" "${AUTH_ARGS[@]}" "$ENGINE_URL/api/tools/mcp" > "$TMP"; then
  need go
  ENGINE_LOG="${UIAI_TOOL_PARITY_ENGINE_LOG:-/tmp/uiai-tool-parity-engine.log}"
  (cd "$ROOT_DIR" && go run ./cmd/uiai-engine > "$ENGINE_LOG" 2>&1) &
  ENGINE_PID=$!
  for _ in $(seq 1 60); do
    if curl -fsS --max-time 2 "$ENGINE_URL/health" >/dev/null 2>&1; then
      break
    fi
    if ! kill -0 "$ENGINE_PID" >/dev/null 2>&1; then
      echo "tool parity engine exited early; log: $ENGINE_LOG" >&2
      sed -n '1,120p' "$ENGINE_LOG" >&2 || true
      exit 1
    fi
    sleep 0.5
  done
  curl -fsS --max-time "$TIMEOUT_SECONDS" "${AUTH_ARGS[@]}" "$ENGINE_URL/api/tools/mcp" > "$TMP"
fi
python3 - <<'PY' "$ROOT_DIR" "$TMP"
import json, re, sys
from pathlib import Path
root=Path(sys.argv[1])
tools_json=Path(sys.argv[2])
mcp_tools=json.loads(tools_json.read_text()).get('tools',[])
mcp_names={t.get('name') for t in mcp_tools if t.get('name')}
pi_src=(root/'.pi/extensions/uiai-engine.ts').read_text()
mcp_src=(root/'mcp/browser-session-mcp.mjs').read_text()
cli_src=(root/'scripts/uiai').read_text()
readme=(root/'README.md').read_text()
parity=(root/'docs/PUBLIC_API_PARITY_MATRIX.md').read_text()
quickstart=(root/'docs/UIAI_FOR_AGENTS_QUICKSTART.md').read_text()
pi_tools=set(re.findall(r'registerTool\s*\(\s*\{[^{}]*?name:\s*"([^"]+)"', pi_src, re.S))
pi_commands=set(re.findall(r'registerCommand\s*\(\s*"([^"]+)"', pi_src))
route_cases=set(re.findall(r'case\s+"([^"]+)"\s*:', mcp_src))
normalized={
 'browser_search':'uiai_search',
 'frame_catalog':'uiai_frame_catalog',
 'frame_render':'uiai_frame_render',
 'uiai_agent_card':'pi_uiai_agent_card',
 'uiai_tool_search':'pi_uiai_tool_search',
 'uiai_tool_graph':'pi_uiai_tool_graph',
}
def pi_candidates(name):
    out={name,'uiai_'+name}
    if name in normalized: out.add(normalized[name])
    if name.startswith('browser_'): out.add('uiai_'+name)
    if name.startswith('frame_'): out.add('uiai_'+name)
    if name.startswith('critique_'): out.add('uiai_'+name)
    if name.startswith('uiai_'): out.add('pi_'+name)
    return out
missing_pi=sorted(name for name in mcp_names if not (pi_candidates(name) & pi_tools))
missing_mcp_routes=sorted(name for name in mcp_names if name not in route_cases)
required_cli=['status','health','errors','tools','packet compose','markdown <url>','research packet','session open','session read','session diagnostics','session close','smoke agent','smoke browser','smoke packet']
missing_cli=[cmd for cmd in required_cli if cmd not in cli_src]
required_docs=['PUBLIC_API_PARITY_MATRIX.md','scripts/uiai research packet','scripts/uiai markdown','uiai_source_to_markdown','source_to_markdown','uiai_focusa_packet_compose','uiai_browser_diagnostics','smoke-mcp-tool-routes.sh','smoke-pi-extension-registration.sh']
doc_blob='\n'.join([readme,parity,quickstart])
missing_docs=[needle for needle in required_docs if needle not in doc_blob]
required_pi_phrases=['/uiai research <query>','/uiai proof <url>','/uiai diagnose <session_id>','runGuidedPacketWorkflow']
missing_pi_phrases=[needle for needle in required_pi_phrases if needle not in pi_src]
problems=[]
if missing_pi: problems.append('missing Pi mirrors: '+', '.join(missing_pi))
if missing_mcp_routes: problems.append('missing MCP routes: '+', '.join(missing_mcp_routes))
if missing_cli: problems.append('missing CLI help/commands: '+', '.join(missing_cli))
if missing_docs: problems.append('missing docs mentions: '+', '.join(missing_docs))
if missing_pi_phrases: problems.append('missing Pi guided execution phrases: '+', '.join(missing_pi_phrases))
report={
 'ok': not problems,
 'mcp_tools': len(mcp_names),
 'pi_tools': len(pi_tools),
 'mcp_routes': len(route_cases),
 'pi_commands': sorted(pi_commands),
 'cli_required': required_cli,
 'problems': problems,
}
print(json.dumps(report, indent=2))
if problems:
    raise SystemExit('tool parity check failed')
PY
