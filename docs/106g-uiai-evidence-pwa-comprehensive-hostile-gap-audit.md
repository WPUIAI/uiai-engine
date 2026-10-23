> Parent authority: https://github.com/WPUIAI/uiai-engine/issues/106
> Canonical source: https://github.com/WPUIAI/uiai-engine/issues/106#issuecomment-5462623618

## Comprehensive hostile gap audit — requirements still not frozen

Audit scope: full Evidence Artifact lifecycle from capture → normalization → Project/Workstream authority → storage/integrity → redaction/access → human/agent verification → actions/closure → distribution/federation → retention/deletion/recovery.

The prior amendments establish the product direction. The following requirements remain absent or insufficiently exact for IR5 implementation. None may be inferred ad hoc.

## P0 — Canonical contract and authority gaps

**EPWA-GAP-001 — Schema ownership.** Freeze which schemas are Focusa-owned/generated versus UIAI-owned, their exact IDs, source files, codegen path, compatibility policy, and prohibition on duplicate DTOs.

**EPWA-GAP-002 — Semantic boundaries.** Define exact distinctions and allowed links among raw artifact, Evidence, Evidence Artifact, Receipt, Verification Result, Judge Result, Review Report, Completion Candidate/Decision, Settlement, and public derivative.

**EPWA-GAP-003 — Lifecycle state machine.** Specify legal transitions for draft, capturing, normalizing, redacting, blocked, published, verified-integrity, expired, revoked, superseded, corrupt, quarantined, deleted-payload/tombstoned, and restored.

**EPWA-GAP-004 — Partial-publication semantics.** Define crash/timeout behavior when manifest, assets, index, share authority, derivative, Receipt, or cleanup succeeds only partially. No state may imply publication when required pieces are absent.

**EPWA-GAP-005 — Identity.** Freeze globally stable artifact/revision/asset/derivative/instance IDs, collision handling, content-hash relationship, idempotency key derivation, and privacy implications of deterministic IDs.

**EPWA-GAP-006 — Scope evolution.** Define behavior when project name/root/worktree changes, Workstream rotates, Workpoint is superseded, evidence belongs to multiple Workstreams, or an artifact is imported into another project. Original scope must not be rewritten.

**EPWA-GAP-007 — Authority drift.** Define what happens when permissions, entitlement, Completion Contract, verifier policy, redaction policy, or action capability changes after publication.

**EPWA-GAP-008 — Exact operations.** Freeze HTTP paths, OpenAPI operation IDs, CLI/MCP/Pi names, side-effect/risk classes, permissions, confirmations, idempotency, Receipts, and typed errors for publish/inspect/verify/export/share/import/revoke/action/judge operations.

**EPWA-GAP-009 — Version negotiation.** Define reader/writer compatibility, unknown-field behavior, schema migrations, downgrade/read-only posture, capability digest negotiation, and cross-version fixtures across Focusa/UIAI/Cockpit/Desktop.

**EPWA-GAP-010 — Artifact collections.** Define grouping/ordering of multiple artifacts under a Workpoint, Acceptance Atom, Completion Case, Workset/release, incident, or project timeline without creating another mutable truth store.

**EPWA-GAP-011 — Readiness packet.** No approved IR3 task graph, IR4 fixtures/errors/migrations/rollback/evidence, or IR5 implementation packet exists. #294 remains a hard gate.

**EPWA-GAP-012 — Existing-work ownership.** Resolve ownership of current untracked `contracts/` and schema-embed trees and exact dirty-state policy before placing canonical schemas there.

## P0 — Cryptographic trust, time, and federation gaps

**EPWA-GAP-013 — Hashes are not authenticity.** Define optional/required instance signatures, signing envelope, algorithm agility, verification, canonicalization, and what “unsigned” means. A SHA-256 alone cannot prove who produced an artifact.

**EPWA-GAP-014 — Key lifecycle.** Define instance key generation/storage, hardware/keychain options, rotation, revocation, compromise recovery, expired keys, trust-store updates, and historical-signature validation.

**EPWA-GAP-015 — Trusted time.** Define capture/publish/evaluation time sources, clock-drift detection, monotonic ordering, timezone rendering, remote-worker skew, optional trusted timestamps, and behavior when time confidence is degraded.

