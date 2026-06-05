#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="${UIAI_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
cd "$ROOT_DIR"
TMPDIR=$(mktemp -d); trap 'rm -rf "$TMPDIR"' EXIT
cat > "$TMPDIR/diagnostics.json" <<'JSON'
{"summary":{"console_errors":1,"failed_requests":1},"console":[{"text":"Authorization Bearer abc should redact"}],"failed_requests":[{"url":"https://example.com/api?token=abc","status":500}],"focusa_evidence":{"evidence_ref":"uiai-diagnostics:session=smoke:seq=1","target_ref":"browser:smoke"}}
JSON
scripts/render-diagnostics-artifact.py "$TMPDIR/diagnostics.json" > "$TMPDIR/diag.txt"
scripts/render-diagnostics-artifact.py docs/examples/focusa-packets/diagnose-packet.example.json > "$TMPDIR/packet.txt"
grep -q 'Diagnostics artifact' "$TMPDIR/diag.txt"
grep -q 'Packet: schema=uiai.focusa_research_diagnostics_packet.v1' "$TMPDIR/packet.txt"
! grep -q 'Bearer abc' "$TMPDIR/diag.txt"
echo "diagnostics artifact render smoke ok"
