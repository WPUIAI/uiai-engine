# UIAI Cockpit Focusa Daemon Pairing, Multi-Daemon Fleet, and Scope Reconciliation Specification

**Document number:** `UIAI-COCKPIT-005`  
**Parent document:** `UIAI-COCKPIT-000`  
**Preceding amendment:** `UIAI-COCKPIT-004`  
**Status:** Proposed normative implementation amendment  
**Version:** 1.0  
**Date:** 2026-08-03  
**Machine-readable companion:** [`UIAI-COCKPIT-005-C01`](contracts/UIAI_COCKPIT_005_C01_FOCUSA_PAIRING_RECONCILIATION_LEDGER_v1.yaml)  
**Call-stack design:** `019fca49-b618-7502-9346-8facffeb7778`

---

# 0. Authority and application order

```text
UIAI-COCKPIT-000
→ UIAI-COCKPIT-001
→ UIAI-COCKPIT-002 + UIAI-COCKPIT-002-C01
→ UIAI-COCKPIT-003
→ UIAI-COCKPIT-004 + UIAI-COCKPIT-004-C01
→ UIAI-COCKPIT-005 + UIAI-COCKPIT-005-C01
```

This amendment is the canonical Cockpit integration specification for Focusa daemon discovery, pairing, secure authentication, multi-daemon selection, and project/scope reconciliation. It incorporates, but does not replace, the daemon and Menubar authority defined by Focusa Specs 53–57.

Operator steering remains authoritative. Focusa cognition, discovery, and prediction are advisory and cannot grant pairing or scope authority.

# 1. Normative source map

| Source | Authority retained |
|---|---|
| Focusa `53-focusa-device-pairing-spec.md` | device pairing, token issuance, ledger semantics |
| Focusa `54-focusa-pairing-room-plan.md` | room lifecycle, bridge ownership, phone/VPS completion |
| Focusa `55-focusa-self-host-architecture.md` | self-host topology and trust boundaries |
| Focusa `56-focusa-pairing-wizard-spec.md` | pairing UX and recovery states |
| Focusa `57-focusa-pairing-revoke-and-repair.md` | revocation, expiry, re-pair, repair |
| `UIAI_OPERATOR_BROWSER_DESKTOP_SPEC_2026-06-19.md` §17.3–17.19 | Cockpit replication, auto-add, credential storage, fleet picker |
| `UIAI-COCKPIT-004` | deep-link ownership, handoff, compatibility, connection planes |
| This document | Cockpit implementation sequence, contracts, TaskGraph, acceptance |

Conflicts fail closed. Focusa Specs 53–57 control daemon behavior; this document controls Cockpit behavior.

# 2. Required outcome

Cockpit SHALL:

1. detect local and reachable Focusa daemons without treating discovery as authentication;
2. pair independently as `client_type=cockpit` through the same governed protocol used by Menubar;
3. store every daemon token only in OS credential storage;
4. support fresh pairing and Menubar-assisted auto-add without copying Menubar credentials;
5. maintain multiple authenticated daemon profiles simultaneously;
6. enumerate projects/scopes from each profile using that daemon's token;
7. require explicit reconciliation when equivalent projects appear on multiple daemons;
8. preserve daemon, project, continuity, Workpoint, and evidence authority boundaries;
9. expose expiry, revoke, repair, version mismatch, and sibling-app absence truthfully;
10. prove local-only, VPS, multi-daemon, cross-platform, and adversarial paths before rollout.

# 3. Terminology

- **Candidate:** discovered daemon endpoint; no authority.
- **Profile:** authenticated Cockpit device relationship with one daemon.
- **Hint:** non-secret endpoint/device metadata; never a token.
- **Path A:** fresh pairing through room/bridge/phone/VPS completion.
- **Path B:** Menubar-assisted Cockpit token minting on the same machine.
- **Fleet:** all authenticated daemon profiles visible to Cockpit.
- **Binding:** daemon-qualified project and continuity selection.
- **Reconciliation:** explicit operator decision relating equivalent project identities across daemon authorities.
- **Truth plane:** authenticated daemon/Engine APIs.
- **Discovery plane:** loopback, Bonjour, Tailscale, saved hints, environment, manual input.

