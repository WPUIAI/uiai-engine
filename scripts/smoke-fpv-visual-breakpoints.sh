#!/usr/bin/env bash
set -euo pipefail
ENGINE_URL=${ENGINE_URL:-http://127.0.0.1:7456}
PUBLIC_HOST=${PUBLIC_HOST:-fpv.wpuiai.com}
OUT_DIR=${OUT_DIR:-/tmp/uiai-fpv-visual}
BASELINE_DIR=${BASELINE_DIR:-tests/fixtures/fpv-visual-baselines}
UPDATE_BASELINE=${UPDATE_BASELINE:-0}
DIFF_THRESHOLD=${DIFF_THRESHOLD:-0.28}
mkdir -p "$OUT_DIR"

cleanup() {
  curl -fsS "$ENGINE_URL/api/session" 2>/dev/null | jq -r '.sessions[].id' | while read -r id; do
    [ -n "$id" ] && curl -fsS -X DELETE "$ENGINE_URL/api/session/$id" >/dev/null 2>&1 || true
  done
}
cleanup

source_json=$(curl -fsS -X POST "$ENGINE_URL/api/session" -H 'Content-Type: application/json' -d '{"url":"https://project-nullframe.vercel.app/","width":1440,"height":1000}')
source_id=$(printf '%s' "$source_json" | jq -r .session.id)
share_json=$(curl -fsS -X POST "$ENGINE_URL/api/fpv/share" -H 'Content-Type: application/json' -d "{\"session_id\":\"$source_id\",\"expires_minutes\":10,\"controls\":true}")
token=$(printf '%s' "$share_json" | jq -r .token)
url="https://$PUBLIC_HOST/m/$token"

status_json=$(curl -ksS "$url/status")
printf '%s' "$status_json" | jq -e '.mode and .transport.primary == "mjpeg" and .context.project.name == "uiai-engine"' >/dev/null
api_code=$(curl -ksS -o /dev/null -w '%{http_code}' "https://$PUBLIC_HOST/api/health" || true)
[ "$api_code" = "404" ] || { echo "expected $PUBLIC_HOST/api/health 404, got $api_code" >&2; exit 1; }

for spec in 375x812 768x1024 1024x900 1440x1000; do
  w=${spec%x*}; h=${spec#*x}
  viewer=$(curl -fsS -X POST "$ENGINE_URL/api/session" -H 'Content-Type: application/json' -d "{\"url\":\"$url\",\"width\":$w,\"height\":$h}")
  vid=$(printf '%s' "$viewer" | jq -r .session.id)
  sleep 2
  shot=$(curl -fsS -X POST "$ENGINE_URL/api/session/$vid/screenshot" -H 'Content-Type: application/json' -d '{"format":"jpeg","quality":70}')
  printf '%s' "$shot" | jq -r .screenshot > "$OUT_DIR/$spec.b64"
  base64 -d "$OUT_DIR/$spec.b64" > "$OUT_DIR/$spec.jpg"
  [ -s "$OUT_DIR/$spec.jpg" ] || { echo "missing screenshot for $spec" >&2; exit 1; }
  if [ "$UPDATE_BASELINE" = "1" ]; then
    mkdir -p "$BASELINE_DIR"
    cp "$OUT_DIR/$spec.jpg" "$BASELINE_DIR/$spec.jpg"
  elif [ -f "$BASELINE_DIR/$spec.jpg" ] && command -v compare >/dev/null 2>&1; then
    metric=$(compare -metric RMSE "$BASELINE_DIR/$spec.jpg" "$OUT_DIR/$spec.jpg" null: 2>&1 | awk '{print $2}' | tr -d '()' || true)
    python3 - <<PY
m=float("${metric:-0}")
th=float("$DIFF_THRESHOLD")
assert m <= th, f"visual diff $spec RMSE normalized {m} > {th}"
PY
  fi
  diag=$(curl -fsS "$ENGINE_URL/api/session/$vid/diagnostics?level=error&limit=40")
  printf '%s' "$diag" | jq -e '(.exceptions|length)==0 and (.console|length)==0' >/dev/null
  curl -fsS -X DELETE "$ENGINE_URL/api/session/$vid" >/dev/null || true
  echo "fpv visual $spec ok"
done

curl -fsS -X DELETE "$ENGINE_URL/api/session/$source_id" >/dev/null || true
echo "fpv visual breakpoints ok: $url out=$OUT_DIR"
