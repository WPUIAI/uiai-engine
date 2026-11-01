# Private authentication resolver admission — proposed v1

Status: **advisory_external / proposed; NOT implementation admission**.
Owner decision: Sir V3 under ARCHITECTURE_AUTHORITY_POLICY.md. No signature,
grant, implementation readiness, runtime capability or release is created here.
Related existing work: #82 (uiai-engine-zsjm), #81, #145, #147;
Startempire-Wire/focusa#299 and #294. This proposal closes none of them.

## Why a separate admission decision is necessary

Inspected main: 1c0a2d3a5a2724fcad7dde7981c2a0be053316c9.
Existing session.go middleware intentionally denies ordinary mutations while
AWAITING_AUTH. auth/resolve only matches a resume token and clears a latch;
it does not inject credentials or verify an authenticated account. Therefore
neither calling resolve before login nor permitting ordinary eval/fill is repair.

Current #82 comments explicitly disclaim a new broker/control/resolve grant.
Current #145 requires a versioned specification before implementation. Merged
PR #4 added C04 as documentation-only; its merge does not turn it into a private
consumer admission. Spec 156 explicitly remains IR0 and requires approved typed
Call Stack Designs and dependency-ordered #294 IR5 packets before source or
credential mutation. PR #239 config-provider credentials do not supply that lane.
The existing public FPV control/share restrictions stay intact; #175 is not a
private resolver implementation. This is the deduplicated proposal for those
existing owners, not a new architecture epic or competing authority.

## Exact decisions requested (none inferred approved)

1. **Transport/consumer boundary:** approve a separately authenticated private
   consumer operation over node-local authenticated IPC, not a public HTTP
   exception or an ordinary agent operation. Proposed operation names below
   remain subject to #258 conflict review. UIAI owns browser execution; Focusa
   retains policy/grant/use/receipt authority. Local OS peer identity alone
   authenticates the transport, not permission to consume a credential.
2. **Authority adapter and IR5 packet:** identify/approve the exact Focusa
   verifier/consume/reconcile operations, schema revision and trusted issuer
   configuration. It must validate current authority, atomic use reservation,
   revocation and fencing. No UIAI issuer, local allow boolean, fabricated
   signed grant, agent-authored manifest, or temporary grant substitute ships.
   The absence of this adapter means admission stays unavailable, not allow.
3. **Isolation and identity verifier:** approve the #76/#145 independently
   allocated private context/profile primitive and exact provider adapter that
   verifies form destination, expected account and post-login identity.
   Ordinary pooled pages are ineligible. Choose exact account-observation
   source and verifier revision in the IR5 packet, not generic success-text JS.

Recommended decision: consider the exact D1–D3 alternatives and bounded choices
in the root decision packet below for design decomposition, not implementation.
The three workstreams below are NOT atomic IR5 packets and cannot be admitted
as written; owning typed designs must resolve every mandatory field first.
This proposal does not request a universal grant or permit secret reveal.
Human takeover, CAPTCHA, passkeys and native presence retain separate handoff
contracts. Recovery/break-glass and unknown renewability are unconditionally
excluded, including from provider discovery/retrieval/testing.

## Proposed private interfaces (NOT deployed routes)

PrivateAuth.Begin(request) -> opaque resolution_ref + value-free state.
PrivateAuth.Status(resolution_ref) -> value-free receipt/status.
PrivateAuth.Cancel(resolution_ref) -> reconciled cancellation/cleanup status.
These are operation intents, not presently callable endpoints. No new route is
mounted on the public session router; no broker socket is added by this PR.

Begin accepts only refs: grant_ref, secret_binding_ref, private_session_ref,
challenge_ref, work_surface_binding_ref, frame_attempt_ref, operation_ref,
account_assertion_ref, pre_observation_ref, lease_generation, idempotency_key.
It accepts no password, OTP, cookie, JS, selector, URL, claimed actor,
resume_token, authenticated boolean, or caller-authored grant document.
Actor, consumer identity, origin/account, form/field bindings and expected
context generation are resolved from the authenticated peer and owning records.
The detailed projected schema and required gates are in the companion JSON.
Opaque refs are subject to exact-peer/scope read authorization, not public bearer
capabilities. Unknown fields, malformed refs, ambiguous versions and missing
owning metadata fail closed before provider retrieval.

