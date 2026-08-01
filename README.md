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

UIAI Engine owns browser/search/session/media/diagnostics execution and stable evidence handles. [Focusa](https://github.com/Startempire-Wire/focusa) owns ProjectIdentity, Workpoints, Trajectory, evidence linkage, predictions, metacognition, continuation, and recovery.

Authentication and entitlement are separate:

- loopback changes network-auth risk only;
- local API, extension, Pi, MCP, Cockpit, webhook, or Focusa pairing tokens do not create product permission;
- a healthy browser pool does not prove a license;
- source access is not an authority-issued Evaluation;
- explicit product/features/time/node/sequence/limits must be verified before execution.

## Main capabilities

- Persistent browser sessions with navigation, read, snapshot, click, fill, type, select, keypress, CSS, JavaScript, viewport, cookies, screenshots, and diagnostics.
- Provider-neutral search and Source-to-Markdown research flows.
- Console, exception, network, failed-request, and structured engine/browser error evidence.
- First-person-view sharing and audited operator steering.
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

## Browser workflow

The intended entitled workflow is:

```text
verify caller + lease/product/feature/limit
→ open session with optional Focusa scope
→ read/snapshot
→ act through selectors/@refs
→ inspect diagnostics after uncertainty/failure
→ capture bounded evidence/packet
→ close session
→ commit/release usage reservation
```

Current direct loopback examples in older docs are as-built development references. They do not establish Evaluation entitlement.

## FPV and public tokenized routes

FPV links can show and optionally control a browser session. Final production tokens must be signed/opaque, short-lived, audience/resource/action scoped, independently audited, and incapable of escalating to general engine execution. Share creation/control is an entitled operation even when a bounded viewer is public by token.

## Focusa integration

Focusa and UIAI remain separate authorities:

- Focusa supplies canonical project/Workpoint scope and may broker a child token from a valid bundle grant.
- UIAI independently validates the `uiai-engine` product, feature, parent lease state, node, time, sequence, and limits.
- UIAI results become Evidence proposals; UIAI does not become Focusa cognitive truth.
- UIAI pressure/health can narrow execution but cannot create or expand entitlement.

## Security and recovery

- Private/internal URLs are blocked by default unless explicitly authorized by deployment policy.
- Logs and error records redact query secrets, fragments, auth headers, cookies, request bodies, keys, and tokens.
- Entitlement checks must occur before browser allocation, model/provider calls, media jobs, persistent session creation, or mutation.
- Production builds reject test roots, test fixture statuses, and development bypasses.
- License failure never deletes operator data.

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
- redaction and data-preserving expiry/uninstall.

## Documentation map

### Licensing and endpoint authority

- [`docs/LICENSING.md`](docs/LICENSING.md)
- [`docs/UIAI_LICENSE_ENTITLEMENT_AND_ONBOARDING_ENFORCEMENT_SPEC_2026-08-01.md`](docs/UIAI_LICENSE_ENTITLEMENT_AND_ONBOARDING_ENFORCEMENT_SPEC_2026-08-01.md)
- [`docs/UIAI_PROTECTED_WORKER_AND_FEATURE_CAPSULE_ADDENDUM_2026-08-01.md`](docs/UIAI_PROTECTED_WORKER_AND_FEATURE_CAPSULE_ADDENDUM_2026-08-01.md)
- [`docs/ENDPOINT_AUTH_MATRIX.md`](docs/ENDPOINT_AUTH_MATRIX.md)
- [`docs/REMOTE_AUTH_EXAMPLES.md`](docs/REMOTE_AUTH_EXAMPLES.md) — current authentication examples, not entitlement proof
- [`docs/SESSION_API.md`](docs/SESSION_API.md) — current API plus migration warning

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
