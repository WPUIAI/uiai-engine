#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="${UIAI_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
cd "$ROOT_DIR"
TMPDIR="$(mktemp -d)"
cleanup(){ local code=$?; if [[ -n "${SITE_PID:-}" ]]; then kill "$SITE_PID" 2>/dev/null || true; wait "$SITE_PID" 2>/dev/null || true; fi; rm -rf "$TMPDIR"; exit "$code"; }
trap cleanup EXIT
mkdir -p "$TMPDIR/site"
printf '<html><body><main><h1>UIAI open result smoke</h1><p>Bounded read target.</p></main></body></html>' > "$TMPDIR/site/index.html"
SITE_PORT="${SITE_PORT:-8797}"
python3 -m http.server "$SITE_PORT" -d "$TMPDIR/site" >"$TMPDIR/site.log" 2>&1 & SITE_PID=$!
for _ in $(seq 1 30); do curl -fsS "http://127.0.0.1:$SITE_PORT/" >/dev/null 2>&1 && break; sleep 0.2; done
cat > "$TMPDIR/search.json" <<JSON
{"results":[{"rank":1,"title":"Smoke result","url":"http://127.0.0.1:$SITE_PORT/","evidence_ref":"uiai-search:smoke:open-result:1"}]}
JSON
scripts/uiai-open-result.sh --search-json "$TMPDIR/search.json" --index 1 --out "$TMPDIR/report.json" >"$TMPDIR/stdout.json"
jq -e '.ok == true and .selected_url != "" and .session_id != null and (.evidence_refs | length) >= 1 and .cleanup.recommended_action == "close_when_done"' "$TMPDIR/report.json" >/dev/null
SID=$(jq -r '.session_id' "$TMPDIR/report.json")
curl -fsS -X DELETE "${UIAI_ENGINE_URL:-http://localhost:7456}/api/session/$SID" >/dev/null || true
echo "open result smoke ok: report=$TMPDIR/report.json"
