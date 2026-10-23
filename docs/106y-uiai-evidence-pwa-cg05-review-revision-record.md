# 106y — UIAI Evidence PWA: CG05 Review Revision Record (FAIL → ACCEPT, cycle 2 of 2)

- **Date:** 2026-09-12
- **Node:** CG-05 T04 capture assembly independent join
- **Reviewer authority:** independent acceptance reviewer, deliberately separate account/model from the implementation agent (executed every command itself)
- **Frozen set (cycle 2):** `/tmp/epwa-review-cg05-c2-o9AWBE/evidence/` per `PROVENANCE.txt`, frozen `2026-09-12T04:51:34.191Z`, repair-branch HEAD `0f8b722d0698aafbf315ea938906f475eedf8e0f` (`cg05-unknown-permission`, PR #229). Full-source frozen slice: `go.mod` at `evidence/` so every internal package builds and tests run.
- **Frozen set (cycle 1):** `/tmp/epwa-review-cg05-ff0JBB/` at release HEAD `1f0625353faebea62bf3aeb8252546912ab78522`.
- **Outcome: cycle-1 VERDICT: FAIL → cycle-2 VERDICT: ACCEPT** — the join closes.

## Cycle-1 blocking point (verbatim)

> "The 'permission' and 'unknown-proof' clauses of the amended row are not evidenced by any committed implementation, test, or digest binding in this frozen set."

## Repair (PR #229, commit `0f8b722`)

- `internal/evidenceartifact/assembly.go`: `CaptureOmission` gains `outcome` + `proof_ref` JSON fields (empty outcome preserves legacy omission semantics).
- `internal/evidenceartifact/assembly_validate.go`: postures `unavailable`/`unknown`/`denied` require a `proof_ref` (receipt proving the posture) — else `ErrAssemblyInvalid`; an `unknown` or `denied` posture can never support a complete-window claim — `ErrCoverageIncomplete`; invented postures are rejected — `ErrAssemblyInvalid`.
- `internal/evidenceartifact/assembly_unknown_test.go` + `assembly_permission_test.go`: unknown and permission-denied captures cannot prove a complete window; postures survive assembly; proof-less postures invalid.

## Cycle-2 reviewer-verified highlights (VERDICT: ACCEPT)

- **B-1 (permission) RESOLVED:** committed implementation + `TestPermissionDeniedCaptureCannotProveCompleteWindow` evidence the clause; a permission-denied capture posture is proof-ref-required and cannot support a complete-window claim.
- **B-2 (unknown-proof) RESOLVED:** committed implementation + `TestUnknownCaptureCannotProveCompleteWindow` evidence the clause; unknown posture is proof-ref-required and never proves absence.
- **Digest binding verified structural:** posture validation sits inside the sealed-hash assembly path, so future digests bind postures automatically via the new JSON fields; the committed 30-run digest atom is byte-identical (digest `bdbd5e67…`, capture file `d31d1e41…`, verify-against green).
- **71/71 evidenceartifact tests PASS** including both new posture tests; sibling packages regression-clean.
- **Blocking gaps: None found.** Hardening H-1..H-5 carried forward; new H-6/H-7 (dedicated negative test for the `unavailable` proof-ref requirement; posture-binding exercise inside a future digest regeneration) recorded as non-blocking follow-ups.

## Disposition

CG-05 **accepted** (both review cycles of its own two-cycle budget used: FAIL at cycle 1, ACCEPT at cycle 2). Repair review record is this file; next join is CG-06 (T05 crypto/time/custody independent join).
