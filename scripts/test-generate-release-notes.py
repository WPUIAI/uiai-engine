#!/usr/bin/env python3
"""Boundary tests for the GitHub release-note publication budget."""

import importlib.util
import unittest
from pathlib import Path

MODULE_PATH = Path(__file__).with_name("generate-release-notes.py")
SPEC = importlib.util.spec_from_file_location("generate_release_notes", MODULE_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class ReleaseNoteBoundsTest(unittest.TestCase):
    def test_notes_within_budget_are_unchanged(self):
        notes = "# Release\n\nShort notes.\n"
        self.assertEqual(MODULE.bound_release_notes(notes, "https://example.test/compare", 1000), notes)

    def test_oversized_notes_are_disclosed_and_bounded(self):
        compare_url = "https://example.test/compare/old...new"
        notes = "# Release\n" + "detail line\n" * 100
        bounded = MODULE.bound_release_notes(notes, compare_url, 400)
        self.assertLessEqual(len(bounded), 400)
        self.assertIn("Release-note detail truncated", bounded)
        self.assertIn(compare_url, bounded)
        self.assertNotIn("detail linedetail", bounded)

    def test_budget_must_fit_disclosure(self):
        with self.assertRaises(ValueError):
            MODULE.bound_release_notes("oversized", "https://example.test", 5)


if __name__ == "__main__":
    unittest.main()
