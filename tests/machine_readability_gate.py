#!/usr/bin/env python3
"""MR-P0-01: validate schema registry identity and the presentation fixture."""
import json
import pathlib

ROOT = pathlib.Path(__file__).resolve().parent.parent


def load(path):
    return json.loads(path.read_text())


def validate_schema_registry(root):
    manifest = load(root / "contracts/schemas/manifest.json")
    seen_ids, seen_schema_ids = set(), set()
    for entry in manifest["schemas"]:
        wire_id = entry["id"]
        canonical_id = entry.get("schema_id", wire_id)
        assert wire_id not in seen_ids, f"duplicate manifest id: {wire_id}"
        assert canonical_id not in seen_schema_ids, f"duplicate canonical schema id: {canonical_id}"
        seen_ids.add(wire_id)
        seen_schema_ids.add(canonical_id)
        src = root / entry["src"]
        assert src.exists(), f"missing schema file {src}"
        schema = load(src)
        assert schema.get("$id") == canonical_id, f"$id mismatch {src}: {schema.get('$id')} != {canonical_id}"
        assert schema.get("$schema"), f"missing $schema in {src}"
        if canonical_id != wire_id:
            property_schema = schema.get("properties", {}).get("schema", {})
            discriminators = property_schema.get("enum", [property_schema.get("const")])
            expected = entry.get("wire_schema_ids", [wire_id])
            assert expected and all(isinstance(value, str) for value in expected), f"missing wire discriminator identities {src}"
            assert set(discriminators) == set(expected), f"wire discriminator mismatch {src}: {discriminators} != {expected}"
    return seen_ids


def main():
    ids = validate_schema_registry(ROOT)
    fixture = load(ROOT / "tests/fixtures/desktop-presentation/valid-contracts.json")
    req = fixture["presentation_request"]
    schema = load(ROOT / "contracts/schemas/uiai.desktop_presentation_request.v1.schema.json")
    for key in schema["required"]:
        assert key in req, f"fixture missing required {key}"
    assert req["schema"] == schema["$id"]
    scope = req.get("scope_ref", {})
    if scope.get("project_root_key") and scope.get("continuity_id"):
        derived = scope["project_root_key"].rstrip("/") + "::" + scope["continuity_id"]
        assert scope["workstream_key"] == derived, f"workstream_key {scope['workstream_key']} != {derived}"
    bundle = load(ROOT / "contracts/schemas/focusa.cockpit.bundle_manifest.v1.schema.json")
    assert bundle["required"] == ["schema", "version", "built_at", "source_commit", "artifacts"]
    print("machine_readability_gate PASS —", len(ids), "schemas, fixture vs desktop_presentation_request OK")


if __name__ == "__main__":
    main()
