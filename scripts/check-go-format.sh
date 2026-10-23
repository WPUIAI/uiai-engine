#!/usr/bin/env bash
# Read-only gate for outgoing, staged, working-tree and untracked Go changes.
set -euo pipefail
ROOT="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
cd "$ROOT"
set --
if git rev-parse --verify HEAD >/dev/null 2>&1; then
  base=HEAD
  if git rev-parse --verify '@{upstream}' >/dev/null 2>&1; then
    base="$(git merge-base HEAD '@{upstream}')"
  fi
  # Include HEAD itself for clean/detached commits; diff HEAD alone would
  # falsely pass already-committed unformatted outgoing source.
  while IFS= read -r -d '' file; do
    if [[ -f "$file" ]]; then set -- "$@" "$file"; fi
  done < <({
    git diff --name-only --diff-filter=ACMRT -z "$base" -- '*.go'
    git diff-tree --root --no-commit-id --name-only --diff-filter=ACMRT -r -z HEAD -- '*.go'
    git ls-files -o --exclude-standard -z -- '*.go'
  } | sort -zu)
else
  while IFS= read -r -d '' file; do
    if [[ -f "$file" ]]; then set -- "$@" "$file"; fi
  done < <(git ls-files -co --exclude-standard -z -- '*.go')
fi
if [[ "$#" -eq 0 ]]; then
  echo "changed Go format: PASS (no changed Go files)"
  exit 0
fi
unformatted="$(gofmt -l "$@")"
if [[ -n "$unformatted" ]]; then
  echo "FAIL changed Go format — run gofmt on:" >&2
  printf '%s\n' "$unformatted" >&2
  gofmt -d "$@" >&2
  exit 1
fi
echo "changed Go format: PASS"
