# 106u — UIAI Evidence PWA: CG03 Independent Acceptance Record

- **Date:** 2026-09-09
- **Node:** CG-03 store independent join (crash/restart/quarantine/retention/backup; power-loss, restore, GC receipts)
- **Reviewer authority:** operator-directed independent LLM review, account `openai-codex-account-2` (model `gpt-6-astra`), separate from the implementation agent.
- **Frozen set:** release checkout `HEAD 09a662a91970a39da9fd25e7c74b199f105f7a23`; `docs/106q` callgraph; `internal/evidenceartifact` store surfaces (store, commit, recovery, retention, layout, backup, restore) plus tests and golden fixture; CG24 live e2e script. Snapshot `/tmp/epwa-review-cg03-iu7wjv`.
- **Outcome: ACCEPTED — 7/7 requirements PASS, zero blocking gaps.** Crash safety (durable staging→commit rename, fsync discipline), restart reconciliation, quarantine aging, retention/tombstone/GC receipts, backup/restore verification, and power-loss receipt semantics all verified with file/line citations. No resurrection risk, silent canonical deletion, tombstone loss, or false-success receipts found.
- **Non-blocking observations (reviewer):** durablefile primitive internals outside frozen set; `GCResult.RemovedManifests` path lacks a dedicated test assertion; O(store) reconciliation cost per commit is a throughput (not correctness) concern; zero-byte backup-verification edge is unproducible in tested flows; degraded-health during fresh staging is conservative and deliberate.
- **Truth notice (reviewer):** acceptance covers the store-surface contract within the frozen evidence set only; it does not certify deployment, settlement, or live execution of the e2e script (read, not run).

## Verbatim verdict

