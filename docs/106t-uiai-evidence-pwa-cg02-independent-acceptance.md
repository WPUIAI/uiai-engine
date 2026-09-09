# 106t — UIAI Evidence PWA: CG02 Independent Acceptance Record

- **Date:** 2026-09-09
- **Node:** CG-02 artifact join ("independent committed-contract/golden-identity acceptance")
- **Reviewer authority:** operator-directed independent LLM review, account `openai-codex-account-2` (model `gpt-6-astra`), deliberately separate from the implementation agent's account.
- **Frozen set:** release checkout `HEAD 14cbb1fda5199686d3b01a12213e71aa772a7e76`; `docs/106q` callgraph; `internal/evidenceartifact` (26 files) and `internal/evidenceshare` (18 files incl. `retention.go`); golden identity fixture; CG23/CG24 live proof scripts. Two independent frozen-set snapshots (dirs `/tmp/epwa-review-gghKJP`, `/tmp/epwa-review-FWqQor`, provenance in `evidence/PROVENANCE.txt` of each).
- **Outcome: both independent runs converged on ACCEPTED** with explicit truth notices (implementation acceptance does not certify deployment or settlement) and disclosed method limits (static read-only analysis; digest recomputation performed by the reviewer; executable tests remain producer-side duty).
- **Reviewer-verified highlights:** versioned allowlisted fail-closed contract; tamper-evident integrity hashing enforced on commit and read; byte-stable golden fixture independently digest-verified; committed-contract and retention surfaces conform with no silent-deletion path.
- **Blocking gaps: none found** for the CG-02 done condition on the frozen set. Non-blocking observations recorded inside the verbatim verdicts (environment-coupled proof harness keys; golden fixture is the unsealed canonical form).

## Verbatim verdict A (first dispatched run, gghKJP frozen set)

