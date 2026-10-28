#!/usr/bin/env bash
# Canonical UIAI release tag creator (legacy filename retained for callers).
# Explicit --tag accepts cockpit-v*, engine-v*, and uiai-engine-v*.
# Prepare source first; --tag-only releases an already prepared, verified SHA.
set -euo pipefail
cd "$(dirname "$0")/.."
BASE="0.1"
TAGS=()
PUSH=0
DRY_RUN=1
TAG_ONLY=0
PREPARE_ONLY=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --base) BASE="${2:?}"; shift 2;;
    --tag) TAGS+=("${2:?}"); shift 2;;
    --push) PUSH=1; DRY_RUN=0; shift;;
    --dry-run) DRY_RUN=1; shift;;
    --tag-only) TAG_ONLY=1; shift;;
    --prepare-only) PREPARE_ONLY=1; shift;;
    -h|--help) echo "Usage: $0 [--base 0.1] [--tag TAG ...] [--prepare-only|--tag-only] [--push]"; exit 0;;
    *) echo "Unknown arg $1" >&2; exit 2;;
  esac
done
if [[ "$TAG_ONLY" == 1 && "$PREPARE_ONLY" == 1 ]]; then echo 'Conflicting release phases' >&2; exit 2; fi
if [[ ${#TAGS[@]} == 0 ]]; then
  latest=$(git for-each-ref --count=1 --sort=-version:refname --format='%(refname:short)' 'refs/tags/cockpit-v*')
  if [[ -z "$latest" ]]; then
    TAGS+=("cockpit-v${BASE}.0-dev")
  else
    ver=${latest#cockpit-v}; ver=${ver%%-*}
    IFS='.' read -r maj min pat <<<"$ver"
    TAGS+=("cockpit-v${maj}.${min}.$((pat+1))-dev")
  fi
fi
FILES=(docs/distribution-manifest.json)
for tag in "${TAGS[@]}"; do
  if [[ ! "$tag" =~ ^(cockpit-v|engine-v|uiai-engine-v)[0-9]+\.[0-9]+\.[0-9]+(-dev)?$ ]]; then
    echo "Invalid UIAI release tag: $tag" >&2; exit 2
  fi
  case "$tag" in
    cockpit-v*) FILES+=(apps/cockpit/package.json apps/cockpit/package-lock.json apps/cockpit/src-tauri/tauri.conf.json apps/cockpit/src-tauri/Cargo.toml apps/cockpit/src/lib/version.ts apps/cockpit/.release-version-stamp);;
    *) FILES+=(cmd/uiai-engine/main.go docs/current/.release-version-stamp);;
  esac
  if git show-ref --verify --quiet "refs/tags/$tag"; then echo "Immutable tag already exists: $tag" >&2; exit 2; fi
  printf 'Release tag: %s\n' "$tag"
done
if [[ "$DRY_RUN" == 1 ]]; then
  echo 'Dry-run only: no source, index, commit, tag, or remote changes.'
  exit 0
fi
if git ls-files --error-unmatch apps/cockpit/src-tauri/Cargo.lock >/dev/null 2>&1; then
  FILES+=(apps/cockpit/src-tauri/Cargo.lock)
fi
# Never overwrite another actor's version edits or commit their staged work.
git diff --quiet -- "${FILES[@]}" || { echo 'Version surfaces have uncommitted changes' >&2; exit 2; }
git diff --cached --quiet || { echo 'Index contains existing staged work' >&2; exit 2; }
if [[ "$TAG_ONLY" == 0 ]]; then
  for tag in "${TAGS[@]}"; do python3 scripts/stamp-uiai-version.py "$tag"; done
  git add -- "${FILES[@]}"
  if ! git diff --cached --quiet; then git commit -m "chore(release): prepare ${TAGS[*]}"; fi
  git push origin HEAD:main
  if [[ "$PREPARE_ONLY" == 1 ]]; then echo 'Source prepared; no tags created.'; exit 0; fi
fi
for tag in "${TAGS[@]}"; do python3 scripts/verify-uiai-version-surfaces.py "$tag"; done
if [[ $(git rev-parse --is-shallow-repository) == true ]]; then
  git -c 'credential.helper=!gh auth git-credential' fetch --deepen=1 origin main
fi
head=$(git rev-parse HEAD)
# Run on the authorized execution host, never the production control host.
unset PREFLIGHT_FAST
bash scripts/local-uiai-preflight.sh --strict
run=$(gh run list --workflow ci.yml --commit "$head" --limit 5 --json databaseId,headSha | python3 -c 'import json,sys; rows=json.load(sys.stdin); print(next((str(x["databaseId"]) for x in rows if x["headSha"]==sys.argv[1]),""))' "$head")
[[ -n "$run" ]] || { echo "Exact-source CI has not registered; retry --tag-only after it registers: $head" >&2; exit 2; }
gh run watch "$run" --exit-status
[[ $(git rev-parse HEAD) == "$head" ]] || { echo 'Source changed during verification' >&2; exit 2; }
for tag in "${TAGS[@]}"; do git tag "$tag"; done
# One remote transaction for a coordinated parent/child release.
git -c 'credential.helper=!gh auth git-credential' push --atomic origin "${TAGS[@]}"
echo 'Immutable tags pushed; release workflow completion is still required.'
