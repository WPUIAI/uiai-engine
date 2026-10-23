> Parent authority: https://github.com/WPUIAI/uiai-engine/issues/106
> Canonical source: https://github.com/WPUIAI/uiai-engine/issues/106#issuecomment-5462539189

## Architecture amendment — universal, Workstream-scoped Evidence PWA

Operator requirement: this must be a polished, extensive **Evidence PWA**, not a generic browser screenshot gallery. Every evidence-producing turn should return a formal, durable Evidence Artifact projection. Browser proof is the first rich module, not the only evidence kind.

### Authority and non-duplication

This issue owns UIAI's immutable artifact assembly, redaction, storage, portable rendering, publication, and browser-specific evidence modules. It must interoperate with—not duplicate—Focusa's canonical completion system:

- `Startempire-Wire/focusa#263`: cross-harness evidence/receipt/artifact capability family.
- `#277`: sole Completion Authority/reducer and terminal settlement.
- `#278`: Acceptance Atom and proof execution runtime.
- `#281` + `WPUIAI/uiai-engine#65`: independent UIAI browser/visual witness.
- `#283`: Completion Verification Cockpit/PWA and Focusa Desktop projection.
- `#291`: operator adjudication UX.
- `#294`: implementation-readiness gate for canonical Focusa contracts.

A UIAI artifact is a durable proof input and human-review surface. It never self-verifies, self-settles, or closes work.

### Required scope and authority binding

Every bundle and every visible PWA must bind to the exact applicable scope:

- project ref/id, canonical display name, and privacy-safe fingerprint;
- Workstream/continuity ref;
- Workpoint ref and applicable Work Item refs;
- trajectory/goal/acceptance-contract refs when present;
- producer actor/node/session/runtime refs;
- source authority and collection method;
- verifier refs and verification independence class;
- canonical Evidence refs, Receipt refs, Acceptance Atom refs, completion-decision ref, and settlement state;
- policy revisions controlling access, disclosure, redaction, retention, expiry, export, and publication.

Public/untrusted projections must never expose raw filesystem paths, private project roots, tokens, secrets, hidden browser state, or unredacted scope metadata. Local authorized Inspect/Developer projections may reveal additional details by policy.

### Versioned evidence bundle

Define one immutable bundle envelope with typed modules rather than one browser-only DTO. Proposed structure (exact schema IDs remain subject to the Focusa readiness/contract process):

1. `identity`: artifact id, schema/version, title, evidence kind(s), created/captured/published timestamps.
2. `scope`: project, Workstream, Workpoint, Work Items, trajectory, target/object refs.
3. `authority`: producer, source authority, evidence authority, verifier(s), closure authority, canonical/advisory/degraded posture.
4. `claims`: bounded claims/predicates, acceptance atoms, expected/observed results, pass/fail/blocked/indeterminate/disputed state.
5. `artifacts`: typed media/files with MIME, dimensions, byte size, capture method, SHA-256, redaction state, verification class, parent/source refs.
6. `browser`: URL/origin policy, viewport matrix, DOM/layout/accessibility measurements, screenshots, visual comparisons, console/network diagnostics, interaction/action trace, cleanup attestation.
7. `execution`: commands/tool operations by safe ref, runtime/engine/build identity, start/end/duration, retries, bounded outputs, exit/result state.
8. `provenance`: source lineage, transformations, annotations, dependency revisions, content-influence/untrusted-input posture.
9. `verification`: verifier identity/class, checks, contradictions, uncertainty, freshness/revalidation posture.
10. `receipts`: action, evidence-capture, publication, adjudication, completion, and settlement refs.
11. `policy`: classification, audience/access class, redaction status, retention/expiry/legal hold, export/share posture.
12. `integrity`: per-file hashes, normalized manifest hash, bundle hash, producer/renderer versions, supersedes/superseded-by lineage.
13. `links`: same-origin PWA, JSON manifest, bounded download/export, canonical Focusa inspect/rehydrate refs, related artifacts.

Large payloads remain ref-first. The manifest stays bounded and machine-readable.

### Automatic evidence-turn publication

All UIAI/MCP/Pi responses that produce `evidence_refs`, screenshots, diagnostics, verification results, or receipts must run the Evidence Artifact publisher **after response assembly and before session cleanup**.

Every response must include a formal publication result, even on failure:

- `published`: durable artifact ref + PWA URL + manifest ref + bundle hash;
- `blocked`: explicit policy/redaction/scope reason and recovery action;
- `failed`: typed failure + durable diagnostics/retry posture;
- `not_applicable`: only for genuinely non-evidentiary turns.

No silent omission. Idempotency must derive from normalized scope + sorted source refs + policy revision + renderer version. Republish with identical inputs returns the same artifact; changed inputs create a new immutable revision linked by supersession.

### Beautiful, portable PWA

