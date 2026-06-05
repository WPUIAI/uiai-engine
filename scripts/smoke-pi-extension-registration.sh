#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENGINE_URL="${UIAI_ENGINE_URL:-http://localhost:7456}"
TIMEOUT_SECONDS="${UIAI_SMOKE_TIMEOUT_SECONDS:-20}"
PI_EXT="$ROOT_DIR/.pi/extensions/uiai-engine.ts"
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

TOOLS_JSON="$(mktemp)"
trap 'rm -f "$TOOLS_JSON"' EXIT
curl -fsS --max-time "$TIMEOUT_SECONDS" "${AUTH_ARGS[@]}" "$ENGINE_URL/api/tools/mcp" > "$TOOLS_JSON"

python3 - <<'PY' "$PI_EXT" "$TOOLS_JSON"
import json
import re
import sys
from pathlib import Path

pi_ext = Path(sys.argv[1])
src = pi_ext.read_text()
mcp_tools = json.loads(Path(sys.argv[2]).read_text()).get("tools", [])

registered_tools = set(re.findall(r"registerTool\s*\(\s*\{[^{}]*?name:\s*\"([^\"]+)\"", src, re.S))
registered_commands = set(re.findall(r"registerCommand\s*\(\s*\"([^\"]+)\"", src))

required_tools = {
    "pi_uiai_agent_card",
    "pi_uiai_tool_search",
    "pi_uiai_tool_graph",
    "uiai_search",
    "uiai_focusa_packet_build",
    "uiai_focusa_packet_compose",
    "uiai_health",
    "uiai_status",
    "uiai_errors",
    "uiai_browser_open",
    "uiai_browser_screenshot",
    "uiai_browser_snapshot",
    "uiai_browser_eval_async",
    "uiai_browser_diagnostics",
    "uiai_browser_diagnostics_clear",
    "uiai_screenshot",
    "uiai_frame_catalog",
    "uiai_frame_render",
}
missing_required = sorted(required_tools - registered_tools)
if missing_required:
    raise SystemExit("Pi extension missing required tools: " + ", ".join(missing_required))

if "uiai" not in registered_commands:
    raise SystemExit("Pi extension missing /uiai command registration")
for phrase in ["Run research packet", "Run diagnostics packet", "Run proof packet", "guidedPrompts"]:
    if phrase not in src:
        raise SystemExit(f"Pi extension missing guided workflow phrase: {phrase}")

missing_mirrors = []
for tool in mcp_tools:
    name = tool.get("name")
    if not name:
        continue
    candidates = {name, "uiai_" + name}
    if name == "browser_search":
        candidates.add("uiai_search")
    if name.startswith("critique_"):
        candidates.add("uiai_" + name)
    if name.startswith("browser_"):
        candidates.add("uiai_" + name)
    elif name.startswith("frame_"):
        candidates.add("uiai_" + name)
    elif name.startswith("uiai_"):
        candidates.add("pi_" + name)
    if not (candidates & registered_tools):
        missing_mirrors.append(name)
if missing_mirrors:
    raise SystemExit("Pi extension missing MCP mirrors: " + ", ".join(sorted(missing_mirrors)))

if not re.search(r"pi\.registerTool\s*=\s*\(\(definition:\s*any\)", src):
    raise SystemExit("Pi extension missing compact default registerTool wrapper")

print(f"pi extension registration ok: tools={len(registered_tools)} commands={len(registered_commands)} mcp_mirrors={len(mcp_tools)}")
PY
