#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="${UIAI_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
SRC_DIR="$ROOT_DIR/.pi/skills"
expected=(uiai-agent uiai-focusa-packet uiai-release uiai-mcp uiai-remote-auth uiai-docs-maintenance uiai-ci-debug uiai-browser-debug vision)
missing=()
for name in "${expected[@]}"; do
  file="$SRC_DIR/$name/SKILL.md"
  if [[ ! -f "$file" ]]; then missing+=("$name:missing"); continue; fi
  if ! grep -q "^name: $name$" "$file"; then missing+=("$name:frontmatter-name"); fi
  if ! grep -q "$name" "$ROOT_DIR/README.md"; then missing+=("$name:README"); fi
done
if [[ ${#missing[@]} -gt 0 ]]; then
  printf 'pi skill smoke failed: %s\n' "${missing[*]}" >&2
  exit 1
fi
printf 'pi skill smoke ok: skills=%d\n' "${#expected[@]}"
