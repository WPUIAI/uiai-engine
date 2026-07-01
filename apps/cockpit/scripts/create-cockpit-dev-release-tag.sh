#!/usr/bin/env bash
# Create the next cockpit-vX.Y.Z-dev tag for Focusa Cockpit.
#
# Usage:
#   scripts/create-cockpit-dev-release-tag.sh --base 0.1          # dry-run
#   scripts/create-cockpit-dev-release-tag.sh --base 0.1 --push   # create + push

set -euo pipefail
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

BASE="0.1"
PUSH=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --base)
      BASE="${2:?--base requires MAJOR.MINOR, e.g. 0.1}"
      shift 2
      ;;
    --push) PUSH=1; shift ;;
    --dry-run) PUSH=0; shift ;;
    -h|--help)
      sed -n '1,12p' "$0"
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 2
      ;;
  esac
done

if ! [[ "$BASE" =~ ^[0-9]+\.[0-9]+$ ]]; then
  echo "Invalid --base '$BASE'; expected MAJOR.MINOR, e.g. 0.1" >&2
  exit 2
fi

LAST="$(git tag --list 'cockpit-v*' --sort=-v:refname | head -1 || true)"
if [ -z "$LAST" ]; then
  NEXT="${BASE}.0"
else
  V="${LAST#cockpit-v}"
  V="${V%-dev}"
  IFS='.' read -r MAJOR MINOR PATCH <<<"$V"
  PATCH=$((PATCH + 1))
  NEXT="${MAJOR}.${MINOR}.${PATCH}"
fi

TAG="cockpit-v${NEXT}-dev"

echo "computed next tag: $TAG"

# Stamp version into apps/cockpit/src/lib/version.ts and tauri.conf.json
apps/cockpit/scripts/stamp-cockpit-version/stamp-cockpit-version "$NEXT-dev"

if [ "$PUSH" = "0" ]; then
  echo "dry-run: would create + push tag $TAG"
  exit 0
fi

git add -A
git -c user.name="$GIT_AUTHOR_NAME" \
    -c user.email="$GIT_AUTHOR_EMAIL" \
    commit -m "cockpit: stamp $TAG" || true

git tag "$TAG"
git push origin "$TAG"
git push origin HEAD:main 2>/dev/null || git push origin main

echo "pushed tag: $TAG"
