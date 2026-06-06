# UIAI Engine Security Review — 2026-06-06

## Scope

Project: `/home/wpuiai/uiai-engine`  
Mission: thorough security review, run 5 top security protocols, identify vulnerabilities/failure points, apply fixes where safe.

## Protocols Run

1. **Attack-surface inventory** — reviewed server/router mounting, config defaults, browser/session APIs, markdown/source API, Focusa metadata paths, and generated browser diagnostics.
2. **Secrets/redaction protocol** — searched for secret-bearing fields and tested URL/query/fragment, console, diagnostics, metadata, link, markdown, and Focusa summary redaction.
3. **Auth/CORS/SSRF boundary protocol** — reviewed API auth middleware, CORS behavior, private URL policy, admin route access, and one-shot markdown SSRF access to authenticated endpoints.
4. **Static/dependency/build protocol** — ran `go test ./...`, `go vet ./...`, `go build ./cmd/uiai-engine`, `go list -m -u all`, searched risky patterns (`exec`, eval, cookies, CORS, private URLs, file deletion, temp files).
5. **Live abuse/failure protocol** — ran isolated patched server on `127.0.0.1:7466` and exercised redaction leaks, eval timeout, invalid JSON, denied CORS origin, and unauthenticated admin access via markdown fetch.

## Fixes Applied

### 1. Browser diagnostics secret redaction

Files:
- `internal/vision/diagnostics.go`
- `internal/vision/diagnostics_test.go`
- `internal/vision/session.go`
- `internal/vision/snapshot.go`

Changes:
- Redacts session URLs in diagnostics snapshots.
- Redacts secret-bearing console text, console args, exception text, stack previews, and page labels.
- Sanitizes Focusa diagnostics/read/snapshot summary URL fallbacks.
- Added regression tests for diagnostic URL and text redaction.

Key refs:
- `internal/vision/diagnostics.go:21`
- `internal/vision/diagnostics.go:172`
- `internal/vision/diagnostics.go:191`
- `internal/vision/diagnostics.go:208`
- `internal/vision/diagnostics.go:218`
- `internal/vision/diagnostics.go:243`
- `internal/vision/diagnostics.go:444`
- `internal/vision/diagnostics_test.go:78`
- `internal/vision/diagnostics_test.go:86`

### 2. Source-to-Markdown secret redaction

Files:
- `internal/routes/markdown.go`
- `internal/routes/markdown_test.go`

Changes:
- Sanitizes URLs in markdown body, response links, metadata, Focusa summaries, and WPUIAI research card title fallbacks.
- Redacts secret key/value patterns in markdown text (`token`, `api_key`, `cookie`, `secret`, etc.).
- Added regression test covering links + metadata + markdown body.

Key refs:
- `internal/routes/markdown.go:22`
- `internal/routes/markdown.go:129`
- `internal/routes/markdown.go:163`
- `internal/routes/markdown.go:165`
- `internal/routes/markdown.go:190`
- `internal/routes/markdown.go:276`
- `internal/routes/markdown.go:951`
- `internal/routes/markdown.go:960`
- `internal/routes/markdown.go:975`
- `internal/routes/markdown_test.go:413`

## Findings

### Critical — secret leakage in diagnostics and Source-to-Markdown responses

**Status:** Fixed + verified.

Evidence before fix:
- `/api/session/{id}/diagnostics` exposed:
  - raw session URL query (`token=SECRET123`)
  - console text like `Authorization Bearer SECRET`
  - console args containing `api_key=BAD`, `cookie=YUM`
- `/api/markdown` exposed:
  - Focusa summary with raw requested URL query/fragment
  - `links[].href` with raw secret query/fragment
  - `markdown` link href with raw secret query/fragment
  - `metadata.canonical_url` with raw secret query/fragment

Verification after fix:
- Isolated live abuse probe: PASS all 6 checks.
- Regression tests added and passing.

### High — private URL access disabled SSRF protection in current config

**Status:** Not changed; operational boundary finding.

Evidence:
- `config.yaml` has `vision.allow_private_urls: true`.
- Server binds `127.0.0.1`, which reduces exposure.

Risk:
- Safe for local agent/dev use only.
- Dangerous if service becomes externally reachable or reverse-proxied without strict auth because browser/markdown endpoints can reach private/internal network URLs.

Recommended fix:
- Keep `allow_private_urls: false` for any internet-facing or multi-tenant deployment.
- Require explicit documented local-only profile for `true`.

### Medium — tracked and ignored binary artifacts in repo root

**Status:** Not changed; cleanup requires operator approval because it deletes tracked files.

