#!/usr/bin/env bash
# Create the next cockpit-vX.Y.Z-dev tag for UIAI Engine Cockpit.
#
# Usage:
#   scripts/create-cockpit-dev-release-tag.sh --base 0.1          # dry-run
#   scripts/create-cockpit-dev-release-tag.sh --base 0.1 --push   # create + push

set -euo pipefail
SELF="$(readlink -f "$0")"
SCRIPT_DIR="$(cd "$(dirname "$SELF")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
cd "$REPO_ROOT"

STAMP="$SCRIPT_DIR/stamp-cockpit-version/stamp-cockpit-version"

BASE="0.1"
BUMP="auto"
CHANNEL="dev"
PUSH=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --base)
      BASE="${2:?--base requires MAJOR.MINOR, e.g. 0.1}"
      shift 2
      ;;
    --bump)
      BUMP="${2:?--bump requires auto, patch, minor, or major}"
      shift 2
      ;;
    --channel)
      CHANNEL="${2:?--channel requires dev or stable}"
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
if ! [[ "$BUMP" =~ ^(auto|patch|minor|major)$ ]]; then
  echo "Invalid --bump '$BUMP'; expected auto, patch, minor, or major" >&2
  exit 2
fi
if ! [[ "$CHANNEL" =~ ^(dev|stable)$ ]]; then
  echo "Invalid --channel '$CHANNEL'; expected dev or stable" >&2
  exit 2
fi

git fetch --quiet --force origin 'refs/tags/cockpit-v*:refs/tags/cockpit-v*'
LAST="$(git tag --list 'cockpit-v*' --sort=-v:refname | head -1 || true)"
if [ -z "$LAST" ]; then
  IFS='.' read -r MAJOR MINOR <<<"$BASE"
  PATCH=0
else
  V="${LAST#cockpit-v}"
  V="${V%-dev}"
  IFS='.' read -r MAJOR MINOR PATCH <<<"$V"
fi

if [ "$BUMP" = "auto" ]; then
  RANGE="HEAD"
  [ -n "$LAST" ] && RANGE="${LAST}..HEAD"
  COMMITS="$(git log "$RANGE" --format='%s%n%b' 2>/dev/null || true)"
  if grep -Eq 'BREAKING[ -]CHANGE:|^[a-z]+(\([^)]*\))?!:' <<<"$COMMITS"; then
    BUMP="major"
  elif grep -Eq '^feat(\([^)]*\))?:' <<<"$COMMITS"; then
    BUMP="minor"
  else
    BUMP="patch"
  fi
fi

case "$BUMP" in
  major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
  minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
  patch) PATCH=$((PATCH + 1)) ;;
esac
NEXT="${MAJOR}.${MINOR}.${PATCH}"
SUFFIX="-dev"
[ "$CHANNEL" = "stable" ] && SUFFIX=""
TAG="cockpit-v${NEXT}${SUFFIX}"

echo "computed semantic $BUMP release: $TAG"

if [ "$PUSH" = "0" ]; then
  echo "dry-run: would stamp, create, and push tag $TAG"
  exit 0
fi

# Stamp version only for an authorized immutable release.
"$STAMP" "$NEXT${SUFFIX}"
git add -A
git -c user.name="$GIT_AUTHOR_NAME" \
    -c user.email="$GIT_AUTHOR_EMAIL" \
    commit -m "cockpit: stamp $TAG" || true

git tag "$TAG"
git push origin "$TAG"

echo "pushed immutable $CHANNEL tag: $TAG (source branch was not rewritten)"