# 4. Non-negotiable invariants

`PAIR-INV-001` Discovery SHALL NOT create authentication or select canonical scope.

`PAIR-INV-002` Menubar and Cockpit SHALL have distinct device identities and tokens.

`PAIR-INV-003` Tokens SHALL NOT enter localStorage, IndexedDB, URLs, QR payload logs, deep links, telemetry, crash reports, or project files.

`PAIR-INV-004` Deep links and bridge messages are intent/transport envelopes, never authentication.

`PAIR-INV-005` Every project binding is qualified by `daemon_id + project_identity + continuity_id`.

`PAIR-INV-006` Equivalent projects on different daemons SHALL NOT be silently merged.

`PAIR-INV-007` Pairing, mutation, takeover, and reconciliation require explicit authority and auditable receipts.

`PAIR-INV-008` Local-only mode SHALL not contact remote pairing endpoints.

`PAIR-INV-009` Unknown schema fields, protocol versions, routes, refs, or capabilities fail closed.

`PAIR-INV-010` Credential-storage failure SHALL leave the profile unauthenticated and the token discarded.

# 5. Connection planes

| Plane | Mechanism | May mutate authority? |
|---|---|---|
| Discovery | loopback, Bonjour, Tailscale, hints, environment, manual URL | No |
| Health | `/.well-known/focusa.json`, daemon `/v1/health` | No |
| Pairing | `/v1/device/pair/*`, governed rooms and bridge | Only through daemon approval |
| Truth | authenticated daemon APIs | Per token scopes |
| Handoff | `focusa://`, `cockpit://` opaque intents | No |
| Fast channel | per-user socket/named pipe | No canonical mutation |

No plane silently substitutes for another.

# 6. Typed contracts

## 6.1 `FocusaDaemonCandidateV1`

Required fields: `schema`, `candidate_id`, `base_url`, `source`, `location`, `observed_at`, `health_status`, `latency_ms`, optional `daemon_id`, `machine_id`, `version`, `capabilities`.

`source` is one of `loopback|bonjour|tailscale|saved_hint|environment|manual`.

## 6.2 `FocusaPlatformHintV1`

Non-secret fields only: `schema`, `daemon_url`, optional `daemon_id`, `device_id`, `paired`, `client_types_seen`, `last_verified_at`. Token-shaped fields are rejected.

## 6.3 `CockpitPairingRoomV1`

Required fields: `schema`, `room_id`, `nonce`, `client_type=cockpit`, `daemon_url`, `status`, `created_at`, `expires_at`, `bridge_owner`, optional `pair_url`, `pair_code`, `menubar_device_id`.

Statuses: `created|awaiting_operator|awaiting_vps|completed|expired|revoked|failed`.

## 6.4 `AuthenticatedDaemonProfileV1`

Required fields: `schema`, `profile_id`, `daemon_id`, `daemon_url`, `device_id`, `client_type=cockpit`, `token_handle`, `scopes`, `source`, `paired_at`, `expires_at`, `last_verified_at`, `status`.

`token_handle` references OS credential storage; it is not token material.

## 6.5 `DaemonProjectCandidateV1`

Required fields: `schema`, `daemon_id`, `profile_id`, `project_root`, `project_id`, `canonical_name`, `identity_status`, `observed_at`; optional `repo_remote`, `continuities`.

## 6.6 `ProjectScopeBindingV1`

Required fields: `schema`, `binding_id`, `daemon_id`, `profile_id`, `project_id`, `project_root`, `continuity_id`, `authority_status`, `selected_at`; optional `workpoint_id`.

