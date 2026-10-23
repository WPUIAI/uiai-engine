> Parent authority: https://github.com/WPUIAI/uiai-engine/issues/106
> Canonical source: https://github.com/WPUIAI/uiai-engine/issues/106#issuecomment-5462658761

## EPWA implementation decomposition — 15 dependency-ordered packets

Parent: #106
Baseline: `653eb32d6f59981cf3bddb6304d369a1e96b7baa` on `main`
Policy: preserve all unrelated dirty/untracked work; no release/deploy/install mutation; no Focusa completion authority in UIAI.

### Dependency DAG

```text
T01 contract core
├─ T02 durable store/index
│  ├─ T03 security/privacy boundary
│  ├─ T04 capture/media assembly
│  └─ T05 crypto/time/federation/custody
├─ T06 Judge View/verification ─┐
├─ T07 Action Deck/collaboration│
├─ T08 PWA/registry/accessibility│
│  └─ T09 distributions          │
│     └─ T10 work connectors     │
└─ T11 CLI/MCP/API/agent parity ─┤
T02–T11 → T12 operations/migration/release
T01–T12 + Focusa #263/#277/#278/#280/#281/#283/#290/#291/#293 → T13 Focusa closure integration
T13 → T14 installed E2E/dogfood
T14 → T15 standards/benchmark/external claim proof
```

### T01 — Immutable Evidence Artifact contract core — **IR5 executable**

Gap coverage: EPWA-GAP-001–007 foundation, 009–010 type foundation, 045–052 metadata foundation, 111–122 lineage foundation.

Exact files allowed:

- `internal/evidenceartifact/types.go`
- `internal/evidenceartifact/normalize.go`
- `internal/evidenceartifact/validate.go`
- `internal/evidenceartifact/hash.go`
- `internal/evidenceartifact/*_test.go`
- `internal/evidenceartifact/testdata/manifest.golden.json`

Exact output:

- UIAI-owned portable manifest constant `uiai.evidence_artifact_manifest.v1`.
- Focusa Project/Workstream/Workpoint/Evidence/Receipt/Completion refs remain opaque strings; UIAI does not accept or settle their authority.
- Typed identity, scope, authority posture, claims, assets, provenance, verification, receipts, policy, integrity, and links.
- No arbitrary executable HTML/JS or untyped action payload.
- `Normalize`, `Validate`, `CanonicalBytes`, and `ComputeManifestSHA256`.
- Deterministic trimming/sorting/deduplication for set-like refs; chronological sequences remain ordered.
- SHA-256 self-field excluded from hash input; asset hashes remain included.
- Strict size/count/string/path/hash/media/access/redaction/truth/verification enums and relative portable asset paths.
- No raw local path in public manifest fields.
- Golden deterministic fixture; mutation changes hash; ordering-only changes do not; invalid scope/path/hash/enum/duplicate IDs fail.

Stable initial errors:

- `ErrInvalidSchema`
- `ErrInvalidIdentity`
- `ErrInvalidScope`
- `ErrInvalidAuthority`
- `ErrInvalidClaim`
- `ErrInvalidAsset`
- `ErrInvalidPolicy`
- `ErrInvalidIntegrity`
- `ErrLimitExceeded`

Side effects: source files only; no storage/network/API/state mutation.
Rollback: remove the new isolated package.
Evidence: targeted Go package tests plus formatting/vetting in later canonical background gate.
Non-goals: route, storage, PWA, share token, Focusa write, judge execution, export, connector, release.

### T02 — Crash-safe artifact store/index/retention

Gap coverage: 004–005, 021–030, 135–137. Depends T01.

Freeze content-addressed asset storage, manifest revision store, index, staging/atomic commit, restart replay, corruption quarantine, quota admission, reference-aware GC, retention/tombstones, backup/restore, cache semantics, range reads, and storage health. Exact schema/migration/paths/errors/fixtures required before IR5.

### T03 — Security/privacy/hostile-content boundary

Gap coverage: 031–044, 125–129, 134, #107. Depends T01–T02.

Harden escaped rendering; CSP/CSRF/CORS/frame/referrer/proxy policy; SSRF; MIME/file/archive/SVG/PDF/Office/media sanitization; malware adapter; PII/secret/OCR/transcript/EXIF detection; irreversible redaction; channel disclosure; rate limits; service-worker security; audit integrity. Threat model and adversarial fixtures required before IR5.

### T04 — Capture/media/anti-curation assembly

Gap coverage: 045–052, 111–117, 126–129, 134. Depends T01–T03.

Assemble browser screenshots, diagnostics, action traces, responsive matrix, image/video/audio, environment identity, actual/surrogate/synthetic labels, contiguous event ranges, selection/omission manifest, anti-cherry-picking coverage, target/account identity, proof-of-absence posture, annotation geometry, comparison policy, transcripts/keyframes, cleanup ordering.

### T05 — Cryptographic identity/time/federation/custody

Gap coverage: 013–020, 111–125, 135–140. Depends T01–T03.