The broker/custodian pushes a single-use role-bound value only through the
approved model-free IPC/FD boundary. Provider path/entry and secret are never
returned through Begin/Status. No config interpolation, ordinary fill/eval,
CDP attachment, raw cookie import, command argv or alternate worker is a fallback.

## Proposed owning call stack

1. Existing detector (vision.AuthChallenge) records a challenge and pauses the
   ordinary session. Private context allocation precedes credential retrieval.
   A pooled challenged session is not promoted or copied into private custody:
   return private_context_required and request a new admitted context. Any new
   context receives its own challenge and observation binding.
2. Authenticated IPC dispatcher resolves exact peer/actor and current owning
   refs. A proposed internal privateauth.Coordinator uses an injected authority
   consumer/verifier; it contains no grant-issue implementation. Names are
   proposals, not existing Go symbols. Check private context, challenge, actor,
   work surface/frame attempt, origin, account assertion, operation/fields,
   provider renewability, expiry/revocation, generation/fence and use budget.
3. Atomically reserve the use with idempotency binding to the complete immutable
   request digest and exact session/context/challenge generation. Same key with
   different binding is rejected; same consumed key returns the same authorized
   receipt without another retrieval or submit. Concurrent use has one winner.
4. Enter exclusive private resolution/capture quarantine BEFORE retrieval.
   Ordinary agent fill/eval/click remain denied. Deny reads, screenshots, stream,
   public share, exports, recorder/network-body/console persistence and other
   observation paths during secret use, including concurrent existing consumers.
   If any capture cannot be fenced or any recorder cannot be stopped, fail closed.
   Quarantine is held through post-login secret cleanup and output scanning.
5. Revalidate authority + exact document/frame/target origin and form destination
   at the last trusted execution boundary. A static approved provider adapter
   performs bound field input and ordinary form submission without user-supplied
   script/selectors. Cross-origin frames, changed form destinations, redirects,
   new challenges, navigation or fencing change invalidate admission before the
   next secret use. Redirect success is not account success. TOTP is renewable,
   just-in-time, challenge-bound and checked for adequate trusted-clock lifetime.
6. Independently observe fresh provider-owned authenticated account identity,
   allowlisted origin and exact context generation. Expected account comes from
   trusted authority/binding records, observed account from the adapter's approved
   fresh identity observation, never the request, HTML claims or injection return.
   A successful submit/HTTP200/challenge disappearance alone cannot resolve.
7. Recheck revocation, expiry, reservation/fence and challenge identity; obtain a
   durable external value-free use receipt, then conditionally resume the exact
   local challenge under its synchronization owner. This is NOT distributed
   atomicity; D2 below specifies crash/reconciliation and revocation-window
   requirements that must be resolved before implementation admission.
   Resume token remains server-private in this path. No call to public resolve is
   used as positive authentication proof. Existing ResolveAuthChallenge is not a
   credential boundary; its wider security disposition remains separately open.
   Proposed source integration must guard against closing, navigation, challenge
   replacement, stale timer callbacks and cancellation between observe and resume.
8. Release quarantine only after field/transient buffer cleanup and no-value
   output projections are verified. Keep authenticated context under its exact
   revocable private lease; subsequent ordinary actions need fresh authorization
   and identity posture, not an unrestricted post-login session. Cleanup failures
   retain quarantine, mark incident and revoke use; never return ordinary captures.

Any failure keeps auth pending (or closes the exact owned context after
reconciliation). An ambiguous submit records possible_use and consumes/reserves
budget according to the authority protocol; no blind retry. Cancel cannot undo a
remote login. Lost response, process crash and active cancellation reconcile by
idempotency key before any subsequent attempt. No fabricated secure-zeroization
or zero-residue claim: receipts name each checked resource and remaining gap.