Reuse the FPV PWA's canonical zero-build visual system and responsive tokens, replacing live-session controls with immutable evidence review:

- polished summary header with Project / Workstream / Workpoint scope and truth-state badge;
- progressive modes: Overview → Evidence → Timeline → Inspect → Developer;
- responsive proof matrix and side-by-side compare;
- evidence gallery with annotations, measurements, exact hashes, and provenance;
- acceptance-atom and contradiction views;
- event/action/receipt timeline;
- metadata and raw bounded manifest inspector;
- accessible status semantics, keyboard navigation, reduced motion, high contrast, 44px touch targets;
- mobile-first 375 / 768 / 1024 / 1440 baseline with fluid `clamp()` behavior and zero page-level horizontal overflow;
- dependency-free, same-origin/relative assets; no CDN, hostname, port, or deployment-path assumptions;
- service-worker/offline-safe immutable bundle where policy permits;
- safe absolute URL derivation from trusted request/proxy origin;
- localhost, LAN, tailnet, reverse proxy, tunnel, public host, and static export compatibility.

The page must present failing, blocked, disputed, stale, or incomplete evidence as beautifully and clearly as passing evidence. Visual polish must never turn uncertainty into success.

### Lightweight bundle and evidentiary media contract

Use the lightest format that preserves evidentiary fidelity:

- semantic static HTML + compact CSS; JavaScript is optional and enhancement-only;
- one bounded normalized JSON manifest; repeated scope/provenance/actor data uses dictionaries and stable refs rather than duplication;
- no framework runtime, CDN, external font, inline base64 media, or repeated diagnostics blobs;
- content-addressed assets deduplicate identical screenshots/clips in storage; a portable export includes only assets referenced by that artifact;
- image previews use responsive thumbnails and lazy loading; exact original evidence remains available by hash on demand;
- full diagnostic/trace payloads stay behind stable refs and compressed/downloadable representations where supported;
- renderer budgets for shell bytes, manifest bytes, initial transfer, media bytes, and total bundle size are recorded in the manifest and tested;
- Overview must become useful before heavy media downloads; Evidence/Inspect progressively load exact assets.

Images and video are first-class evidence—not decoration:

- screenshots/images prove exact visual state, layout, responsive behavior, annotations, comparisons, and measurements;
- short video proves temporal/interaction behavior such as menus, drag/drop, focus movement, animation, multi-step flows, retries, recovery, and settlement transitions;
- each media item records evidentiary purpose, source event/action range, MIME/codec, dimensions, duration/frame rate when applicable, byte size, capture method, timestamps, SHA-256, redaction state, and verification class;
- interaction clips synchronize timestamps with the structured action/event/Receipt timeline so a reviewer can jump from an event to the relevant frame/segment;
- videos provide a poster image, controls, captions/transcript when speech or meaningful audio exists, no autoplay, muted-by-default posture, and `preload="metadata"`;
- browser-compatible lightweight derivatives may improve viewing, but the exact original evidentiary asset and its hash remain authoritative;
- media is included only when it proves a declared claim/Acceptance Atom; decorative, duplicate, or unscoped captures are excluded.

### Lifecycle and persistence gates

- Publish must complete before `browser_close`; the artifact survives session closure and engine restart.
- Screenshot/media assets are copied into the immutable bundle; no live-session paths remain.
- Path traversal, markup/script injection, MIME confusion, oversized assets, unsafe URL/origin data, and secret patterns fail closed.
- Default access is read-only and non-indexed; public-safe publication requires explicit redaction/policy approval.
- Retention is declared and enforceable; expiry/revocation preserves canonical ledger lineage without keeping disallowed payloads.
- Cleanup attestation records session closure and temporary-resource disposal independently of publication success.

### Added acceptance

1. One Focusa Homepage proof publishes a rich four-viewport bundle and remains usable after browser close and engine restart.
2. The same renderer accepts at least one non-browser evidence bundle (for example test/release/API evidence).
3. Pi, MCP, HTTP, CLI/OpenAPI consumers return the same artifact identity, scope, hash, and publication state.
4. Focusa Desktop/Cockpit opens the artifact with exact Project/Workstream/Workpoint scope and no parallel truth state.
5. Public-safe and local-authorized projections demonstrably differ according to policy.
6. Missing/blocked artifact publication prevents a caller from presenting verified completion; existence of an artifact alone does not grant completion.
7. Responsive, accessibility, restart-persistence, idempotency, redaction, injection, traversal, MIME/size, reverse-proxy-origin, offline/export, and cross-version tests pass.
8. Shell/manifest/initial-transfer/total-bundle budgets pass; repeated media is content-deduplicated and full-resolution assets do not block initial render.
9. Image evidence proves static visual claims; one timestamp-synchronized video clip proves an interaction claim with poster, controls, and structured action linkage.
