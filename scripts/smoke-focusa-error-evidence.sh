#!/usr/bin/env bash
set -euo pipefail

ENGINE_URL="${UIAI_ENGINE_URL:-http://localhost:7456}"
TIMEOUT_SECONDS="${UIAI_SMOKE_TIMEOUT_SECONDS:-20}"
OUT="${OUT:-/tmp/uiai-error-focusa-evidence.json}"
AUTH_ARGS=()
if [[ -n "${UIAI_API_KEY:-}" ]]; then
  AUTH_ARGS=(-H "X-API-Key: ${UIAI_API_KEY}")
elif [[ -n "${UIAI_BEARER_TOKEN:-}" ]]; then
  AUTH_ARGS=(-H "Authorization: Bearer ${UIAI_BEARER_TOKEN}")
fi

need(){ command -v "$1" >/dev/null 2>&1 || { echo "missing required command: $1" >&2; exit 1; }; }
need curl
need jq

curl_json(){ curl -fsS --max-time "$TIMEOUT_SECONDS" "${AUTH_ARGS[@]}" "$@"; }

# Trigger a harmless, bounded browser/session error if possible so /api/errors has a fresh event.
set +e
curl_json -X POST "$ENGINE_URL/api/session" \
  -H "Content-Type: application/json" \
  -d '{"url":"file:///tmp/uiai-focusa-evidence-smoke.html","width":800,"height":600}' >/tmp/uiai-error-focusa-trigger.json 2>/dev/null
set -e

errors_json="$(curl_json "$ENGINE_URL/api/errors?limit=5")"
error_event="$(printf '%s' "$errors_json" | jq -c '.events[0] // empty')"
if [[ -z "$error_event" ]]; then
  echo "no /api/errors event available for Focusa evidence smoke" >&2
  exit 1
fi

error_id="$(printf '%s' "$error_event" | jq -r '.id')"
error_class="$(printf '%s' "$error_event" | jq -r '.class // "unknown"')"
source="$(printf '%s' "$error_event" | jq -r '.source // "engine"')"
message="$(printf '%s' "$error_event" | jq -r '.message // "UIAI error event"')"

evidence_ref="uiai-error:${error_id}"
diagnostics_ref="/api/errors?limit=20&source=${source}&class=${error_class}"

jq -n \
  --arg target_ref "UIAI /api/errors ${source}/${error_class}" \
  --arg result "UIAI error evidence available: ${error_id} class=${error_class} message=${message}" \
  --arg evidence_ref "$evidence_ref" \
  --arg diagnostics_ref "$diagnostics_ref" \
  --arg intake_tool "focusa_evidence_capture" \
  --argjson event "$error_event" \
  '{ok:true, focusa_evidence:{target_ref:$target_ref,result:$result,evidence_ref:$evidence_ref,diagnostics_ref:$diagnostics_ref,intake_tool:$intake_tool}, event:{id:$event.id, source:$event.source, class:$event.class, status:$event.status, diagnostics:$diagnostics_ref}}' > "$OUT"

jq -e '.ok == true and (.focusa_evidence.evidence_ref | startswith("uiai-error:"))' "$OUT" >/dev/null
printf 'focusa uiai_error evidence smoke ok: evidence_ref=%s diagnostics_ref=%s out=%s\n' "$evidence_ref" "$diagnostics_ref" "$OUT"
