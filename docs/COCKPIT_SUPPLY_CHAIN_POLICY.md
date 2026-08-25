# Cockpit Supply Chain — Pinned + Audited

- Rust: `Cargo.lock` committed, `cargo audit` in CI (deny un-audited crates).
- Node: `package-lock.json` committed, `npm ci` + `npm audit --audit-level=high` (fail on high).
- Signing: `TAURI_SIGNING_PRIVATE_KEY` only from GitHub Secrets; no local key.
- Verification: `scripts/verify-uiai-version-surfaces.py`, `audit-schema.py`, `cockpit_cross_product_invariants_test.py` in `ci.yml`/`cockpit-release.yml`.
