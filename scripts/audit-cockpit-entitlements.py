#!/usr/bin/env python3
"""Fail closed when Cockpit signing entitlements widen credential authority."""
from pathlib import Path
import plistlib, sys

path = Path(__file__).parents[1] / "apps/cockpit/src-tauri/entitlements.plist"
with path.open("rb") as source:
    values = plistlib.load(source)
expected = {
    "com.apple.security.app-sandbox": True,
    "com.apple.security.network.client": True,
    "com.apple.security.files.user-selected.read-write": True,
    "com.apple.security.cs.allow-jit": False,
    "com.apple.security.cs.allow-unsigned-executable-memory": False,
    "keychain-access-groups": ["$(AppIdentifierPrefix)com.wpuiai.uiaiengine.cockpit"],
    "com.apple.security.application-groups": ["group.com.focusa.shared"],
}
if values != expected:
    print("Cockpit entitlements differ from the least-privilege policy", file=sys.stderr)
    print(f"expected={expected!r}\nobserved={values!r}", file=sys.stderr)
    raise SystemExit(1)
print("Cockpit entitlements: least-privilege policy verified")
