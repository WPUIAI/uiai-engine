# UIAI Engine

[![Browser Reliability](https://github.com/WPUIAI/uiai-engine/actions/workflows/browser-reliability.yml/badge.svg)](https://github.com/WPUIAI/uiai-engine/actions/workflows/browser-reliability.yml)
![Go](https://img.shields.io/badge/go-1.25.5-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-BSL%201.1-blue)
![Source Available](https://img.shields.io/badge/source-available-brightgreen)
![Commercial License](https://img.shields.io/badge/commercial%20license-available-purple)
[![Focusa](https://img.shields.io/badge/Focusa-powered-7c3aed)](https://github.com/Startempire-Wire/focusa)

UIAI Engine is an **agent-first browser and proof backend**. It provides persistent browser sessions, screenshots, page reads/snapshots, selector and `@ref` actions, diagnostics, public-source research, media/proof artifacts, Pi/MCP integration, and Focusa evidence handoff.

> **Evaluator/customer distribution status:** Current code still permits selected unauthenticated loopback execution and reconstructs broad features from tier/token state. Those are legacy implementation facts, not the approved licensing model. New evaluator/customer distribution is release-blocked until every execution path requires an authority-issued signed `uiai-engine` entitlement or a narrower verified child token.

## Product responsibility

UIAI Engine owns browser/search/session/media/diagnostics execution, browser/computer actuator behavior, observation freshness, and stable evidence handles. [Focusa](https://github.com/Startempire-Wire/focusa) owns ProjectIdentity, Workpoints, Trajectory, evidence linkage, predictions, metacognition, continuation, recovery, and Voice/Conversation semantic lineage. Veragensia owns OS-level execution enforcement, secure attention, general desktop composition, audio-device integration, resource/runtime incarnation, and the voice-native Agent Computer experience.

### Veragensia Agent Computer composition

Within a supported **full Veragensia Agent Computer profile, UIAI Engine + Cockpit/browser/computer surfaces are deliberately listed canonical first-party defaults** for browser/computer execution, observation, diagnostics, operator oversight, control, and proof. UIAI is not merely one optional browser candidate in that composition.

The other deliberately listed first-party defaults are Focusa daemon/core, Focusa Desktop, Pi + the Focusa Pi extension as the reference/default Focusa-aware harness, Veragensia enforcement/control substrate, Voice/Conversation service bound to Focusa Spec 181, and Veragensia session/shell integration. Those surfaces retain their own responsibilities; listing them together does not merge their authority domains.

Veragensia SHOULD reuse UIAI's existing Agent-First Browser contracts—compact capability discovery, versioned observations, observation-bound actions, semantic references, Focusa-directed verification, provenance/influence controls, execution capsules, Cockpit/FPV oversight, control-lease fencing, and visual computer-use fallback—rather than creating a parallel browser automation authority.

Cross-system binding: [`docs/UIAI_VERAGENSIA_COMPUTER_CONTROL_AND_VOICE_BINDING_2026-09-04.md`](docs/UIAI_VERAGENSIA_COMPUTER_CONTROL_AND_VOICE_BINDING_2026-09-04.md).

Authentication and entitlement are separate:

- loopback changes network-auth risk only;
- local API, extension, Pi, MCP, Cockpit, webhook, Focusa pairing, Veragensia root, or voice input do not create product permission;
- a healthy browser pool does not prove a license;
- source access is not an authority-issued Evaluation;
- explicit product/features/time/node/sequence/limits must be verified before execution.

## Voice-native relationship

A full Veragensia `voice_complete` profile may initiate UIAI browser/computer work entirely by natural conversation.

Canonical path:

```text
human speech
→ Focusa utterance/current ask
→ Focusa authority / canonical operation
→ Veragensia execution routing
→ UIAI observation/action/verification
→ Evidence / Focusa settlement
→ Focusa ExpressionOutput
→ Veragensia spoken response
```

**Voice changes the human interface; it does not create a parallel UIAI permission or execution path.**

Examples such as:

```text
"Open GitHub and tell me whether CI passed."
"Fill this form from the customer record but do not submit it."
"Give me control of the browser."
"Submit now."
```

must preserve the same UIAI observation/version checks, provenance/influence firewall, entitlement, control lease, evidence and postcondition verification as non-voice operation.

UIAI does not become the canonical microphone/ASR/TTS or Conversation Ledger owner. Browser-page microphone/media capability is separate from the trusted Veragensia Voice/Conversation capture service.

## Main capabilities

- Persistent browser sessions with navigation, read, snapshot, click, fill, type, select, keypress, CSS, JavaScript, viewport, cookies, screenshots, and diagnostics.
- Provider-neutral search and Source-to-Markdown research flows.
- Console, exception, network, failed-request, and structured engine/browser error evidence.
- First-person-view sharing and audited operator steering/takeover.
- Agent-First Browser contracts with versioned observations, stale-safe actions, semantic deltas, verification and execution capsules.
- Existing operator-control lease/takeover reconciliation design with generation/fencing, local safety freeze, operator delta, and mandatory re-observation.
- Critique, UI reverse/reference analysis, section detection, layout comparison, style enhancement, copilot, intake, workflow, design-system, content-map, comparison, migration, and media surfaces.
- Pi extension, MCP bridge, CLI wrapper, OpenAI/MCP schemas, agent cards, tool search, graph, and docs metadata.
- Focusa-ready bounded evidence and research/diagnostics packet handoff.

Selected commercially valuable implementations may move behind privately built signed workers or encrypted feature capsules. Patching a public middleware return value must not create a missing protected worker, node-bound content key, operation token, or official update.

## Current runtime architecture

- Go server with Chi router.
- Rod-backed Chromium/vision pool.
- Local state/usage/job stores.
- Pi extension and Node MCP bridge.
- Optional AI provider configuration.
- Local binary/systemd deployment and browser reliability workflow.

Main entry points:

- `cmd/uiai-engine/main.go`
- `internal/server/server.go`
- `internal/auth/auth.go`
- `internal/license/entitlements.go`
- `internal/config/config.go`
- `.pi/extensions/uiai-engine.ts`
- `mcp/browser-session-mcp.mjs`
- `scripts/uiai`

## Veragensia enforcement relationship

Veragensia Doc 193 governs the OS/container/device/network capability envelope available to a UIAI workload; UIAI still independently enforces its own product/session/origin/action rules.

```text
Veragensia machine capability
!=
UIAI product entitlement
!=
Focusa work/operation authority
```

All applicable gates must pass.

A UIAI worker may carry a Veragensia WorkloadIdentity/RuntimeAttestation and run inside a Doc-193 EnforcementPlan, but those objects do not let UIAI bypass Focusa or UIAI authorization.

For general desktop control, Veragensia Doc 194 may compose UIAI observations/control leases with non-browser DesktopObservation state. UIAI remains authoritative for its browser runtime identity; Veragensia must not rewrite browser document/navigation/frame freshness into weaker desktop-only identity.

## Mandatory licensing target

Every official runtime—including Evaluation—uses this state model:

```text
unactivated | active_evaluation | active_paid | offline_grace
| expired | revoked | invalid
```

Missing/expired/revoked/invalid state starts recovery-only. Recovery preserves health, license onboarding/refresh/doctor, safe diagnostics, operator-owned data, applicable export, and uninstall guidance; it denies new browser/analysis/media/control execution.

Required onboarding:

```text
standalone UIAI or Focusa bundle
→ authority device code
→ verified account/email and current terms
→ separate promotional-email choice
→ explicit uiai-engine product/features/limits grant
→ node activation and signed lease
→ independent engine verification
→ bounded walkthrough
```

Focusa-brokered tokens must be short-lived, audience-bound, feature/node/client scoped, tied to the parent lease id/sequence/digest, and no broader or longer-lived than the parent grant.

## Current use and testing caution

The current APIs and quickstarts document as-built browser behavior for development and migration. Command availability does not prove entitlement. Do not expose the current engine to new evaluators/customers as a licensed product until the mandatory entitlement and endpoint coverage gates pass.

The Veragensia control/voice binding is architecture direction; existing UIAI proposed takeover/control-lease documents remain subject to their own implementation/closure status. A documentation binding MUST NOT be described as shipped functionality.

Development/build example:

```bash
go test ./internal/auth ./internal/license ./internal/server
go build ./cmd/uiai-engine
scripts/uiai health
```

A developer build is not a customer/Evaluation license and must use test trust roots and synthetic fixtures only.

## Agent discovery

Primary metadata entry points:

- `/api/tools/agent-card`
- `/api/tools/search?q=<keyword>`
- `/api/tools/graph`
- `/api/tools/docs`
- `/api/tools/openai`
- `/api/tools/mcp`
- Pi `pi_uiai_agent_card`
- CLI `scripts/uiai tools search <query>`

Locked tools may remain discoverable, but descriptors must include their product-qualified `license_feature`, limit posture, and pre-side-effect denial behavior.

A voice interpreter should discover these same capabilities through Focusa/Veragensia rather than maintain a separate speech-command catalog.

## Browser workflow

The intended entitled workflow is:

```text
verify caller + lease/product/feature/limit
→ open session with optional Focusa scope
→ read/snapshot
→ act through selectors/@refs / strongest semantic actuator
→ inspect diagnostics after uncertainty/failure
→ capture bounded evidence/packet
→ close session
→ commit/release usage reservation
```

Current direct loopback examples in older docs are as-built development references. They do not establish Evaluation entitlement.

## Observation and control invariants

UIAI/Veragensia cross-system operation preserves:

- browser document/navigation/frame identity;
- exact expected observation on action;
- stale/resync failure rather than acting on replaced state;
- explicit coordinate-space mapping for visual fallback;
- one active control-lease holder per actuator scope;
- lease generation/fencing against stale controllers;
- local safety freeze distinct from Focusa canonical pause;
- operator delta capture;
- mandatory re-observation and authority/credential refresh before agent resume;
- pending unknown side effects blocking unsafe resume.

A spoken "continue" after human takeover is a steering input, not proof that these reconciliation requirements are satisfied.

## FPV and public tokenized routes

FPV links can show and optionally control a browser session. Final production tokens must be signed/opaque, short-lived, audience/resource/action scoped, independently audited, and incapable of escalating to general engine execution. Share creation/control is an entitled operation even when a bounded viewer is public by token.

Voice commands to start/stop/take over FPV/control remain subject to the same token/control-lease/Focusa authority rules.

## Focusa integration

Focusa and UIAI remain separate authorities:

- Focusa supplies canonical project/Workpoint scope and may broker a child token from a valid bundle grant.
- Focusa Spec 181 owns Voice/Conversation, participant/utterance/transcript correction and Conversation Ledger semantics.
- UIAI independently validates the `uiai-engine` product, feature, parent lease state, node, time, sequence, and limits.
- UIAI results become Evidence proposals; UIAI does not become Focusa cognitive or conversation truth.
- Research and diagnostics handoff uses `uiai.focusa_research_diagnostics_packet.v1`; `uiai_focusa_packet_build` and `uiai_focusa_packet_compose` preserve bounded packet metadata and the next Focusa action.
- UIAI pressure/health can narrow execution but cannot create or expand entitlement.
- A spoken instruction may be linked to the UIAI execution capsule/receipt chain, but UIAI does not own the canonical transcript history merely because it executed the action.

## Influence firewall and audio

UIAI's browser-content provenance/influence firewall remains essential in voice-native operation.

Untrusted webpage text/audio, page-generated speech, ads, WebMCP metadata/results, downloads, remote responses or browser media may inform reasoning but cannot:

- redefine the human's Focusa utterance;
- grant authority;
- expand data egress;
- unlock credentials;
- suppress Evidence;
- impersonate a trusted Secure Attention prompt merely by sounding similar.

Trusted Veragensia microphone capture permission does not automatically grant microphone access to browser pages or UIAI sessions.

## Security and recovery

- Private/internal URLs are blocked by default unless explicitly authorized by deployment policy.
- Logs and error records redact query secrets, fragments, auth headers, cookies, request bodies, keys, and tokens.
- Entitlement checks must occur before browser allocation, model/provider calls, media jobs, persistent session creation, or mutation.
- Production builds reject test roots, test fixture statuses, and development bypasses.
- License failure never deletes operator data.
- Runtime/session replacement invalidates stale browser/control references and requires reconciliation.
- Voice/Conversation continuity may survive through Focusa ledger refs without resurrecting stale UIAI actuator authority.

## Required release proof

- authentication/entitlement separation;
- no loopback bypass;
- reverse-proxy remote cannot inherit loopback permission;
- local/extension token without parent grant denied;
- wrong product/feature/node/time/sequence denied;
- atomic concurrent limit reservations;
- child token cannot exceed or outlive parent;
- public/share/status routes cannot enumerate private work;
- standalone and Focusa-brokered onboarding parity;
- protected worker/capsule direct/replay/copy/substitution/downgrade tests;
- redaction and data-preserving expiry/uninstall;
- stale observation/control-generation rejection for implemented takeover surfaces;
- cross-system voice requests use normal UIAI authority/verification paths where voice integration is claimed;
- browser-page audio cannot impersonate trusted approval where Veragensia integration is claimed.

## Clearest benefits

- Public sources become usable proof through Source-to-Markdown, stable evidence handles, and bounded diagnostics.
- Search surfaces include `/api/search`, `uiai_search`, `browser_search`, `scripts/uiai-open-result.sh`, and a Wikipedia OpenSearch fallback; `/api/markdown` and `browser sessions/actions/diagnostics` are likewise bounded and auditable.
- Browser sessions/actions/diagnostics remain separate from Focusa authority and are inspected with `browser_diagnostics`.
- Source-to-Markdown returns `uiai.source_markdown.v1`, `source_to_markdown`, `wpuiai.research_card`, and bounded JSONL records where requested.
- Diagnostics errors are available at `/api/errors` and through `uiai_errors`; `uiai_browser_diagnostics` is the preferred bounded session diagnostic; the Focusa packet route is `/api/agent/research-packet`; `uiai_focusa_packet_compose` and `uiai_focusa_packet_build` produce proposal-only handoff packets.
- Available repository skills include `.pi/skills/uiai-agent/SKILL.md`, `.pi/skills/uiai-focusa-packet/SKILL.md`, `.pi/skills/uiai-mcp/SKILL.md`, `.pi/skills/uiai-release/SKILL.md`, `.pi/skills/uiai-remote-auth/SKILL.md`, `.pi/skills/uiai-docs-maintenance/SKILL.md`, `.pi/skills/uiai-ci-debug/SKILL.md`, and `.pi/skills/uiai-browser-debug/SKILL.md`.

## Documentation map

### Veragensia integration

- [`docs/UIAI_VERAGENSIA_COMPUTER_CONTROL_AND_VOICE_BINDING_2026-09-04.md`](docs/UIAI_VERAGENSIA_COMPUTER_CONTROL_AND_VOICE_BINDING_2026-09-04.md) — cross-system control, observation, voice and authority binding.
- [`docs/UIAI_COCKPIT_002_AGENT_FIRST_BROWSER_AMENDMENT_2026-07-19_v1.0.md`](docs/UIAI_COCKPIT_002_AGENT_FIRST_BROWSER_AMENDMENT_2026-07-19_v1.0.md) — Agent-First Browser exchange/observation/action/provenance contract.
- [`docs/contracts/UIAI_COCKPIT_008_C03_OPERATOR_CONTROL_LEASE_TAKEOVER_RECONCILIATION_v1.yaml`](docs/contracts/UIAI_COCKPIT_008_C03_OPERATOR_CONTROL_LEASE_TAKEOVER_RECONCILIATION_v1.yaml) — proposed control lease/takeover/re-observation contract; implementation status remains explicit in that document.

### Licensing and endpoint authority

- [`docs/LICENSING.md`](docs/LICENSING.md)
- [`docs/UIAI_LICENSE_ENTITLEMENT_AND_ONBOARDING_ENFORCEMENT_SPEC_2026-08-01.md`](docs/UIAI_LICENSE_ENTITLEMENT_AND_ONBOARDING_ENFORCEMENT_SPEC_2026-08-01.md)
- [`docs/UIAI_PROTECTED_WORKER_AND_FEATURE_CAPSULE_ADDENDUM_2026-08-01.md`](docs/UIAI_PROTECTED_WORKER_AND_FEATURE_CAPSULE_ADDENDUM_2026-08-01.md)
- [`docs/ENDPOINT_AUTH_MATRIX.md`](docs/ENDPOINT_AUTH_MATRIX.md)
- [`docs/REMOTE_AUTH_EXAMPLES.md`](docs/REMOTE_AUTH_EXAMPLES.md) — current authentication examples, not entitlement proof
- [`docs/SESSION_API.md`](docs/SESSION_API.md) — current API plus migration warning

### Cockpit and desktop architecture

- [`docs/UIAI_COCKPIT_DOCUMENT_REGISTER.md`](docs/UIAI_COCKPIT_DOCUMENT_REGISTER.md) — numbered Cockpit master, decisions, amendments, and companions
- [`docs/UIAI_COCKPIT_004_DESKTOP_SESSION_PRESENTATION_AND_MENUBAR_HANDOFF_SPEC_2026-08-03_v1.0.md`](docs/UIAI_COCKPIT_004_DESKTOP_SESSION_PRESENTATION_AND_MENUBAR_HANDOFF_SPEC_2026-08-03_v1.0.md) — packaged runtime, same-session Cockpit presentation, and Focusa Menubar handoff
- [`docs/contracts/UIAI_COCKPIT_004_C01_DESKTOP_SESSION_PRESENTATION_HANDOFF_LEDGER_v1.yaml`](docs/contracts/UIAI_COCKPIT_004_C01_DESKTOP_SESSION_PRESENTATION_HANDOFF_LEDGER_v1.yaml) — stable requirements, task graph, metrics, and proof contract

### Agent/browser operation

- [`docs/UIAI_FOR_AGENTS_QUICKSTART.md`](docs/UIAI_FOR_AGENTS_QUICKSTART.md)
- [`docs/AGENT_DISCOVERY_INDEX.md`](docs/AGENT_DISCOVERY_INDEX.md)
- [`docs/AGENT_UX_COOKBOOK.md`](docs/AGENT_UX_COOKBOOK.md)
- [`docs/BROWSER_DIAGNOSTICS_SPEC.md`](docs/BROWSER_DIAGNOSTICS_SPEC.md)
- [`docs/UIAI_FOCUSA_PI_HAND_IN_GLOVE_SPEC.md`](docs/UIAI_FOCUSA_PI_HAND_IN_GLOVE_SPEC.md)
- [`docs/UIAI_AGENT_FPV_PWA_SPEC_2026-06-09.md`](docs/UIAI_AGENT_FPV_PWA_SPEC_2026-06-09.md)
- [`docs/PUBLIC_API_PARITY_MATRIX.md`](docs/PUBLIC_API_PARITY_MATRIX.md)
- [`docs/ENGINE_RELEASE_DEPLOY_RUNBOOK.md`](docs/ENGINE_RELEASE_DEPLOY_RUNBOOK.md)

## License

UIAI Engine is source-available under BSL 1.1. See [`LICENSE`](LICENSE) and [`docs/LICENSING.md`](docs/LICENSING.md). Source-use rights and official product entitlement are separate boundaries.
