#!/usr/bin/env python3
"""Cross-product invariants: tag lane, audit stream, signing identity, runner reuse."""
import pathlib, re, sys
ROOT = pathlib.Path(__file__).resolve().parents[1]

def fail(msg): print(f"FAIL: {msg}"); sys.exit(1)
def ok(msg): print(f"PASS: {msg}")

# 1 tag lane separation
cockpit_tags = list((ROOT/".github/workflows").glob("cockpit*.yml"))
engine_tags = list((ROOT/".github/workflows").glob("engine*.yml"))
# Check cockpit workflows only trigger on cockpit-v* and not v*
for p in [x for x in cockpit_tags if 'release' in x.name or 'dev-release' in x.name]:
    t=p.read_text()
    if "cockpit-v*" in t: ok(f"{p.name} triggers cockpit-v*")
    else: fail(f"{p.name} missing cockpit-v* trigger")
    # ensure not triggering on generic v* without cockpit prefix in release workflows
    if p.name=="cockpit-release.yml" and re.search(r"tags:\s*\n\s*-\s*'v\*'", t):
        fail("cockpit-release should not trigger on generic v*")
    ok(f"{p.name} tag lane OK")
# Engine should not trigger on cockpit-v*
for p in engine_tags:
    t=p.read_text()
    if "cockpit-v*" in t: fail(f"{p.name} should not trigger cockpit-v*")
    ok(f"{p.name} lane OK")

# 2 audit stream separation
cockpit_audit = ROOT/"release-proof/cockpit/audit.jsonl"
daemon_audit = ROOT/"release-proof/audit/audit.jsonl"
if not cockpit_audit.exists(): fail("cockpit audit missing")
if daemon_audit.exists() and cockpit_audit.read_text()==daemon_audit.read_text(): fail("audit merge detected")
ok("audit streams separate")

# 3 signing identity reuse (same secret name)
signers = []
for p in (ROOT/".github/workflows").glob("*.yml"):
    if "TAURI_SIGNING_PRIVATE_KEY" in p.read_text():
        signers.append(p.name)
if len(signers)==0: fail("no signing identity found")
ok(f"signing identity reuse {signers}")

# 4 runner reuse
ci = (ROOT/".github/workflows/ci.yml").read_text()
if "macos-latest" not in ci: fail("ci.yml missing macos runner for cockpit")
if "ubuntu-latest" not in ci: fail("ci.yml missing ubuntu runner for engine")
ok("runner reuse OK")

print("cross-product invariants PASS")
