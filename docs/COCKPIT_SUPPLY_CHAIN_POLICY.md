# Cockpit Supply Chain — Pinned + Audited

- Rust: `Cargo.lock` committed, `cargo audit` in CI (deny un-audited crates).
- Node: `package-lock.json` committed, `npm ci` + `npm audit --audit-level=high` (fail on high).
- Signing: `TAURI_SIGNING_PRIVATE_KEY` only from GitHub Secrets; no local key.
- Verification: `scripts/verify-uiai-version-surfaces.py`, `audit-schema.py`, `cockpit_cross_product_invariants_test.py` in `ci.yml`/`cockpit-release.yml`.

## Cockpit audit validation

The audit recorder calls `python3 scripts/audit-schema.py validate
release-proof/cockpit/audit.jsonl`, not inline escaped Python. The canonical
validator supports legacy addition/failure records and the versioned Cockpit
producer contract in `apps/cockpit/scripts/cockpit_audit.py`; it rejects malformed
rows with file/line diagnostics without printing payloads. Validation never
rewrites the audit file. Manual dispatch reaches validation, and unchanged
audits create no commit. Hosted/scheduled proof is separate from local fixtures;
never dispatch paid CI solely to manufacture acceptance evidence.