## 6.7 `ScopeReconciliationDecisionV1`

Required fields: `schema`, `decision_id`, `left_binding_id`, `right_binding_id`, `relation`, `operator_confirmed`, `created_at`, `evidence_refs`.

Relations: `same_project_separate_authority|preferred_profile|mirror_read_only|not_same_project`.

No relation transfers credentials or canonical state.

# 7. Secure storage

| Platform | Required backend |
|---|---|
| macOS | Keychain |
| Linux | Secret Service/libsecret |
| Windows | Credential Manager |

Cockpit service name: `UIAI Engine Cockpit Token`.

Stored secret entry: `{profile_id, daemon_id, daemon_url, device_id, token, scopes, expires_at}`.

Shared `focusa-platform` hint contains no token. Cross-app access requires a signed shared Keychain access group or the local read-only compatibility manifest. If neither is available, Path B is unavailable and Path A remains functional.

# 8. Discovery sequence

1. probe `127.0.0.1:8787/v1/health`;
2. inspect local sibling manifest;
3. query Bonjour using the Focusa service type;
4. query approved Tailscale/MagicDNS names;
5. load non-secret saved hints;
6. read explicit environment/CLI bootstrap values;
7. accept manual HTTPS URL;
8. deduplicate by verified `daemon_id`, otherwise normalized URL;
9. display all candidates and their source/health;
10. never mark a candidate paired without authenticated proof.

# 9. Path A — fresh pairing

1. Operator selects a candidate.
2. Cockpit verifies health and pairing capabilities.
3. Cockpit calls `/v1/device/pair/start` with `client_type=cockpit`, device display name, platform, requested scopes, and bounded ScopeContext.
4. Daemon returns room/code/nonce/TTL.
5. Cockpit becomes bridge owner for Cockpit-created rooms using `focusa_start_bridge_callback`.
6. Cockpit renders daemon-provided pair URL/QR without embedding secrets in logs.
7. Phone/VPS completes approval under Focusa Specs 53–54.
8. Cockpit polls canonical room status every 1.5 seconds with backoff after pressure signals.
9. On completion Cockpit consumes the completion once, validates nonce, room, client type, daemon, TTL, and scopes.
10. Cockpit writes the token to OS credential storage.
11. Only after successful storage, Cockpit creates an authenticated profile and verifies an authenticated daemon read.
12. Cockpit clears bridge/room transient state and emits a receipt.
13. Failure at any step discards transient token material and presents recovery.

# 10. Path B — Menubar-assisted auto-add

All gates must pass:

- same machine/user;
- cloud profile permits auto-add;
- Menubar hint or manifest is compatible;
- daemon URL matches a paired Menubar profile;
- operator has not dismissed the prompt;
- daemon supports inherited pairing proof.

Flow:

1. Cockpit shows the explicit auto-add prompt.
2. Operator chooses `Add this machine`, `Pair from scratch`, or `Details`.
3. Auto-add calls `/v1/device/pair/start` with `client_type=cockpit` and Menubar device cross-reference.
4. Daemon proves ownership and mints a new Cockpit device/token.
5. Cockpit stores its own token and records `source=auto_via_menubar`.
6. Menubar token is never read or copied.
7. If proof fails, Cockpit offers Path A without mutation.

# 11. Multi-daemon fleet

- Zero profiles: pairing call-to-action.
- One healthy profile: default selection is permitted.
- More than one: show fleet picker before project authority selection.
- Mark local/remote, health, version, scopes, expiry, source, and last use.
- Menubar-active daemon may be marked but not silently preferred over operator choice.
- `All` means pair/enumerate independently, never shared authority.

# 12. Project and scope reconciliation

