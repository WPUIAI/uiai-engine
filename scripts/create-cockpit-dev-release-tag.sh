#!/usr/bin/env bash
# Sibling of Focusa scripts/create-dev-release-tag.sh — cockpit lane.
# Computes next cockpit-vX.Y.Z-dev tag, stamps version, prints deltas.
# Default dry-run: scripts/create-cockpit-dev-release-tag.sh
# Push (when allowed): scripts/create-cockpit-dev-release-tag.sh --push
set -euo pipefail
cd "$(dirname "$0")/.."

BASE="0.1"
EXACT_TAG=""
PUSH=0
DRY_RUN=1

while [[ $# -gt 0 ]]; do
  case "$1" in
    --base) BASE="${2:?}"; shift 2;;
    --tag) EXACT_TAG="${2:?}"; shift 2;;
    --push) PUSH=1; DRY_RUN=0; shift;;
    --dry-run) DRY_RUN=1; shift;;
    -h|--help) echo "Usage: $0 [--base 0.1] [--tag cockpit-v0.1.5-dev] [--push]"; exit 0;;
    *) echo "Unknown arg $1" >&2; exit 2;;
  esac
done

if [[ -n "$EXACT_TAG" ]]; then
  TAG="$EXACT_TAG"
else
  latest=$(git tag -l "cockpit-v*" --sort=-v:refname | head -n1 || true)
  if [[ -z "$latest" ]]; then
    TAG="cockpit-v${BASE}.0-dev"
  else
    # bump patch: cockpit-v0.1.4 -> cockpit-v0.1.5-dev
    ver=${latest#cockpit-v}
    ver=${ver%%-*} # strip -dev
    IFS='.' read -r maj min pat <<<"$ver"
    pat=$((pat+1))
    TAG="cockpit-v${maj}.${min}.${pat}-dev"
  fi
fi

echo "Next cockpit tag: $TAG"
ver=${TAG#cockpit-v}
ver=${ver%%-*}
echo "Version: $ver"

# Dry-run: stamp, show diff, then revert stamp artifacts
if [[ "$DRY_RUN" == "1" ]]; then
  echo "Dry-run: stamping $TAG"
  python3 scripts/stamp-uiai-version.py "$TAG" --cockpit-only
  echo "--- version surfaces ---"
  python3 scripts/verify-uiai-version-surfaces.py "$TAG" || true
  echo "--- git diff (stamped, not committed) ---"
  git diff --stat
  # revert to keep working tree clean
  git checkout -- apps/cockpit/package.json apps/cockpit/src-tauri/tauri.conf.json apps/cockpit/src-tauri/Cargo.toml apps/cockpit/src/lib/version.ts 2>/dev/null || true
  git checkout -- apps/cockpit/.release-version-stamp docs/distribution-manifest.json 2>/dev/null || true
  echo "Dry-run complete — no tag pushed (per no-publish policy)"
  exit 0
fi

# Push path (blocked by operator no-publish policy — kept for completeness)
echo "Push requested for $TAG — stamping and pushing"
python3 scripts/stamp-uiai-version.py "$TAG" --cockpit-only
python3 scripts/verify-uiai-version-surfaces.py "$TAG"
git add apps/cockpit/package.json apps/cockpit/src-tauri/tauri.conf.json apps/cockpit/src-tauri/Cargo.toml apps/cockpit/src/lib/version.ts apps/cockpit/.release-version-stamp docs/distribution-manifest.json
git commit -m "chore(release): stamp $TAG" || true
git tag "$TAG"
git push origin HEAD:main 2>&1 | head
git push origin "$TAG" 2>&1 | head
echo "Pushed $TAG"
