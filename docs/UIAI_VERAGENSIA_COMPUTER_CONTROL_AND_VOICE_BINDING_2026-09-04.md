# UIAI Engine × Veragensia Computer Control, Voice, and Ambient Operator Binding

**Status:** Cross-system architecture binding, revised 2026-09-05; implementation status remains governed by each referenced UIAI/Veragensia/Focusa contract.
**Canonical human architecture authority:** Verious Smith III under `docs/ARCHITECTURE_AUTHORITY_POLICY.md`.
**Focusa companions:** Specs 181–184 for Conversation, Project Foreman, Radar, and Ambient Operator.
**Veragensia companion:** Doc 199 for Companion sync/Omarchy integration in addition to Docs 193–197.

## 1. Purpose

UIAI Engine remains the canonical first-party browser/computer execution and proof surface inside a full Veragensia Agent Computer.

Veragensia defines broader OS primitives for:

- machine execution enforcement;
- trusted human control and Secure Attention;
- generalized desktop observations;
- stable ResourceRefs/runtime incarnations;
- platform/workload attestation;
- voice-native operation;
- paired mobile/wearable Ambient Operator integration.

Focusa defines:

- Voice/Conversation lineage;
- Project Foreman Workstream intelligence;
- Radar proactive Signals/Episodes;
- Ambient Operator semantic/presence/sync contracts.

This document binds those primitives to existing UIAI contracts without creating a second UIAI browser/control authority, second Foreman, or second Radar.

## 2. Ownership

```text
Focusa
    current ask · Workstream/Workpoint · authority
    Conversation Ledger · ExpressionOutput · settlement
    Project Foreman · Radar · Ambient Operator semantics

Veragensia
    OS enforcement · secure attention · audio devices
    desktop observation composition · runtime incarnation
    human-control reserve · Companion sync · voice-native experience

UIAI Engine
    browser/computer execution · browser observations
    actuator control · diagnostics · proof · execution capsules
```

A voice surface, mobile Companion, Veragensia shell, Focusa Desktop, Radar projection, or Foreman UI must not reimplement UIAI browser action semantics.

## 3. Voice/mobile request path

Voice and mobile commands follow:

```text
human speech/mobile intent
→ Focusa Conversation utterance/current ask
→ Wirebot or exact Project Foreman routing
→ canonical Focusa operation / authority
→ Veragensia chooses execution surface
→ UIAI capability / browser-computer action
→ UIAI observation/evidence/diagnostics
→ Focusa reconciliation/Receipt
→ Focusa ExpressionOutput
→ Veragensia/mobile spoken or visual response
```

Voice, phone, earbuds, Radar, and Foreman do not receive special UIAI permission classes.

## 4. Project Foreman binding

Focusa Spec 182 owns the Project Foreman as a persistent Workstream-scoped project-intelligence role projection.

UIAI requests MAY carry:

```yaml
workstream_ref:
foreman_ref:
workpoint_ref:
utterance_ref:
```

for provenance/scope correlation.

UIAI MUST NOT:

- construct a new Foreman from browser history;
- treat a browser session as the Foreman's memory;
- grant browser capability because a caller claims `foreman_ref`;
- persist a competing project-intelligence store.

Browser observations and execution results return to Focusa as evidence/operational facts for the Foreman to consume.

## 5. Radar observation bridge

Focusa Spec 183 Radar may consume **existing UIAI structured observations, diagnostics, execution capsules, verification results and Evidence refs**.

Preferred path:

```text
UIAI native event / BrowserObservation / diagnostics
→ bounded Focusa observation adapter
→ RadarObservation
→ fingerprint / Episode / Signal
```

Rules:

- UIAI remains source authority for the browser observation itself;
- Radar stores source refs/digests and bounded interpretation, not a competing browser snapshot truth;
- use structured events/semantic state before screenshots/vision;
- do not add broad polling or screenshot loops merely because Radar exists;
- UIAI content/influence firewall remains active before any external page/tool content can affect action.

## 6. Control lease reuse

UIAI already defines `uiai.cockpit_operator_control_lease_takeover_reconciliation.v1` with:

- one active holder per actuator scope;
- lease generation;
- fencing token;
- exact work-surface/session binding;
- immediate local safety freeze;
- local freeze distinct from Focusa canonical pause;
- OperatorDelta receipt;
- mandatory re-observation before agent resume;
- credential-grant revalidation;
- stale/context-change blocking.

Veragensia Doc 194 generalizes those semantics to application/full-desktop control.

**UIAI remains primitive owner for UIAI actuator leases.** Veragensia may project/compose them into a broader `ComputerControlLease` but MUST NOT loosen their fencing or freshness requirements.

## 7. General desktop observation binding

UIAI browser observations remain browser-runtime authoritative.

Veragensia `DesktopObservation` may reference them:

```yaml
veragensia_desktop_observation:
  surface_ref: browser_window
  runtime_incarnation_ref:
  uiai_observation_ref:
  stream_coordinate_space_ref:
  compositor_coordinate_space_ref:
```

