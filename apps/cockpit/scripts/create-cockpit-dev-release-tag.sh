#!/usr/bin/env bash
# create-cockpit-dev-release-tag.sh — sibling of Focusa create-dev-release-tag.sh.
# Computes next cockpit-vX.Y.Z-dev tag, stamps version, pushes tag and main.

set -euo pipefail

BASE="${BASE:-0.1}"
DRY_RUN="${DRY_RUN:-false}"
PUSH="${PUSH:-true}"

REPO_ROOT="$(git rev-parse --show-toplevel)"
APPS_COCKPIT="${REPO_ROOT}/apps/cockpit"
cd "$REPO_ROOT"

LAST_TAG="$(git tag --list 'cockpit-v*' --sort=-v:refname | head -n 1 || true)"
if [ -z "$LAST_TAG" ]; then
  NEXT="${BASE}.0"
else
  LAST_VERSION="${LAST_TAG#cockpit-v}"
  LAST_VERSION="${LAST_VERSION%-dev}"
  IFS='.' read -r MAJOR MINOR PATCH <<<"$LAST_VERSION"
  PATCH=$((PATCH + 1))
  NEXT="${MAJOR}.${MINOR}.${PATCH}"
fi

TAG="cockpit-v${NEXT}-dev"

if [ "$DRY_RUN" = "true" ]; then
  echo "DRY_RUN: would create tag ${TAG}"
  exit 0
fi

# Stamp version
(cd "$APPS_COCKPIT" && node scripts/stamp-cockpit-version/version.ts "${NEXT}-dev" || true)
git add -A
git commit -m "cockpit: stamp ${TAG}" || true

git tag "$TAG"

if [ "$PUSH" = "true" ]; then
  git push origin main --follow-tags
  echo "pushed tag: ${TAG}"
fi

echo "${TAG}"