**EPWA-GAP-016 — Self-host discovery.** Define instance identity/discovery, public capability metadata, version/capability negotiation, trust bootstrap, and prevention of host/instance impersonation without requiring Focusa Cloud.

**EPWA-GAP-017 — Federation/mirroring.** Define source vs mirror authority, import verification, duplicate/conflicting mirrors, supersession across instances, split-brain resolution, source-instance loss, and revocation propagation limits.

**EPWA-GAP-018 — Downloaded-copy truth.** Make explicit that revoking a hosted link cannot retract prior PDFs, emails, decks, archives, screenshots, or imported copies. Derivatives must carry source access/retention limitations.

**EPWA-GAP-019 — Cross-instance action trust.** Define authenticated handoff, destination selection, nonce/replay defense, scope preservation, confused-deputy prevention, and which instance may execute each action.

**EPWA-GAP-020 — QR/deep-link validity.** Define QR payload safety, expiry/revocation, app-link/universal-link behavior, unsupported clients, and prevention of internal-host leakage.

## P0 — Storage, durability, and recovery gaps

**EPWA-GAP-021 — Durable store.** Freeze exact directory/database/index layout, file permissions, tenant boundaries, metadata index, and storage-backend abstraction.

**EPWA-GAP-022 — Atomicity/crash consistency.** Define staging, atomic rename, fsync/durability level, transaction boundaries, orphan cleanup, restart replay, and power-loss fixtures.

**EPWA-GAP-023 — Corruption handling.** Add periodic integrity scrub, manifest/asset mismatch detection, quarantine, repair/re-fetch policy, operator alerts, and no-success behavior for corrupt artifacts.

**EPWA-GAP-024 — Backup/restore/DR.** Define backup scope, encryption, restore ordering, RPO/RTO, artifact/index/share-authority consistency, disaster recovery, and restore verification.

**EPWA-GAP-025 — Retention/GC.** Define reference-aware garbage collection, superseded revisions, derivatives, mirrors, legal holds, tombstones, cleanup Receipts, and protection against deleting assets shared by multiple artifacts.

**EPWA-GAP-026 — Quotas/disk pressure.** Define per-instance/project/workstream/user limits, media budgets, admission control, low-disk behavior, notification, cleanup priority, and fail-closed publication.

**EPWA-GAP-027 — Deduplication privacy.** Content-addressed dedup must not reveal another tenant/project’s asset existence, bypass ACLs, or couple deletion/retention across security domains.

**EPWA-GAP-028 — Encryption at rest.** Define encryption requirements, key ownership/rotation, backup encryption, temporary/staging files, thumbnails/OCR/transcripts, and memory/log exposure.

**EPWA-GAP-029 — Cache correctness.** Define immutable caching, ETags, cache keys by audience/access policy, purge/revocation behavior, proxy/CDN avoidance or configuration, and prevention of private/public cache confusion.

**EPWA-GAP-030 — Search/index recovery.** Define index rebuild from immutable manifests, stale index detection, pagination, sorting, and no-loss behavior when the index is corrupted.

## P0 — Security, privacy, and hostile-content gaps

**EPWA-GAP-031 — Legacy XSS.** #107 remains open. The new renderer cannot ship atop the unsafe interpolation path.

**EPWA-GAP-032 — Web security baseline.** Freeze CSP, CSRF, CORS, clickjacking/frame-ancestors, Referrer-Policy, Permissions-Policy, nosniff, HSTS deployment posture, cookie flags, and trusted-proxy/Host-header rules.

**EPWA-GAP-033 — SSRF.** Remote URL/media ingestion, unfurling, email images, embeds, and import must block loopback/private/link-local/metadata endpoints, redirects, DNS rebinding, unsupported schemes, and credential forwarding.

**EPWA-GAP-034 — Hostile files.** Define MIME sniffing, extension mismatch, size/decompression limits, zip/tar path traversal, archive/zip bombs, malformed images/video/PDF/PPTX/Office files, and quarantine/scanning policy.

**EPWA-GAP-035 — Active content.** Sanitize or reject SVG scripts/external refs, HTML, PDFs with active content, Office macros/links, media metadata, and imported service workers/manifests.

