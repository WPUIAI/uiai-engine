#!/usr/bin/env python3
import pathlib
ROOT=pathlib.Path(__file__).resolve().parents[1]
for p in ["docs/COCKPIT_ROLLBACK_RUNBOOK.md","docs/cockpit-bundle-manifest.schema.json","release-proof/cockpit/audit.jsonl"]:
    assert (ROOT/p).exists(), f"missing {p}"
print("hardening completeness PASS")
