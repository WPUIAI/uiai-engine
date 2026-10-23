> Parent authority: https://github.com/WPUIAI/uiai-engine/issues/106
> Canonical source: https://github.com/WPUIAI/uiai-engine/issues/106#issuecomment-5462634711

## Comprehensive gap audit addendum — anti-curation, chain-of-custody, and long-term trust

The 110-gap register remains valid. A second adversarial pass adds the following requirements; total identified unresolved requirements: **140**.

**EPWA-GAP-111 — Chain of custody.** Record every custody transition from capture worker/session through staging, normalization, redaction, derivative generation, export, import, mirror, judge, and archival, including actor/process/instance, time confidence, input/output hashes, and policy.

**EPWA-GAP-112 — Selection/omission manifest.** The publisher must declare what eligible evidence was included, excluded, truncated, unavailable, or rejected and why. A curated bundle cannot imply that omitted contrary evidence does not exist.

**EPWA-GAP-113 — Independent ground-truth escape hatch.** Verification policy must specify when judges/reviewers may or must inspect canonical source state, live target, repository revision, external API, or independent recapture instead of relying solely on the producer-assembled artifact.

**EPWA-GAP-114 — Anti-cherry-picking coverage.** Define required viewport/theme/locale/state/error/negative-test matrices, randomized or policy-selected samples, and detection of producer-selected “best” frames/runs.

**EPWA-GAP-115 — Event-range continuity.** Bind interaction/video/diagnostics to contiguous event/action sequence ranges with gap markers. Missing events, dropped frames, cleared diagnostics, or restarted sessions must be visible.

**EPWA-GAP-116 — Target/account identity.** Prove or explicitly qualify which origin, environment, tenant/account, deployment, revision, user role, and authentication context was observed. A correct page on the wrong account/environment is not valid proof.

**EPWA-GAP-117 — Proof of absence.** Define what evidence can establish absence/non-occurrence (no overflow, no errors, no request, no secret leak), observation window and channels, and when the correct result is only “not observed.”

**EPWA-GAP-118 — Corrections/retractions.** Define correction, withdrawal, invalidation, retraction reason, affected claims/derivatives/judge results, public notification, and immutable preservation of prior lineage without continuing to present it as current.

**EPWA-GAP-119 — Provenance trust levels.** Distinguish self-asserted, engine-observed, independently observed, externally attested, operator-attested, hardware-backed, imported-unsigned, and unverifiable provenance. A valid signature does not prove truthful capture.

**EPWA-GAP-120 — Actor/delegation identity.** Bind human, agent, model, harness, worker, service account, delegated authority, approver, verifier, exporter, importer, and sharer identities without exposing unnecessary personal information.

**EPWA-GAP-121 — Imported action distrust.** Action manifests embedded in foreign/imported artifacts are untrusted suggestions. The receiving instance must regenerate eligible actions from its local operation registry, scope, authority, and policy before execution.

**EPWA-GAP-122 — Policy precedence.** Define deterministic conflict resolution among artifact policy, Project/Workstream policy, Completion Contract, instance policy, destination policy, legal hold, operator decision, and public-share policy.

**EPWA-GAP-123 — Legal/eDiscovery posture.** Define chain-of-custody export, litigation/legal hold, audit discovery, signed declarations, retention suspension, jurisdiction metadata, and explicit statement that technical artifacts are not automatically legally admissible.

**EPWA-GAP-124 — Public authenticity UX.** A recipient needs a simple offline/online verify flow for hash/signature/schema/revocation/supersession without trusting visual branding. Verification failures must be obvious and nontechnical.

**EPWA-GAP-125 — Moderation/takedown.** Define abuse, copyright/privacy complaints, accidental public disclosure, emergency revoke, takedown records, mirror notification, and limitations once derivatives were downloaded or emailed.

**EPWA-GAP-126 — Redaction quality assurance.** Add false-negative/false-positive fixtures, OCR-after-redaction checks, hidden-layer/metadata inspection, sampled human review for high-risk exports, and regression gates for classifier/model changes.

**EPWA-GAP-127 — OCR/transcript/translation fidelity.** Record engine/model/version/language/confidence, diarization, uncertain spans, edits, source timestamps, translation provenance, and prohibition on treating generated text as exact source content.

**EPWA-GAP-128 — Visual/color fidelity.** Define device pixel ratio, zoom, color profile, HDR/SDR, gamma, font availability, browser scaling, image orientation, compression, transparency, and comparison behavior so export/transcoding does not alter material visual evidence.

**EPWA-GAP-129 — Capture permissions.** Define OS/browser camera, microphone, screen-recording, clipboard, file, notification, and accessibility permissions; consent; denied-permission posture; and proof that capture did not silently omit a protected channel.

**EPWA-GAP-130 — Resource scheduling.** Define browser-page pool, encoder/transcriber/PDF/deck/judge worker queues, priorities, per-project fairness, cancellation, backpressure, cooldown, and protection against evidence jobs starving live agent work.

**EPWA-GAP-131 — Judge contamination/gaming.** Guard against benchmark leakage, repeated prompt optimization to a known judge, producer-written persuasive summaries, model training contamination, and optimizing appearance instead of actual outcome.

**EPWA-GAP-132 — Judge order/context bias.** Test evidence ordering, verbosity, status-label anchoring, brand/style bias, prior verdict exposure, and compare blind/randomized evaluation where required.

**EPWA-GAP-133 — Provider execution provenance.** Record requested versus actual model/provider/version, routing/fallback, temperature/seed/determinism controls where available, tool access, region/data-use posture, and silent provider substitution.

**EPWA-GAP-134 — Data minimization.** Capture only fields/media needed for declared claims; define crop/region capture, bounded diagnostics, selective transcript, retention minimization, and why each sensitive item is necessary.

**EPWA-GAP-135 — Long-term format survival.** Define migration/validation for obsolete codecs, PDF/PPTX/schema/browser formats, preservation of originals, derivative regeneration, and no silent semantic drift during archival conversion.

**EPWA-GAP-136 — Air-gapped verification.** Portable bundles need offline schema/hash/signature verification, embedded verification instructions/tools or reproducible algorithm description, and explicit inability to check current revocation/supersession while offline.

**EPWA-GAP-137 — Link rot/source loss.** Define behavior when source instance/domain/target/repository/media disappears, including archival refs, mirror provenance, unavailable-source labels, and no false live-verification claims.

**EPWA-GAP-138 — Analytics/tracking policy.** Default to no tracking. If self-hosted access analytics exist, define consent, minimization, IP/user-agent handling, retention, access, email tracking prohibition, and separation from verification truth.

**EPWA-GAP-139 — Canonical-state snapshot linkage.** An action or judge result should cite the exact canonical state/revision it read. Later state changes must not retroactively make the old action/result appear current.

**EPWA-GAP-140 — Human-verifiable printed identity.** Printed/PDF/slide/email forms need short artifact/revision identifiers, human-readable verification instructions, QR/link, and checksum/signature presentation that is useful without overwhelming or implying that a truncated hash is cryptographic verification.

These gaps reinforce a central rule: the Evidence PWA is the preferred verification entry point, but it must never become a producer-curated evidence monopoly. Independent source inspection and omission visibility remain mandatory where the Completion Contract requires them.
