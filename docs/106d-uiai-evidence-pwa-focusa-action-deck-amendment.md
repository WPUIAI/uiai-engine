> Parent authority: https://github.com/WPUIAI/uiai-engine/issues/106
> Canonical source: https://github.com/WPUIAI/uiai-engine/issues/106#issuecomment-5462573424

## Interaction amendment — Focusa Action Deck on every Evidence PWA

Each Evidence PWA artifact must expose a polished, context-sensitive group of useful Focusa-centric actions. Reuse Cockpit's existing declarative `ReportInteractionManifest` / `ReportActionController` contract; do not create arbitrary executable buttons or a parallel action router.

### Core rule

Actions are derived from the artifact's exact Project / Workstream-continuity / Workpoint / Work Item scope, truth posture, missing Acceptance Atoms, freshness, authority, permissions, entitlement, audience/access class, and online/paired state.

Every action declares:

- stable interaction/action id and human label;
- canonical capability/operation id;
- exact target refs and required scope;
- context-ref policy;
- side-effect and risk class;
- permission/entitlement/availability result with plain-language reason;
- preview and approval/confirmation policy;
- idempotency/reconciliation posture;
- expected Receipt/result schema;
- online/paired authority requirement;
- safe handoff/deep-link destination;
- recovery action and non-authoritative fallback.

No action executes arbitrary prompts, scripts, raw tools, or browser commands. No action expands scope from artifact content. Artifact/page content remains untrusted input and never grants capability authority.

### Action groups

#### 1. Scope & orientation

- Open Project / Workstream / Workpoint.
- Resume the exact Workpoint.
- View trajectory and current gap.
- Open Context Cognition for the bounded artifact scope.
- Inspect source object, Work Items, related artifacts, and lineage.

Existing capability candidates include `focusa.workpoint.resume`, `focusa.trajectory.view`, `focusa.trajectory.assess`, and `focusa.context_cognition.packet` when available and authorized.

#### 2. Evidence

- Inspect canonical Evidence and provenance.
- Link this artifact to the active/exact Workpoint.
- Capture or intake the artifact as Evidence.
- Compare against prior/superseding artifacts or an approved baseline.
- Verify manifest/bundle/media hashes.
- Open exact screenshot/video/diagnostic segment.
- Download/export a policy-safe bundle.

Existing capability candidates include `focusa.workpoint.link_evidence`, `focusa.evidence.capture`, `focusa.workspace.artifact.intake`, and read/inspect operations.

#### 3. Verification

- Review Evidence.
- Request recapture.
- Request reverification/reproof.
- Rerun the exact producing capability where policy allows.
- Run a missing visual/interaction/accessibility check.
- Inspect Acceptance Atoms, contradictions, freshness, verifier independence, and diagnostics.

Use Cockpit action kinds `review_evidence`, `request_recapture`, `request_reverification`, and `rerun_capability`. Pending #278/#281 capabilities remain visibly unavailable until canonically released; the PWA must not invent them locally.

#### 4. Resolution & continued work

- Request changes.
- Comment or add an immutable annotation overlay.
- Create a scoped follow-up.
- Propose a repair/verification Workpoint.
- Checkpoint current work before switching.
- Record a blocker/open question through the authorized Focusa path.

Existing capabilities include `focusa.workpoint.checkpoint`, `focusa.trajectory.propose_workpoint`, and governed task-plan operations. Use Cockpit action kinds `request_changes`, `comment`, `annotate`, and `create_followup`.

#### 5. Adjudication & closure

- Open the Completion Verification Case.
- Preview the completion evaluation inputs.
- Request canonical completion evaluation when #277/#278 operations exist.
- Review contradiction/dispute/exception state.
- Request operator adjudication.
- Reopen/reproof after invalidation where canonical authority permits.
- Inspect the final decision and settlement Receipt.

There is never an ungoverned `Mark done`, `Approve proof`, or local success override. An artifact action may request/preview canonical evaluation; #277 alone decides and settles completion.

#### 6. Share, retention, and lifecycle

- Create a redaction preview.
- Publish/open a public-safe read-only derivative.
- Copy canonical artifact/manifest link.
- Export the portable bundle.
- Inspect or request retention/expiry changes.
- Revoke a share or restrict access.
- View supersession/revision lineage.

All mutations require the appropriate authority and Receipt. The immutable artifact body is never rewritten by comments, actions, or lifecycle changes; those become external events/Receipts or a new artifact revision.

### Context-sensitive primary action

The shell should promote one safe primary action and keep the rest in grouped disclosure:

