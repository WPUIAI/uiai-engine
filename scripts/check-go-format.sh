#!/usr/bin/env bash
set -euo pipefail

ROOT="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
cd "$ROOT"
if git rev-parse --verify HEAD >/dev/null 2>&1; then
  mapfile -d '' go_files < <({
    git diff --name-only --diff-filter=ACMRT -z HEAD -- '*.go'
    git ls-files -o --exclude-standard -z -- '*.go'
  } | sort -zu)
else
  mapfile -d '' go_files < <(git ls-files -co --exclude-standard -z -- '*.go')
fi
if [[ "${#go_files[@]}" -eq 0 ]]; then
  echo "changed Go format: PASS (no changed Go files)"
  exit 0
fi
unformatted="$(gofmt -l "${go_files[@]}")"
if [[ -n "$unformatted" ]]; then
  echo "FAIL changed Go format — run gofmt on:" >&2
  printf '%s\n' "$unformatted" >&2
  exit 1
fi
echo "changed Go format: PASS"
