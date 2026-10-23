import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent.parent
HOOK = ROOT / "scripts/git-hooks-pre-commit"


class ScopedPreCommitTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.repo = Path(self.tmp.name) / "repo"
        self.repo.mkdir()
        self.git("init", "-q")
        self.git("config", "user.email", "fixture@example.invalid")
        self.git("config", "user.name", "Fixture")
        (self.repo / ".beads").mkdir()
        (self.repo / ".beads/issues.jsonl").write_text('{"id":"baseline"}\n')
        self.git("add", ".")
        self.git("commit", "-qm", "baseline")
        shutil.copy(HOOK, self.repo / ".git/hooks/pre-commit")
        (self.repo / ".git/hooks/pre-commit").chmod(0o755)
        # A poisonous stub proves that no implicit bd flush occurs.
        self.bin = Path(self.tmp.name) / "bin"
        self.bin.mkdir()
        (self.bin / "bd").write_text('#!/bin/sh\necho called > "$BD_CALLED"\nexit 99\n')
        (self.bin / "bd").chmod(0o755)
        self.env = dict(os.environ, PATH=str(self.bin) + os.pathsep + os.environ["PATH"], BD_CALLED=str(Path(self.tmp.name) / "bd-called"))

    def git(self, *args):
        return subprocess.check_output(["git", "-C", str(self.repo), *args], stderr=subprocess.STDOUT, env=getattr(self, "env", None))

    def commit_selected(self, expected):
        (self.repo / "selected.txt").write_text("intentional change")
        self.git("add", "selected.txt")
        ledger_before = (self.repo / ".beads/issues.jsonl").read_bytes()
        index_before = self.git("write-tree")
        self.git("commit", "-qm", "selected task")
        self.assertEqual(self.git("rev-parse", "HEAD^{tree}"), index_before)
        self.assertEqual((self.repo / ".beads/issues.jsonl").read_bytes(), ledger_before)
        self.assertEqual(set(self.git("diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD").decode().splitlines()), set(expected))
        self.assertFalse((Path(self.tmp.name) / "bd-called").exists())

    def test_clean_ledger(self):
        self.commit_selected(["selected.txt"])

    def test_unstaged_other_agent_ledger_stays_out(self):
        (self.repo / ".beads/issues.jsonl").write_text('{"id":"other-agent"}\n')
        self.commit_selected(["selected.txt"])

    def test_intentionally_staged_ledger_is_preserved(self):
        (self.repo / ".beads/issues.jsonl").write_text('{"id":"intentional"}\n')
        self.git("add", ".beads/issues.jsonl")
        self.commit_selected(["selected.txt", ".beads/issues.jsonl"])

    def test_partially_staged_ledger_stays_exact(self):
        (self.repo / ".beads/issues.jsonl").write_text('{"id":"intentional"}\n')
        self.git("add", ".beads/issues.jsonl")
        (self.repo / ".beads/issues.jsonl").write_text('{"id":"intentional"}\n{"id":"other-agent"}\n')
        self.commit_selected(["selected.txt", ".beads/issues.jsonl"])
        self.assertEqual(self.git("show", "HEAD:.beads/issues.jsonl"), b'{"id":"intentional"}\n')

    def test_linked_worktree_does_not_flush_primary_ledger(self):
        primary = self.repo
        secondary = Path(self.tmp.name) / "task-worktree"
        self.git("worktree", "add", "--detach", str(secondary), "HEAD")
        (primary / ".beads/issues.jsonl").write_text('{"id":"primary-agent"}\n')
        self.repo = secondary
        try:
            self.commit_selected(["selected.txt"])
            self.assertEqual((primary / ".beads/issues.jsonl").read_text(), '{"id":"primary-agent"}\n')
        finally:
            self.repo = primary


if __name__ == "__main__":
    unittest.main()
