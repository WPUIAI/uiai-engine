#!/usr/bin/env python3
# auto-heal-cockpit-audit.py — sibling of Focusa scripts/auto-heal-audit.py.
# Synthesizes self_heal rows in release-proof/cockpit/audit.jsonl.

import argparse
import datetime as dt
import json
import os
import sys
from pathlib import Path

SCHEMA = "focusa.cockpit.audit.v1"


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("--kind", default="self_heal")
    p.add_argument("--reason", default="unspecified")
    p.add_argument("--out", default="release-proof/cockpit/audit.jsonl")
    args = p.parse_args()

    out = Path(args.out)
    out.parent.mkdir(parents=True, exist_ok=True)

    row = {
        "schema": SCHEMA,
        "kind": args.kind,
        "reason": args.reason,
        "at": dt.datetime.utcnow().isoformat() + "Z",
    }
    with out.open("a", encoding="utf-8") as f:
        f.write(json.dumps(row) + "\n")

    print(f"auto-heal-cockpit-audit: appended {row['kind']} to {out}")
    return 0


if __name__ == "__main__":
    sys.exit(main())