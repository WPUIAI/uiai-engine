from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parent.parent


class AppVeyorPolicyTests(unittest.TestCase):
    def test_windows_checks_cannot_mask_failure(self):
        text = (ROOT / ".appveyor.yml").read_text()
        block = text.split("test_script:", 1)[1].split("deploy:", 1)[0]
        commands = re.findall(r"^\s*-\s*(go\s+(?:vet|test)\s+[^\n]+)$", block, re.MULTILINE)
        self.assertEqual(len(commands), 2, "Windows vet and tests must both run")
        for command in commands:
            self.assertNotIn("||", command)
            self.assertNotIn("exit 0", command.lower())
            self.assertNotIn("continueonerror", command.lower())
        self.assertNotRegex(text, r"(?im)^\s*(?:test_script|build_script):\s*(?:false|off)")

    def test_toolchain_version_has_one_project_owner(self):
        text = (ROOT / ".appveyor.yml").read_text()
        self.assertNotRegex(text, r"(?m)^\s+GO_VERSION:", "Windows must derive its version from go.mod")
        self.assertIn("'go.mod'", text)
        self.assertIn("$match.Groups[1].Value", text)
        self.assertIn('"go$version.windows-amd64.zip"', text)
        self.assertIn("Go toolchain version missing from go.mod", text)
        self.assertRegex(text, r"(?m)^\s+GO_SHA256: [a-f0-9]{64}$")

    def test_linux_security_gate_is_unmasked(self):
        text = (ROOT / ".github/workflows/ci.yml").read_text()
        commands = re.findall(r"(?m)^\s+run: (go run golang.org/x/vuln/cmd/govulncheck@[^\n]+)$", text)
        self.assertEqual(commands, ["go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./..."], "Source security findings must fail CI")

    def test_open_pr_suppresses_duplicate_branch_build(self):
        text = (ROOT / ".appveyor.yml").read_text()
        self.assertRegex(text, r"(?m)^skip_branch_with_pr:\s*true\s*$")
        self.assertNotRegex(text, r"(?im)^skip_pull_requests:\s*true")


if __name__ == "__main__":
    unittest.main()
