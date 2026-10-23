#!/usr/bin/env python3
"""Execute the deploy script's exact readiness gate with isolated curl fixtures."""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

SOURCE = Path(__file__).with_name("deploy-engine-ovh.sh").read_text()
GATE = SOURCE.split("# BEGIN HEALTH READINESS GATE", 1)[1].split("\n", 1)[1].split("# END HEALTH READINESS GATE", 1)[0]


class DeployHealthTests(unittest.TestCase):
    def run_gate(self, statuses):
        with tempfile.TemporaryDirectory(prefix="epwa-health-test-") as root:
            root = Path(root)
            curl = root / "curl"
            curl.write_text('''#!/usr/bin/env python3
import os, pathlib, sys
root = pathlib.Path(os.environ["HEALTH_FIXTURE"])
state = root / "count"
i = int(state.read_text()) if state.exists() else 0
state.write_text(str(i + 1))
statuses = os.environ["HEALTH_STATUSES"].split(",")
status = statuses[min(i, len(statuses) - 1)]
args = sys.argv[1:]
output = pathlib.Path(args[args.index("-o") + 1])
if status != "000":
    output.write_text('{"status":"healthy"}' if status == "200" else '{"error":"not ready"}')
print(status, end="")
sys.exit(7 if status == "000" else 0)
''')
            curl.chmod(0o755)
            sleep = root / "sleep"
            sleep.write_text("#!/bin/sh\nexit 0\n")
            sleep.chmod(0o755)
            env = {**os.environ, "PATH": str(root) + os.pathsep + os.environ["PATH"],
                   "TMPDIR": str(root), "HEALTH_FIXTURE": str(root),
                   "HEALTH_STATUSES": ",".join(statuses)}
            result = subprocess.run(["bash", "-c", 'set -euo pipefail\nhealth_url=http://fixture/health\n' + GATE],
                                    env=env, capture_output=True, text=True, timeout=10)
            return result, int((root / "count").read_text())

    def test_waits_for_startup_and_requires_success(self):
        result, count = self.run_gate(["000", "503", "200"])
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(count, 3)
        self.assertIn("health_http_code=200", result.stdout)

    def test_unauthorized_is_not_healthy(self):
        result, count = self.run_gate(["401"])
        self.assertEqual(result.returncode, 4)
        self.assertEqual(count, 20)

    def test_connection_failure_has_bounded_attempts(self):
        result, count = self.run_gate(["000"])
        self.assertEqual(result.returncode, 4)
        self.assertEqual(count, 20)
        self.assertNotIn("missing authentication header", result.stdout)


if __name__ == "__main__":
    unittest.main()