## Dependency-ordered implementation packets for owner review

- P1, authority/isolation prerequisites: approve and implement the exact Focusa
  authority adapter + atomic reservation/reconciliation and private context
  allocation/generation/capacity/cleanup contract. Actual issuer stays Focusa.
  Pinned cross-repository revisions and golden value-free fixtures required.
- P2, private coordinator and consumer: admitted private IPC plus bound browser
  adapter and complete capture quarantine. Integrate with existing session
  state under the session synchronization owner; no public middleware bypass.
  Potential owning seams: internal/vision/auth_challenge.go, session lifecycle,
  existing diagnostics/snapshot/FPV/recorder paths. Exact new files/symbols must
  be enumerated in the approved IR5 packet, not silently authorized by this list.
- P3, actual retained Chromium contract suite and parent independent review:
  full positive synthetic login through the admitted dispatcher and owning
  coordinator, using the real session/challenge transitions and account verifier.
  No replica state machine or mocked-only test is end-to-end acceptance.

## Release-blocking executable acceptance required after admission

Companion matrix IDs PA01–PA15 are required retained test-source cases, not
claimed test results. Use only synthetic renewable credentials and owned local
fixtures. Positive fixture has a real form, server-side session cookie and a
fresh account endpoint; expected and wrong-account fixtures must differ. Test
through the actual proposed private dispatcher and authority adapter seam with
an explicitly test-only issuer/ledger that cannot be built/registered in release.
A fake authority stub tests the consumer boundary only; production issuer
interoperability still requires pinned owning-service receipts.

Scan byte and decoded textual representations; adversarial fixture echoes the
sentinel in DOM/title/eval/console/errors/structured output and tries cross-origin
exfiltration. For screenshots/streams, assert capture never occurred while bound
or quarantined, not merely that a text grep found nothing in PNG bytes. Inspect
process argv, engine logs, events, diagnostics, receipts, persisted artifacts and
post-close resources; report coverage limits honestly. No OCR/heap-wipe proof by
implication. Ordinary formatting and public view-only guards stay protected.

Parent must independently inspect source and reproduce positive + adversarial
suite at the exact candidate commit before any deployment. Deployment then
requires exact binary/revision provenance and installed two-worker acceptance;
source fixture passes cannot establish production/private login capability.
Live owner-site login/editor/object-removal-without-saving/discard/persistence
checks are a later separate acceptance, with no automatic customer message.

## Scope and actual verification of this proposal

This branch changes proposal docs, a machine-readable acceptance matrix, a
proposal consistency test, and supported distribution manifest stamp metadata.
The canonical scripts/stamp-uiai-version.py regenerates source_commit and
generated_at for the same cockpit-v0.3.2 version under normal pre-push policy;
no version, compatibility range, artifact payload, tag or release is changed.
The source stamp may name the ancestor at regeneration, not the final docs head.
A fresh manifest timestamp is NOT installed compatibility or deployment proof. It adds NO runtime, private endpoint, grant,
credential read, browser action, policy activation, production deployment or
claim that a website is fixed. The consistency tests verify proposal integrity,
not authentication, authorization correctness or any runtime test above.

Sources: #82 comments 6371125282, 6412611393, 6413439573; #81 and #145 current
bodies; #145 comment 5475400006; merged PR #4 (349974621e513f20115a04dc12749fa83e580b06);
C04 status proposed_documentation_only; Focusa #299 and current Spec156 §§1,
implementation readiness and atomic decomposition. Architectural adoption still
requires the exact root-authority approval; merged presence is not approval.

## Root decision packet — bounded advisory revision (NO auth runtime)

