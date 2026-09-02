#!/usr/bin/env bash
set -euo pipefail

ROOT="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
cd "$ROOT"

if [[ -n "${UIAI_FORMAT_BASE:-}" ]] && ! git rev-parse --verify "$UIAI_FORMAT_BASE^{commit}" >/dev/null 2>&1; then
  echo "invalid UIAI_FORMAT_BASE: $UIAI_FORMAT_BASE" >&2
  exit 1
fi

collect_go_files() {
  if git rev-parse --verify HEAD >/dev/null 2>&1; then
    local base="${UIAI_FORMAT_BASE:-}"
    if [[ -n "$base" ]]; then
      : # Explicit base was validated before collection.
    elif git rev-parse --verify '@{upstream}^{commit}' >/dev/null 2>&1; then
      base="$(git merge-base HEAD '@{upstream}')"
    elif git rev-parse --verify 'origin/main^{commit}' >/dev/null 2>&1; then
      base="$(git merge-base HEAD origin/main)"
    elif git rev-parse --verify 'HEAD^{commit}^' >/dev/null 2>&1; then
      base="HEAD^"
    fi

    if [[ -n "$base" ]]; then
      git diff --name-only --diff-filter=ACMRT -z "$base" HEAD -- '*.go'
    else
      git ls-files -z -- '*.go'
    fi
    git diff --name-only --diff-filter=ACMRT -z HEAD -- '*.go'
    git ls-files -o --exclude-standard -z -- '*.go'
  else
    git ls-files -co --exclude-standard -z -- '*.go'
  fi
}

changed_file="$(mktemp)"
trap 'unlink "$changed_file"' EXIT
collect_go_files | sort -zu >"$changed_file"
mapfile -d '' go_files <"$changed_file"
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