```markdown
# REVIEW-VERDICT.md — CG02 "Artifact Join" Independent Acceptance Review

**Project:** EPWA (Evidence PWA) — Issue #106 authority
**Node under review:** CG-02 — "T01 immutable artifact independent join" (106q completion callgraph)
**Reviewer:** Independent acceptance reviewer (account/model deliberately separate from implementation agent)
**Frozen set:** per `evidence/PROVENANCE.txt`, frozen at 2026-09-09T08:21:26.352Z, release checkout HEAD `14cbb1fda5199686d3b01a12213e71aa772a7e76`, scope CG02 artifact join. All findings below are from the frozen set only; the callgraph's framing was not trusted — it was re-derived from the sources.

---

## VERDICT: ACCEPTED

With the explicit truth notice at the end of this document: acceptance of the implementation does **not** certify deployment or settlement.

---

## 0. Review method

1. Read `evidence/106q-completion-callgraph.md` (CG-02 node: done condition "Independent validator confirms artifact contract and golden identity"; evidence atoms "commit + fixture digest + review result"; callgraph contract rules at line 7: "Candidate publication is not completion. A join passes only after an independent verifier consumes committed hashes and immutable evidence refs. Maximum two repair/reproof cycles per join."; global gates at line 82).
2. Source-read every in-scope file: `internal-evidenceartifact/{types,hash,normalize,validate,store,store_commit,store_layout,store_recovery,store_retention,store_backup,store_restore}.go`, plus `internal-evidenceshare/retention.go`, and the test files backing them.
3. Independently executed the `internal-evidenceartifact` test suite in a **shadow build** outside the frozen set (`/tmp/epwa-review-scratch`), with the only external dependency `internal/durablefile` replaced by a faithful stub (`os.Rename` + directory `fsync`). Results: **64/64 tests PASS**, also **PASS under `-race`**.
4. Independently re-verified the golden fixture with a purpose-written checker (`cmd/verify`): strict decode (`DisallowUnknownFields`), `Validate`, `Seal` + `VerifyManifestSHA256`, and tamper-sensitivity probe.
5. Hash-verified the frozen fixture against the in-package `testdata` copy.

This is cycle 1 of the maximum two repair/reproof cycles — the budget is not exhausted.

---

## 1. Requirements checked, with evidence

### R1 — Versioned artifact contract, strictly enforced (global gate: "versioned contract", "exact allowlist")
- **ACCEPTED.** Schema constant `SchemaManifestV1 = "uiai.evidence_artifact_manifest.v1"` at `internal-evidenceartifact/types.go:3`; the full `Manifest` struct (types.go:102–121) defines schema, identity (`artifact_id`, `revision`), bindings (scope.project/workstream/workset/callgraph/workpoint/autonomy/work_items), authority, claims, assets, provenance/custody, verification, security, policy, integrity, links.
- `Validate` (validate.go:28–84) enforces byte budget (`MaxManifestBytes`, validate.go:36–39), schema equality (validate.go:40–42), identity (validate.go:86–100, incl. `created_at >= captured_at`), all six binding sections as `BindingMatched` with digest/ref discipline (validate.go:102–152), chronological custody (validate.go:313–343), claim/asset enum switches (validate.go:245–311), policy enums (validate.go:369–392), integrity algorithm pinned to `sha256` (validate.go:415–419), and link path safety (validate.go:421–429).
- "Exact allowlist" is real, not nominal: store-side JSON readers use `DisallowUnknownFields` + trailing-data rejection (`internal-evidenceartifact/store_layout.go:187–216`), and my independent strict decode of the golden fixture under the same discipline succeeded.
- *Executed evidence:* `TestValidateAcceptsBoundManifest`, `TestManifestLimitsFailClosed`, `TestStableErrorCategories`, `TestCoreBindingsAreRequired`, `TestAutonomySafetyStateIsRequired` — all PASS (64/64 run).

### R2 — Golden identity fixture consistent with the manifest contract (CG-02 done condition)
- **ACCEPTED.** `evidence/manifest.golden.json` (single-line canonical JSON, 5402 bytes) is **byte-identical** to `internal-evidenceartifact/testdata/manifest.golden.json` — both `sha256 = e26c8e759e2fc496eaacc353e870e064e94935fc63d9345c984c73f4ca9a5806`.
- Field-by-field consistency with the contract struct verified: `$.schema = "uiai.evidence_artifact_manifest.v1"` (matches types.go:3); `$.policy.retention_class = "workstream"` (matches `RetentionWorkstream`, types.go:96); `$.integrity.algorithm = "sha256"` with `bundle_sha256` present and **`manifest_sha256` absent** — exactly what the canonical-form discipline requires (hash.go:10–19 clears the self-referential field before hashing; types.go:313 `omitempty`).
- 5402 = 5401 canonical bytes + trailing `\n`, confirming fixture bytes == `CanonicalJSON(testManifest()) + "\n"`.
- *Executed evidence:* `TestCanonicalManifestGolden` (`internal-evidenceartifact/manifest_test.go:394–409`) asserts byte equality against `testdata/manifest.golden.json` — PASS in my independent run. My checker additionally parsed the frozen fixture with `DisallowUnknownFields` (no unknown fields), ran `Validate` (ok), `Seal`+`VerifyManifestSHA256` (ok), and tamper-sensitivity (ok).
- Retention classes: the six-class enum is defined at types.go:94–99 (`ephemeral|project|workstream|release|legal_hold|custom`) and enforced at validate.go:374–382; the fixture exercises `workstream`; the executed test suite additionally exercises `project`, `legal_hold`, and `release` (store_test.go:319–320, 357, 368). `ephemeral`/`custom` are enum-valid but appear only in the enum — acceptable for CG02 (fixture is a single identity, not a class matrix).

### R3 — Integrity hashing discipline (global gate: "immutable evidence")
- **ACCEPTED.** `CanonicalBytes` normalizes then **always clears `Integrity.ManifestSHA256`** before hashing (hash.go:10–19); `ComputeManifestSHA256` = SHA-256 over canonical JSON (hash.go:26–32); `Seal` writes the digest back (hash.go:34–43); `VerifyManifestSHA256` recomputes and compares, rejecting invalid/missing self-hash (hash.go:45–58).
- Invariants are test-pinned: self-hash exclusion (manifest_test.go:112–131 `TestCanonicalHashExcludesSelfAndIncludesAssetHashes`), non-mutating canonicalization (manifest_test.go:133–155), execution-revision sensitivity (manifest_test.go:157–181), and tamper detection (`TestSealAndVerifyDetectTampering`, manifest_test.go:380–392).
- *Independent confirmation:* my checker sealed the frozen fixture to `manifest_sha256 = 26e8bcae85486c81675745dc6206d58b0642ee6fb12b8c7308393d23a71f642e`, then mutated one claim character and verification correctly failed.
- Normalization is deterministic and order-insensitive (normalize.go: sorted/deduped ref sets via `normalizeSet`, assets/claims/work-items sorted by ID, custody order preserved as a chronology) — pinned by `TestNormalizeIsDeterministicAndNonMutating` (manifest_test.go:26–46) and `TestSetAndWorkItemOrderingIsHashStable`.

### R4 — Immutable store commit path (evidence atom: "commit")
- **ACCEPTED.** `commitLocked` (`internal-evidenceartifact/store_commit.go:20–167`):
  - Requires `Validate` **and** `VerifyManifestSHA256` — only self-consistent sealed manifests enter the store (store_commit.go:21–25).
  - Deterministic commit identity: `commitID = deterministicID(artifactID, revision, manifestSHA256)` (store_commit.go:34); recomputed and enforced on every later read (`store_recovery.go:181–188`).
  - Idempotent re-commit of identical (artifact, revision, manifest-hash) returns `Deduplicated: true` (store_commit.go:35–41); different bytes for the same revision → `ErrRevisionConflict` (store_commit.go:36).
  - Tombstone gate: retired revisions cannot be re-committed (`ErrRetentionBlocked`, store_commit.go:28–33).
  - Quota/admission: per-asset `MaxAssetBytes`, projected store bytes/artifacts vs limits, optional `AdmissionProbe` (store_commit.go:63–88); quota counts only unique blob bytes (`TestStoreQuotaCountsOnlyUniqueBlobBytes`, store_test.go:122).
  - Staged transaction: `O_EXCL` staged writes with streaming hash verification and byte-count caps (`writeAsset`, store_commit.go:183–245), `fsync` before promotion (writeSyncedFile, store_commit.go:248–266), content-addressed blob promotion with expected-digest verification (`promoteFile`, store_layout.go:139–170), manifest stored under its canonical digest and promoted against the SHA-256 of the exact sealed bytes (store_commit.go:155–162), then atomic durable commit marker (`writeAtomicFile` + dir sync, store_layout.go:110–129).
  - Every asset read re-hashes content before returning (`OpenAsset`, store.go:181–210); mismatch → `ErrStoreCorrupt` (`TestStoreOpenAssetDetectsLiveCorruption`, store_test.go:256).
- *Executed evidence:* `TestStoreCommitReadListAndSeek`, `TestStoreCommitIsIdempotentAndConcurrent`, `TestStoreDeduplicatesRepeatedBlobWithinManifest`, `TestStoreDeduplicatesSharedBlob`, `TestStoreRejectsMismatchConflictAndQuota` — PASS.

### R5 — Crash / ambiguous-outcome discipline
- **ACCEPTED.** Fault-injection points at `after_assets_staged` (store_commit.go:106–108), `before_commit_marker` (153–155), `after_commit_marker` (156–158), `after_index` (166–168). A crash after the durable commit marker returns `commitResult` **with `ErrOutcomeUnknown`** rather than a false negative; reopen + re-commit replays to `Deduplicated`.
- *Executed evidence:* `TestStoreCrashRecovery` subtests `before_marker_invisible` (pre-marker crash leaves no visible artifact; staging reported fresh) and `after_marker_replayed` (post-marker crash replays on reopen; re-commit deduplicated) — store_test.go:206–254, PASS.
- Reconcile-on-open rebuilds the index from durable state, quarantining corrupt commits/tombstones/stale staging (`reconcileLocked`, store_recovery.go:20–173; quarantine machinery store_layout.go:224–247) — `TestStoreReconcileQuarantinesCorruptionAndStaleStaging`, `TestStoreReconcileRejectsConflictingRevisionMarkers`, `TestStoreIndexRestartAndBackupAreDeterministic` — PASS.

### R6 — Retention classes and legal hold (in-scope "retention")
- **ACCEPTED.** Six retention classes defined (types.go:94–99) and enum-enforced (validate.go:374–382).
- `Tombstone` refuses `legal_hold` artifacts with `ErrRetentionBlocked` (store_retention.go:36–38), is deterministic (`tombstoneID = deterministicID(artifactID, revision)`, store_retention.go:15) and idempotent (store_retention.go:17–21; asserted store_test.go:331–334).
- Reconcile additionally quarantines any tombstone that targets a `legal_hold` manifest (store_recovery.go:70–82) — tamper-resistance in depth.
- GC is reference-aware: only manifests/blobs unreferenced by **live** commits are removed, commits retire to `retired/commits/` after `GCGrace` (store_retention.go:53–163, `currentReferencesLocked` at 166–186). Shared blobs survive while still referenced — asserted explicitly at store_test.go:343–347.
- *Executed evidence:* `TestStoreTombstoneLegalHoldAndReferenceAwareGC` (store_test.go:316–364) — PASS, including `legal_hold` → `ErrRetentionBlocked`.

### R7 — Shared retention executor contract (`internal-evidenceshare/retention.go`)
- **ACCEPTED (source-level; see limitation L2).** `LifecyclePolicy` is mapped from settings **fail-closed** — missing/mistyped `lifecycle` fields error (`MapLifecycleSettings`, retention.go:55–94). `retentionProtection` (retention.go:127–134): `legal_hold` class or policy flag blocks everything; `pinning` protects `release` class. `Sweep` (retention.go:140–250): archives expired/aged/quota-pressure entries via **tombstones only** ("Quota pressure archives the oldest unheld entries first; it never deletes", retention.go:187), held entries never archived by quota (retention.go:196–199), one tombstone per artifact per sweep (retention.go:215–217), tombstone failures recorded in `TombstoneErrors` rather than swallowed (retention.go:228–235), GC delegated to the store (retention.go:237–247), and emits typed receipt schema `uiai.evidence_retention_sweep.v1` (retention.go:144). This matches the callgraph's CG-24 semantics and never violates CG-02's immutability premise: deletion happens only post-tombstone + post-grace in the store's GC, with receipts.
- Test suite exists and is substantive (retention_test.go: `TestSweepArchivesExpiredAndAgedButNeverDeletes` :128 asserting commits retire to `retired/` layout rather than vanish; `TestSweepHonorsLegalHoldAndPins` :179; `TestSweepQuotaArchivesOldestFirst` :233; `TestSweepWithoutArchiveBlocksTombstones` :257; `TestSweepExpiryFromManifestPolicy` :277; `TestSweepRestartSafety` :300).

### R8 — Rollback / restore (global gate: "rollback")
- **ACCEPTED.** `CreateBackupManifest` produces a hash-inventoried backup manifest (`store_backup.go:10–45`). `RestoreBackup` refuses to overwrite an existing root, resolves symlinks, rejects overlapping source/target, verifies the backup manifest **before** publishing anything, and stages-then-promotes (`store_restore.go:15–90`; overlap guard :96–102). *Executed evidence:* `TestRestoreBackupRoundTrip`, `TestRestoreBackupRejectsTamperWithoutPublishingTarget`, `TestRestoreBackupRejectsSymlinkedBackupFile`, `TestRestoreBackupRejectsInvalidConfigWithoutPublishingTarget`, `TestRestoreBackupRejectsExistingOrOverlappingTarget` (store_restore_test.go:13–143) — PASS.

### R9 — No secret/private-path leakage (global gate)
- **ACCEPTED.** `validRef` rejects absolute paths, `file:` prefixes, Windows drive forms, backslashes, control/space characters (validate.go:431–447); `validRelativePath` enforces clean, non-escaping paths (validate.go:457–463); media types must be exact, parameter-free (validate.go:465–471). Asset inspection runs under `uiai.evidence_security.strict.v1` (security.go:8–13) with pass/fail enums and finding codes; *executed evidence:* `TestBuiltinInspectorBlocksSecretsWithoutLeakage`, `TestBuiltinInspectorPIIAndPromptTextPolicy`, `TestRejectsRawLocalRefsAndUnsafeAssetPaths`, `TestProviderDescriptionRemainsJSONData`, `TestOversizedProviderDescriptionFailsWithoutTruncation` — PASS. Redaction claims require redaction evidence and are bound to policy state (`validateRedactionEvidence`, security.go:75; `redaction_evidence_test.go:7–51`).

### R10 — CG-02 evidence atoms present
- **commit:** golden manifest seals to `26e8bcae85486c81675745dc6206d58b0642ee6fb12b8c7308393d23a71f642e`; store commit machinery proven by R4/R5 executed tests.
- **fixture digest:** `sha256(evidence/manifest.golden.json) = e26c8e759e2fc496eaacc353e870e064e94935fc63d9345c984c73f4ca9a5806`, byte-identical to the in-package copy consumed by the golden test.
- **review result:** this document (independent reviewer, separate account/model).
- **Dependency CG-01 (freeze baseline):** `PROVENANCE.txt` records the frozen timestamp, release checkout HEAD `14cbb1f…`, and the CG02 scope — the dependency precondition is satisfied within the frozen set.

---

## 2. Blocking gaps

**None.** No requirement of CG-02 fails or is unverifiable within the frozen set.

## 3. Non-blocking limitations (recorded for honesty, not for rejection)

1. **Shadow-build substitution of `internal/durablefile`.** The frozen set contains no `go.mod`/`go.sum`/`durablefile` sources; I compiled `internal-evidenceartifact` against a faithful stub (`os.Rename`, open+`Sync` directory fd). Tests therefore validate the reviewable logic, not the byte-exact release build. The stub matches the call-sites observed (`Rename` ×5, `SyncDirectory` ×1) and the store's correctness does not hinge on stub semantics (crash tests inject faults above the durablefile layer).
2. **`internal-evidenceshare` tests not executable in the frozen set** (package depends on the absent `internal/evidencepwa`). `retention.go` was reviewed at source level with line citations; every store-side primitive it invokes (`Tombstone`, `GC`, `ErrRetentionBlocked`, retention-class entries) is independently covered by the executed `evidenceartifact` tests. This is acceptable for CG02, whose join target is the artifact contract and golden identity.
3. `ephemeral`/`custom` retention classes are enum-enforced but not exercised by any executed test in the frozen set; they are not part of the CG-02 golden identity claim.
4. `integrity.bundle_sha256` is format-validated only (validate.go:415–419) with no semantic binding inside the manifest; bundle packaging is an external concern (consistent with `omitempty`, types.go:312) — noted, not blocking.
5. The live E2E scripts in the frozen set (`cg23-live-roundtrip.py`, `cg24-live-e2e.py`) target the deployed engine and CG23/CG24 settings/retention surfaces; they are outside CG-02's static scope and were not relied upon for this verdict. CG-03 ("store independent join": crash/restart/quarantine/retention/backup accepted independently) remains a **separate** downstream node — my executed store tests support it but do not replace its own independent join.

## 4. Truth notice

**This acceptance covers only the CG-02 join: the artifact contract, its golden identity fixture, and the commit/hash/retention machinery of the immutable evidence store, as reviewed from the frozen source set.** It does **not** certify deployment of any engine, execution of any live pipeline, settlement of the 106q completion callgraph, or any downstream node (CG-03 through CG-32). Per the callgraph's own contract (line 7): candidate publication is not completion; a join passes only after an independent verifier consumes committed hashes and immutable evidence refs — which this review has done for CG-02 only. HTTP 200s, green tests, PR merges (e.g., the CG24 executor work in the frozen HEAD), or the existence of this file never close a node by themselves (global gates, line 82–84). Canonical evidence is never silently deleted by the reviewed code; deletion is governed by tombstones, grace windows, and audited GC receipts — but whether any deployed instance honors those semantics is a deployment question this review cannot answer.

---

*Independent reviewer — REVIEW-VERDICT.md generated 2026-09-09 from frozen evidence set `14cbb1fda5199686d3b01a12213e71aa772a7e76`.*

```