1. Query each selected profile independently.
2. Verify project identity with that daemon.
3. Produce daemon-qualified candidates.
4. Group likely equivalents by durable project ID, repository remote, and verified root evidence; fingerprints remain advisory.
5. Display conflicts and confidence.
6. Require an explicit reconciliation decision.
7. Persist only relation metadata and evidence refs.
8. Keep active binding singular for mutations.
9. Cross-daemon reads may be shown together only with clear provenance.
10. Never copy Workpoints, tokens, scope, evidence, or trajectory state as a side effect of reconciliation.

# 13. Expiry, revoke, and repair

- Detect expiry before authenticated requests.
- Mark profile `expired`, preserve non-secret metadata, and prompt re-pair.
- Revoke through canonical daemon API; delete local credential only after receipt or explicit local-forget choice.
- Repair validates daemon identity before replacing token handle.
- URL change with same daemon ID requires operator confirmation.
- Same URL with changed daemon ID is a conflict, not silent replacement.

# 14. Compatibility manifest

Cockpit and Menubar expose `/.well-known/focusa.json` with `focusa.app.manifest.v2`, app/version/channel, protocol versions, pairing capabilities, and client type. It is read-only and contains no credentials.

Path B requires compatible `pairing`, `bridge`, and `scope_context` protocol versions.

# 15. Error taxonomy

Required classes: `daemon_unreachable`, `daemon_identity_changed`, `pairing_unsupported`, `room_expired`, `nonce_mismatch`, `client_type_mismatch`, `scope_rejected`, `credential_store_locked`, `credential_store_denied`, `token_expired`, `token_revoked`, `menubar_absent`, `menubar_incompatible`, `multi_daemon_conflict`, `project_identity_conflict`, `reconciliation_required`, `local_only_blocked_remote`.

Every error supplies bounded operator recovery and avoids secret material.

# 16. Observability and receipts

Emit structured, redacted events for discovery, pairing start/status/completion, secure-storage result, profile verification, fleet selection, reconciliation, expiry, revoke, and repair. Store IDs and evidence refs, never tokens, QR payloads, authorization headers, or full bridge payloads.

# 17. Required call stack

```text
Cockpit UI
→ pairing/fleet/reconciliation controller
→ typed contract validator
→ discovery, daemon, sibling-manifest, and credential adapters
→ Tauri bridge/keyring/Bonjour commands
→ Focusa daemon pairing and truth APIs
→ redacted receipts/evidence
```

Existing surfaces to extend:

- `apps/cockpit/src/lib/focusa-daemon-discovery.ts`
- `apps/cockpit/src/lib/focusa-projects.ts`
- `apps/cockpit/src/lib/bridge/tauri.ts`
- `apps/cockpit/src-tauri/src/bridge.rs`
- `apps/cockpit/src-tauri/src/bonjour.rs`
- `apps/cockpit/src/routes/settings/+page.svelte`
- `apps/cockpit/src/routes/nodes-services/+page.svelte`

# 18. Executable TaskGraph

## T005-00 — Register and freeze authority

Create this spec, companion ledger, register entry, source map, and parent bead. Acceptance: no unresolved authority owner or secret-storage ambiguity.

## T005-01 — Contracts and adversarial fixtures

Implement all §6 contracts in TypeScript/Rust plus valid/invalid fixtures. Depends on T005-00.

## T005-02 — Compatibility manifest

Implement sibling manifest server/client and protocol negotiation. Depends on T005-01.

## T005-03 — Discovery adapters

Complete loopback, sibling, Bonjour, Tailscale, hints, environment, and manual discovery with deduplication. Depends on T005-01.

## T005-04 — Secure credential storage

Implement Keychain/libsecret/Credential Manager adapter, signed entitlements, token-handle API, and redaction tests. Depends on T005-01.

## T005-05 — Path A pairing

Implement room start, bridge ownership, QR, status polling, completion validation, secure persist, profile verification, and cleanup. Depends on T005-02, T005-03, T005-04.

## T005-06 — Path B auto-add

Implement Menubar hint/manifest, explicit prompt, device cross-reference proof, new Cockpit token, denial/fallback. Depends on T005-02, T005-04, T005-05.

