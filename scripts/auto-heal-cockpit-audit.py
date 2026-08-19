#!/usr/bin/env python3
"""Auto-heal cockpit audit: verify version surfaces, frontend gate, heal if needed."""
import json, subprocess, sys
from pathlib import Path
ROOT = Path(__file__).resolve().parents[1]

def run(cmd): 
    print(f"$ {' '.join(cmd)}")
    r = subprocess.run(cmd, cwd=ROOT)
    return r.returncode == 0

checks = []
# version surfaces
ok = run(["python3","scripts/verify-uiai-version-surfaces.py", "cockpit-v0.1.0-dev"])
checks.append(("verify-version", ok))
# svelte-check
ok = subprocess.run(["npm","run","check"], cwd=ROOT/"apps/cockpit").returncode == 0
checks.append(("svelte-check", ok))
# build
ok = subprocess.run(["npm","run","build"], cwd=ROOT/"apps/cockpit").returncode == 0
checks.append(("build", ok))

failed = [k for k,v in checks if not v]
if failed:
    print(f"HEAL needed: {failed}")
    sys.exit(1)
print("auto-heal-cockpit-audit PASS")
