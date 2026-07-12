#!/usr/bin/env bash
# Safely install repo-local UIAI Pi skills into ~/.pi/skills convenience copies.
set -euo pipefail
ROOT_DIR="${UIAI_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
SRC_DIR="$ROOT_DIR/.pi/skills"
DEST_DIR="${UIAI_PI_SKILLS_DEST:-$HOME/.pi/skills}"
MODE="dry-run"
BACKUP=1
FORCE_NO_BACKUP=0
usage(){ cat <<'USAGE'
Usage: scripts/install-pi-skills.sh [--dry-run|--apply] [--dest DIR] [--force-no-backup]

Copies repo-local .pi/skills/* into the global Pi skills directory for convenience.
Repo-local skills remain canonical. Dry-run is default.

Options:
  --dry-run          Print planned changes and diff hints; no writes (default)
  --apply            Copy skills into destination
  --dest DIR         Destination skills directory (default: ~/.pi/skills)
  --force-no-backup  Allow overwriting changed destination copies without backup
USAGE
}
while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) MODE="dry-run"; shift ;;
    --apply) MODE="apply"; shift ;;
    --dest) DEST_DIR="${2:?missing dest}"; shift 2 ;;
    --force-no-backup) BACKUP=0; FORCE_NO_BACKUP=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done
if [[ ! -d "$SRC_DIR" ]]; then echo "missing source skills: $SRC_DIR" >&2; exit 3; fi
if [[ "$(id -u)" == "0" && "${UIAI_ALLOW_ROOT_SKILL_SYNC:-}" != "1" ]]; then
  echo "refusing root skill sync without UIAI_ALLOW_ROOT_SKILL_SYNC=1; run as the target Pi user" >&2
  exit 4
fi
# macOS ships Bash 3.2, which has arrays but not mapfile/readarray.
skills=()
while IFS= read -r skill; do
  skills+=("$skill")
done < <(find "$SRC_DIR" -mindepth 1 -maxdepth 1 -type d -exec test -f '{}/SKILL.md' \; -print | sort)
if [[ ${#skills[@]} -eq 0 ]]; then echo "no repo-local skills found in $SRC_DIR" >&2; exit 5; fi
backup_root="$DEST_DIR/.uiai-backups/$(date +%Y%m%d-%H%M%S)"
changed=0
printf 'uiai skill sync mode=%s source=%s dest=%s skills=%d\n' "$MODE" "$SRC_DIR" "$DEST_DIR" "${#skills[@]}"
for src in "${skills[@]}"; do
  name="$(basename "$src")"
  dest="$DEST_DIR/$name"
  status="new"
  if [[ -d "$dest" ]]; then
    if diff -qr "$src" "$dest" >/dev/null 2>&1; then status="identical"; else status="changed"; fi
  fi
  printf '%s: %s -> %s\n' "$status" "$src" "$dest"
  [[ "$status" == "identical" ]] && continue
  changed=$((changed+1))
  if [[ "$MODE" == "dry-run" ]]; then
    if [[ -d "$dest" ]]; then printf '  diff: diff -ru %q %q || true\n' "$dest" "$src"; fi
    continue
  fi
  mkdir -p "$DEST_DIR"
  if [[ -e "$dest" ]]; then
    if [[ "$BACKUP" == "1" ]]; then
      mkdir -p "$backup_root"
      cp -a "$dest" "$backup_root/$name"
      printf '  backup: %s\n' "$backup_root/$name"
    elif [[ "$FORCE_NO_BACKUP" != "1" ]]; then
      echo "internal error: overwrite without backup requires --force-no-backup" >&2; exit 6
    fi
    rm -rf "$dest"
  fi
  cp -a "$src" "$dest"
  printf '  installed: %s\n' "$dest"
done
if [[ "$MODE" == "dry-run" ]]; then
  printf 'dry-run complete: changed_or_new=%d; rerun with --apply to install.\n' "$changed"
else
  printf 'apply complete: changed_or_new=%d\n' "$changed"
fi
