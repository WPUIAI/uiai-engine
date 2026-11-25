import os
import subprocess
import tempfile
import unittest

SCRIPT = os.path.join(os.path.dirname(__file__), "..", "scripts", "check-ovh-lb-health.sh")

VALID = """http://:7460 {
	bind 127.0.0.1
	reverse_proxy 127.0.0.1:7457 127.0.0.1:7456 {
		lb_policy first
		health_uri /api/health/browser
		health_interval 5s
	}
}
"""

OBSOLETE = VALID.replace("health_uri /api/health/browser", "health_uri /health")

NO_PROBE = VALID.replace("\t\thealth_uri /api/health/browser\n", "")


class CheckOvhLbHealth(unittest.TestCase):
    def run_script(self, content):
        with tempfile.NamedTemporaryFile("w", suffix=".Caddyfile", delete=False) as f:
            f.write(content)
            path = f.name
        try:
            return subprocess.run(
                ["bash", SCRIPT, path], capture_output=True, text=True, timeout=30
            )
        finally:
            os.unlink(path)

    def test_current_contract_passes(self):
        result = self.run_script(VALID)
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_obsolete_health_path_is_rejected(self):
        result = self.run_script(OBSOLETE)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("obsolete", result.stderr + result.stdout)

    def test_missing_probe_is_rejected(self):
        result = self.run_script(NO_PROBE)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("without health_uri", result.stderr + result.stdout)


if __name__ == "__main__":
    unittest.main()
