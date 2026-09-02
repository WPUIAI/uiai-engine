#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
cd "$TMP"
git init -q
git config user.email test@example.invalid
git config user.name Test

cat > main.go <<'GO'
package main

func main() { println("baseline") }
GO
git add main.go
git commit -qm baseline
git remote add origin "$TMP/nonexistent-remote.git"
git update-ref refs/remotes/origin/main HEAD
git branch --set-upstream-to=origin/main >/dev/null
if UIAI_FORMAT_BASE=missing-ref bash "$ROOT/scripts/check-go-format.sh" "$TMP" >output.txt 2>&1; then
  echo "formatter check accepted invalid explicit base" >&2
  exit 1
fi
grep -q '^invalid UIAI_FORMAT_BASE: missing-ref$' output.txt || { echo "invalid base error is not actionable" >&2; exit 1; }
unlink output.txt

cat > main.go <<'GO'
package main
func main(){println("changed")}
GO
before_worktree="$(sha256sum main.go | cut -d' ' -f1)"
if bash "$ROOT/scripts/check-go-format.sh" "$TMP" >output.txt 2>&1; then
  echo "formatter check accepted changed unformatted source" >&2
  exit 1
fi
[[ "$before_worktree" == "$(sha256sum main.go | cut -d' ' -f1)" ]] || { echo "formatter check mutated source" >&2; exit 1; }
grep -q '^main.go$' output.txt || { echo "formatter check omitted exact path" >&2; exit 1; }
! grep -q 'PASS' output.txt || { echo "formatter check reported PASS with a diff" >&2; exit 1; }

git add main.go
before_index="$(git rev-parse :main.go)"
if bash "$ROOT/scripts/check-go-format.sh" "$TMP" >output.txt 2>&1; then
  echo "formatter check accepted staged unformatted source" >&2
  exit 1
fi
[[ "$before_index" == "$(git rev-parse :main.go)" ]] || { echo "formatter check mutated index" >&2; exit 1; }

git commit -qm unformatted
unlink output.txt
[[ -z "$(git status --porcelain)" ]] || { echo "fixture worktree not clean" >&2; exit 1; }
if bash "$ROOT/scripts/check-go-format.sh" "$TMP" >output.txt 2>&1; then
  echo "formatter check ignored committed unformatted source" >&2
  exit 1
fi
grep -q '^main.go$' output.txt || { echo "committed diff omitted exact path" >&2; exit 1; }

git checkout --detach -q
if bash "$ROOT/scripts/check-go-format.sh" "$TMP" >output.txt 2>&1; then
  echo "detached-worktree check ignored committed unformatted source" >&2
  exit 1
fi

gofmt -w main.go
git add main.go
git commit -qm formatted
formatted="$(sha256sum main.go | cut -d' ' -f1)"
printf 'leave me untouched\n' > unrelated.txt
unrelated="$(sha256sum unrelated.txt | cut -d' ' -f1)"
bash "$ROOT/scripts/check-go-format.sh" "$TMP" >output.txt
[[ "$formatted" == "$(sha256sum main.go | cut -d' ' -f1)" ]] || { echo "passing formatter check mutated source" >&2; exit 1; }
[[ "$unrelated" == "$(sha256sum unrelated.txt | cut -d' ' -f1)" ]] || { echo "formatter check mutated unrelated dirty file" >&2; exit 1; }
grep -q '^changed Go format: PASS$' output.txt || { echo "passing formatter output is misleading" >&2; exit 1; }

echo "check-go-format regression: PASS"
