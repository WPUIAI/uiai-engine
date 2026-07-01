#!/usr/bin/env bash
# cockpit-doctor.sh — Linux-safe doctor.
#
# Runs the gates that work without a Tauri toolchain:
#   - svelte-check (TypeScript + Svelte types)
#   - vitest (JS unit tests)
#   - sveltekit build (SPA static output)
#   - cockpit:smoke (backend mock-external mode)
#
# Rust + Tauri bundle compilation happens on GitHub Actions runners
# (ubuntu-latest for `cargo check`, macos-latest for the macOS bundle).
# Never attempt to build Tauri locally — see §14 release gates.

set -euo pipefail

APPS_COCKPIT="${APPS_COCKPIT:-apps/cockpit}"
PNPM="${PNPM:-pnpm}"

if [ ! -d "$APPS_COCKPIT" ]; then
  echo "cockpit-doctor: $APPS_COCKPIT not found" >&2
  exit 1
fi

cd "$APPS_COCKPIT"

echo "== svelte-check =="
$PNPM check

echo "== vitest =="
$PNPM test

echo "== web build (SPA) =="
$PNPM build

echo "== backend smoke (mock external) =="
$PNPM cockpit:smoke --mode local-only --mock-external

echo
echo "cockpit-doctor: OK (Linux-safe gates)."
echo "Note: Rust + Tauri bundle compile on GitHub Actions."