The Veragensia projection does not rewrite UIAI document/navigation/frame identity.

For non-browser applications, Veragensia applies analogous observation-bound semantics without pretending UIAI owns those applications' native object models.

## 8. Observation-bound actions

Voice/mobile/Foreman requests never reduce an action to an unguarded coordinate click when UIAI has a stronger ref.

Preferred UIAI path:

```text
current UIAI observation
→ semantic/@ref target
→ observation-bound action
→ runtime revalidation
→ action
→ verification/diagnostics
```

If Veragensia computer-use fallback supplies coordinates, they remain bound to explicit surface/observation/coordinate-space generation under Doc 194.

## 9. Human voice/mobile takeover

Examples:

```text
"Stop moving the mouse."
"Give me control of the browser."
"Pause this task."
"I selected the right account; continue."
```

Mobile equivalents may use trusted UI/touch controls.

Binding:

```text
spoken/mobile intent
→ Focusa intervention intent
→ immediate UIAI local safety freeze when needed
→ control lease generation/fencing transition
→ operator control
→ UIAI/Veragensia OperatorDelta
→ Focusa reconciliation
→ new UIAI/desktop observation
→ authority + credential refresh
→ resume / redirect / block / stop
```

The mere end of human input is not permission for an agent to resume.

## 10. Mobile FPV / observation

An authorized Ambient Operator Companion MAY receive a bounded UIAI FPV or observation projection when remote visual context materially helps the human.

That path requires:

- exact user/device pairing;
- exact UIAI session/surface scope;
- short-lived view/control authorization;
- privacy/redaction policy;
- current runtime incarnation;
- distinct `view` vs `control` capabilities;
- normal control lease for actuation.

A share/view token does not become general UIAI execution permission.

## 11. Voice and browser provenance

A user must later be able to traverse:

```text
spoken/mobile instruction
→ Focusa utterance_ref
→ Foreman/Workstream ref
→ action_proposal_ref
→ UIAI observation/action/execution capsule
→ browser evidence
→ Focusa Receipt/outcome
→ spoken/visual response
```

UIAI does not own or store the canonical human/agent Conversation Ledger merely because it produced browser evidence during the conversation.

## 12. Voice/audio boundary

UIAI Engine does not become the general microphone/TTS service for Veragensia/Ambient Operator.

- Veragensia/mobile OS owns trusted audio-device integration.
- Focusa owns conversation/utterance/expression semantics.
- UIAI may interact with webpages/applications containing audio or media under normal capability rules.
- Microphone use by a browser page remains a browser/device capability and does not inherit the trusted Voice/Conversation service grant.
- A website or remote audio stream cannot impersonate the trusted Veragensia/Focusa voice merely by sounding similar.

## 13. Influence firewall generalization

UIAI already treats page/tool/browser content as untrusted influence.

Veragensia/Focusa voice and Ambient integration preserves this law:

- webpage audio/text;
- page-generated speech;
- WebMCP descriptions/results;
- downloads;
- remote participants;
- content transcribed from a meeting;

may inform reasoning or become Radar/Conversation candidates but cannot expand capability, grant credentials, approve an action, redefine the operator's spoken command, or suppress Evidence.

## 14. Runtime incarnation

Browser contexts/targets/documents are already generation/version sensitive. Veragensia Doc 195 adds Agent Computer runtime-incarnation boundaries.

After browser/session/runtime replacement:

- stale UIAI refs remain invalid;
- control lease revalidates;
- Veragensia DesktopObservation refreshes;
- credential posture refreshes;
- mobile FPV/view refs refresh;
- voice conversation may continue through Focusa ledger refs but does not revive stale actuator authority.

## 15. Enforcement relationship

Veragensia Doc 193 controls OS/container/device/network access available to the UIAI worker/runtime.

UIAI's own product entitlement, session/origin/operation authorization and safety policies remain separately enforced.

```text
Veragensia machine capability
!=
UIAI product entitlement
!=
Focusa operation authority
```

All applicable gates must pass.

## 16. Ambient Operator acceptance

Cross-system closure requires proof that:

1. a voice/mobile request invokes normal UIAI capability/authority paths rather than a modality-specific bypass;
2. spoken/mobile takeover causes correct lease/fencing transition;
3. operator changes force fresh UIAI observation before resume;
4. stale browser refs cannot survive Veragensia runtime replacement;
5. UIAI evidence remains traceable to originating Focusa utterance, Workstream/Foreman and final Receipt;
6. browser-page audio cannot impersonate secure Veragensia approval;
7. trusted Veragensia/mobile microphone capability does not automatically expose microphone access to browser pages;
8. Radar can consume bounded UIAI events/Evidence without a duplicate browser watcher;
9. mobile FPV view authority cannot escalate into control/execution;
10. voice-complete UIAI browser tasks can be completed without keyboard/pointer while preserving normal UIAI verification and safety.

## 17. Final principle

> **Voice and Ambient Operator change how and where the human asks. Foreman changes which Workstream intelligence answers. Radar changes what gets noticed. None changes what UIAI must prove before it acts.**
