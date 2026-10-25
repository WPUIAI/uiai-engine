from pathlib import Path
import shutil
import json
import os
import subprocess
import tempfile
import textwrap
import unittest

ROOT = Path(__file__).resolve().parent.parent
WORKFLOW = ROOT / ".github/workflows/cockpit-audit-recorder.yml"


class CockpitAuditWorkflowTests(unittest.TestCase):
    def validation_command(self):
        workflow = WORKFLOW.read_text()
        block = workflow.split("- name: Validate audit JSONL shape", 1)[1].split("- name:", 1)[0]
        return textwrap.dedent(block.split("run: |\n", 1)[1]).strip()

    def validate(self, content):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            scripts = root / "apps/cockpit/scripts"
            scripts.mkdir(parents=True)
            shutil.copy(ROOT / "apps/cockpit/scripts/cockpit_audit.py", scripts)
            (root / "scripts").mkdir()
            shutil.copy(ROOT / "scripts/audit-schema.py", root / "scripts/audit-schema.py")
            audit = root / "release-proof/cockpit/audit.jsonl"
            audit.parent.mkdir(parents=True)
            audit.write_text(content)
            before = audit.read_bytes()
            result = subprocess.run(["bash", "-euc", self.validation_command()], cwd=root, capture_output=True, text=True, timeout=8)
            self.assertEqual(audit.read_bytes(), before)
            return result

    def test_checked_in_workflow_command_accepts_valid_jsonl(self):
        result = self.validate('{"schema":"uaiengine.cockpit.audit.v1","kind":"cockpit_ci_failure","workflow":"Fixture","run_id":"1","at":"2026-10-23T00:00:00Z"}\n\n')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("audit schema OK (1 rows", result.stdout)
        self.assertNotIn("python3 -c", self.validation_command())
        self.assertNotIn("\\n", self.validation_command())

    def test_errors_identify_line_without_echoing_payload(self):
        for row in ['not-json', '{"schema":"wrong","secret":"synthetic-value"}', '[]']:
            with self.subTest(row=row):
                result = self.validate('{"schema":"uaiengine.cockpit.audit.v1"}\n' + row + '\n')
                self.assertNotEqual(result.returncode, 0)
                self.assertIn("audit.jsonl:2:", result.stderr)
                self.assertNotIn("synthetic-value", result.stderr)

    def test_existing_legacy_cockpit_audit_is_accepted(self):
        result = self.validate((ROOT / "release-proof/cockpit/audit.jsonl").read_text())
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_manual_dispatch_is_admitted(self):
        self.assertIn("github.event_name == 'workflow_dispatch'", WORKFLOW.read_text())

    def test_unchanged_audit_does_not_commit(self):
        workflow = WORKFLOW.read_text()
        block = workflow.split("- name: Commit audit updates", 1)[1]
        command = textwrap.dedent(block.split("run: |\n", 1)[1]).strip()
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            def git(*args):
                return subprocess.check_output(["git", "-C", str(root), *args], stderr=subprocess.STDOUT)
            git("init", "-q")
            git("config", "user.name", "Fixture")
            git("config", "user.email", "fixture@example.invalid")
            audit = root / "release-proof/cockpit/audit.jsonl"
            audit.parent.mkdir(parents=True)
            audit.write_text('{"schema":"uaiengine.cockpit.audit.v1"}\n')
            git("add", ".")
            git("commit", "-qm", "baseline")
            before = git("rev-parse", "HEAD")
            result = subprocess.run(["bash", "-euc", command], cwd=root, capture_output=True, text=True, timeout=8)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("no audit changes", result.stdout)
            self.assertEqual(git("rev-parse", "HEAD"), before)

    def test_real_qualifying_failure_through_checked_in_producer(self):
        metadata = json.loads((ROOT / "release-proof/cockpit/qualifying-failure-replay.json").read_text())
        self.assertEqual(metadata["conclusion"], "failure")
        self.assertIn(metadata["workflow"], ("Cockpit CI", "Cockpit Release", "Deploy Cockpit"))
        block = WORKFLOW.read_text().split("- name: Record CI failure", 1)[1].split("- name: Validate audit JSONL shape", 1)[0]
        command = textwrap.dedent(block.split("run: |\n", 1)[1]).split("python3 apps/cockpit/scripts/auto-heal-cockpit-audit.py", 1)[0]
        # Exercise the real recorder only; branch creation/re-dispatch is not
        # authorized or needed by this isolated parser/producer regression.
        env = dict(os.environ, RUN_ID=metadata["run_id"], WORKFLOW=metadata["workflow"], HEAD_SHA=metadata["head_sha"], BRANCH=metadata["branch"], EVENT=metadata["event"], ATTEMPT="1", LOG_URL=metadata["log_url"], TS=metadata["at"])
        with tempfile.TemporaryDirectory() as tmp:
            result = subprocess.run(["bash", "-euc", command], cwd=tmp, env=env, capture_output=True, text=True, timeout=8)
            self.assertEqual(result.returncode, 0, result.stderr)
            data = (Path(tmp) / "release-proof/cockpit/audit.jsonl").read_text()
            self.assertLessEqual(len(data.encode()), 4096)
            rows = [json.loads(line) for line in data.splitlines() if line.strip()]
            self.assertEqual(len(rows), 1)
            self.assertEqual(rows[0]["run_id"], metadata["run_id"])
            validation = self.validate(data)
            self.assertEqual(validation.returncode, 0, validation.stderr)
            self.assertIn("audit schema OK (1 rows", validation.stdout)


if __name__ == "__main__":
    unittest.main()