## T005-07 — Authenticated profile registry

Implement profile lifecycle, health, active profile, scope checks, expiry state, and token-handle resolution. Depends on T005-05 and T005-06.

## T005-08 — Fleet picker

Implement zero/one/many behavior and explicit multi-daemon selection. Depends on T005-07.

## T005-09 — Project/scope enumeration

Query and validate daemon-qualified projects, continuities, and Workpoints. Depends on T005-07.

## T005-10 — Reconciliation

Implement candidate grouping, explicit relation decisions, provenance, and mutation guards. Depends on T005-08 and T005-09.

## T005-11 — Expiry, revoke, and repair

Implement canonical revoke, local forget, expiry, identity-change conflict, and re-pair. Depends on T005-07.

## T005-12 — UX, accessibility, and observability

Complete settings/nodes surfaces, keyboard/screen-reader behavior, redacted events, and recovery. Depends on T005-08, T005-10, T005-11.

## T005-13 — Cross-platform adapters

Prove macOS, Linux, Windows, sandbox, local-only, and CI paths. Depends on T005-04, T005-12.

## T005-14 — E2E and signed rollout

Run the full matrix, security review, signed package tests, rollback, and release proof. Depends on T005-13.

# 19. Acceptance matrix

| Case | Required proof |
|---|---|
| Local daemon only | auto-detected; no remote traffic; project selection works |
| Fresh VPS pairing | Path A mints/stores distinct Cockpit token |
| Menubar paired | Path B mints new Cockpit token without copying Menubar token |
| Menubar absent/stopped | Path A remains available |
| One profile | safe default selection |
| Multiple profiles | picker shown; no silent choice |
| Same project on two daemons | explicit reconciliation required |
| Token expired/revoked | mutation blocked; repair offered |
| Keychain locked/denied | no authenticated profile created |
| Daemon identity changed | fail closed with conflict |
| Protocol mismatch | Path B blocked; compatible recovery offered |
| Local-only mode | remote pairing blocked before network request |
| Malformed/oversized payload | rejected without persistence or log leakage |
| Crash during completion | no plaintext token residue; idempotent recovery |
| Cross-platform | native credential and discovery adapters pass |
| Signed install/update/rollback | profiles remain valid or truthfully require repair |

# 20. Required test suites

- contract fixtures and unknown-field rejection;
- discovery source/deduplication/timeout tests;
- bridge nonce, TTL, replay, ownership, and one-time completion tests;
- credential storage lock/deny/write/read/delete tests;
- Path A state-machine tests;
- Path B proof, denial, and fallback tests;
- profile expiry/revoke/identity-change tests;
- fleet zero/one/many tests;
- project reconciliation provenance and mutation-guard tests;
- accessibility and recovery-state tests;
- secret scanning of logs, storage, URLs, events, screenshots, and artifacts;
- macOS/Linux/Windows compile/package tests;
- local daemon, VPS, multi-daemon, Menubar absent/stopped, CI, update, rollback E2E.

# 21. Rollout gates

1. Advisory/dev only with fake daemon fixtures.
2. Local daemon real integration.
3. VPS Path A staging.
4. Path B staging with signed Menubar/Cockpit builds.
5. Multi-daemon and reconciliation staging.
6. Cross-platform credential proof.
7. Signed dev OTA with rollback.
8. Operator approval for stable promotion.

# 22. Definition of done

All T005-00..T005-14 tasks are closed with evidence; no token enters browser storage; local/VPS/multi-daemon paths pass; project authority remains daemon-qualified; revoke/repair works; signed release and rollback are proven.

# 23. Explicit non-goals

- automatic daemon-state merging;
- token sharing between Menubar and Cockpit;
- using deep links as authentication;
- replacing Focusa daemon pairing authority;
- background pairing without an approved profile/policy;
- cross-daemon Workpoint or trajectory migration as a reconciliation side effect.
