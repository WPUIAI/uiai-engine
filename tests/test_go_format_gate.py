from pathlib import Path
import subprocess
import os
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent.parent
SCRIPT = ROOT / "scripts/check-go-format.sh"
FORMATTED = "package example\n\nfunc example() {}\n"
UNFORMATTED = "package example\nfunc example( ){ }\n"


class GoFormatGateTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.repo = Path(self.tmp.name)
        self.git("init", "-q")
        self.git("config", "user.email", "fixture@example.invalid")
        self.git("config", "user.name", "Fixture")
        (self.repo / "main.go").write_text(FORMATTED)
        self.git("add", ".")
        self.git("commit", "-qm", "baseline")

    def git(self, *args):
        return subprocess.check_output(["git", "-C", str(self.repo), *args], stderr=subprocess.STDOUT)

    def snapshot(self):
        files = {str(p.relative_to(self.repo)): p.read_bytes() for p in self.repo.rglob("*") if p.is_file() and ".git" not in p.relative_to(self.repo).parts}
        return files, self.git("write-tree"), self.git("diff", "--cached", "--binary")

    def check(self, success):
        before = self.snapshot()
        result = subprocess.run([os.environ.get("UIAI_TEST_BASH", "bash"), str(SCRIPT), str(self.repo)], capture_output=True, text=True, timeout=10)
        self.assertEqual(result.returncode == 0, success, result.stdout + result.stderr)
        self.assertEqual(self.snapshot(), before, "gate changed source or index")
        if not success:
            self.assertIn("main.go", result.stderr)
            self.assertIn("diff ", result.stderr)
            self.assertNotIn("PASS", result.stdout)
        return result

    def test_clean_committed_unformatted_fails(self):
        (self.repo / "main.go").write_text(UNFORMATTED)
        self.git("add", "main.go")
        self.git("commit", "-qm", "unformatted")
        self.check(False)

    def test_formatted_with_unrelated_dirty_file_is_unchanged(self):
        (self.repo / "other-agent.txt").write_text("preserve this")
        self.check(True)

    def test_partial_index_and_worktree_are_unchanged(self):
        (self.repo / "main.go").write_text(FORMATTED + "\n// intentional staged change\n")
        self.git("add", "main.go")
        (self.repo / "main.go").write_text(UNFORMATTED)
        (self.repo / "other-agent.txt").write_text("concurrent work")
        self.check(False)

    def test_detached_head_committed_unformatted_fails(self):
        (self.repo / "main.go").write_text(UNFORMATTED)
        self.git("add", "main.go")
        self.git("commit", "-qm", "unformatted")
        self.git("checkout", "--detach", "-q")
        self.check(False)

    def test_task_isolated_worktree_is_unchanged(self):
        primary = self.repo
        other = primary / "isolated-fixture"
        self.git("worktree", "add", "--detach", str(other), "HEAD")
        self.repo = other
        try:
            (other / "main.go").write_text(UNFORMATTED)
            self.check(False)
            self.assertEqual((primary / "main.go").read_text(), FORMATTED)
        finally:
            self.repo = primary

    def test_outgoing_older_commit_is_checked(self):
        baseline = self.git("rev-parse", "HEAD").decode().strip()
        self.git("update-ref", "refs/remotes/origin/main", baseline)
        self.git("config", "remote.origin.url", str(self.repo / "synthetic-remote"))
        self.git("config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
        branch = self.git("branch", "--show-current").decode().strip()
        self.git("config", f"branch.{branch}.remote", "origin")
        self.git("config", f"branch.{branch}.merge", "refs/heads/main")
        (self.repo / "main.go").write_text(UNFORMATTED)
        self.git("add", "main.go")
        self.git("commit", "-qm", "earlier outgoing source")
        (self.repo / "README.md").write_text("later docs-only commit")
        self.git("add", "README.md")
        self.git("commit", "-qm", "docs")
        self.check(False)


if __name__ == "__main__":
    unittest.main()
