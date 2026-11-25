#!/usr/bin/env bash
# #309: reject obsolete Caddy LB health paths before they cause false worker
# outages. The engine health contract is /api/health/browser; the legacy
# /health probe 404s on current engine builds and silently marks healthy
# workers unavailable to the shared route.
#
# Usage: check-ovh-lb-health.sh [caddyfile]
#   CADDYFILE env var or default /etc/caddy/uiai-rr.Caddyfile on the LB host.
# Exit 0 = every health_uri matches the current contract.
# Exit 1 = obsolete/missing health probe found (or unreadable config).

set -euo pipefail

CADDYFILE="${1:-${CADDYFILE:-/etc/caddy/uiai-rr.Caddyfile}}"
EXPECTED="health_uri /api/health/browser"

if [[ ! -r "$CADDYFILE" ]]; then
  echo "FAIL: cannot read $CADDYFILE" >&2
  exit 1
fi

if ! grep -q "reverse_proxy" "$CADDYFILE"; then
  echo "FAIL: $CADDYFILE has no reverse_proxy block to health-check" >&2
  exit 1
fi

health_lines=$(grep -c "health_uri" "$CADDYFILE" || true)
if [[ "$health_lines" -eq 0 ]]; then
  echo "FAIL: reverse_proxy without health_uri — an unhealthy worker would keep receiving traffic" >&2
  exit 1
fi

if grep -q "health_uri /health$\|health_uri /health[^/]" "$CADDYFILE"; then
  echo "FAIL: obsolete health_uri /health probe found — engine contract is /api/health/browser" >&2
  exit 1
fi

if ! grep -q "$EXPECTED" "$CADDYFILE"; then
  echo "FAIL: expected '$EXPECTED' not found in $CADDYFILE" >&2
  exit 1
fi

echo "OK: $CADDYFILE probes the current engine health contract (/api/health/browser)"
