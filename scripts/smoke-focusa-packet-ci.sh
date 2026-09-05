#!/usr/bin/env bash
# Isolated CI smoke for /api/agent/research-packet. Starts temp engine, runs packet smoke, prints bounded logs on failure.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="${UIAI_ROOT:-$(cd "$SCRIPT_DIR/.." && pwd)}"
ENGINE_PORT="${ENGINE_PORT:-7472}"
OUT="${OUT:-/tmp/uiai-browser-reliability/focusa-packet-smoke.json}"
ENGINE_BIN="${ENGINE_BIN:-/tmp/uiai-engine-packet-smoke}"
ENGINE_LOG="${ENGINE_LOG:-/tmp/uiai-packet-smoke-engine.log}"
# Isolated contract fixture only; deployed delivery proof must use its audited public HTTPS origin.
export UIAI_EPWA_PUBLIC_BASE_URL="${UIAI_EPWA_PUBLIC_BASE_URL:-https://epwa-ci.invalid/}"
[[ "$UIAI_EPWA_PUBLIC_BASE_URL" == https://* ]] || { echo "UIAI_EPWA_PUBLIC_BASE_URL must use HTTPS" >&2; exit 2; }
cd "$ROOT_DIR"
go build -o "$ENGINE_BIN" ./cmd/uiai-engine
TMPDIR="$(mktemp -d)"
cleanup(){
  local code=$?
  if [[ -n "${ENGINE_PID:-}" ]]; then kill "$ENGINE_PID" 2>/dev/null || true; wait "$ENGINE_PID" 2>/dev/null || true; fi
  rm -rf "$TMPDIR" "$ENGINE_BIN"
  exit "$code"
}
trap cleanup EXIT
cp config.yaml "$TMPDIR/config.yaml"
python3 - "$TMPDIR/config.yaml" "$ENGINE_PORT" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]); port=sys.argv[2]; s=p.read_text()
s=s.replace('port: 7456', f'port: {port}', 1)
s=s.replace('data_dir: "/home/wpuiai/uiai-engine/data"', f'data_dir: "{p.parent / "data"}"', 1)
s=s.replace('share_dir: "/home/wpuiai/ai-api/shares"', f'share_dir: "{p.parent / "shares"}"', 1)
s=s.replace('script_dir: "/home/wpuiai/public_html/wp-content/plugins/wpuiai/assets/templates/devices"', f'script_dir: "{p.parent / "device-templates"}"', 1)
s=s.replace('health_file: "/var/log/uiai/ip-pool-health.json"', f'health_file: "{p.parent / "ip-pool-health.json"}"', 1)
s=s.replace('log_file: "/var/log/uiai/captcha-stats.jsonl"', f'log_file: "{p.parent / "captcha-stats.jsonl"}"', 1)
s=s.replace('file: "/var/log/uiai/engine.log"', f'file: "{p.parent / "engine.log"}"', 1)
p.write_text(s)
PY
"$ENGINE_BIN" -config "$TMPDIR/config.yaml" >"$ENGINE_LOG" 2>&1 & ENGINE_PID=$!
for _ in $(seq 1 60); do
  if curl -fsS "http://127.0.0.1:$ENGINE_PORT/health" >/dev/null 2>&1; then break; fi
  sleep 1
done
if ! curl -fsS "http://127.0.0.1:$ENGINE_PORT/health" >/dev/null; then
  echo "uiai packet smoke startup failed: engine health unavailable on port $ENGINE_PORT" >&2
  echo "--- engine log ($ENGINE_LOG) ---" >&2
  sed -n '1,220p' "$ENGINE_LOG" >&2 || true
  exit 7
fi
mkdir -p "$(dirname "$OUT")"
printf '{}\n' >"$OUT"
ENGINE_URL="http://127.0.0.1:$ENGINE_PORT" OUT="$OUT" "$SCRIPT_DIR/smoke-focusa-packet.sh"
python3 - "$OUT" <<'PY'
import json, sys
p=sys.argv[1]
d=json.load(open(p))
packet=d.get('packet',{})
assert d.get('ok') is True, d
assert packet.get('schema') == 'uiai.artifact_result.v2', packet
assert packet.get('artifact_schema') == 'uiai.focusa_research_diagnostics_packet.v1', packet
assert packet.get('delivery_state') == 'ready', packet
assert str(packet.get('artifact_url','')).startswith('https://'), packet
assert str(packet.get('portable_url','')).startswith('https://'), packet
assert packet.get('recommended_focusa',{}).get('preferred_tool') == 'focusa_browser_diagnostics_intake', packet.get('recommended_focusa')
assert d.get('packet_bytes', 999999) <= 8192, d.get('packet_bytes')
print(f"packet endpoint ci smoke ok: out={p} bytes={d.get('packet_bytes')} preferred_tool={packet.get('recommended_focusa',{}).get('preferred_tool')}")
PY
