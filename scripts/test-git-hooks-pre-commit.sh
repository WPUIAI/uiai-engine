#!/usr/bin/env bash
set -euo pipefail

ROOT="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
HOOK="$ROOT/scripts/git-hooks-pre-commit"
TMP="$(mktemp -d)"
trap 'rm -rf -- "$TMP"' EXIT
export PATH="$TMP/bin:$PATH"
mkdir -p "$TMP/bin"

cat >"$TMP/bin/bd" <<'BD'
#!/usr/bin/env bash
set -euo pipefail
[[ "${1:-}" == sync && "${2:-}" == --flush-only ]]
[[ -z "${BD_CALL_MARKER:-}" ]] || printf 'called\n' >>"$BD_CALL_MARKER"
[[ "${BD_FAIL:-0}" != 1 ]] || exit 42
[[ -z "${BD_FLUSH_CONTENT:-}" ]] || printf '%s\n' "$BD_FLUSH_CONTENT" >.beads/issues.jsonl
BD
chmod +x "$TMP/bin/bd"

new_fixture() {
  local name="$1"
  local repo="$TMP/$name"
  mkdir -p "$repo/.beads"
  git -C "$repo" init -q
  git -C "$repo" config user.email test@example.invalid
  git -C "$repo" config user.name Test
  printf '{"id":"baseline"}\n' >"$repo/.beads/issues.jsonl"
  printf 'baseline\n' >"$repo/app.txt"
  git -C "$repo" add .beads/issues.jsonl app.txt
  git -C "$repo" commit -qm baseline
  printf '%s\n' "$repo"
}

index_digest() { git -C "$1" ls-files --stage -z | sha256sum | awk '{print $1}'; }
file_digest() { sha256sum "$1" | awk '{print $1}'; }

# Clean ledger and no flush delta: hook succeeds without changing the index.
repo="$(new_fixture clean)"
printf 'feature\n' >"$repo/feature.txt"
git -C "$repo" add feature.txt
before_index="$(index_digest "$repo")"
(cd "$repo" && "$HOOK")
[[ "$before_index" == "$(index_digest "$repo")" ]]
[[ "$(git -C "$repo" diff --cached --name-only)" == feature.txt ]]

# Existing unstaged ledger changes: fail before invoking bd and preserve bytes/index.
repo="$(new_fixture unstaged)"
printf 'feature\n' >"$repo/feature.txt"
git -C "$repo" add feature.txt
printf '{"id":"concurrent"}\n' >"$repo/.beads/issues.jsonl"
marker="$TMP/unstaged-called"
before_ledger="$(file_digest "$repo/.beads/issues.jsonl")"
before_index="$(index_digest "$repo")"
if (cd "$repo" && BD_CALL_MARKER="$marker" "$HOOK" >"$TMP/unstaged.out" 2>&1); then
  echo "unstaged ledger was accepted" >&2
  exit 1
fi
[[ ! -e "$marker" ]]
[[ "$before_ledger" == "$(file_digest "$repo/.beads/issues.jsonl")" ]]
[[ "$before_index" == "$(index_digest "$repo")" ]]
grep -q 'has unstaged changes' "$TMP/unstaged.out"

# Partially staged ledger: fail before flush and preserve staged + worktree forms.
repo="$(new_fixture partial)"
printf '{"id":"staged"}\n' >"$repo/.beads/issues.jsonl"
git -C "$repo" add .beads/issues.jsonl
printf '{"id":"unstaged-after-stage"}\n' >"$repo/.beads/issues.jsonl"
marker="$TMP/partial-called"
before_ledger="$(file_digest "$repo/.beads/issues.jsonl")"
before_index="$(index_digest "$repo")"
if (cd "$repo" && BD_CALL_MARKER="$marker" "$HOOK" >"$TMP/partial.out" 2>&1); then
  echo "partially staged ledger was accepted" >&2
  exit 1
fi
[[ ! -e "$marker" ]]
[[ "$before_ledger" == "$(file_digest "$repo/.beads/issues.jsonl")" ]]
[[ "$before_index" == "$(index_digest "$repo")" ]]
grep -q 'partially staged' "$TMP/partial.out"

# A clean flush that creates ledger work must remain unstaged and block the commit.
repo="$(new_fixture flush-delta)"
printf 'feature\n' >"$repo/feature.txt"
git -C "$repo" add feature.txt
before_index="$(index_digest "$repo")"
if (cd "$repo" && BD_FLUSH_CONTENT='{"id":"flushed"}' "$HOOK" >"$TMP/flush.out" 2>&1); then
  echo "flush delta was accepted" >&2
  exit 1
fi
[[ "$before_index" == "$(index_digest "$repo")" ]]
[[ "$(git -C "$repo" diff --cached --name-only)" == feature.txt ]]
grep -q 'Beads flush updated' "$TMP/flush.out"
grep -q '"id":"flushed"' "$repo/.beads/issues.jsonl"

# An intentionally staged ledger with no additional flush delta remains valid.
repo="$(new_fixture staged)"
printf '{"id":"intentional"}\n' >"$repo/.beads/issues.jsonl"
git -C "$repo" add .beads/issues.jsonl
before_index="$(index_digest "$repo")"
(cd "$repo" && "$HOOK")
[[ "$before_index" == "$(index_digest "$repo")" ]]

# Flush failures block without changing the index.
repo="$(new_fixture flush-failure)"
printf 'feature\n' >"$repo/feature.txt"
git -C "$repo" add feature.txt
before_index="$(index_digest "$repo")"
if (cd "$repo" && BD_FAIL=1 "$HOOK" >"$TMP/failure.out" 2>&1); then
  echo "failed flush was accepted" >&2
  exit 1
fi
[[ "$before_index" == "$(index_digest "$repo")" ]]
grep -q 'bd sync --flush-only failed' "$TMP/failure.out"

# Linked worktrees never call bd or mutate the shared main-worktree ledger.
repo="$(new_fixture linked-main)"
linked="$TMP/linked-worktree"
git -C "$repo" worktree add -q -b linked-test "$linked"
marker="$TMP/linked-called"
before_ledger="$(file_digest "$repo/.beads/issues.jsonl")"
printf 'linked feature\n' >"$linked/linked.txt"
git -C "$linked" add linked.txt
before_index="$(index_digest "$linked")"
(cd "$linked" && BD_CALL_MARKER="$marker" "$HOOK" >"$TMP/linked.out" 2>&1)
[[ ! -e "$marker" ]]
[[ "$before_ledger" == "$(file_digest "$repo/.beads/issues.jsonl")" ]]
[[ "$before_index" == "$(index_digest "$linked")" ]]
grep -q 'linked worktree detected' "$TMP/linked.out"

printf 'safe pre-commit Beads hook regression: PASS\n'
