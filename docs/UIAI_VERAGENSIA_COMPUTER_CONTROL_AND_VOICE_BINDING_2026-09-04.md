# UIAI Engine × Veragensia Computer Control and Voice Binding

**Status:** Cross-system architecture binding, 2026-09-04; implementation status remains governed by each referenced UIAI/Veragensia contract.
**Canonical human architecture authority:** Verious Smith III under `docs/ARCHITECTURE_AUTHORITY_POLICY.md`.

## 1. Purpose

UIAI Engine remains the canonical first-party browser/computer execution and proof surface inside a full Veragensia Agent Computer.

Veragensia now defines broader OS primitives for:

- machine execution enforcement;
- trusted human control and Secure Attention;
- generalized desktop observations;
- stable ResourceRefs/runtime incarnations;
- platform/workload attestation;
- voice-native operation.

This document binds those primitives to existing UIAI contracts without creating a second UIAI browser/control authority.

## 2. Ownership

```text
Focusa
    current ask · project/Workpoint · authority
    Conversation Ledger · ExpressionOutput · settlement

Veragensia
    OS enforcement · secure attention · audio devices
    desktop observation composition · runtime incarnation
    human-control reserve · voice-native experience

UIAI Engine
    browser/computer execution · browser observations
    actuator control · diagnostics · proof · execution capsules
```

A voice surface, Veragensia shell, or Focusa Desktop must not reimplement UIAI browser action semantics.

## 3. Voice request path

Voice commands follow:

```text
human speech
→ Focusa Voice/Conversation utterance/current ask
→ canonical operation / authority
→ Veragensia chooses execution surface
→ UIAI capability / browser-computer action
→ UIAI observation/evidence/diagnostics
→ Focusa reconciliation/Receipt
→ Focusa ExpressionOutput
→ Veragensia spoken response
```

Voice does not receive a special UIAI permission class.

## 4. Control lease reuse

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

## 5. General desktop observation binding

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

## 6. Observation-bound actions

Voice or other modality requests never reduce an action to an unguarded coordinate click when UIAI has a stronger ref.

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

## 7. Human voice takeover

Examples:

```text
"Stop moving the mouse."
"Give me control of the browser."
"Pause this task."
"I selected the right account; continue."
```

Binding:

```text
spoken utterance
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

## 8. Voice and browser provenance

A user must later be able to traverse:

```text
spoken instruction
→ Focusa utterance_ref
→ action_proposal_ref
→ UIAI observation/action/execution capsule
→ browser evidence
→ Focusa Receipt/outcome
→ spoken response
```

UIAI does not own or store the canonical human/agent Conversation Ledger merely because it produced browser evidence during the conversation.

## 9. Voice/audio boundary

UIAI Engine does not become the general microphone/TTS service for Veragensia.

- Veragensia owns trusted OS audio-device integration.
- Focusa owns conversation/utterance/expression semantics.
- UIAI may interact with webpages/applications containing audio or media under normal capability rules.
- Microphone use by a browser page remains a browser/device capability and does not inherit the trusted Voice/Conversation service grant.
- A website or remote audio stream cannot impersonate the trusted Veragensia/Focusa voice merely by sounding similar.

## 10. Influence firewall generalization

UIAI already treats page/tool/browser content as untrusted influence.

Veragensia/Focusa voice integration preserves this law:

- webpage audio/text;
- page-generated speech;
- WebMCP descriptions/results;
- downloads;
- remote participants;

may inform reasoning but cannot expand capability, grant credentials, approve an action, redefine the operator's spoken command, or suppress Evidence.

## 11. Runtime incarnation

Browser contexts/targets/documents are already generation/version sensitive. Veragensia Doc 195 adds Agent Computer runtime-incarnation boundaries.

After browser/session/runtime replacement:

- stale UIAI refs remain invalid;
- control lease revalidates;
- Veragensia DesktopObservation refreshes;
- credential posture refreshes;
- voice conversation may continue through Focusa ledger refs but does not revive stale actuator authority.

## 12. Enforcement relationship

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

## 13. Acceptance

Cross-system closure requires proof that:

1. a voice request invokes normal UIAI capability/authority paths rather than a voice-only bypass;
2. spoken takeover causes correct lease/fencing transition;
3. operator changes force fresh UIAI observation before resume;
4. stale browser refs cannot survive Veragensia runtime replacement;
5. UIAI evidence remains traceable to the originating Focusa utterance and final Receipt;
6. browser-page audio cannot impersonate secure Veragensia approval;
7. trusted Veragensia microphone capability does not automatically expose microphone access to browser pages;
8. voice-complete UIAI browser tasks can be completed without keyboard/pointer while preserving normal UIAI verification and safety.

## 14. Final principle

> Voice changes how the human asks. It does not change what UIAI must prove before it acts.