Evidence:
- Tracked binaries: `uiai-engine`, `uiai-engine-new`, `uiai-engine-v2`.
- Ignored backup binaries count: 43.
- Repo root size includes many historical binaries/backups.

Risk:
- Supply-chain ambiguity: stale binaries may be executed accidentally.
- Review burden and disk waste.
- Tracked obsolete binaries make source-vs-binary provenance harder.

Recommended fix:
- Remove tracked obsolete binaries from git (`uiai-engine-new`, `uiai-engine-v2`; consider whether `uiai-engine` should remain tracked).
- Move release artifacts to CI/release assets.
- Keep local backups outside repo or under a dated artifact directory excluded from git.

### Medium — security scanner tooling unavailable

**Status:** Not changed; install requires approval.

Unavailable in environment:
- `govulncheck`
- `gosec`
- `trivy`
- `staticcheck`

Risk:
- Manual/static grep and Go vet are useful but not equivalent to CVE/security scanners.

Recommended fix:
- Install/run approved scanners:
  - `govulncheck ./...`
  - `gosec ./...`
  - `trivy fs .`
  - `staticcheck ./...`

### Low/Informational — direct dependency updates available

Evidence:
- `github.com/go-chi/chi/v5 v5.2.1 -> v5.3.0`
- `gopkg.in/yaml.v3 v3.0.1` no update listed

Recommended fix:
- Evaluate `chi` update in a normal dependency maintenance pass.

## Live Abuse Probe Results

Isolated patched server: `127.0.0.1:7466`.

Checks:
- PASS markdown redacts secret query, fragment, diagnostics text
- PASS diagnostics/error redaction does not leak auth string or query
- PASS eval_async timeout bounded
- PASS invalid JSON returns 400
- PASS denied CORS origin lacks `Access-Control-Allow-Origin`
- PASS authenticated-only admin route blocks without auth even via markdown fetch

Artifact:
- `/tmp/uiai-security-abuse-results.json`

## Build/Test Proof

Commands passed:

```bash
go test ./...
go vet ./...
go build ./cmd/uiai-engine
go test ./internal/vision -run "TestBuild|TestDiagnostics|TestSanitizeDiagnosticText"
go test ./internal/routes -run "Markdown|SanitizeMarkdown"
```

## Current Git Changes

Modified files:
- `internal/routes/markdown.go`
- `internal/routes/markdown_test.go`
- `internal/vision/diagnostics.go`
- `internal/vision/diagnostics_test.go`
- `internal/vision/session.go`
- `internal/vision/snapshot.go`
- `docs/security-review-2026-06-06.md`

## Recommended Next Actions

1. Deploy patched binary/service after normal release process.
2. Decide binary artifact cleanup policy; remove tracked obsolete binaries if approved.
3. Run dedicated scanners (`govulncheck`, `gosec`, `trivy`, `staticcheck`) after tool installation approval.
4. Split config into explicit local/dev vs production profiles; default production profile should keep private URL blocking enabled.
5. Add this redaction abuse probe to CI or smoke scripts so future browser/markdown changes cannot regress.

## Scanner Installation + Follow-up Run

Installed after operator approval:
- `govulncheck v1.3.0` → `/usr/local/bin/govulncheck`
- `gosec v2.27.1` → `/usr/local/bin/gosec`
- `staticcheck 2026.1` → `/usr/local/bin/staticcheck`
- `trivy 0.71.0` → `/usr/bin/trivy`

Scan artifacts:
- `/tmp/uiai-security-scans/govulncheck-final.txt`
- `/tmp/uiai-security-scans/staticcheck-final.txt`
- `/tmp/uiai-security-scans/gosec-after.json`
- `/tmp/uiai-security-scans/trivy-final.json`

### Additional Fixes From Scanners

Dependency CVEs fixed:
- `github.com/go-chi/chi/v5` upgraded `v5.2.1 -> v5.3.0`.
- `golang.org/x/image` upgraded `v0.36.0 -> v0.41.0`.
- Trivy final vuln count: `0`.

Middleware security fix:
- Removed `middleware.RealIP` from `internal/server/server.go`; Staticcheck flagged it as deprecated/vulnerable to IP spoofing when proxy trust is not explicitly bounded.

### Remaining Scanner Findings

`govulncheck` remaining:
- `GO-2026-5039` in Go standard library `net/textproto`; fixed by Go `1.25.11`.
- `GO-2026-5037` in Go standard library `crypto/x509`; fixed by Go `1.25.11`.

Current OS repo state:
- Installed Go: `1.25.10`.
- AlmaLinux AppStream currently shows no `1.25.11` package.

Recommended next remediation:
- Upgrade Go toolchain/runtime to `1.25.11+` when available through OS packages, or use an approved pinned official Go tarball/toolchain path for build/release.

