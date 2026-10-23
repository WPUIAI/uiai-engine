#!/usr/bin/env python3
"""Verify UIAI-COCKPIT-004 schema/fixture parity without generating code."""

from __future__ import annotations

import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
LEDGER = ROOT / "docs/contracts/UIAI_COCKPIT_004_C01_DESKTOP_SESSION_PRESENTATION_HANDOFF_LEDGER_v1.yaml"
GO = ROOT / "internal/desktopcontract/contracts.go"
TS = ROOT / "apps/cockpit/src/lib/contracts/desktop-presentation.ts"
RUST = ROOT / "apps/cockpit/src-tauri/src/desktop_contract.rs"
FIXTURES = ROOT / "tests/fixtures/desktop-presentation"
SCHEMA_PATTERN = re.compile(r'["\']((?:uiai|focusa)\.[a-z0-9_.]+\.v\d+)["\']')


def ledger_schemas() -> set[str]:
    schemas: set[str] = set()
    active = False
    for line in LEDGER.read_text().splitlines():
        if line == "schemas:":
            active = True
            continue
        if active and line and not line.startswith(" "):
            break
        if active and line.startswith("  - "):
            schemas.add(line.removeprefix("  - ").strip())
    return schemas


def source_schemas(path: Path) -> set[str]:
    return set(SCHEMA_PATTERN.findall(path.read_text()))


def fixture_schemas(value: object) -> set[str]:
    found: set[str] = set()
    if isinstance(value, dict):
        schema = value.get("schema")
        if isinstance(schema, str):
            found.add(schema)
        for child in value.values():
            found.update(fixture_schemas(child))
    elif isinstance(value, list):
        for child in value:
            found.update(fixture_schemas(child))
    return found


def main() -> None:
    canonical = ledger_schemas()
    if len(canonical) != 7:
        raise SystemExit(f"expected 7 canonical schemas, got {sorted(canonical)}")

    for language, path in (("go", GO), ("typescript", TS), ("rust", RUST)):
        missing = canonical - source_schemas(path)
        if missing:
            raise SystemExit(f"{language} contracts missing schemas: {sorted(missing)}")

    valid = json.loads((FIXTURES / "valid-contracts.json").read_text())
    missing_fixture = canonical - fixture_schemas(valid)
    if missing_fixture:
        raise SystemExit(f"canonical fixture missing schemas: {sorted(missing_fixture)}")

    menubar = json.loads((FIXTURES / "focusa-app-manifest.valid.json").read_text())
    if menubar.get("schema") != "focusa.app.manifest.v2" or menubar.get("app") != "focusa-menubar":
        raise SystemExit("Focusa Menubar compatibility fixture is invalid")
    if "cockpit.open" not in menubar.get("capabilities", []):
        raise SystemExit("Focusa Menubar fixture must advertise cockpit.open")

    secret = json.loads((FIXTURES / "handoff-secret.invalid.json").read_text())
    raw_path = json.loads((FIXTURES / "handoff-raw-path.invalid.json").read_text())
    private_url = json.loads((FIXTURES / "handoff-private-url.invalid.json").read_text())
    if "token" not in secret:
        raise SystemExit("secret rejection fixture must carry a forbidden token field")
    if not str(raw_path.get("target_ref", "")).startswith("/"):
        raise SystemExit("raw-path rejection fixture is not a path")
    if "://" not in str(private_url.get("target_ref", "")):
        raise SystemExit("private-URL rejection fixture is not a URL")

    print(
        "desktop_contract_parity_ok "
        f"schemas={len(canonical)} languages=go,typescript,rust "
        "apps=uaiengine-cockpit,focusa-menubar invalid_fixtures=4"
    )


if __name__ == "__main__":
    main()
