#!/usr/bin/env bash
set -euo pipefail
ENGINE_URL=${ENGINE_URL:-http://127.0.0.1:7456}
PUBLIC_HOST=${PUBLIC_HOST:-fpv.wpuiai.com}
OUT_DIR=${OUT_DIR:-/tmp/uiai-fpv-visual}
BASELINE_DIR=${BASELINE_DIR:-tests/fixtures/fpv-visual-baselines}
UPDATE_BASELINE=${UPDATE_BASELINE:-0}
DIFF_THRESHOLD=${DIFF_THRESHOLD:-0.28}
: "${UIAI_EVIDENCE_SCOPE_JSON:?UIAI_EVIDENCE_SCOPE_JSON is required for EPWA-producing browser smoke}"
jq -e 'type == "object"' >/dev/null <<<"$UIAI_EVIDENCE_SCOPE_JSON" || { echo "UIAI_EVIDENCE_SCOPE_JSON must be a JSON object" >&2; exit 2; }
mkdir -p "$OUT_DIR"

require_epwa() {
  jq -e '
    .epwa_delivery.schema == "uiai.epwa_delivery.v1" and
    .delivery_state == "ready" and .epwa_delivery.state == .delivery_state and
    .epwa_delivery.artifact.artifact_ref == .artifact_ref and
    (.artifact_url | type == "string" and startswith("https://")) and
    (.portable_url | type == "string" and startswith("https://")) and
    .artifact_url == .epwa_delivery.epwa.record_url and
    .portable_url == .epwa_delivery.epwa.portable_url and
    ([.. | objects | keys[]] | any(. == "screenshot" or . == "imageBase64" or . == "image_base64" or . == "artifact_path" or . == "result_path" or . == "result_url") | not)' >/dev/null
}

cleanup() {
  curl -fsS "$ENGINE_URL/api/session" 2>/dev/null | jq -r '.sessions[].id' | while read -r id; do
    [ -n "$id" ] && curl -fsS -X DELETE "$ENGINE_URL/api/session/$id" >/dev/null 2>&1 || true
  done
}
cleanup

source_payload=$(jq -cn --argjson scope "$UIAI_EVIDENCE_SCOPE_JSON" '{url:"https://project-nullframe.vercel.app/",width:1440,height:1000,focusa_scope:$scope}')
source_json=$(curl -fsS -X POST "$ENGINE_URL/api/session" -H 'Content-Type: application/json' -d "$source_payload")
printf '%s' "$source_json" | require_epwa
source_id=$(printf '%s' "$source_json" | jq -r .session.id)
share_json=$(curl -fsS -X POST "$ENGINE_URL/api/fpv/share" -H 'Content-Type: application/json' -d "{\"session_id\":\"$source_id\",\"expires_minutes\":10,\"controls\":true}")
printf '%s' "$share_json" | require_epwa
token=$(printf '%s' "$share_json" | jq -r .token)
url="https://$PUBLIC_HOST/m/$token"

status_json=$(curl -ksS "$url/status")
printf '%s' "$status_json" | jq -e '.mode and .transport.primary == "cdp_screencast" and .transport.mjpeg_url and .context.project.name == "uiai-engine"' >/dev/null
api_code=$(curl -ksS -o /dev/null -w '%{http_code}' "https://$PUBLIC_HOST/api/health" || true)
[ "$api_code" = "404" ] || { echo "expected $PUBLIC_HOST/api/health 404, got $api_code" >&2; exit 1; }

for spec in 375x812 768x1024 1024x900 1440x1000; do
  w=${spec%x*}; h=${spec#*x}
  viewer_payload=$(jq -cn --arg url "$url" --argjson width "$w" --argjson height "$h" --argjson scope "$UIAI_EVIDENCE_SCOPE_JSON" '{url:$url,width:$width,height:$height,focusa_scope:$scope}')
  viewer=$(curl -fsS -X POST "$ENGINE_URL/api/session" -H 'Content-Type: application/json' -d "$viewer_payload")
  printf '%s' "$viewer" | require_epwa
  vid=$(printf '%s' "$viewer" | jq -r .session.id)
  sleep 2
  shot=$(curl -fsS -X POST "$ENGINE_URL/api/session/$vid/screenshot" -H 'Content-Type: application/json' -d '{"format":"jpeg","quality":70}')
  printf '%s' "$shot" | require_epwa
  portable_url=$(printf '%s' "$shot" | jq -r .portable_url)
  curl -fsS "$portable_url" -o "$OUT_DIR/$spec.zip"
  unzip -p "$OUT_DIR/$spec.zip" screenshot.jpeg > "$OUT_DIR/$spec.jpg"
  [ -s "$OUT_DIR/$spec.jpg" ] || { echo "EPWA portable package omitted screenshot.jpeg for $spec" >&2; exit 1; }
  if [ "$UPDATE_BASELINE" = "1" ]; then
    mkdir -p "$BASELINE_DIR"
    cp "$OUT_DIR/$spec.jpg" "$BASELINE_DIR/$spec.jpg"
  elif [ -f "$BASELINE_DIR/$spec.jpg" ]; then
    if ! command -v compare >/dev/null 2>&1; then
      echo "ImageMagick compare is required for FPV baseline diff when $BASELINE_DIR/$spec.jpg exists" >&2
      exit 1
    fi
    diff_path="$OUT_DIR/$spec.diff.jpg"
    metric=$(compare -metric RMSE "$BASELINE_DIR/$spec.jpg" "$OUT_DIR/$spec.jpg" "$diff_path" 2>&1 | awk '{print $2}' | tr -d '()' || true)
    if ! python3 - <<PY
m=float("${metric:-0}")
th=float("$DIFF_THRESHOLD")
assert m <= th
PY
    then
      echo "visual diff failed for $spec: RMSE normalized ${metric:-unknown} > $DIFF_THRESHOLD" >&2
      echo "baseline=$BASELINE_DIR/$spec.jpg" >&2
      echo "current=$OUT_DIR/$spec.jpg" >&2
      echo "diff=$diff_path" >&2
      exit 1
    fi
  else
    echo "fpv visual $spec baseline missing; set UPDATE_BASELINE=1 to capture $BASELINE_DIR/$spec.jpg"
  fi
  diag=$(curl -fsS "$ENGINE_URL/api/session/$vid/diagnostics?level=error&limit=40")
  printf '%s' "$diag" | require_epwa
  printf '%s' "$diag" | jq -e '(.exceptions|length)==0 and (.console|length)==0' >/dev/null
  curl -fsS -X DELETE "$ENGINE_URL/api/session/$vid" >/dev/null || true
  echo "fpv visual $spec ok"
done

curl -fsS -X DELETE "$ENGINE_URL/api/session/$source_id" >/dev/null || true
echo "fpv visual breakpoints ok: $url out=$OUT_DIR"