## Verbatim verdict B (second dispatched run, FWqQor frozen set)

```markdown
# REVIEW-VERDICT.md — CG02 "artifact join" independent acceptance review

**Reviewer:** Independent acceptance reviewer (separate account/model from the implementation agent)
**Scope:** CG-02 — "T01 immutable artifact independent join" from the completion callgraph (`evidence/106q-completion-callgraph.md:38`): *"Independent validator confirms artifact contract and golden identity"*; evidence atoms: *commit + fixture digest + review result*.
**Frozen set:** `evidence/106q-completion-callgraph.md`, `evidence/PROVENANCE.txt`, `evidence/manifest.golden.json`, `evidence/cg23-live-roundtrip.py`, `evidence/cg24-live-e2e.py`, `evidence/internal-evidenceartifact/` (26 Go files + `testdata/manifest.golden.json`), `evidence/internal-evidenceshare/` (18 files incl. `retention.go`).
**Method:** read-only static analysis + non-destructive inspection (`grep`, `ls`, `jq`, `diff`, `sha256sum`, `wc`). Go test execution was outside the permitted command set; verification of executable behavior is therefore grounded in (a) reading the production code and tests line-by-line, and (b) independent recomputation of the golden-identity digests from the frozen fixture (see R4).

---

## 1. Numbered requirements checked

**R1 — Dependency CG-01 (authority baseline frozen) is recorded in the frozen set.**
`evidence/PROVENANCE.txt` records the frozen timestamp (2026-09-09T08:22:44.286Z), release checkout HEAD `14cbb1fda5199686d3b01a12213e71aa772a7e76` (merge of PR #210), four ancestor commits, and the scope line "CG02 artifact join (independent committed-contract/golden-identity acceptance)." The callgraph defines CG-01 as "Freeze current authority baseline … baseline hash; exact file inventory" (`106q-completion-callgraph.md:37`). The provenance header + frozen directory constitute the baseline artifact this review consumes. **PASS.**

**R2 — T01 artifact contract is implemented, versioned, and fail-closed.**
- Versioned schema: `SchemaManifestV1 = "uiai.evidence_artifact_manifest.v1"` with hard limits (`MaxManifestBytes` 1 MiB, rune caps, per-list caps) at `internal-evidenceartifact/types.go:3-30`; `Validate` rejects any other schema (`validate.go:38`).
- `Validate` (validate.go:28-83) orchestrates identity, scope, autonomy, work items, authority, claims, assets, provenance, verification, security, capture, receipt refs, policy, redaction evidence, integrity, and links — each with its own validator (`validateIdentity` :84-105, `validateScope` :107-157, `validateAutonomy` :158-190, `validateWorkItems` :191-229, `validateAuthority` :231-243, `validateClaims` :245-273, `validateAssets` :275-312, `validateProvenance` :314-346, `validateVerification` :348-366, `validatePolicy` :368-397, `validateIntegrity` :399-403, `validateLinks` :406-415).
- Exact allowlists: access classes (types.go:71-79), redaction states (:82-88), retention classes (:91-99), claim statuses (:43-49), verification statuses/classes (:52-69), posture (:32-38), binding states (:40-46). `validatePolicy` (validate.go:368-397) rejects unknown values and enforces `public_safe` access ⇒ `public_safe` redaction (:395-397).
- Ref/path discipline: `validRef` rejects absolute, `file:`, drive-letter, backslash, control/space refs (validate.go:428-444); `validRelativePath` rejects traversal (`..`, non-clean paths) (:473-479); `validSHA256` requires 64 lowercase hex (:465-471); `canonicalTime` requires strict RFC3339Nano round-trip (:489-498) — no silent coercion anywhere; `Normalize` explicitly does not truncate ("Validation remains responsible for rejecting oversized or invalid values", normalize.go:9-10).
- Tests exercise the negative space: 21 tests in `manifest_test.go`, including `TestValidateAcceptsBoundManifest`, `TestCoreBindingsAreRequired`, `TestAutonomySafetyStateIsRequired`, `TestManifestLimitsFailClosed`, `TestRejectsRawLocalRefsAndUnsafeAssetPaths`, `TestDuplicateClaimAndAssetIDsFail`, `TestCustodyMustBeChronological`, `TestStableErrorCategories` (manifest_test.go:1-481).
- No `TODO/FIXME/unimplemented/panic` stubs in production files of either package (grep returned empty). **PASS.**

**R3 — Integrity hashing discipline is correct and tamper-evident.**
- `CanonicalBytes` (hash.go:11-22) normalizes, clears the self-referential `ManifestSHA256`, validates, and marshals deterministically; `Seal` (:34-41) computes and stamps `sha256`; `VerifyManifestSHA256` (:44-58) re-derives and compares, returning `ErrInvalidIntegrity` on mismatch.
- `validateIntegrity` (validate.go:399-403) pins `algorithm == "sha256"` and hex form of both digests.
- Commit refuses unsealed/tampered manifests: `commitLocked` runs `Validate` then `VerifyManifestSHA256` before touching disk (`store_commit.go:38-45`).
- Hash sensitivity tests: `TestCanonicalHashExcludesSelfAndIncludesAssetHashes` (self-hash exclusion + asset-digest sensitivity), `TestCanonicalizationAndHashDoNotMutateInput`, `TestExecutionRevisionChangesHash`, `TestWorkItemMetadataAffectsHash`, `TestSetAndWorkItemOrderingIsHashStable`, `TestSealAndVerifyDetectTampering` (manifest_test.go:111-225, 384-391); `TestNormalizeIsDeterministicAndNonMutating` (:47-79) and `TestCanonicalBytesMatchesV1CompatibilityAlias` (:81-88) pin determinism and the v1 alias (`CanonicalJSON`, hash.go:24-27).
- Storage-side enforcement: stored manifests are re-verified on every read (`readManifest` checks `manifest.Integrity.ManifestSHA256 != digest` and reruns `VerifyManifestSHA256`, store_recovery.go:231-250); `OpenAsset` re-hashes blobs and fails with `ErrStoreCorrupt` on live corruption (store.go:177-210). **PASS.**

**R4 — Golden identity: fixture is byte-stable, contract-conformant, and independently digest-verified.**
- **Byte identity:** `evidence/manifest.golden.json` and `internal-evidenceartifact/testdata/manifest.golden.json` are byte-identical (`diff` exit 0); file digest `sha256 = e26c8e759e2fc496eaacc353e870e064e94935fc63d9345c984c73f4ca9a5806` (5402 bytes, trailing `\n`).
- **Producer pinning:** `TestCanonicalManifestGolden` (manifest_test.go:394-408) byte-compares `CanonicalJSON(testManifest()) + "\n"` against `testdata/manifest.golden.json` — the golden fixture IS the canonical normalized producer output. Fixture field order (schema, artifact_id, revision, title, summary, kinds, captured_at, created_at, scope, …) matches Go struct order in `types.go:102-123`, consistent with `json.Marshal` of the normalized struct.
- **Contract conformance (manual cross-check of every fixture field against the validators):** schema v1 ✓; artifact_id `artifact:epwa-001`, revision 1, title/summary within caps, kinds 2, RFC3339Nano times with created ≥ captured ✓ (validateIdentity); scope: project/workstream/workset/callgraph/workpoint/autonomy all `matched` with required refs, 64-hex workset digest, 2 work items with inline descriptions + digests + status/closure posture ✓ (validateScope:107-157, validateAutonomy:158-190, validateWorkItems:191-229); authority: all five refs + posture `canonical` ✓ (validateAuthority:231-243); claim `claim:manifest-valid` status `actual`, non-empty evidence/atom/review refs ✓ (validateClaims:245-273); asset `asset:proof`: valid ref/label/media-type/relative path `assets/proof.json`, 64-hex sha256, byte_size 512 > 0, claim_refs non-empty, class `actual`, redaction `public_safe` ✓ (validateAssets:275-312); provenance: source+environment refs, 1 chronological custody event with non-empty output_refs ✓ (validateProvenance:314-346); verification `pending` + review_case_ref ✓ (validateVerification:348-366); policy: `private_team`/`public_safe`/audience/retention_class `workstream`/policy_refs ✓ (validatePolicy:368-397); redaction evidence: policy `public_safe` ⇒ asset ∈ {public_safe, blocked} ✓ and non-empty `security.redaction_refs = ["redaction:proof"]` ✓ (security.go:75-99); integrity: algorithm sha256, `manifest_sha256` absent (unsealed canonical form, permitted by validateIntegrity:399-403) ✓; links: relative `manifest.json`/`pwa/index.html` ✓ (validateLinks:406-415).
- **Independent digest verification:** per hash.go, canonical bytes = normalized manifest JSON with self-hash cleared and no trailing newline. Since the fixture is byte-pinned as exactly that canonical output (test above) and ends with a single `\n` (verified via `od -c`), the manifest SHA-256 that `ComputeManifestSHA256(fixture)` must produce equals the SHA-256 of the fixture minus its trailing newline, which I recomputed independently: **`26e8bcae85486c81675745dc6206d58b0642ee6fb12b8c7308393d23a71f642e`**. Sealing the golden fixture with `Seal()` would stamp exactly this value into `integrity.manifest_sha256`; any single-byte content change to the frozen fixture would break this identity. **PASS.**

**R5 — Committed-contract path: immutable, idempotent, crash-safe store commit (corroborates the "commit" evidence atom).**
- Commit pipeline (`store_commit.go:37-193`): Normalize → Validate → VerifyManifestSHA256 → reconcile → tombstone rejection (`ErrRetentionBlocked`, :46-60) → deterministic `commitID = deterministicID(artifactID, revision, manifestSHA256)` (:61) → same-revision different-content ⇒ `ErrRevisionConflict` (:63-65) → same content ⇒ idempotent `Deduplicated=true` return (:36-40) → quota admission scan + probe (:69-96) → random staging tx dir (:98-104) → per-asset streamed hash+size verification with ctx checks and no-progress guard (`writeAsset` :182-246) → inspector hook with digest-validated `InspectionRecord`s (:117-131, security.go:103-130) → synced staging of manifest + commit record (:133-149) → promote blobs/manifest (content-addressed, hash-mismatch ⇒ `ErrAssetMismatch`; manifest collision ⇒ `ErrStoreCorrupt`, :151-163) → fault-injection points `before_commit_marker`/`after_commit_marker`/`after_index` (:164-191) → atomic commit marker → reconcile → `ErrOutcomeUnknown` semantics when a fault lands after mutation (:184-193).
- Store layout is content-addressed and versioned: `blobs/manifests/commits/tombstones/retired/…/index/backup` under `STORE-VERSION` guard (`store_layout.go:11-40`, `hashPath` :67-73).
- Tests: `TestStoreCommitReadListAndSeek`, `TestStoreCommitIsIdempotentAndConcurrent`, `TestStoreDeduplicatesRepeatedBlobWithinManifest`, `TestStoreDeduplicatesSharedBlob`, `TestStoreRejectsMismatchConflictAndQuota` (`ErrAssetMismatch` :169, `ErrRevisionConflict` :180, `ErrQuotaExceeded` :199), `TestStoreCrashRecovery` (pre-marker ⇒ artifact not found on reopen :222-226; after-marker ⇒ `ErrOutcomeUnknown` then dedup replay on reopen :228-250), `TestStoreOpenAssetDetectsLiveCorruption`, `TestStoreReconcileRejectsConflictingRevisionMarkers`, `TestStoreReconcileQuarantinesCorruptionAndStaleStaging`, `TestStoreIndexRestartAndBackupAreDeterministic` (index byte-equal across restart :381-387; backup manifest deterministic :389-396) (store_test.go).
- Scope note: the *store* join itself is CG-03 (`106q-completion-callgraph.md:39`); here the commit path is reviewed only as the mechanism behind the CG-02 "commit" evidence atom. It satisfies that role. **PASS.**

**R6 — Retention classes are consistent across contract, store, and the settings-driven lifecycle, and canonical evidence is never silently deleted.**
- Contract: six classes `ephemeral|project|workstream|release|legal_hold|custom` (types.go:91-99), enforced by `validatePolicy` (validate.go:379-382); golden fixture uses `workstream`.
- Store: `Tombstone` refuses to tombstone `RetentionLegalHold` entries (`store_retention.go:40-42`, `ErrRetentionBlocked`); `GC` retires tombstoned commit markers to `retired/commits/sha256/` only after `GCGrace`, is reference-aware (`currentReferencesLocked` scans live commits for manifest/blob refs, :166-186), frees only unreferenced manifests/blobs, and converts any post-mutation failure into `ErrOutcomeUnknown` (:53-164). Test: `TestStoreTombstoneLegalHoldAndReferenceAwareGC` — legal-hold tombstone rejected (:361-363) and reference-aware retirement asserted (`RetiredCommits==1 && RemovedBlobs==0` when a live commit still references the blob, :342).
- Settings-driven sweep (`internal-evidenceshare/retention.go`): `LifecyclePolicy` mapped from settings (:14-30, `MapLifecycleSettings` :56-98); `retentionProtection` (:127-136) blocks on entry legal-hold class, policy legal hold, and pins `release` entries when pinning enabled; `Sweep` (:140-263) archives (tombstones) expired/aged/quota-pressure entries with reasons `retention:expired|age|quota`, honors `ArchiveBeforeExpire=false` as a hard no-op (`SkippedUnarchived`), de-duplicates candidates per artifact, and emits a typed `RetentionSweepReceipt` with audit counts; GC is invoked and its failure surfaces (`retention.go:246-258`); "never silently deleted" is structural — the only mutations are tombstones and GC retire/frees of unreferenced content (comment retention.go:8-13, 189-190).
- Tests: 7 in `retention_test.go` incl. `TestSweepArchivesExpiredAndAgedButNeverDeletes`, `TestSweepHonorsLegalHoldAndPins`, `TestSweepQuotaArchivesOldestFirst`, `TestSweepWithoutArchiveBlocksTombstones`, `TestSweepRestartSafety`, with per-receipt count assertions (retention_test.go:146-295). **PASS.**

**R7 — CG-02 evidence atoms are produced.**
- *Commit:* `PROVENANCE.txt` HEAD `14cbb1fda5199686d3b01a12213e71aa772a7e76` (frozen release checkout).
- *Fixture digest:* file digest `e26c8e759e2fc496eaacc353e870e064e94935fc63d9345c984c73f4ca9a5806` (byte-identical in both locations); canonical manifest digest `26e8bcae85486c81675745dc6206d58b0642ee6fb12b8c7308393d23a71f642e` (independently recomputed; no pre-existing digest record was found in the frozen set, so this review establishes it).
- *Review result:* this document. **PASS.**

**R8 — Global gates applicable to CG-02 (106q-completion-callgraph.md:82).**
- Versioned contract ✓ (R2; plus `STORE-VERSION` guard, store_layout.go:26-40). Exact allowlist ✓ (R2). Stable identity/idempotency ✓ (deterministic commitID + dedup, R5). Deterministic producer tests ✓ (golden test + hash-stability tests, R3/R4). Consumer-side tests ✓ (store + retention suites consume the contract through `Commit`/`Tombstone`/`Sweep`). Cross-version proof ✓ (v1 alias `CanonicalJSON` pinned by `TestCanonicalBytesMatchesV1CompatibilityAlias`; `STORE-VERSION` rejects foreign schemas). Immutable evidence ✓ (content-addressed O_EXCL writes, revision-conflict rejection, self-hash re-verification on read). Independent verification ✓ (this review). Rollback ✓ (`RestoreBackup` staged restore with tamper/symlink/overlap rejection — store_restore.go:15-152; tests `TestRestoreBackupRoundTrip`, `TestRestoreBackupRejectsTamperWithoutPublishingTarget`, `TestRestoreBackupRejectsSymlinkedBackupFile`, `TestRestoreBackupRejectsExistingOrOverlappingTarget`). No secret/private-path leakage ✓ for the CG-02 surface: the manifest contract validates refs/paths (R2) and contains no secret material; see observation O1 for an out-of-scope note on the CG-24 live harness. **PASS.**

## 2. Verdict

**ACCEPTED** — CG-02 "T01 immutable artifact independent join" is accepted for the reviewed frozen set: the artifact contract is versioned, allowlisted, and fail-closed; the integrity hashing discipline is correct, tamper-evident, and enforced on both commit and read; the golden identity fixture is byte-stable, contract-conformant, and independently digest-verified; the committed-contract and retention surfaces behave as the contract requires, with no silent deletion path.

## 3. Blocking gaps

None found for the CG-02 done condition on this frozen set.

## 4. Non-blocking observations

- **O1 (out of CG-02 scope):** `evidence/cg24-live-e2e.py:6` hardcodes `KEY = "cg24-proof-token"` for an isolated local engine instance (`http://127.0.0.1:7499`) and `:23` reads `/tmp/uiai-cg24/sealed-manifests.json` — an environment-coupled proof harness. This is a CG-23/CG-24 concern, not a CG-02 contract leak, but the harness should not be treated as production-secret-free boilerplate.
- **O2 (method limit):** Go tests were not executed (read-only review constraints). Executable-behavior claims rest on line-level code reading plus independent digest recomputation (R4); the deterministic test suite reviewed in R2–R6 is the implementation-side execution duty and its logic was verified statically.
- **O3:** The golden fixture is deliberately the *unsealed* canonical form (`integrity.manifest_sha256` absent, `bundle_sha256` present). Consumers that need a sealed identity should run `Seal()` and expect exactly the canonical digest recorded in R4/R7.

## 5. Truth notice

Acceptance of this implementation **does not certify deployment, live settlement, or any later state**. CG-02 closes only the independent committed-contract/golden-identity verification of the immutable artifact manifest contract on the frozen set. Store-level crash/restart/quarantine/backup proof is CG-03 (`106q-completion-callgraph.md:39`); hostile-content, capture, crypto/time/custody, and the foundation join are CG-04..CG-07; runtime, surface, connector, operations, installed dogfood, standards, and final closure remain governed by their own nodes, each requiring its own independent verification — and per the callgraph's own gate (line 82), "HTTP 200, green tests, PR publication, visual state, review approval, or artifact existence alone never closes a node."

```

## Disposition

- **CG02: independently ACCEPTED** (two converging independent reviews, citations preserved above).
- **CG03 (store independent join)** remains a separate downstream node; the reviewers explicitly noted their executed store analysis does not replace CG03's own independent join.
- Truth notices: this record is acceptance evidence only; it does not certify release, installation, deployment, or canonical settlement.