Define signatures/canonicalization/algorithm agility, instance keys/rotation/revocation, time confidence, self-host discovery, imports/mirrors/split-brain/source loss, chain of custody, trust levels, actor/delegation identity, policy precedence, public verification, corrections/retractions, air-gap and printed verification.

### T06 — Judge View and independent verification runtime

Gap coverage: 053–064, 113–117, 119–120, 131–133. Depends T01, T04, T05.

Freeze Judge View/request/result schemas, information-set hashing, citations, modality, calibration/golden sets, bias/order tests, model/provider drift, disagreement, budgets, isolation, high-consequence policy, appeals, freshness, independent ground-truth access, immutable verification results. UIAI emits proof inputs; Focusa #278/#277 retain verification/completion authority.

### T07 — Focusa Action Deck and collaboration

Gap coverage: 065–070, 121–122, 139. Depends T01, T05, T06 and approved Focusa operation registry.

Declarative actions only: inspect/link/capture/reproof/follow-up/adjudication/share/export. Freeze capability refresh, preview/confirmation, anti-replay, idempotency/reconciliation, partial effects, imported-action distrust, comments/annotations/suggestions, mutable review thread vs immutable artifact, notifications, bulk operations, follow-up traceability.

### T08 — Self-hosted Evidence PWA, registry, and accessibility

Gap coverage: 071–078, 124, 137–140. Depends T01–T05.

Build FPV-token-based semantic article; Overview/Evidence/Timeline/Inspect/Developer; registry/list/search/filter/collections; local/LAN/tailnet/private/unlisted/public-safe hosting; relative subpath-safe assets; PWA/offline/update; large-set virtualization; LowMem; localization/RTL; WCAG 2.2 AA; all degraded states.

### T09 — Print/PDF/email/slides/portable derivatives

Gap coverage: 079–085, 127–128, 135, 140. Depends T01–T05, T08.

Implement print CSS, formal PDF, email text/HTML, Markdown/rich text/HTML/JSON/CSV/archive, PPTX/self-contained HTML slides/presentation PDF. Freeze PDF/A/PDF-UA posture, viewer/client matrices, fonts/color, keyframes/transcripts, SMTP/delivery truth, archive safety, licensing, deterministic derivative IDs/hashes/Receipts.

### T10 — Composable work-software connectors

Gap coverage: 086–092. Depends T03, T07, T09.

Neutral block schema; destination adapters; exact workspace/channel authority; schema drift/degradation; idempotency/retry/rate/dead-letter; external retention/delete truth; webhook signatures; safe unfurl/embed. Initial proof: one issue tracker, document/wiki, and chat/card surface.

### T11 — CLI/MCP/REST/OpenAPI/Pi/LLM parity

Gap coverage: 093–100. Depends T01–T10 contract surfaces.

One generated contract; progressive discovery; content negotiation; field projection/cursors/ranges; async jobs/cancel/resume; binary-free ref-first defaults; stable citations; token-budget/omission truth; typed errors; credential safety; generated clients; cross-harness conformance.

### T12 — Operations, migration, reliability, and release

Gap coverage: 101–110, 130, 135–138. Depends T02–T11.

SLOs/telemetry/doctor/support; resource scheduling/backpressure; legacy `/api/share`, `/v/{token}`, screenshot and evidence-ref migration; compatibility matrix; threat/red-team/fuzz/property/chaos/power-loss/backup-restore/load tests; browser/document/email/viewer matrices; production-consistency five proofs; ownership/runbooks/release/rollback.

### T13 — Focusa completion and ecosystem integration

Gap coverage: 007–012, 053–070, 093–110, 139 plus Focusa #263/#277/#278/#280/#281/#283/#284/#290/#291/#293/#294/#295. Depends T01–T12 and approved Focusa IR5 packets.

Wire preferred verification artifact through Project/Workstream/Workpoint, Context Cognition, Workset/release, CallGraph, Silent Sessions, provider close, Desktop/Cockpit, agent final outputs, Acceptance Atoms, revalidation/reopen, and sole Completion Authority—without UIAI becoming a truth writer.

### T14 — Installed end-to-end dogfood and independent proof

Depends T13.

Dogfood real projects: publish → close/restart → inspect human/agent → multimodal judge → action/reproof → PDF/email/deck/connector → export/import second instance → revoke/expire → invalidation/reopen → Focusa settlement. Include failure/security/recovery, cross-version, consumer-side, independent verifier, and installed binaries.

### T15 — Standards, benchmarks, and public claim governance

Gap coverage: 141–150. Depends T14.

Standards applicability/conformance matrix; public corpus; competitive cohort and metrics; external security/privacy/accessibility/provenance/judge review; independent implementation interoperability; dated public claim evidence packet; challenge/correction/expiry governance. No superlative claim before this packet passes.

## Admission rule

Only T01 is IR5 now. T02–T15 are dependency packets and must each receive exact baseline, files, state transitions, operation/event/error schemas, fixtures, rollback, and evidence before source mutation. Passing T01 does not imply the Evidence PWA exists or that any gap beyond its listed foundation is complete.
