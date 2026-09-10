# 106x — UIAI Evidence PWA: CG04 Independent Review Revision Record (cycle 2 FAIL → cycle 2 repair → re-review PASS)

- **Date:** 2026-09-10
- **Node:** CG-04 hostile-content independent join (T03)
- **Continues:** `106v` (cycle-1 FAIL record)
- **Repair/reproof budget:** two cycles maximum — both cycles used; the re-review below is the final permitted cycle for this join.

## Cycle 2 — independent re-review VERDICT: FAIL (layout + binding residuals)

- **Reviewer authority:** separate account (`openai-codex-account-2` / `gpt-6-astra`), not the implementation agent
- **Frozen set:** `/tmp/epwa-review-cg04-rr-XA6kxK/` — frozen slice with corpus + canonical scan report after PR #218
- **Outcome:** VERDICT: FAIL with these exact gaps:
  - **T1/T2 — harness layout defects (AI packaging errors, not repo defects):** frozen slice lacked `internal/durablefile` and had renamed the package directory (`internal-evidenceartifact`) so import paths did not match module paths.
  - **B-1 residual — report↔corpus binding** used the canonical in-memory re-marshal hash instead of the committed `corpus.json` artifact-byte hash.
  - **Class enumeration unverifiable in the frozen slice:** G-2/G-4 classes needed committed fixtures; amended-row clauses (OCR/transcript, CSP/CSRF/CORS/SSRF/archive, public/private disclosure) needed cited committed test evidence; one affirmative public-safe active-content acceptance class was required.
- **Durable verdict file:** `/tmp/epwa-review-cg04-rr-XA6kxK/REVIEW-VERDICT.md`

## Cycle-2 repair (PR #220, merged `21c67290e5bfe5900368d572ed7a7cc6fc146da3`)

- Report now binds the **committed corpus.json artifact bytes** (B-1 residual closed): `LoadThreatCorpus` records `FileSHA256`, the report refuses in-memory-only corpora.
- Corpus grows to **31 hash-bound fixtures** with per-entry typed `access_class`/`redaction_state`; new entries enumerate G-2 (JPEG COM/ICC metadata), G-3 (binary polyglot embedding script + secret), G-4 (binary magic re-declared as text/plain), G-5 (paraphrased prompt injection), G-6 (JWT and service-account credential breadth), G-7 (CSV formula injection), G-8 (redacted-claim soft signal), plus the affirmative public-safe active-content acceptance class (`threat:text-active-untrusted-public`).
- Uncovered classes stay typed `uncovered_hardening`; GitHub push protection rejected the synthetic Slack-token fixture, replaced with a non-secret service-account JSON fixture.
- Frozen-slice layout fixed in the review harness itself: the cycle-2/3 freeze copies the **full source tree** (`go.mod`, `go.sum`, `cmd/`, `internal/`, `contracts/`) — no directory renames, no missing siblings.
- Scan at merge: **31 entries, 31 matched, 14 uncovered documented, 0 unexpected**; `go test ./...` 30/30 packages green; CI green on #220.

## Cycle 3 — final re-review VERDICT: PASS

- **Reviewer authority:** separate account (`openai-codex-account-2` / `gpt-6-astra`); execution transcripts embedded in the verdict
- **Frozen set:** `/tmp/epwa-review-cg04-c2-7Tky1k/evidence/` — full source tree frozen `2026-09-10T03:40:24.494Z`, release HEAD `21c67290e5bfe5900368d572ed7a7cc6fc146da3`
- **Reviewer-executed proof (independently, from the frozen set):**
  1. `go test ./internal/evidenceartifact ./internal/evidenceshare ./internal/vision ./internal/routes` — all green
  2. `go run ./cmd/epwa-threat-scan --corpus ... --fixtures ... --code-ref re-review-cycle2 --verify-against .../scan-report.json` — byte-for-byte verified
  3. Corpus/report hashes recomputed independently; fixture SHA-256 bindings confirmed; artifact-byte binding of the report confirmed
- **Outcome:** VERDICT: PASS — B-1, B-2, R-5, R-6 all closed; G-1..G-8 enumerated concretely with matched outcomes; affirmative public-safe active-content class present and matching.
- **Durable verdict file:** `/tmp/epwa-review-cg04-c2-7Tky1k/REVIEW-VERDICT.md`
- **Disclosed reviewer incident:** the reviewer's initial `go list -mod=mod` probe rewrote the frozen `go.mod`; detected by byte-diff against the release checkout, restored pristine, and the full verification re-executed on the pristine tree with identical results.
- **Truth notice (verbatim obligations carried):** CG-04's done condition is attested met for the clauses cited in the verdict; clauses with no committed test evidence are stated plainly and pass only as static structural mitigations verified by the review itself.

## Hardening carried forward (non-blocking; next repair cycle's test additions)

1. **CSRF:** no committed test evidence; structural token-auth/no-cookie mitigation only — add an explicit CSRF no-cookie invariant test.
2. **CORS:** control untested; SSE wildcard origin on a read-only stream — add a CORS header test.
3. **OCR leak checks:** no committed test evidence; `internal/reference` untested — add OCR-output-through-inspector test.
4. **Transcript body-content leak checks:** no dedicated test — add transcript body scan test.
5. The 14 typed `uncovered_hardening` attacker classes in the corpus remain documented exposure surfaces.

## Disposition

- CG-04 join **accepted** under the graph's two-cycle budget; the two-cycle budget is exhausted.
- Hardening items above are follow-up test work, tracked so they are not silently lost; they do not reopen CG-04 unless they surface an exploitable path.
- This record closes only CG-04's join. CG-05+ downstream joins, task completion, provider closure, and settlement are not closed by this verdict.
