import importlib.util
import json
from pathlib import Path
import shutil
import tempfile
import unittest
from urllib.parse import unquote, urlsplit

ROOT = Path(__file__).resolve().parent.parent
spec = importlib.util.spec_from_file_location("machine_gate", ROOT / "tests/machine_readability_gate.py")
gate = importlib.util.module_from_spec(spec)
spec.loader.exec_module(gate)


class SchemaRegistryTests(unittest.TestCase):
    def test_all_manifest_entries(self):
        entries = gate.load(ROOT / "contracts/schemas/manifest.json")["schemas"]
        self.assertEqual(len(gate.validate_schema_registry(ROOT)), len(entries))
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            shutil.copytree(ROOT / "contracts/schemas", root / "contracts/schemas")
            for entry in entries:
                with self.subTest(schema=entry["id"]):
                    path = root / entry["src"]
                    original = path.read_text()
                    changed = json.loads(original)
                    changed["$id"] = "wrong-canonical-identity"
                    path.write_text(json.dumps(changed))
                    with self.assertRaisesRegex(AssertionError, r"\$id mismatch"):
                        gate.validate_schema_registry(root)
                    path.write_text(original)

    def test_wire_identity_is_not_waived_by_canonical_url(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            shutil.copytree(ROOT / "contracts/schemas", root / "contracts/schemas")
            manifest = gate.load(root / "contracts/schemas/manifest.json")
            for entry in manifest["schemas"]:
                if entry.get("schema_id"):
                    with self.subTest(schema=entry["id"]):
                        path = root / entry["src"]
                        original = path.read_text()
                        changed = json.loads(original)
                        changed["properties"]["schema"] = {"const": "wrong-wire-identity"}
                        path.write_text(json.dumps(changed))
                        with self.assertRaisesRegex(AssertionError, "wire discriminator mismatch"):
                            gate.validate_schema_registry(root)
                        path.write_text(original)

    def test_generated_embedded_and_relative_refs(self):
        document = gate.load(ROOT / "docs/api/openapi.json")
        self.assertEqual(document, gate.load(ROOT / "internal/routes/openapi_embed.json"))
        entries = gate.load(ROOT / "contracts/schemas/manifest.json")["schemas"]
        for entry in entries:
            with self.subTest(schema=entry["id"]):
                path = ROOT / entry["src"]
                schema = gate.load(path)
                key = entry["id"].replace(".", "_").replace("-", "_")
                component = document["components"]["schemas"][key]
                self.assertEqual(component["$id"], schema["$id"])
                self.assertEqual(schema, gate.load(ROOT / "internal/routes/schemas_embed" / (entry["id"] + ".schema.json")))
                self.check_refs(schema, path)

    def check_refs(self, node, source):
        if isinstance(node, dict):
            if "$ref" in node:
                reference = urlsplit(node["$ref"])
                if not reference.scheme:
                    target = source.parent / reference.path if reference.path else source
                    resolved = gate.load(target)
                    fragment = unquote(reference.fragment)
                    if fragment.startswith("/"):
                        for part in fragment[1:].split("/"):
                            resolved = resolved[part.replace("~1", "/").replace("~0", "~")]
            for value in node.values():
                self.check_refs(value, source)
        elif isinstance(node, list):
            for value in node:
                self.check_refs(value, source)


if __name__ == "__main__":
    unittest.main()
