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
cat > main.go <<'GO'
package main
func main(){println("changed")}
GO
before="$(sha256sum main.go | cut -d' ' -f1)"
if bash "$ROOT/scripts/check-go-format.sh" "$TMP" >output.txt 2>&1; then
  echo "formatter check accepted changed unformatted source" >&2
  exit 1
fi
after="$(sha256sum main.go | cut -d' ' -f1)"
[[ "$before" == "$after" ]] || { echo "formatter check mutated source" >&2; exit 1; }
grep -q '^main.go$' output.txt || { echo "formatter check omitted exact path" >&2; exit 1; }
gofmt -w main.go
formatted="$(sha256sum main.go | cut -d' ' -f1)"
bash "$ROOT/scripts/check-go-format.sh" "$TMP" >output.txt
[[ "$formatted" == "$(sha256sum main.go | cut -d' ' -f1)" ]] || { echo "passing formatter check mutated source" >&2; exit 1; }
echo "check-go-format regression: PASS"
