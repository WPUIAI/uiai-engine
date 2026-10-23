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

    def test_open_pr_suppresses_duplicate_branch_build(self):
        text = (ROOT / ".appveyor.yml").read_text()
        self.assertRegex(text, r"(?m)^skip_branch_with_pr:\s*true\s*$")
        self.assertNotRegex(text, r"(?im)^skip_pull_requests:\s*true")


if __name__ == "__main__":
    unittest.main()
