#!/usr/bin/env bash
# Shared strict Rust gate for local release preflight and the publisher.
set -euo pipefail
# Match the publisher's existing stable channel without changing host defaults.
export RUSTUP_TOOLCHAIN=stable
cd "$(dirname "${BASH_SOURCE[0]}")/../apps/cockpit/src-tauri"
cargo test
cargo clippy -- -D warnings
