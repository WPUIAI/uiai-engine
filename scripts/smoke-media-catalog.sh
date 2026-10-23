#!/usr/bin/env bash
# Probe media route health without treating authentication as entitlement.
set -euo pipefail
ENGINE_URL="${UIAI_ENGINE_URL:-http://localhost:7456}"
set -- -sS --max-time "${UIAI_SMOKE_TIMEOUT_SECONDS:-20}" -w $'\n%{http_code}'
if [[ -n "${UIAI_API_KEY:-}" ]]; then
  set -- "$@" -H "X-API-Key: ${UIAI_API_KEY}"
elif [[ -n "${UIAI_BEARER_TOKEN:-}" ]]; then
  set -- "$@" -H "Authorization: Bearer ${UIAI_BEARER_TOKEN}"
fi
response=$(curl "$@" "$ENGINE_URL/api/media/frame/catalog")
status=${response##*$'\n'}
body=${response%$'\n'*}
case "$status" in
  200)
    printf '%s' "$body" | jq -e '.count | type == "number" and . > 0' >/dev/null
    printf '%s\n' 'media catalog: available; catalog content verified'
    ;;
  402)
    printf '%s\n' 'media catalog: route reachable; entitlement required (HTTP 402); catalog content not verified'
    ;;
  *)
    printf 'media catalog: unexpected HTTP %s\n' "$status" >&2
    exit 1
    ;;
esac