**EPWA-GAP-036 — Malware scanning.** Define optional/required scanner adapters, scanner unavailable posture, signature/version freshness, result refs, quarantine, and public-share block rules.

**EPWA-GAP-037 — Secret/PII discovery.** Freeze classifiers for credentials, cookies, auth headers, personal data, faces, emails, account identifiers, hidden DOM, OCR/transcripts, diagnostics, URLs/query strings, EXIF, and document metadata.

**EPWA-GAP-038 — Irreversible redaction.** Blurring/overlays may be reversible or preserve source pixels. Define destructive pixel/text/audio removal, metadata stripping, derivative re-encoding, verification, and retention of originals under separate authority.

**EPWA-GAP-039 — Channel-specific privacy.** Public PWA, PDF, email, slide deck, unfurl, connector, Judge View, and archive need separate allow/deny field and media policies—not one `public_safe` boolean.

**EPWA-GAP-040 — Consent and lawful handling.** Define capture/share consent, third-party/customer data, minors/high-risk data, data residency, purpose limitation, export, right-to-delete, legal hold, audit access, and public-proof approvals.

**EPWA-GAP-041 — Abuse/enumeration.** Add rate limits, token entropy, brute-force protection, list/search authorization, download bandwidth controls, bot/indexing rules, abuse reporting, and suspicious-access audit.

**EPWA-GAP-042 — Service-worker security.** Define cache versioning, update/rollback, scope confinement, revocation/private-data purge, offline stale warnings, import behavior, and cache-poisoning tests.

**EPWA-GAP-043 — Prompt/data exfiltration.** Ensure Judge Views, connector payloads, comments, OCR, transcripts, captions, filenames, and metadata cannot alter trusted instructions or induce unauthorized network/tool use.

**EPWA-GAP-044 — Audit integrity.** Define append-only access/share/action/export/judge logs, actor/instance attribution, tamper detection, privacy redaction, retention, and operator inspection.

## P0 — Capture, provenance, media authenticity, and reproducibility gaps

**EPWA-GAP-045 — Completeness snapshot.** Freeze exactly which evidence refs/results are captured at turn finalization and how late-arriving diagnostics/jobs are handled without producing a misleading “complete” bundle.

**EPWA-GAP-046 — Environment fingerprint.** Record OS/arch, engine/browser/runtime/build, viewport/DPR, fonts/color scheme/locale/timezone, relevant feature flags, network profile, target revision/deploy identity, and degraded/unavailable fields.

**EPWA-GAP-047 — Actual vs surrogate/synthetic.** Define mandatory labels and closure eligibility for real capture, generated illustration, mock, placeholder, transformed media, AI-generated content, reconstructed state, and unavailable native proof.

**EPWA-GAP-048 — Media authenticity.** Define capture provenance, transcoding lineage, frame drops, variable frame rate, timestamps, audio/video synchronization, codec changes, edit/cut manifests, and detection/labeling of manipulated or synthetic media.

**EPWA-GAP-049 — Interaction causality.** Link action intent → pre-state → actual side effect → post-state → diagnostics/network delta → visual frames. Video alone must not imply causation.

**EPWA-GAP-050 — Annotation geometry.** Define coordinate systems, responsive scaling, crop/rotation transforms, source asset revision, author/time, overlapping annotations, and immutable overlays.

**EPWA-GAP-051 — Visual comparison policy.** Freeze baseline selection, anti-alias/font/render noise tolerances, masks, perceptual thresholds, dynamic-region handling, and prohibition on cherry-picked viewports.

**EPWA-GAP-052 — Audio privacy/accessibility.** Define audio capture consent, redaction/muting, transcript accuracy/confidence, speaker identity handling, captions, meaningful-sound descriptions, and no-audio proof cases.

## P0 — Verification, LLM judge, and closure gaps

**EPWA-GAP-053 — Judge schema not approved.** Judge View, evaluation request/result, allowed verdicts, citations, information-set hash, and immutable result/Receipt remain conceptual.

**EPWA-GAP-054 — Calibration.** Define judge golden sets, calibration curves, false-positive/false-negative thresholds, inter-rater reliability, regression gates, and promotion/rollback of judge policies/prompts/models.

