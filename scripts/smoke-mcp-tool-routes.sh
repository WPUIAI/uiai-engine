#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE_URL="${UIAI_ENGINE_URL:-http://localhost:7456}"
TIMEOUT_SECONDS="${UIAI_SMOKE_TIMEOUT_SECONDS:-20}"
BRIDGE="$ROOT_DIR/mcp/browser-session-mcp.mjs"
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

ENGINE_TOOLS_JSON="$(mktemp)"
trap 'rm -f "$ENGINE_TOOLS_JSON"' EXIT
curl -fsS --max-time "$TIMEOUT_SECONDS" "${AUTH_ARGS[@]}" "$ENGINE_URL/api/tools/mcp" > "$ENGINE_TOOLS_JSON"

python3 - <<'PY' "$BRIDGE" "$ENGINE_TOOLS_JSON"
import json
import re
import sys
from pathlib import Path

bridge = Path(sys.argv[1]).read_text()
engine_tools = json.loads(Path(sys.argv[2]).read_text()).get("tools", [])
engine_names = {tool.get("name") for tool in engine_tools if tool.get("name")}

core_match = re.search(r"const\s+BRIDGE_CORE_TOOLS\s*=\s*\[(.*?)\];", bridge, re.S)
if not core_match:
    raise SystemExit("BRIDGE_CORE_TOOLS block not found")
core_names = set(re.findall(r"\bname:\s*\"([^\"]+)\"", core_match.group(1)))

call_match = re.search(r"async\s+function\s+toolsCall\s*\([^)]*\)\s*\{(.*?)(?:\n}\n?$)", bridge, re.S)
if not call_match:
    raise SystemExit("toolsCall function not found")
call_names = set(re.findall(r"case\s+\"([^\"]+)\"\s*:", call_match.group(1)))

advertised = engine_names | core_names
missing = sorted(advertised - call_names)
stale_cases = sorted(call_names - advertised)
if missing:
    print("MCP advertised tools missing toolsCall route:", ", ".join(missing), file=sys.stderr)
    raise SystemExit(1)
print(f"mcp tool route parity ok: advertised={len(advertised)} routed={len(call_names)} extra_routes={len(stale_cases)}")
if stale_cases:
    print("extra routed-only tools:", ", ".join(stale_cases))
PY
