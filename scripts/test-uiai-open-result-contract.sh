#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="${UIAI_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
cd "$ROOT_DIR"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

printf '{"results":[]}' >"$TMPDIR/empty.json"
if UIAI_EVIDENCE_SCOPE_JSON='{"project_ref":' scripts/uiai-open-result.sh --search-json "$TMPDIR/empty.json" >"$TMPDIR/malformed.json" 2>"$TMPDIR/malformed.err"; then
  echo "malformed evidence scope was accepted" >&2
  exit 1
fi
python3 - "$TMPDIR/malformed.json" <<'PY'
import json, sys
result=json.load(open(sys.argv[1], encoding='utf-8'))
assert result == {
    'ok': False,
    'error_class': 'invalid_evidence_scope',
    'message': 'UIAI_EVIDENCE_SCOPE_JSON must be valid JSON: Expecting value',
}, result
PY

cat >"$TMPDIR/nested-raw.json" <<'JSON'
{
  "delivery_state": "ready",
  "artifact_ref": "artifact:test",
  "artifact_url": "https://epwa-ci.invalid/records/artifact-test/",
  "portable_url": "https://epwa-ci.invalid/packages/artifact-test.zip",
  "epwa_delivery": {
    "schema": "uiai.epwa_delivery.v1",
    "state": "ready",
    "artifact": {"artifact_ref": "artifact:test"},
    "epwa": {
      "record_url": "https://epwa-ci.invalid/records/artifact-test/",
      "portable_url": "https://epwa-ci.invalid/packages/artifact-test.zip"
    }
  },
  "results": [],
  "session": {"screenshot": "forbidden-inline-bytes"}
}
JSON
scope='{"project_ref":"project:test","workstream_ref":"workstream:test","workset_ref":"workset:test","callgraph_ref":"callgraph:test","workpoint_ref":"workpoint:test","work_item_ref":"work-item:test","continuity_ref":"continuity:test"}'
if UIAI_EVIDENCE_SCOPE_JSON="$scope" scripts/uiai-open-result.sh --search-json "$TMPDIR/nested-raw.json" >"$TMPDIR/raw.out" 2>"$TMPDIR/raw.err"; then
  echo "nested raw artifact field was accepted" >&2
  exit 1
fi
grep -F 'search: forbidden raw artifact field: $.session.screenshot' "$TMPDIR/raw.err" >/dev/null

echo "uiai-open-result contract: PASS"
