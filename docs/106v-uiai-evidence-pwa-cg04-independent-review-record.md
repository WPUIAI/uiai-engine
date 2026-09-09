# 106v — UIAI Evidence PWA: CG04 Independent Review Record (FAIL → exact gaps returned)

- **Date:** 2026-09-09
- **Node:** CG-04 hostile-content independent join (T03)
- **Reviewer authority:** operator-directed independent review, deliberately separate account from the implementation agent
- **Frozen set:** `evidence/` as of `PROVENANCE.txt` frozen `2026-09-09T12:12:39.475Z`; release checkout HEAD `aa2c0cfccd63de526f453f9f6dee245b6dcd7a22` (CG03 record merge); frozen scope line: "CG04 hostile-content independent join (adversarial fixtures, sanitization, redaction, leak checks; threat corpus + scan report)"
- **Outcome: VERDICT: FAIL** — mechanisms statically verified present and coherent, but the required evidence atoms do not exist in the frozen set. The join does not close. Repair/reproof cycle 1 of 2 used.
- **Durable verdict file:** `/tmp/epwa-review-cg04-fFGBtC/REVIEW-VERDICT.md` (read-only review; nothing outside that file was written)

## Reviewer-verified highlights

- Sanitization refusal, redaction-state discipline, leak prevention, media-type sniffing/EXIF rejection, and determinism/identity are present, coherent, internally tested, and deterministically bound to asset hashes — statically verified against their named adversarial fixtures.

## Blocking gaps (exact, returned to the model)

1. **B-1 — Threat corpus + scan report artifacts absent.** The CG04 evidence atoms (`threat corpus + scan report`) do not exist in the frozen set: no corpus artifact, no scan report, no test producing them. Required: a durable, committed, hash-bound threat-corpus artifact enumerating the adversarial fixtures — including hashes for the uncovered classes the reviewer named (G-1 metadata asymmetry for GIF/MP4/WebM/WAV/OGG/MP3; G-2; G-4) — plus a scan report recording results per corpus entry.
2. **B-2 — Frozen slice is not independently executable.** The frozen `evidence/internal-evidenceartifact/` slice lacks `go.mod` and non-frozen siblings (`Attestation`, `TrustBundle`, `Seal`, `deterministicID`, store implementation referenced by tests); `go test` fails to build. Required: either a self-contained frozen evidence slice that builds and runs, or the CG04 join record must bind the exact executed commit/manifest.

## Hardening observations (non-blocking for CG04's own done condition)

- G-1: metadata asymmetry — `inspectMedia` rejects EXIF/text chunks only for PNG/JPEG/WebP; GIF/MP4/WebM/WAV/OGG/MP3 containers pass with only magic-byte verification, so device/location metadata can persist in accepted evidence. The corpus must at least document these classes.
- G-2, G-4: additional uncovered attacker classes named in the verdict; corpus must enumerate them even where the code has no defense yet.

## Post-amendment note

This review froze **before** the closure-governance amendments (PRs #213 `5bf38da7`, #214 `64581411`). The merged CG-04 row now additionally names CSP/CSRF/CORS/SSRF/archive controls and public/private disclosure checks. The next CG04 reproof set must satisfy the amended row and the reviewer's B-1/B-2 requirements; the corpus and scan report should be produced as committed, hash-bound artifacts (not ad-hoc files) so the independent reviewer can consume them from a fresh frozen set.

## Truth notice

This record does not close CG-04, any downstream join (CG-05 onward), the task, provider closure, or settlement. Static verification only (L-1/L-2/L-3 limitations in the verdict). CG-04 remains open with exactly the gaps above until an executable, hash-bound corpus + scan report exists and an independent reviewer accepts a fresh frozen set.
