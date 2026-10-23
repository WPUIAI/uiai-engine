#!/usr/bin/env python3
"""Machine-readability gate — MR-P0-01: every schema id has a JSON Schema file and fixture validates."""
import json, pathlib, sys
ROOT = pathlib.Path(__file__).parent.parent
SCHEMAS = ROOT / "contracts/schemas"
MANIFEST = SCHEMAS / "manifest.json"

def _load(p): return json.loads(p.read_text())

manifest = _load(MANIFEST)
ids = {s["id"] for s in manifest["schemas"]}
# 1) every schema file exists and $id matches
for entry in manifest["schemas"]:
    src = ROOT / entry["src"]
    assert src.exists(), f"missing schema file {src}"
    sch = _load(src)
    assert sch.get("$id") == entry["id"], f"$id mismatch {src}: {sch.get('$id')} != {entry['id']}"
    assert sch.get("$schema"), f"missing $schema in {src}"

# 2) fixture validates structurally against desktop_presentation_request schema
fixture = _load(ROOT / "tests/fixtures/desktop-presentation/valid-contracts.json")
req = fixture["presentation_request"]
schema = _load(SCHEMAS / "uiai.desktop_presentation_request.v1.schema.json")
for k in schema["required"]:
    assert k in req, f"fixture missing required {k}"
assert req["schema"] == schema["$id"]
# scope workstream invariant already covered by TS parser, double-check here
sr = req.get("scope_ref", {})
if sr.get("project_root_key") and sr.get("continuity_id"):
    derived = sr["project_root_key"].rstrip("/") + "::" + sr["continuity_id"]
    assert sr["workstream_key"] == derived, f"workstream_key {sr['workstream_key']} != {derived}"

# 3) bundle manifest schema still valid draft-07 shape
bundle = _load(SCHEMAS / "focusa.cockpit.bundle_manifest.v1.schema.json")
assert bundle["required"] == ["schema","version","built_at","source_commit","artifacts"]

print("machine_readability_gate PASS —", len(ids), "schemas, fixture vs desktop_presentation_request OK")
