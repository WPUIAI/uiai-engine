#!/usr/bin/env bash
set -euo pipefail

ENGINE_URL="${UIAI_ENGINE_URL:-http://localhost:7456}"
TIMEOUT_SECONDS="${UIAI_SMOKE_TIMEOUT_SECONDS:-20}"
OUT="${OUT:-/tmp/uiai-failed-network-diagnostics.json}"
SITE_PORT="${UIAI_FAILED_NETWORK_SITE_PORT:-18567}"
TMPDIR="$(mktemp -d)"
AUTH_ARGS=()
if [[ -n "${UIAI_API_KEY:-}" ]]; then
  AUTH_ARGS=(-H "X-API-Key: ${UIAI_API_KEY}")
elif [[ -n "${UIAI_BEARER_TOKEN:-}" ]]; then
  AUTH_ARGS=(-H "Authorization: Bearer ${UIAI_BEARER_TOKEN}")
fi
cleanup(){
  [[ -n "${SITE_PID:-}" ]] && kill "$SITE_PID" 2>/dev/null || true
  rm -rf "$TMPDIR"
}
trap cleanup EXIT

need(){ command -v "$1" >/dev/null 2>&1 || { echo "missing required command: $1" >&2; exit 1; }; }
need curl
need jq
need python3

mkdir -p "$TMPDIR/site"
cat > "$TMPDIR/site/index.html" <<'HTML'
<!doctype html>
<title>UIAI failed network diagnostics smoke</title>
<script>
  fetch('/missing-uiai-network-proof.json')
    .catch((err) => console.error('failed-network-proof', err && err.message));
</script>
<body>failed network diagnostics proof</body>
HTML
python3 -m http.server "$SITE_PORT" -d "$TMPDIR/site" >/tmp/uiai-failed-network-site.log 2>&1 & SITE_PID=$!
sleep 0.3

curl_json(){ curl -fsS --max-time "$TIMEOUT_SECONDS" "${AUTH_ARGS[@]}" "$@"; }
TARGET="http://127.0.0.1:${SITE_PORT}/"
open_json="$(curl_json -X POST "$ENGINE_URL/api/session" -H "Content-Type: application/json" -d "{\"url\":\"$TARGET\",\"width\":800,\"height\":600}")"
SID="$(printf '%s' "$open_json" | jq -r '.session.id')"
if [[ -z "$SID" || "$SID" == "null" ]]; then
  echo "failed to open UIAI session" >&2
  exit 1
fi
sleep 0.6
diag="$(curl_json "$ENGINE_URL/api/session/$SID/diagnostics?limit=50&failed_only=true")"
curl_json -X DELETE "$ENGINE_URL/api/session/$SID" >/dev/null || true
printf '%s' "$diag" > "$OUT"

jq -e '.summary.failed_requests >= 1 and ([.failed_requests[].url] | any(contains("/missing-uiai-network-proof.json")))' "$OUT" >/dev/null
printf 'failed-network diagnostics smoke ok: session=%s out=%s failed_requests=%s\n' "$SID" "$OUT" "$(jq -r '.summary.failed_requests' "$OUT")"