**EPWA-GAP-055 — Bias/fairness.** Define bias tests across visual style, language, disability/accessibility, locale, skin tone/people imagery where applicable, brand unfamiliarity, and model/provider differences.

**EPWA-GAP-056 — Model drift/retirement.** Define behavior when a judge model/version disappears, silently changes, is deprecated, or becomes incompatible; preserve reproducibility without pretending exact reruns are possible.

**EPWA-GAP-057 — Judge disagreement.** Freeze policies for multiple judges, human overrides, ties, outliers, contradictory evidence, correlated models/providers, and when disagreement forces disputed/indeterminate state.

**EPWA-GAP-058 — Judge economics.** Define token/media/spend budgets, expensive multimodal admission, caching, duplicate evaluation prevention, provider outage, rate limiting, and cost per verified outcome.

**EPWA-GAP-059 — High-consequence policy.** Define domains where LLM judgment is advisory only or requires named human/domain verifier, dual control, regulated records, or prohibition.

**EPWA-GAP-060 — Appeal/adjudication.** Define operator dispute, evidence correction, recusal/conflict, re-evaluation, accepted-risk/exception, and immutable appeal lineage.

**EPWA-GAP-061 — Sufficiency/coverage.** Define how required evidence dimensions, viewport/media coverage, negative tests, accessibility, diagnostics, native/live proof, and missing evidence map to each Acceptance Atom.

**EPWA-GAP-062 — Freshness/revalidation.** Bind proof validity to source/deploy/runtime/policy/dependency revisions and #280 watchdog rules; define automatic invalidation, reproof, reopen, and stale public derivatives.

**EPWA-GAP-063 — Closure enforcement.** The “no artifact/no done” rule is documented but not wired into provider close, Workpoint, Workset/release, Silent Sessions, CallGraph, generated UI, agent final output, or installed binaries.

**EPWA-GAP-064 — Judge isolation.** Define sandbox/network/tool restrictions, credential/data egress, provider data-use policy, prompt storage, and deletion/retention for judge inputs/outputs.

## P1 — Action, collaboration, and workflow gaps

**EPWA-GAP-065 — Action transport.** Freeze exact local/remote/deep-link/connector execution protocol, authentication, anti-replay, capability refresh, and stale action-manifest invalidation.

**EPWA-GAP-066 — Partial action effects.** Define reconciliation and compensation when an action runs but the PWA loses the response, times out, reconnects, or destination state differs.

**EPWA-GAP-067 — Collaboration concurrency.** Define comments, annotations, suggestions, mentions, assignments, presence, simultaneous edits, moderation, offline reconciliation, and immutable artifact vs mutable review-thread boundaries under #290.

**EPWA-GAP-068 — Notifications.** Define who receives capture/failure/share/expiry/reproof/dispute/settlement notifications, channel preference, deduplication, quiet hours, and sensitive-content redaction.

**EPWA-GAP-069 — Bulk operations.** Define project/workstream evidence registry, list/search/filter/tag/archive, bulk export/retention/reproof, and prohibition on bulk mutation crossing scope/authority.

**EPWA-GAP-070 — Follow-up traceability.** Tasks/Workpoints created from findings must retain exact artifact/finding/Atom refs and later resolve back into a new proof revision.

## P1 — Human UX, accessibility, localization, and scale gaps

**EPWA-GAP-071 — Accessibility standard.** Freeze WCAG 2.2 AA targets, automated/manual screen-reader matrix, focus restoration, zoom/reflow, color/contrast, cognitive load, captions/audio descriptions, reduced motion, and accessible errors.

**EPWA-GAP-072 — Document accessibility.** Define PDF/UA or declared equivalent target, tagged PDF verification, accessible PPTX/HTML slides, reading order, alt text, table semantics, language metadata, and known exporter limitations.

**EPWA-GAP-073 — Localization.** Define locale/timezone/number/date/duration formatting, RTL, translated status/error/action vocabulary, font fallback, line expansion, and canonical machine values independent of display locale.

**EPWA-GAP-074 — Large evidence sets.** Define virtualized/lazy rendering, search, section indexes, media pagination, timeline clustering, progressive disclosure, and usable print/export truncation/appendix rules.