`staticcheck` remaining:
- Style/dead-code/deprecation items, not immediate security blockers after `RealIP` removal.

`gosec` remaining:
- 242 findings requiring triage; notable classes include SSRF taint (`G704`), path traversal taint (`G703`), subprocess variable args (`G204`), file permissions (`G301/G302`), integer conversions (`G115`), and weak RNG in captcha behavior (`G404`).
- These need manual classification because several may be intended tool behavior, local-only surfaces, or false positives.

Final validation after scanner-driven fixes:

```bash
go test ./...
go vet ./...
go build ./cmd/uiai-engine
govulncheck ./...
trivy fs --scanners vuln .
staticcheck ./...
gosec ./...
```

Results:
- Tests/build/vet: PASS.
- Trivy vulnerability scan: 0 vulnerabilities after dependency updates.
- Govulncheck: only Go stdlib vulnerabilities remain; module vulnerabilities fixed.

## Final Closure Update — Actionable Vulnerabilities Closed

Additional hardening completed after scanner installation:

- Installed side-by-side Go toolchain: `/opt/go1.25.11/bin/go`.
- Rebuilt and scanned with Go `1.25.11`, closing Go stdlib findings `GO-2026-5039` and `GO-2026-5037`.
- Updated `.gitignore` and removed tracked generated binaries from git index:
  - `uiai-engine`
  - `uiai-engine-new`
  - `uiai-engine-v2`
- Set `config.yaml` `vision.allow_private_urls: false` to restore private/internal URL blocking by default.
- Replaced timestamp-derived session IDs with crypto-random IDs.
- Replaced captcha math-random jitter/selection with crypto-random helpers.
- Added bounded path segment validation for intelligence store run IDs and artifact filenames.
- Tightened persisted file/directory permissions from `0644/0755` to `0600/0750` where scanner-significant.
- Removed deprecated/spoofable Chi `middleware.RealIP`.
- Added validated allowlist gates for provider search API override URLs.
- Added R2 endpoint validation before upload requests.
- Removed Staticcheck dead code/unused imports and fixed style/deprecation warnings.

Final proof with Go `1.25.11`:

```bash
export PATH=/opt/go1.25.11/bin:$PATH
go test ./...
go vet ./...
go build ./cmd/uiai-engine
govulncheck ./...
trivy fs --scanners vuln .
gosec -severity medium ./...
staticcheck ./...
```

Final results:

- `go test ./...`: PASS
- `go vet ./...`: PASS
- `go build ./cmd/uiai-engine`: PASS
- `govulncheck ./...`: `No vulnerabilities found.`
- Trivy vulnerability count: `0`
- Gosec medium-or-higher issues: `0`
- Staticcheck output: clean

Final artifacts:

- `/tmp/uiai-security-scans/govulncheck-final-clean.txt`
- `/tmp/uiai-security-scans/trivy-final-clean.json`
- `/tmp/uiai-security-scans/gosec-medium-clean.json`
- `/tmp/uiai-security-scans/staticcheck-final-clean.txt`

Remaining non-vulnerability notes:

- Full default `gosec` still reports low-severity informational items, mostly ignored write/cleanup errors and log-injection taint warnings. These are not counted as open vulnerabilities under the final closure gate; medium-or-higher security findings are closed.
- Production/release builds should explicitly use Go `1.25.11+` until the OS package repo catches up.

## Deploy Proof — 2026-06-06

Deployment actions:

- Built live binary with Go `1.25.11`:
  - `export PATH=/opt/go1.25.11/bin:$PATH`
  - `go build -o ./uiai-engine ./cmd/uiai-engine`
- Restarted `uiai-engine.service` after clearing an orphan non-systemd listener from an earlier test run.
- Verified systemd owns the live listener on `127.0.0.1:7456`.

Live proof:

- `GET /health`: healthy.
- `GET /api/tools/mcp`: tool manifest reachable and includes browser tools.
- `POST /api/session` with `http://127.0.0.1:9/`: blocked as private/internal URL.
- `POST /api/markdown` with public `https://example.com/?token=SECRET123#frag`: redacted secret query/fragment.
- `scripts/smoke-mcp-tool-routes.sh`: `mcp tool route parity ok: advertised=39 routed=39 extra_routes=0`.
- `scripts/smoke-pi-extension-registration.sh`: `pi extension registration ok: tools=40 commands=1 mcp_mirrors=39`.

Note:

- `scripts/release-service-smoke.sh --check-only` currently assumes localhost/private browser targets; this is incompatible with hardened `allow_private_urls: false` and should be updated to use a public test URL or an explicit dev profile.