| Artifact state | Primary action |
|---|---|
| not linked | Link to Workpoint |
| required proof missing | Capture required evidence |
| stale/invalidated | Request reverification |
| failed | Create repair Workpoint |
| blocked | Resolve/open blocker |
| disputed/contradictory | Open adjudication |
| verification complete | Review completion case |
| public/offline/unpaired | Open in Focusa / connect authority |

Unavailable actions remain visible when useful, with exact reason and recovery—not silently hidden.

### Portable execution modes

- **Local authorized / paired:** invoke the canonical operation through the standard guard/router/adapter/event path.
- **Remote public-safe:** read-only by default; mutations require authenticated handoff into Focusa Desktop/Cockpit/Cloud.
- **Offline export:** inspect, compare, verify hashes, and export remain available; mutations are disabled, not queued as fake success.
- **Expired/revoked:** preserve policy-safe local inspection and recovery guidance while denying unauthorized calls.

The PWA must not hard-code a daemon origin. Action destinations are relative trusted endpoints or approved connector/deep-link descriptors supplied by the manifest/runtime policy.

### Self-hosted portable evidence article

Every user's self-hosted Focusa/UIAI instance must be able to publish and share the complete Evidence PWA without Focusa Cloud, a CDN, external fonts, analytics, or a central rendering service.

The artifact must:

- use relative same-origin assets and remain valid under any deployment subpath;
- derive displayed absolute share URLs only from configured/trusted request-proxy authority, never a hard-coded host or port;
- install/run as a PWA with subpath-safe manifest and service-worker scope where policy permits;
- support explicit access classes: `local`, `lan`, `tailnet`, `private_team`, `unlisted`, and `public_safe`;
- default to read-only and non-indexed; public publication requires a redaction/policy preflight;
- issue revocable/expiring share authority separately from immutable artifact content so access changes never alter the bundle hash;
- keep bearer/share tokens out of manifests, portable bundles, logs, receipts, analytics, and artifact hashes; store only one-way token verifiers server-side;
- report `published` only when the instance persisted the artifact and generated a valid access route; external reachability is a separately measured state and must not be fabricated;
- provide copy-link, QR, open, revoke, restrict, extend-expiry, and export controls when authorized;
- work behind localhost, LAN, tailnet, reverse proxy, tunnel, custom domain, IPv4/IPv6, TLS termination, and non-root path deployments;
- preserve useful read-only inspection when disconnected from the source instance.

Portable export/import:

- export a dependency-free directory/archive containing the bounded manifest, semantic article shell, policy-safe referenced assets, hashes, and optional source-instance signature;
- identify the source instance by an opaque instance ref/public-key fingerprint rather than leaking internal hostname/topology;
- importing/mirroring on another self-hosted instance verifies all hashes/signatures, preserves original producer/scope/provenance, records importer/mirror lineage, and never rewrites the source artifact;
- globally stable artifact/revision refs prevent collision across projects and instances;
- content-addressed assets may deduplicate locally, while each exported article remains self-contained;
- static exports remain read-only. Mutating Focusa actions require authenticated handoff to an allowed source/target instance and exact Project/Workstream authority.

The source self-hosted instance remains access/retention/share authority for its hosted copy. Focusa Completion Authority remains the only completion truth authority; hosting or mirroring an article grants neither Evidence acceptance nor closure.

### Action UX

- Desktop: compact right action rail plus command palette/search.
- Mobile: 44px minimum controls, sticky primary action, grouped bottom sheet.
- Overview shows at most the highest-value actions; Inspect/Developer exposes policy, capability id, scope, side effects, and expected Receipt.
- Destructive/high-consequence actions use preview → confirmation → execution → Receipt.
- Results appear as external activity/Receipt updates without mutating the frozen proof artifact.
- Keyboard, screen-reader, reduced-motion, contrast, focus-order, and disabled-reason semantics are acceptance gates.

### Conformance

1. The same artifact revision produces the same eligible action set for identical capability/authority context.
2. Changing Project/Workstream/Workpoint scope invalidates stale action manifests.
3. Pi, HTTP/OpenAPI, MCP, CLI, Cockpit, Desktop, and PWA route an action to the same operation id and Receipt semantics.
4. Public/offline views cannot execute mutations.
5. Missing permission/entitlement/confirmation is explicit and fail-closed.
6. Action results cannot alter artifact hashes or self-settle completion.
7. At least one read-only, evidence-link, reverification, follow-up, authenticated handoff, unavailable, and high-risk preview path is proven end-to-end.
8. Two self-hosted instances publish/import the same portable article, preserve source scope/provenance/hash identity, and keep mutation authority separate.
9. Local/LAN/tailnet/private/unlisted/public-safe, subpath/reverse-proxy, revoke/expiry, offline export, token-redaction, and no-central-service cases pass.

Focusa integration remains governed by Startempire-Wire/focusa#263/#277/#278/#283/#291/#294.