**EPWA-GAP-075 — Low bandwidth/LowMem.** Define semantic-first mode, image/video quality ladders, range requests, poster/keyframe priority, paused background work, and truthful omitted-media posture.

**EPWA-GAP-076 — Browser/PWA matrix.** Freeze supported browser/mobile versions, installability, offline/update behavior, Web Share fallback, print differences, media codec support, storage eviction, and private browsing behavior.

**EPWA-GAP-077 — Branding/theming.** Define safe project branding and white-label customization without altering status colors/labels, hiding caveats, injecting CSS/scripts, or breaking print/accessibility.

**EPWA-GAP-078 — Empty/degraded states.** Every section/action/export needs loading, absent, unauthorized, blocked, stale, corrupt, partially available, offline, expired, and unsupported-client UX.

## P1 — PDF, presentation, email, and format conformance gaps

**EPWA-GAP-079 — PDF profile.** Choose PDF/A archival posture, PDF/UA accessibility target, font embedding/substitution, color profile, signatures, attachments, metadata, deterministic rendering limits, and viewer conformance matrix.

**EPWA-GAP-080 — Presentation formats.** Freeze PPTX library/implementation, native editability floor, ODP decision, font/media compatibility, speaker notes, master/layout strategy, 16:9/4:3 validation, and PowerPoint/Keynote/LibreOffice/Google Slides matrix.

**EPWA-GAP-081 — Email transport.** Define SMTP/provider adapters, credentials, DKIM/SPF/DMARC posture, TLS, bounce/complaint/suppression handling, retry/dead-letter, attachment limits, recipient consent, and delivery-vs-read truth.

**EPWA-GAP-082 — Email client matrix.** Define tested clients, dark mode, blocked images, plain-text parity, CID/attachment behavior, link expiry, forwards, and no-tracking requirement.

**EPWA-GAP-083 — Markdown/rich-text fidelity.** Define lossy-field warnings, table/media fallback, annotation representation, stable references, and roundtrip/non-roundtrip expectations.

**EPWA-GAP-084 — Archive safety.** Define portable archive format, deterministic file ordering/timestamps, path safety, compression limits, manifest placement, signature verification, and safe import.

**EPWA-GAP-085 — Media licensing.** Preserve copyright/license/attribution/consent restrictions for screenshots, brand assets, documents, fonts, audio, and video across every derivative/channel.

## P1 — Composable work-software connector gaps

**EPWA-GAP-086 — Neutral block schema.** The composable block model needs exact versioned schemas, nesting/ordering, capability negotiation, stable block refs, and loss/degradation reporting.

**EPWA-GAP-087 — Mapping drift.** Define destination schema/API version drift, unsupported blocks, field truncation, markdown quirks, attachment conversion, and connector compatibility tests.

**EPWA-GAP-088 — Destination authority.** Define account/workspace/project/channel selection, permissions, data classification/egress, approval, and prevention of posting into the wrong tenant or channel.

**EPWA-GAP-089 — Delivery idempotency.** Define duplicate prevention, retries, rate limits, partial attachments, remote mutation reconciliation, update-vs-new behavior, and dead-letter recovery.

**EPWA-GAP-090 — External retention/deletion.** Track destination IDs/URLs, external retention/visibility, edit/delete success, inaccessible destinations, and the fact that external copies may persist.

**EPWA-GAP-091 — Webhook security.** Define signatures, timestamp/replay windows, endpoint verification, secret rotation, payload size, delivery retries, and redacted logs.

**EPWA-GAP-092 — Unfurl/embed policy.** Define Open Graph/oEmbed-style fields/routes, sandboxing, thumbnail generation, cache expiry, private-link behavior, and no-mutation embeds.

## P1 — Agent/CLI/MCP/API/LLM gaps

**EPWA-GAP-093 — Generated parity.** No approved source schema/codegen/conformance test yet guarantees identical REST, CLI, MCP, Pi, SDK, Cockpit, and LLM semantics.

**EPWA-GAP-094 — Async job contract.** PDF/deck/archive/media/judge/email/connector operations need job IDs, progress, cancellation, restart persistence, idempotency, result refs, errors, and completion Receipts.

