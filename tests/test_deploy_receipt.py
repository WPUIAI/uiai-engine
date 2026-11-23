#!/usr/bin/env python3
"""#318: the engine upgrade path must rewrite installation-receipt.json as
part of activation, after the new binary passes health. These tests pin the
canonical deploy script's receipt contract: ordering, bound fields, owner
and atomic write, and fail-closed behavior when health never passes."""
import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parent.parent
SCRIPT = (ROOT / "scripts" / "deploy-engine-ovh.sh").read_text()


class DeployInstallationReceiptTest(unittest.TestCase):
    def test_receipt_written_after_health_readiness_gate(self):
        gate_end = SCRIPT.index("# END HEALTH READINESS GATE")
        receipt_begin = SCRIPT.index("# BEGIN INSTALLATION RECEIPT")
        self.assertLess(gate_end, receipt_begin, "receipt must follow the health gate")

    def test_health_failure_aborts_before_receipt(self):
        failure_exit = SCRIPT.index("health readiness failed")
        receipt_begin = SCRIPT.index("# BEGIN INSTALLATION RECEIPT")
        self.assertLess(failure_exit, receipt_begin, "a failed activation must never record a receipt")

    def test_receipt_binds_version_sha_health_and_config(self):
        for token in (
            "uiai.installation_receipt.v1",
            '"version": version_output',
            '"binary_sha256": binary_sha',
            '"health_http_code": http_code or "unknown"',
            '"health_version": None',
            '"config_path": config_path or "unknown"',
            '"config_sha256": config_sha or None',
        ):
            self.assertIn(token, SCRIPT, f"receipt field binding missing: {token}")

    def test_receipt_records_only_verified_health_version(self):
        self.assertIn(
            'if isinstance(health, dict) and isinstance(health.get("version"), str) and health["version"]:',
            SCRIPT,
            "health_version must come from the engine surface, never be fabricated",
        )

    def test_receipt_written_atomically_as_install_owner(self):
        self.assertIn(
            'owner_run install -m 0640 "$receipt_tmp" "$install_root/installation-receipt.json"',
            SCRIPT,
        )
        self.assertIn(
            'version_output=$("$binary_path" -version)',
            SCRIPT,
        )
        binary_sha = SCRIPT.index('binary_sha=$(sha256sum "$binary_path"')
        receipt_write = SCRIPT.index('owner_run install -m 0640 "$receipt_tmp"')
        self.assertLess(binary_sha, receipt_write, "receipt sha must be computed from the installed binary")


if __name__ == "__main__":
    unittest.main()