This section is the deduplicated decision input to PR291 / UIAI #145 / Focusa
#299 and #294, not a replacement epic, approved Call Stack Design, IR5 packet,
or instruction to #82's protected uiai-engine-zsjm writer. Sir V3 is the exact
promotion owner; the parent may do reversible engineering and packet preparation.
Each recommendation below is **advisory_external, owner_decision_required**.
Accepting a recommendation for design completion does NOT admit implementation.

### Pinned source seams and limits of evidence

UIAI references below are relative to immutable base
[1c0a2d3a5a2724fcad7dde7981c2a0be053316c9](https://github.com/WPUIAI/uiai-engine/tree/1c0a2d3a5a2724fcad7dde7981c2a0be053316c9).
Focusa references are relative to
[8cfb52eac5fdc34d14b80f8e6c51e7e6ab06a29a](https://github.com/Startempire-Wire/focusa/tree/8cfb52eac5fdc34d14b80f8e6c51e7e6ab06a29a).
Paths/symbols are source seams, not claims of implemented private consumers.

- S1: UIAI internal/routes/session.go, MountSessionRoutes (lines 86–143): GET,
  DELETE, auth/resolve and auth/cancel bypass the pending-auth mutation guard.
  internal/vision/auth_challenge.go, ResolveAuthChallenge/CancelAuthChallenge
  (lines 140–186): token-only latch/timer/event transition under s.mu, with no
  external authority/account check. CurrentAuthChallenge and events expose the
  legacy resume token; proposed private projection must not reuse that disclosure.
- S2: UIAI internal/vision/session.go, openOnceScoped/closeSession/Screenshot/
  ScreenshotFull/ReadPageContext (lines 288, 414, 530, 572, 1116), and
  internal/vision/pool_pages.go, createPoolPage/closePoolPage: existing pool pages
  already have independently disposable Incognito storage contexts. That is
  storage separation, NOT a private credential-consumer admission.
- S3: UIAI internal/browserprofile/manager.go, Manager.Open/reserveProfileLock/
  Close; launcher.go, Launch/Runtime; session_access.go, Manager.SessionPage:
  managed profile/runtime lifecycle and internal raw-page access exist. Neither
  the name 'operator' nor SessionPage is a credential authorization primitive.
- S4: UIAI internal/vision/session_screencast.go, CDPScreencast,
  screencastHub.broadcast/stop; internal/routes/fpv.go, MountFPVPublicRoutes/
  fpvScreenshotJPEG; internal/routes/fpv_stream.go, fpvWriteStream: existing
  long-lived producers, buffered recipients and route-level projections.
- S5: UIAI internal/vision/diagnostics.go, initDiagnostics/recordConsole/
  recordException/recordRequest/recordResponse/DiagnosticsWithOptions;
  internal/routes/epwa_delivery.go, writeSessionSnapshot/publishSessionSnapshotEPWA;
  internal/routes/vision_interactive.go, screenshotPNG/storeCapture: capture,
  asynchronous recorders, publication and persistence owners, not a proven fence.
- S6: UIAI internal/vision/pool.go, ValidateNavigationURL; launcher
  configuration in internal/browserprofile/launcher.go, BuildChromiumLauncher:
  navigation validation, proxy and tracker settings do NOT prove complete
  hostile-page request egress control. No selected private egress enforcement
  primitive or approved trusted-account adapter was found in the inspected seams.
- S7: UIAI docs/ARCHITECTURE_AUTHORITY_POLICY.md §§1–4,6,8; Focusa
  docs/156-focusa-project-scoped-credential-authority-secret-broker-delegated-autonomy-mfa-totp-and-cross-surface-injection-spec.md
  lines 19–39,56–60,971–1015: authority/custody/use separation, no recovery codes,
  IR0, approved typed design and dependency-ordered #294 IR5. Issue #145
  comment 5475400006 requires versioned spec first; #294 comment 5296742792
  requires exact readiness/injection/account/origin/recovery/test bindings.
  Exact Focusa private reserve/consume API signatures, schema revisions,
  issuer configuration and human implementation assignees remain UNKNOWN here.

### D1 — private IPC / principal mapping / bounded schema

**Alternatives:** (A) node-local Unix-domain socket with OS peer credentials plus
owning principal/consumer registration and current Focusa scope checks;
(B) supervisor-created inherited socketpair/FD per admitted consumer, additionally
bound to its registration/generation; (C) loopback HTTP with bearer token.
Recommend **A for a Linux-only first design**, with B retained if existing
supervisor ownership can be demonstrated. C is not recommended: listener/token
reuse expands exposure and resembles the forbidden public-route exception.
Windows/named-pipe and remote consumers remain unsupported, not silently mapped.
S1 supplies the route seam to keep separate; S7 owns admission. There is no
existing private dispatcher in these sources to claim reused authentication.

**Recommended candidate contract, NOT deployed schema:** one versioned,
length-delimited UTF-8 JSON message per frame; reject duplicate keys, unknown
fields, invalid UTF-8, trailing data and unsupported versions before lookup.
Propose 16 KiB frame cap, refs of 1–256 ASCII bytes matching
[A-Za-z0-9][A-Za-z0-9._:-]*, positive bounded 64-bit generation; operation names
remain intents pending #258 conflict review. Proposed limits: four outstanding
requests per registered peer, one active resolution per context, five-second
read deadline, 30-second dispatch deadline. These values are review choices,
not existing service limits or permission to time out an uncertain side effect.
Deadline expiry after use enters reconciliation; it never releases budget.
No secret belongs in framing/errors/status or unknown-field diagnostics.

Begin retains only the listed refs/generation/idempotency key; version and
operation are protocol envelope fields, not arbitrary Begin input. Status and
Cancel candidate bodies contain ONLY resolution_ref and lease_generation;
responses contain opaque resolution_ref, value-free state, bounded typed error
code and authorized receipt_ref, never grant material, tokens or raw provider
errors. Exact state/error enums must be frozen in the admitted owning schema;
this proposal deliberately does not invent live operation signatures.

Socket directory/socket proposal: service-owned 0700/0600, no shared /tmp
listener, no inherited world/group access; verify owner and peer UID/GID/PID
using platform peer credentials and protected service registration. UID alone
is not actor identity. Map peer service + launch generation to immutable
consumer registration, then resolve actor/work-surface/operation via owning
records; never trust caller actor or a PID string. PID reuse, restart, stale
FD, shared UID without disambiguating registration, unknown principal and
cross-principal Status/Cancel fail closed. Restarts revoke old connection
bindings; outstanding operations reconcile durably, never infer unused.
Resource exhaustion rejects before provider access. No FD secret delivery
format is selected: custody owner must define ownership/close/timeout/cleanup
and prohibit argv/env/log/model transit. Tradeoff: small Linux scope and strong
local containment versus platform limitation and explicit registration burden;
OS root compromise is outside peer-credential protection, not solved by it.

**Admission prerequisites:** rooted D1 choice; exact dispatcher file/symbol,
platform/registration owner, byte schema/enums and golden malformed/oversized/
duplicate/impersonation/restart/status/cancel fixtures; #258 disposition; rate
and resource policy. All remain requirements, not runtime changes here.

### D2 — external reservation / durable consume / local resume

**Alternatives:** (A) Focusa-owned durable reserve/use/reconcile ledger plus
UIAI durable intent/outcome journal and fail-closed reconciliation;
(B) distributed two-phase commit spanning authority and local browser state;
(C) local mutex plus best-effort remote receipt. Recommend **A**. B cannot roll
back a browser form submission and requires a new distributed protocol/owners;
C loses budget/account/fence truth on crash and is rejected. S1 has only a
local mutex; S7 reserves external authority. No existing external API names,
signatures, signed objects or atomic commit guarantees are supplied by A.

Proposed protocol requirements (state labels here are conceptual):
1. Before retrieval, external authority durably reserves the exact immutable
   request digest, peer/principal/context/challenge/frame/account bindings,
   authority revision, lease generation and fencing epoch with one idempotency
   key. UIAI persists the corresponding intent. Duplicate identical requests
   reconcile; conflicting keys/bindings reject. Offline authority is denial.
2. Reservation expires without use ONLY when the owning ledger proves no-use.
   Before any delivery/input/submit, persist use-start intent; any uncertainty
   from then onward is possible_use, never automatic expiry/free-budget retry.
   Provider retrieval itself may consume limited use: exact provider accounting
   point is UNKNOWN until the authority/custody packet defines it.
3. After bound submit and fresh trusted account observation, request durable
   external use/outcome receipt. Local resume may occur only on independently
   validated receipt binding the exact digest, observation freshness, current
   authorization/fence and local still-pending challenge generation.
4. Do not hold s.mu across network calls. Recheck under the local synchronization
   owner before compare-and-transition; persist outcome and receipt reference
   for recovery. A local resume and external consume are NOT a distributed
   atomic transaction. No exactly-once remote login claim is made.
5. A revocation check and resume have a race. The approved authority contract
   must define a linearization point or bounded authorized window/lease plus
   max clock skew and propagation budget, and immediate local fencing on known
   revocation. No numeric safety window is inferred here. If no accepted window
   can be established, do not resume. Later revocation fences an authenticated
   lease as well; login success does not create perpetual authority.

| Crash / race window | Required recovery, never blind retry |
|---|---|
| External reserve succeeded, reply/local intent lost | Query authoritative binding by same key; do not allocate another key/use |
| Intent durable, no delivery known | Owning ledger/custodian proves no-use before release; otherwise possible_use |
| Delivery/input/submit in flight or response lost | Quarantine; possible_use; account observation/reconcile owner decides; no second submit |
| Account observed, consume pending/ack lost | Reconcile external receipt/revocation/fence by exact binding; observation may expire |
| Consume durable, local resume not durable | No new consume/input; re-observe if needed, validate current authority then local compare-and-transition |
| Resume durable, reply/event lost | Authorized status returns same receipt; deduplicate events, never repeat remote login |
| Revoke/expire/cancel/close/restart at any window | Fence exact generation, preserve quarantine, reconcile use and clean owned context; cancellation cannot undo remote login |

UIAI owns local journal, challenge/lease/capture lifecycle; Focusa owns grant,
reservation/use budget, receipt semantics, revocation and canonical reconcile;
custodian owns delivery/use evidence. Human assignment, storage, retention,
query availability, receipt format/trust roots and protocol revision are
UNKNOWN. #299/#294 must name them, including authority outage and orphan reaper.
Security tradeoff: conservative budget loss/held quarantine and lower
availability versus pretending a distributed/browser transaction is atomic.
**Admission prerequisites:** exact pinned external operations/schema/trust,
clock/revocation contract, durable state/error/transition fixtures at every
window, ownership/rollback/reaper and dependency-ordered IR5; no local issuer.

### D3 — independent context / trusted fresh provider account

**Alternatives:** (A) independently allocate a dedicated private Incognito
context using S2's low-level disposal pattern, NEVER promote/check out an
ordinary pool session; (B) dedicated managed ephemeral browser process/profile
using S3's lifecycle with a private storage/launch policy; (C) reuse operator
persistent profile or challenged pool page. Recommend **B for the first
security-sensitive bounded design**, with A a later capacity optimization only
after same-process capture/CDP/egress separation is proven. C is rejected:
ambient cookies/account confusion and pooled promotion violate admission.
B costs memory/startup/capacity and still does NOT solve hostile-page egress,
raw CDP access or account verification merely by process separation.

Require new private session identity, private process/context/target IDs,
monotonic generation on allocation/restart, one admitted holder, explicit per-node
capacity and reject-before-retrieval on exhaustion; no fallback to shared pool
or operator profile. Bind cleanup to exact owned process/context/temporary
profile, include popups/workers/downloads/cache/crash artifacts and verify no
foreign profile closed. S3's runtime and SessionPage need capability-scoped
access/fencing; raw-page getter availability is not proof of private safety.
Sandbox/launch flags, debugging endpoint ownership, proxy/DNS policy, secrets
in temp storage and crash reporting require specific inspected choices.

Trusted account alternatives: provider-supported authenticated account endpoint
read by a privileged adapter in the exact browser context, with HTTPS origin,
response provenance, no-cache/challenge-bound freshness and provider-defined
stable subject+tenant normalization; or a provider-owned signed identity
assertion with approved issuer/audience/nonce/time verification. Recommend the
**provider-supported authenticated endpoint when one is documented and can be
bound to the exact context**; otherwise keep that provider unsupported. An
endpoint response exposed through page JS, service-worker/cache replay, DOM
text, generic WordPress success URL, HTTP200, injected account claims or form
submit return is not trusted evidence. Canonical expected account/tenant comes
from Focusa binding, not caller. Missing/ambiguous normalization, unavailable
endpoint, wrong account/tenant, stale response, context change or redirect all
keep pending. Exact provider endpoint/adapter revision, authenticated transport,
service-worker bypass/provenance, freshness bound and tenant mapping remain
UNKNOWN; do not infer a WordPress route or claim Western login supported.
**Admission prerequisites:** rooted primitive choice, profile/capacity/disposal
policy, trusted-provider contract, exact identity/tenant/error fixtures and
private/private plus pooled/private storage and capture/egress regressions.

### Exact unresolved capture and egress owner inventory

The following enumerates inspected source owners and the missing enforcement
contract. Component owner is not a named assigned human; all human assignments
are UNKNOWN pending #145/#299/#294 coordination. This is not an estate-wide
complete audit. Every reachable owner must acknowledge a fence before retrieval.

| ID | Existing owner / source seam | Required unresolved contract |
|---|---|---|
| C1 | vision Session Screenshot/ScreenshotFull/ReadPageContext/DOMInfo and snapshot.go | Block direct read/capture and action-returned snapshots; prevent in-flight completion/publication after fence |
| C2 | vision CDPScreencast/screencastHub and routes fpvWriteStream/fpvScreenshotJPEG | Stop existing producers; discard buffered frames; invalidate subscribers; acknowledge drain before use, no reattach on legacy resolve |
| C3 | vision initDiagnostics/recordConsole/recordException/recordRequest/recordResponse | Stop/drain asynchronous callbacks before use; isolate preexisting buffers and prevent late append; no response/body/value persistence |
| C4 | routes writeSessionSnapshot/publishSessionSnapshotEPWA, screenshotPNG/storeCapture; evidenceartifact Store.OpenAsset | Fence publication/export/share and queued writers; inventory prior artifacts/retention/revocation, not claim deletions without receipt |
| C5 | browserprofile Manager.SessionPage/Runtime; vision session_eval.go; pool Screenshot | Enumerate all raw-page/CDP/alternate pool consumers; deny attachment/use of private target; dedicated process alone is insufficient |
| C6 | vision auth_challenge.go events; routes sessionInfoPayload/writeSessionError; observability ErrorStore.Record | Value-free challenge/status/event/error/log projection; legacy token exposure not private proof; no payloads in diagnostics |
| E1 | pool ValidateNavigationURL; browserprofile BuildChromiumLauncher proxy/tracker config | Select actual pre-input network enforcement owner; navigation checks are not fetch/XHR/beacon/WebSocket/DNS/subresource policy |
| E2 | private target/frame/document lifecycle + provider form adapter (not implemented) | Before input, bind trusted document/frame/form and restrict destinations, redirects, popups, workers/service-workers/downloads; reject on policy change |
| E3 | privileged provider account adapter (not selected) | Exact endpoint/provenance/cache/service-worker/tenant/freshness contract separate from hostile page; block page-mediated observation |

A same-origin malicious script can read a password field and leak through an
otherwise allowed destination. Form.action checking, tracker blocking, output
redaction, or a destination allowlist alone does NOT solve this. The provider
adapter threat model must either justify trusted credential-entry document and
strict request/method/path/body/initiator constraints, or select a provider-owned
non-page delivery mechanism; unsupported sites remain denied. Allowed login
requests themselves carry credentials: complete 'no credential egress' is false.
Require synthetic hostile input listeners, fetch/XHR/beacon/WebSocket,
redirect/popups/worker/service-worker/cached response and pending-request
fixtures with actual network-observation receipts. Exact enforcement primitive
and deployment owner remain UNKNOWN, mandatory before runtime admission.

### Concurrent public-resolve cases (required, NOT executed)

All cases use the real MountSessionRoutes/ResolveAuthChallenge, session
synchronization and private fence owners after admission. Existing token-only
semantics are not changed by this proposal. A separate generation-bound private
fence must not be derived solely from authChallenge != nil; exact changes are
for admitted packet review, not an HTTP bypass here.

- R1: public resolve wins before private reservation/quarantine: stale challenge
  cannot retrieve; private Begin must reject/reconcile without effects.
- R2: resolve between reservation and quarantine acknowledgement: no retrieval
  until all owners fence exact generation; invalidated binding reconciles use.
- R3: resolve with valid legacy token during retrieval/input/submit: cannot
  resume ordinary mutations, release captures or authorize a second consumer.
- R4: resolve after account observation while consume pending/ack lost: no
  private resume without durable matching receipt and current authority window.
- R5: duplicate public resolve after private local resume, including nil challenge:
  idempotent legacy success cannot release fence twice or become account proof.
- R6: resolve races cancel/close/expiry/navigation/replaced challenge/restart:
  old token/timer cannot affect new generation; only exact owned context cleanup.
- R7: already-open GET/read/stream/share and ordinary action queued before resolve:
  no buffered/late capture or guard time-of-check/time-of-use escape during use.
- R8: foreign principal guesses/obtains legacy token or private resolution_ref:
  no private status/cancel/use/account proof; exact peer authorization persists.

### Compact owner choice and admission checklist

Sir V3: consider **D1-A Linux registered-peer IPC; D2-A external durable ledger
plus local recovery journal (no distributed atomicity claim); D3-B dedicated
ephemeral profile/process plus documented trusted account endpoint or deny**.
The requested decision is promotion of these exact bounded design directions
for existing #145/#299/#294 decomposition, NOT a credential grant, API approval,
runtime admission, root signature or production release. Alternatives and costs
above remain explicit; no answer is inferred from repository/account identity.

Before any implementation, the owning #294/Spec103/#295 typed design must bind:
exact rooted decision/provenance; admitted scope and forbidden effects; actual
human/component owners; pinned repos/revisions/files/symbols and API schemas;
issuer/trust roots and peer mapping; immutable request/account/origin/context
bindings; exact states/errors/transitions/time/revocation/crash rules; C1–C6 and
E1–E3 acknowledged inventory; R1–R8 plus PA01–PA15 executable golden fixtures;
capacity/retention/cleanup/rollback/reconcile owners; dependency graph and
no-unresolved-mandatory-fields IR5 receipts. P1/P2/P3 above are coarse workstreams,
NOT those atomic packets. #258 owns name conflict review; #82 stays protected.

- [x] Advisory alternatives, recommendations and inspected seams enumerated.
- [ ] Exact root promotion and typed design / dependency-ordered IR5: NOT DONE.
- [ ] Exact IPC/Focusa/isolation/provider/capture/egress contracts: unresolved.
- [ ] Auth runtime and retained Chromium PA01–PA15/R1–R8: NOT DONE.
- [ ] Parent independent exact-head review; installed two-worker acceptance: open.
- [ ] Original website login/editor/remove WITHOUT saving/discard/persistence:
  NOT DONE. Neither this packet nor proposal tests can close that original gate.