**EPWA-GAP-095 — Streaming/ranges.** Define byte ranges, video segments, large transcript/diagnostic pagination, backpressure, cancellation, and partial download verification.

**EPWA-GAP-096 — Content negotiation.** Freeze media types, Accept behavior, profiles, filename/content-disposition safety, unsupported format errors, and cache variation.

**EPWA-GAP-097 — Agent citations.** Define canonical citation grammar for artifact/figure/slide/frame/timestamp/finding/Atom/Receipt refs and validation against the cited revision.

**EPWA-GAP-098 — Token-budget truth.** Define summary generation, omissions, token accounting, truncation markers, retrieval priorities, and no-summary-as-proof rule.

**EPWA-GAP-099 — Error taxonomy.** Freeze typed errors/recovery for scope, authority, integrity, policy, media, storage, export, connector, judge, network, compatibility, quota, and partial-effect failures.

**EPWA-GAP-100 — SDK/automation safety.** Decide SDK targets, generated clients, retry defaults, secret handling, pagination helpers, and prevention of clients locally interpreting completion/authority.

## P1 — Operations, observability, migration, and release gaps

**EPWA-GAP-101 — SLOs/budgets.** Define publish latency, first useful render, manifest/shell/media/total sizes, export times, uptime, error rate, durability, recovery, and cost budgets.

**EPWA-GAP-102 — Telemetry.** Add privacy-safe metrics for publication, blocked redaction, integrity failures, export/share/action/judge jobs, stale proofs, storage pressure, connector delivery, and cleanup—without exposing evidence content.

**EPWA-GAP-103 — Health/support.** Define health endpoints, doctor checks, bounded support bundles, operator diagnostics, safe logs, and exact recovery for store/index/key/renderer/connector failures.

**EPWA-GAP-104 — Legacy migration.** Define disposition for `/api/share`, `/v/{token}`, persisted shares, session screenshot paths, existing Focusa evidence refs, old clients, and rollback without breaking valid links.

**EPWA-GAP-105 — Release compatibility.** Publish Focusa/UIAI/Cockpit/Desktop/schema/renderer/format/connector/judge capability matrix and fail closed on incompatible installed components.

**EPWA-GAP-106 — Threat model/red team.** Add hostile artifact/comment/page/connector/judge fixtures, tenant escape, token leak, prompt injection, path traversal, SSRF, XSS, CSRF, file bombs, cache poisoning, signature/key compromise, and malicious import.

**EPWA-GAP-107 — Reliability testing.** Add property/fuzz tests, restart/power-loss/partial-write chaos, backup restore, index rebuild, disk full, concurrent publish/import/revoke, network partition, and source-instance loss.

**EPWA-GAP-108 — Client/viewer matrices.** Test supported browsers/PWAs, screen readers, printers, PDF viewers, PowerPoint/Keynote/LibreOffice/Slides, email clients, and selected work-software adapters.

**EPWA-GAP-109 — Installed E2E.** Source/unit tests are insufficient. Require producer tests, consumer-side tests, cross-version interop, and live installed end-to-end dogfood under the production-consistency policy.

**EPWA-GAP-110 — Operational ownership.** Assign source owners, security response, schema governance, connector maintenance, judge policy owners, retention/backup operators, incident runbooks, and support boundaries.

## Recommended gate order

1. **IR2 completion:** freeze current call stacks, ownership, threat model, data classes, exact authority boundaries.
2. **IR3:** decompose EPWA-GAP-001–110 into dependency-ordered atomic packets; no omnibus implementation task.
3. **IR4:** bind schemas, fixtures, errors, migrations, rollback, red-team cases, consumer tests, and installed evidence.
4. **IR5:** execute only packets with no unresolved mandatory inference.
5. **IR6:** independent installed proof: self-hosted publish → agent/human verification → actions → distribution/import → revalidation → closure decision, including failure/recovery cases.

## Minimum P0 closure blockers for the first production slice

The first production slice cannot be called safe until at least EPWA-GAP-001–064, EPWA-GAP-094, EPWA-GAP-099, and EPWA-GAP-101–110 have approved dispositions. P1 features may be phased only when omission is explicit, truthful, non-blocking to required evidence, and represented as unavailable—not silently deferred.
