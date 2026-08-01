#!/usr/bin/env python3
"""Fail when active UIAI docs reintroduce loopback/missing-license entitlement claims."""

from __future__ import annotations

import hashlib
import pathlib
import sys

ROOT = pathlib.Path(__file__).resolve().parents[1]
FILES = [
    "README.md",
    "docs/LICENSING.md",
    "docs/ENDPOINT_AUTH_MATRIX.md",
    "docs/SESSION_API.md",
    "docs/UIAI_FOR_AGENTS_QUICKSTART.md",
    "docs/REMOTE_AUTH_EXAMPLES.md",
    "docs/UIAI_LICENSE_ENTITLEMENT_AND_ONBOARDING_ENFORCEMENT_SPEC_2026-08-01.md",
    "docs/UIAI_PROTECTED_WORKER_AND_FEATURE_CAPSULE_ADDENDUM_2026-08-01.md",
    "docs/UIAI_ENTITLEMENT_SUPERSESSION_MATRIX_2026-08-01.yaml",
]

ACTIVE_GUIDES = FILES[:6]

FORBIDDEN = [
    "loopback remains frictionless for local agents",
    "unauthenticated on loopback",
    "no license / loopback eval",
    "extension tokens are always at least pro tier",
]


def main() -> int:
    failures: list[str] = []
    data: dict[str, str] = {}
    for rel in FILES:
        p = ROOT / rel
        if not p.is_file():
            failures.append(f"missing active licensing document: {rel}")
            continue
        data[rel] = p.read_text(encoding="utf-8")

    for rel in ACTIVE_GUIDES:
        low = data.get(rel, "").lower()
        if "authority-issued" not in low and "authority issued" not in low:
            failures.append(f"{rel}: missing authority-issued entitlement boundary")
        if "authentication" not in low or "entitlement" not in low:
            failures.append(f"{rel}: must explicitly separate authentication and entitlement")
        if "recovery-only" not in low and "recovery only" not in low:
            failures.append(f"{rel}: missing recovery-only posture")

    for rel, text in data.items():
        low = text.lower()
        for pattern in FORBIDDEN:
            if pattern.lower() in low:
                failures.append(f"{rel}: contains forbidden active entitlement claim: {pattern}")

    matrix = data.get("docs/UIAI_ENTITLEMENT_SUPERSESSION_MATRIX_2026-08-01.yaml", "")
    for token in (
        "internal/license/entitlements.go",
        "internal/auth/auth.go",
        "internal/server/server.go",
        "docs/SESSION_API.md",
        "docs/REMOTE_AUTH_EXAMPLES.md",
    ):
        if token not in matrix:
            failures.append(f"UIAI supersession matrix missing {token}")

    endpoint = data.get("docs/ENDPOINT_AUTH_MATRIX.md", "")
    for token in (
        "Required middleware order",
        "Current-code contradictions to remove",
        "uiai.session.execute",
        "reverse-proxy",
    ):
        if token not in endpoint:
            failures.append(f"endpoint auth/entitlement matrix missing {token}")

    mandatory = data.get(
        "docs/UIAI_LICENSE_ENTITLEMENT_AND_ONBOARDING_ENFORCEMENT_SPEC_2026-08-01.md",
        "",
    )
    for token in (
        "Identity and entitlement separation",
        "Focusa-brokered bundle onboarding",
        "Required tests",
    ):
        if token not in mandatory:
            failures.append(f"mandatory entitlement spec missing section/token: {token}")

    protected = data.get(
        "docs/UIAI_PROTECTED_WORKER_AND_FEATURE_CAPSULE_ADDENDUM_2026-08-01.md",
        "",
    )
    for token in ("Protected worker", "IPC contract", "Capsule delivery"):
        if token not in protected:
            failures.append(f"protected-worker addendum missing section/token: {token}")

    if failures:
        print("UIAI licensing documentation consistency FAILED", file=sys.stderr)
        for failure in failures:
            print(f"- {failure}", file=sys.stderr)
        return 1

    digest = hashlib.sha256()
    for rel in FILES:
        digest.update(rel.encode())
        digest.update(b"\0")
        digest.update(data[rel].encode())
        digest.update(b"\0")
    print("UIAI licensing documentation consistency passed")
    print(f"active_files={len(FILES)}")
    print(f"documentation_digest=sha256:{digest.hexdigest()}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
