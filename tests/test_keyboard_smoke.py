import importlib.util
import json
from pathlib import Path
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parent.parent
spec = importlib.util.spec_from_file_location("keyboard_smoke", ROOT / "scripts/smoke-keyboard-delivery.py")
smoke = importlib.util.module_from_spec(spec)
spec.loader.exec_module(smoke)


class KeyboardSmokeTests(unittest.TestCase):
    def probe(self, keys, focus=None):
        opened, closed = [], []
        def fake(method, path, body=None):
            if path == "/api/status":
                return {"type": "status"}
            if path == "/api/session":
                sid = "fixture-" + str(len(opened))
                opened.append(sid)
                return {"session": {"id": sid}}
            if method == "DELETE":
                closed.append(path.rsplit("/", 1)[-1])
                return {"status": "closed"}
            if path.endswith("/eval") and "JSON.stringify" in body["js"]:
                return {"result": json.dumps({"keys": keys, "focused": True, "keydown_focus": [True, True] if focus is None else focus}), "delivery_state": "blocked"}
            return {"result": "probe-ready"}
        return fake, opened, closed

    def test_acknowledgement_alone_cannot_pass(self):
        fake, opened, closed = self.probe([])
        with patch.object(smoke, "request", fake):
            with self.assertRaisesRegex(RuntimeError, "without focused DOM keydown"):
                smoke.run()
        self.assertEqual(closed, list(reversed(opened)))

    def test_post_snapshot_focus_cannot_mask_unfocused_keydown(self):
        fake, opened, closed = self.probe(["a", "Enter"], [False, False])
        with patch.object(smoke, "request", fake):
            with self.assertRaisesRegex(RuntimeError, "without focused DOM keydown"):
                smoke.run()
        self.assertEqual(closed, list(reversed(opened)))

    def test_input_proof_does_not_claim_artifact_delivery(self):
        fake, opened, closed = self.probe(["a", "Enter"])
        with patch.object(smoke, "request", fake):
            result = smoke.run()
        self.assertEqual(result["input_delivery"], "verified")
        self.assertEqual(result["artifact_delivery_state"], "blocked")
        self.assertIn("not proof", result["truth_notice"])
        self.assertEqual(closed, list(reversed(opened)))


if __name__ == "__main__":
    unittest.main()