```markdown
# REVIEW-VERDICT — CG03 "store independent join" (EPWA)

- Reviewer: independent acceptance reviewer (account/model deliberately separate from the implementation agent).
- Review date: 2026-09-09 (analysis session).
- Frozen evidence set: `evidence/` per `PROVENANCE.txt` — frozen at 2026-09-09T09:23:05.432Z, release checkout HEAD `09a662a91970a39da9fd25e7c74b199f105f7a23` ("docs(epwa): record converging CG02 independent acceptance…"), scope line: "CG03 store independent join (crash/restart/quarantine/retention/backup power-loss/restore/GC receipts)".
- Mode: read-only inspection of the frozen set only (grep/ls/sed/jq-class commands). The live e2e script was read, not executed. Nothing in the CG02 acceptance was taken on trust; the store surface was verified directly from source and tests in the frozen set.

## Contract under review

From `evidence/106q-completion-callgraph.md`, node table line 39:

> | CG-03 — T02 store independent join | CG-02 | UIAI verifier | crash/restart/quarantine/retention/backup proof independently accepted | power-loss, restore, GC receipts |

Done-condition (verified): "crash/restart/quarantine/retention/backup proof independently accepted"; evidence atoms: "power-loss, restore, GC receipts". Global gates (same file, lines 80–82) apply: deterministic producer tests, immutable evidence, independent verification; "HTTP 200, green tests, PR publication, visual state, review approval, or artifact existence alone never closes a node."

Downstream context: CG-24 (lifecycle/retention executor, line 60) consumes these store semantics; the provenance notes CG24's settings-driven sweep route is merged. CG-03 is judged on the store surface itself.

## Numbered requirements and findings

### R1 — Crash safety of the commit path — PASS

Implementation (`evidence/internal-evidenceartifact/store_commit.go`):
- Commit is staged under `root/staging/<random txID>/` (line ~86), never written directly into canonical layout. Staged assets stream through `writeAsset` (lines 194–250) with `O_CREATE|O_EXCL`, incremental SHA-256 and byte-size enforcement against the manifest (`written != asset.ByteSize || hex… != asset.SHA256 → ErrAssetMismatch`), then `file.Sync()` + `file.Close()` before any promotion; failed writes remove the staged file.
- Staged `manifest.json` and `commit.json` via `writeSyncedFile` (fsync + close), followed by `syncDir(staging)` (lines 134, 153–156).
- Promotion via `promoteFile` (`store_layout.go:146–180`): content-addressed, idempotent; if target exists it re-hashes (size+SHA256) and discards the staged copy; otherwise `durablefile.Rename(staged, target)`, `chmod 0o640`, `syncDir(target dir)`. A lost rename race re-enters and re-verifies (lines 158–161).
- Single visibility point: the durable commit marker `commits/sha256/<commitID>.json` written by `writeAtomicFile` (`store_layout.go:112–139`) — temp file fsync → `durablefile.Rename` → `syncDir(dir)`. Before the marker there is no commit file, so nothing is visible after a crash; after the marker the commit replays. `CommitID` is deterministic: `deterministicID(artifactID, revision, manifestSHA256)` (`store_commit.go:32`), and re-commit of the same (artifact, revision, manifest hash) returns `Deduplicated: true` while a different manifest hash for the same revision returns `ErrRevisionConflict` (`store_commit.go:33–38`).
- Fault-injection seams at exactly the durability-relevant stages: `after_assets_staged` (line 104), `before_commit_marker` (158), `after_commit_marker` (164 → returns `commitResult` + `ErrOutcomeUnknown`), `after_index` (173 → same). `ErrOutcomeUnknown` is the honest "uncertain durability" receipt; it never fabricates success.
- `commitLocked` re-runs `reconcileLocked()` before every commit (line 23) and blocks re-commit of tombstoned revisions (`ErrRetentionBlocked`, lines 26–29).

Tests (`store_test.go`):
- `TestStoreCrashRecovery` (206–254): `before_marker_invisible` — fault at `before_commit_marker`, commit fails with `ErrStoreUnavailable`, reopen via `OpenStore(cfg)` → `GetManifest` = `ErrArtifactNotFound`, and `health.FreshStaging != 0` (leftover staging is counted, not hidden). `after_marker_replayed` — fault at `after_commit_marker` → `ErrOutcomeUnknown`, reopen → artifact present, re-commit → `Deduplicated: true`. Both crash windows are exercised and both resolve correctly on replay.
- `TestStoreCommitIsIdempotentAndConcurrent` (64–96): 12 concurrent workers, exactly one CommitID and one entry — idempotency under concurrency.
- `TestStoreRejectsMismatchConflictAndQuota` (164–204): mismatch/conflict/quota rejection paths.

Residual dependency: rename and directory-fsync primitives come from `github.com/WPUIAI/uiai-engine/internal/durablefile` (`durablefile.Rename`, `durablefile.SyncDirectory`). The call-site discipline (fsync file → rename → fsync dir) is fully verified here; the primitive's own implementation is outside the frozen set and could not be inspected. This is a trust boundary, not a defect found in the frozen set.

### R2 — Restart consistency via reconciliation on open — PASS

Implementation (`store.go:136–159`, `store_recovery.go`):
- `OpenStore` → `initializeLayout` (`store_layout.go:24–52`): creates the full layout including `tombstones/`, `retired/commits/`, `staging`, `quarantine`, `backup` (`storeDirs`, lines 13–23); `STORE-VERSION` gate rejects any foreign schema with `ErrStoreCorrupt` (lines 31–45) — an old/unknown store refuses to open rather than being silently migrated.
- `reconcileLocked` (`store_recovery.go:15–158`) on every open and before every commit:
  1. `reconcileStagingLocked` (284–313): staging entries older than `StagingQuarantineAge` are moved into quarantine; fresh staging counted in `health.FreshStaging` and marks health `degraded` (lines 147–150) — conservative, state is never silently dropped.
  2. `scanTombstonesLocked` (253–274): every tombstone file parsed and validated (`validTombstone`: schema, `TombstoneID == deterministicID(artifactID, revision)`, canonical `CreatedAt`, refs); corrupt tombstones quarantined.
  3. Per-commit deep validation `validateCommitFile` (170–224): schema/CommitID recomputation, canonical time, manifest re-read from disk + `VerifyManifestSHA256` (`readManifest`, 226–247), asset/inspection correspondence, **re-hash of every blob on disk** and **re-run of the asset inspection** compared to the stored inspection record. Corruption is detected on open, not on read.
  4. Duplicate `(artifact, revision)` groups → **all** conflicting candidates quarantined as `revision-conflict` (58–78); legal-hold tombstones quarantined (45–57); orphan tombstones (commit neither live nor retired) quarantined (107–123).
  5. Index rebuilt deterministically (sorted entries/tombstones) and written atomically to `index/index.v1.json` (136–142); `StoreHealth` exposes LiveArtifacts/TombstonedArtifacts/CorruptRecords/FreshStaging/Quarantined/StoredBytes/IndexGeneration.

Tests: `TestStoreReconcileRejectsConflictingRevisionMarkers` (270–285: hand-installed conflicting marker via `installCommitForTest`; both records quarantined, `List()` empty); `TestStoreReconcileQuarantinesCorruptionAndStaleStaging` (287–314: corrupted blob → commit quarantined and artifact not served; stale staging dir aged via `os.Chtimes` → quarantined; health `degraded`); `TestStoreOpenAssetDetectsLiveCorruption` (256–268); `TestStoreIndexRestartAndBackupAreDeterministic` (366–404: index bytes byte-identical across restart — restart determinism, not just correctness).

Live evidence: `evidence/cg24-live-e2e.py` lines 80–101 kills the real engine process (`pkill`), restarts it (`setsid /tmp/445-uiai-cg24-engine --config /tmp/uiai-cg24/config.yaml`), polls `/api/health` to 200, then asserts the surviving artifact still serves 200 and the tombstoned artifact still returns 404/410 ("tombstone persists after restart"). This exercises the OpenStore reconcile path against a real on-disk store behind a real HTTP instance — the strongest restart proof in the frozen set (see §Evidence classification for its status).

### R3 — Staging quarantine — PASS

- Quarantine mechanism `quarantinePath` (`store_layout.go:214–234`): `durablefile.Rename` of the offending file/dir into `quarantine/<timestamp>-<kind>-<digest12>-<nonce>/`, plus an atomically written `quarantine.json` metadata record (kind, digest, quarantined_at), then fsync of the source directory. Quarantine is content-preserving (move, not delete) and receipted.
- Aging: `reconcileStagingLocked` (`store_recovery.go:284–313`) uses `StagingQuarantineAge` against the entry mtime; fresh staging is surfaced in health rather than acted on.
- The commit path itself contributes to this discipline: a failed/uncertain commit leaves its staging dir (fault paths return before `os.RemoveAll(staging)`, `store_commit.go:176`), which is either quarantined after aging or cleaned on success (`RemoveAll`, line 176; removal failure → `ErrOutcomeUnknown`).
- Test: `TestStoreReconcileQuarantinesCorruptionAndStaleStaging` ages a staging dir past `StagingQuarantineAge` and asserts `health.Quarantined >= 2` (stale staging + quarantined corrupt commit) and `status == "degraded"`.

### R4 — Retention / tombstones / GC receipts — PASS

Implementation (`store_retention.go`):
- `Tombstone()` (13–63): input validation; **idempotent read-back** (existing valid tombstone returned unchanged, lines 20–27); `RetentionLegalHold` → `ErrRetentionBlocked` (46–48); tombstone record carries `CommitID`, `Reason`, `AuthorityRef`, canonical `CreatedAt`, `TombstoneID = deterministicID(artifactID, revision)`; written via `writeAtomicJSON` (atomic + fsync), then `reconcileLocked()`; reconcile failure → `ErrOutcomeUnknown`.
- Hide-at-tombstone, delete-at-grace: once tombstoned, `GetManifest` returns `ErrArtifactNotFound` immediately (index rebuilt without the entry), while the commit record physically survives until `GC()` retires it — no resurrection window.
- `GC()` (65–172): tombstones older than `GCGrace` (parsed via `canonicalTime`) are **retired, not deleted** — `durablefile.Rename(commits/<id>.json → retired/commits/<id>.json)` with fsync of both directories (87–105): the retired-commit layout is the durable proof-of-deletion, and the tombstone itself is retained as an audit receipt. Live references are then recomputed by `currentReferencesLocked` (175–194), which fully re-validates every remaining live commit; unreferenced manifests and blobs are removed with an fsync of the affected directory after each removal (106–164). Shared blobs are reference-aware and preserved. `mutated` flag: any error after the first mutation → `ErrOutcomeUnknown` (fail-closed uncertainty). Canonical evidence is never silently deleted: deletion is always tombstone → grace → retire (record kept in `retired/commits/`) → unreferenced-data sweep.
- Receipts: `GCResult{RetiredCommits, RemovedManifests, RemovedBlobs, FreedBytes}` (`store.go:121–126`); `Tombstone` record is the deletion receipt; both are returned to callers.

Tests: `TestStoreTombstoneLegalHoldAndReferenceAwareGC` (316–364) verifies tombstone idempotency, immediate hiding, `clock.Add(2 * GCGrace)` → first GC `RetiredCommits == 1, RemovedBlobs == 0` with the **shared blob still present** (reference-awareness, lines 340–343), second tombstone + 2×GCGrace → `RemovedBlobs == 1`, and legal-hold tombstone → `ErrRetentionBlocked`. Live: `cg24-live-e2e.py` asserts a `uiai.evidence_retention_sweep.v1` sweep receipt with `expired_tombstoned`, `quota_tombstoned`, `remaining`, and a non-null `gc` object on the real engine (lines 58–71).

### R5 — Backup — PASS

Implementation (`store_backup.go`):
- `CreateBackupManifest` (26–67): inventories an allowlisted scope only — `STORE-VERSION`, `blobs/sha256`, `manifests/sha256`, `commits/sha256`, `tombstones/sha256`, `retired/commits/sha256` (`backupRoots`, 108–116). Staging, quarantine, index, and prior backups are excluded as derived/ephemeral state — correct: quarantined material and the index are reconstructible and must not be silently canonized. Every file is hashed (SHA-256 + size), paths validated against `validBackupPath` (120–133), records sorted, self-hash computed over the manifest with `ManifestSHA256` blanked (`backupHash`, 136–142), manifest written atomically to `backup/<digest>.json`. `StoreGeneration` binds the manifest to the reconciled index generation.
- `VerifyBackupManifest` (69–103): recomputes the self-hash (manifest tamper detection), enforces schema, strict path allowlist, strict ascending order, duplicate rejection, then **re-hashes every file at its expected path/size** (content tamper detection). Fails closed with `ErrBackupInvalid`.

Tests: `TestStoreIndexRestartAndBackupAreDeterministic` (366–404) asserts two consecutive backup manifests are `reflect.DeepEqual` (deterministic), verification passes, tampering a blob file → `ErrBackupInvalid`, and tampering a manifest field (`Files[0].ByteSize++`) → `ErrBackupInvalid` — both tamper classes covered.

### R6 — Restore — PASS

Implementation (`store_restore.go`):
- `RestoreBackup` (18–85): config validated first; refuses if the target root already exists (never merges into a live store); refuses overlapping source/target roots in both directions with `EvalSymlinks` on the parent (30–50); verifies the manifest against the **source** (`verifyBackupManifestAt(sourceRoot, manifest)`, line 54); copies every file into a private staging dir with per-file fsync + re-hash + per-directory fsync (`restoreBackupFile`, 87–122, rejecting non-regular files and symlink escapes via `Lstat` + `EvalSymlinks` containment); **re-verifies the manifest against the staged copy** (line 66); then opens the staged copy as a full store (`OpenStore` with `Root = stage`, 69–74) — running the entire deep reconciliation, including blob re-hashing and inspection re-runs; `health.IndexGeneration` must equal `manifest.StoreGeneration` (75–77); only then `syncDir(stage)` → `durablefile.Rename(stage, targetRoot)` → `syncDir(parent)` (79–90). Any failure removes the staging dir and never publishes the target.
- This is a two-phase-verified, generation-checked, atomically promoted restore — the restore receipt is the returned `StoreHealth` bound to the backup's `StoreGeneration`.

Tests (`store_restore_test.go`): `TestRestoreBackupRoundTrip` (13–46: manifest digest equality, asset byte equality, generation match); `TestRestoreBackupRejectsTamperWithoutPublishingTarget` (48–70: asserts target does **not** exist after failed verification); `TestRestoreBackupRejectsSymlinkedBackupFile` (72–105: symlink substitution in the backup scope rejected, target not published); `TestRestoreBackupRejectsInvalidConfigWithoutPublishingTarget` (107–123); `TestRestoreBackupRejectsExistingOrOverlappingTarget` (125–142: existing and nested targets).

### R7 — Power-loss durability — PASS (semantics proven; physical claim bounded)

- Mechanisms: every durable state transition uses fsync(file) → atomic rename → fsync(parent dir) (`writeAtomicFile`, `store_layout.go:112–139`; `promoteFile`, 146–180; retire, `store_retention.go:87–105`; quarantine, `store_layout.go:214–234`), with per-file fsync in staged writes (`writeAsset`, `writeSyncedFile`) and per-directory fsync after GC removals. The commit marker is the single atomic visibility point; `ErrOutcomeUnknown` guarantees the caller is never told "committed" when the outcome is uncertain — this is the power-loss receipt contract (uncertainty is surfaced, not laundered).
- Evidence: fault injection at `before_commit_marker` / `after_commit_marker` / `after_index` covers the crash windows that matter for persistence (`TestStoreCrashRecovery`). No true power-loss test (page-cache drop / simulated device loss) exists in the frozen set; physical durability therefore rests on the verified call-site fsync discipline plus the external `durablefile` primitives (not in the frozen set). The live restart in `cg24-live-e2e.py` is a process kill/restart on a real filesystem — stronger than fault injection, but still not a power-loss simulation. Assessed: PASS on the contract's "power-loss receipts" atom (the receipt mechanism exists and is tested); the physical durability claim is bounded by this residual.

## Evidence classification — producer tests vs live executable evidence

1. **Deterministic producer tests (source in frozen set; unit-scoped, clock-injected via `testClock`)**: `store_test.go` — `TestStoreCommitReadListAndSeek`, `TestStoreCommitIsIdempotentAndConcurrent`, `TestStoreDeduplicatesRepeatedBlobWithinManifest`, `TestStoreQuotaCountsOnlyUniqueBlobBytes`, `TestStoreDeduplicatesSharedBlob`, `TestStoreRejectsMismatchConflictAndQuota`, `TestStoreCrashRecovery`, `TestStoreOpenAssetDetectsLiveCorruption`, `TestStoreReconcileRejectsConflictingRevisionMarkers`, `TestStoreReconcileQuarantinesCorruptionAndStaleStaging`, `TestStoreTombstoneLegalHoldAndReferenceAwareGC`, `TestStoreIndexRestartAndBackupAreDeterministic`; `store_restore_test.go` — 5 restore tests listed under R6. These are producer-side; the frozen set contains the test source but **no recorded run log/receipt**, so I verified the assertions against the implementation rather than against a green-run artifact.
2. **Live executable evidence**: `evidence/cg24-live-e2e.py` — present in the frozen set as an executable script against a real HTTP engine (`/api/evidence/artifacts/manifest`, `/api/screenshot/settings/retention`, `/api/health`) with a real process restart (lines 80–101) proving tombstone persistence and survivor serving across restart, plus a `uiai.evidence_retention_sweep.v1` GC/sweep receipt. I verified the script's assertions match the store semantics reviewed above; I did not execute it (read-only review), and **no captured run output/receipt from that script is included in the frozen set**. It is executable evidence, not a frozen run record.
3. Naming discrepancy in the review request: the frozen set contains **no `store_retention_test.go`**. Retention/tombstone/GC coverage lives in `store_test.go` (`TestStoreTombstoneLegalHoldAndReferenceAwareGC`). This does not create a coverage gap — the required behaviors are tested — but the request's file list does not match the frozen inventory exactly, and this review followed the frozen set.

## Non-blocking observations

1. `durablefile.Rename` / `durablefile.SyncDirectory` (uiai-engine internal package) are load-bearing for crash safety and are outside the frozen set; call-site discipline is verified, primitive internals are not.
2. No test asserts `GCResult.RemovedManifests > 0` (manifest GC path) or the post-mutation `ErrOutcomeUnknown` branch of `GC()`; both paths are implemented (store_retention.go:106–124, 96–102) but only partially exercised.
3. Reconciliation re-hashes every blob and re-runs inspections on every `OpenStore` and every `Commit` — strong integrity, O(store) cost per commit; a throughput concern, not a correctness one.
4. `verifyBackupManifestAt` rejects `ByteSize <= 0` records; a hypothetical zero-byte file inside the backup scope would fail verification. No such file is producible in the tested flows (STORE-VERSION is 17 bytes; committed assets are size-verified).
5. Health is marked `degraded` whenever fresh staging exists (store_recovery.go:147–150); conservative and deliberate, but operators will see degraded health during normal long-running commits.

## Blocking gaps

None found. Every CG-03 done-condition — crash safety, restart consistency, staging quarantine, retention/tombstone/GC receipts, backup, restore, power-loss receipt semantics — is implemented with specific, auditable mechanisms and backed by producer tests in the frozen set, with restart/tombstone/GC behavior additionally covered by the live e2e script. No evidence of resurrection risk, silent deletion of canonical evidence, tombstone loss across restart, or false-success receipts was found in the frozen set.

## Truth notice

This verdict is an independent acceptance of the **store-surface contract for CG03 within the frozen evidence set only**. It does **not** certify: deployment of any binary or environment; live settlement of the EPWA release; execution of `cg24-live-e2e.py` (the script was read, not run, and no run receipt exists in the frozen set); CG-24 and later callgraph nodes; CG-02's own acceptance (re-derived here, not assumed); provider synchronization; or any operational/SLO property of a running engine. Green tests, this document, or artifact existence do not close the node per the callgraph's own global gates; independent verification is one input to settlement, not settlement itself.

— Independent acceptance reviewer, CG03.

```

## Disposition

- **CG03: independently ACCEPTED** (citations preserved above). CG04 (hostile-content join) is the next downstream node and remains its own join.
- Reviewer non-blocking observation 2 (add `RemovedManifests` test assertion) is recorded as follow-up test hardening, not a gap.
- This record is acceptance evidence only; it does not certify release, installation, deployment, or canonical settlement.
