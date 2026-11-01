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

Recommended decision: accept this bounded password/renewable-TOTP-only design
for decomposition, then admit the three dependency-ordered packets below.
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
7. Recheck revocation, expiry, reservation/fence and challenge identity; record a
   value-free use receipt and authorized result atomically with challenge resume.
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

This branch changes proposal docs, a machine-readable acceptance matrix and a
proposal consistency test only. It adds NO runtime, private endpoint, grant,
credential read, browser action, policy activation, production deployment or
claim that a website is fixed. The consistency tests verify proposal integrity,
not authentication, authorization correctness or any runtime test above.

Sources: #82 comments 6371125282, 6412611393, 6413439573; #81 and #145 current
bodies; #145 comment 5475400006; merged PR #4 (349974621e513f20115a04dc12749fa83e580b06);
C04 status proposed_documentation_only; Focusa #299 and current Spec156 §§1,
implementation readiness and atomic decomposition. Architectural adoption still
requires the exact root-authority approval; merged presence is not approval.
