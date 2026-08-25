# UIAI Cockpit Browser Identity Gap-Closure Matrix

**Document number:** `UIAI-COCKPIT-009`
**Status:** proposed normative gap-closure amendment v1.0
**Date:** 2026-08-02
**Repository:** `WPUIAI/uiai-engine`
**Primary parent:** `UIAI-COCKPIT-005`
**Also amends:** `UIAI-COCKPIT-004`, `UIAI-COCKPIT-008`, `SESSION_API.md`, `BROWSER_DIAGNOSTICS_SPEC.md`, `CAPTCHA_SOLVER_SPEC.md`, and `FPV_REMAINING_GAPS_SPEC.md`

## 0. Decision

The browser identity research does not require another browser-profile master specification. `UIAI-COCKPIT-005` remains the canonical research and product contract for `detect`, `no_detect`, `operator`, `research`, and `auto` browser modes.

This amendment closes the gap between that contract and the current implementation by assigning every discovered defect to an owner, code surface, implementation phase, regression proof, and closure condition.

A requirement is not complete because a type, setting, route, UI control, or document exists. Completion requires operational behavior through the canonical agent/browser path plus evidence from the required regression scenario.

## 1. Authority and safety boundary

UIAI Engine owns browser processes, profiles, contexts, observations, actions, diagnostics, network identity, challenge handling, and browser-runtime evidence. Cockpit presents and controls those capabilities. Focusa may authorize, scope, and link work, but it does not implement browser identity.

The purpose of `no_detect` and `operator` modes is coherent, governed representation of an approved browser identity. They do not authorize account abuse, access-control bypass, CAPTCHA circumvention, deceptive identity claims, or activity prohibited by a destination's rules.

## 2. Evidence basis

This matrix is based on:

1. the 2026-07-31 UIAI browser self-audit;
2. official documentation research covering Camoufox, Multilogin, GoLogin, AdsPower, Kameleo, Octo Browser, Dolphin Anty, and Incogniton;
3. static review of the `spec/uiai-operational-browser-profiles` branch;
4. the current UIAI Engine session, content-extraction, CAPTCHA, FPV, and browser-profile implementations; and
5. the requirements and acceptance conditions already defined by Cockpit 004, 005, and 008.

The audit proved these baseline defects without recording raw IP addresses, cookies, account data, or share tokens:

- automation exposure through `navigator.webdriver`;
- macOS user-agent versus Linux platform contradiction;
- empty User-Agent Client Hints;
- browser timezone versus approved route-region mismatch;
- impossible screen, viewport, and outer-window geometry;
- software WebGL renderer exposure;
- datacenter-class egress rather than operator-device egress;
- no cookie persistence across ordinary UIAI session reopen;
- empty Bing extraction and low-information Multilogin SPA extraction;
- unsupported Turnstile solving and absent DataDome implementation; and
- automatic creation and disclosure of a public FPV control share during ordinary session creation.

## 3. Current implementation posture

The draft browser-profile branch currently provides:

- canonical profile types and modes;
- configuration loading, inheritance, validation, and stable digest generation;
- explicit profile/mode/domain-rule selection;
- a profile-scoped browser manager;
- in-process exclusive profile locks;
- Chromium launch and document-start identity patches;
- separate browser-profile routes;
- partial CAPTCHA/IP-pool route reuse; and
- initial Cockpit Browser Profile settings.

The draft does not yet satisfy Cockpit 005 acceptance. Its own open-work list correctly identifies Camoufox, shared challenge detection, fingerprint consistency, leak checks, runtime selection, research packs, and broader resource governance as unfinished.

## 4. Gap-closure matrix

| ID | Gap | Current state | Owning contract | Primary code surfaces | Required proof |
|---|---|---|---|---|---|
| `BGC-001` | Canonical session/tool integration | Partial | Cockpit 005 | `internal/routes/session.go`, `internal/routes/browser_profiles.go`, `internal/server/browser_profiles.go`, Pi/MCP tool schemas | The ordinary Pi/MCP browser-open path selects and reports every supported mode without using a parallel private API |
| `BGC-002` | Fingerprint coherence | Missing | Cockpit 005 + 004 | `internal/browserprofile/launcher.go`, `registry.go` | A self-audit proves coherent UA, UA-CH, platform, locale, timezone, screen, viewport, WebGL, hardware, and automation posture |
| `BGC-003` | Declarative settings without runtime effect | Partial | Cockpit 005 | `internal/browserprofile/types.go`, launch/network/storage/challenge adapters | Every accepted setting has a runtime consumer or validation rejects it as unsupported |
| `BGC-004` | Persistent operator persona lifecycle | Partial | Cockpit 005 + 008 | profile manager, secret broker, user-data storage, Cockpit settings | Encrypted persona create/open/lock/close/reopen/revoke/delete tests pass without secret disclosure |
| `BGC-005` | Camoufox engine integration | Missing | Cockpit 005 | engine adapter and capability negotiation | Native protocol adapter passes the shared engine conformance suite; no Rod/CDP compatibility is assumed for Firefox |
| `BGC-006` | Challenge classification, solving, and handoff | Partial | Cockpit 005 + 004 | `internal/captcha`, FPV handoff, challenge events | reCAPTCHA, Turnstile, DataDome, text, image-grid, dynamic-grid, and audio fixtures classify correctly and route to solver, operator, or typed unsupported outcome |
| `BGC-007` | Empty and low-information page extraction | Missing | Cockpit 004 | `internal/vision/session.go` | Bing-like and rendered-SPA fixtures trigger bounded DOM/accessibility fallback instead of returning empty or irrelevant fallback text |
| `BGC-008` | Public FPV share created implicitly | Contradictory | Cockpit 000 + FPV security | `internal/routes/session.go`, `internal/routes/fpv.go` | Ordinary session creation creates no public bearer share; explicit shares default read-only and enforce TTL/revocation/view policy |
| `BGC-009` | Raw authentication state lifecycle | Partial | Cockpit 005 + 008 | `SESSION_API.md`, auth save/load, secret broker | Auth state is encrypted at rest, opaque in normal output, origin-scoped, revocable, and never documented as a plaintext temporary-file workflow |
| `BGC-010` | `auto` mode is only rule/default selection | Partial | Cockpit 005 | profile registry, policy adapter, selection receipt | Auto selection evaluates allowed policy inputs and returns a bounded receipt with candidates, decision, reasons, and new-context requirements |
| `BGC-011` | Profile identity absent from canonical evidence chain | Partial | Cockpit 005 + 002 | sessions, observations, receipts, capsules, evidence adapters | Effective profile digest and route classification appear in every canonical browser result without credentials or raw identity material |
| `BGC-012` | Insufficient conformance and regression coverage | Missing | Cockpit 004 + 005 | Go tests, Cockpit tests, UIAI Engine Eval | Launcher, manager, routes, persistence, profiles, challenges, extraction, FPV defaults, engine parity, and Pi/MCP selection suites pass |
| `BGC-013` | Open implementation work lacks durable decomposition | Missing | Cockpit 003 + 005 | issue/task provider and contract ledger | Every open phase and matrix row has an owner, task reference, dependency, done condition, and evidence reference |
| `BGC-014` | Browser-profile terminology collision | Partial | Cockpit 002 + 005 | schemas, UI labels, docs | `BrowserIdentityMode`/`BrowserProfile` remains distinct from Cockpit response profiles and Focusa preload profiles in every schema and UI |

## 5. Normative requirements

### `UIAI-COCKPIT-009/BGC-001` — one canonical browser-open path

All public UIAI clients SHALL be able to request a browser profile or mode through the canonical session-open contract. Legacy clients MAY omit the field and receive the configured default. A separate browser-profile route MAY remain as a management surface, but it SHALL NOT be the only route that receives the profile engine.

The session-open result SHALL include:

- requested mode/profile;
- effective mode/profile;
- stable profile digest;
- engine and persistence class;
- bounded route classification;
- selection rationale; and
- whether a new context/process was created.

### `UIAI-COCKPIT-009/BGC-002` — coherence before navigation

A `no_detect`, `operator`, or `research` session SHALL fail preflight before destination navigation when its effective identity contains an impossible or unsupported combination.

The coherence probe SHALL cover at least:

- UA and browser-core version;
- UA Client Hints;
- OS and `navigator.platform`;
- language, locale, timezone, and geolocation;
- screen, viewport, outer-window geometry, scale, and touch posture;
- hardware concurrency and device memory;
- Canvas, Audio, fonts, ClientRects, and WebGL policy;
- WebRTC, DNS, proxy, and route geography; and
- automation exposure appropriate to the selected mode.

Randomizing one field without a coherent bundle is forbidden.

### `UIAI-COCKPIT-009/BGC-003` — no inert accepted settings

Each profile field SHALL declare one of:

- implemented and enforced;
- implemented as observation only;
- unsupported by the selected engine; or
- deferred and therefore rejected for execution.

The engine SHALL NOT accept a setting, display it as active, and silently ignore it.

### `UIAI-COCKPIT-009/BGC-004` — governed operator persona

An operator persona SHALL have:

- an encrypted persistent store;
- stable operator and persona references;
- allowed-origin/account bindings;
- process-safe and host-safe exclusive locking;
- explicit import/export with redaction and confirmation;
- MFA, passkey, native-prompt, and operator takeover posture;
- configurable recording suppression during authentication;
- secure close, revoke, rotate, and delete operations; and
- evidence that contains refs and outcomes rather than cookies or secrets.

### `UIAI-COCKPIT-009/BGC-005` — engine adapter truth

Chromium/Rod, operator-browser bridge, and Camoufox/Firefox SHALL advertise typed capabilities. Unsupported features fail closed. Engine parity is determined by conformance tests, not by assuming a remote endpoint speaks CDP.

Camoufox remains experimental until its adapter proves session, page, observation, action, artifact, challenge, cleanup, and evidence parity.

### `UIAI-COCKPIT-009/BGC-006` — challenge event and human handoff

Challenge handling SHALL separate:

1. detection and classification;
2. policy admission;
3. solver capability and attempt;
4. route/profile retry;
5. operator FPV handoff; and
6. terminal unsupported or prohibited outcome.

Detection does not imply solver support. Every attempt SHALL record challenge class, profile digest, route class, solver, timing, result, bounded evidence refs, and cleanup posture.

### `UIAI-COCKPIT-009/BGC-007` — extraction fallback

`browser_read` SHALL classify an extraction as low-information when rendered content exists but the selected main-content result is empty, disproportionately navigational, or substantially below visible DOM/accessibility text.

The bounded fallback order is:

```text
explicit selector
→ main-content extraction
→ visible DOM text
→ accessibility text
→ typed low-information result with recommended next tool
```

Fallback metadata SHALL identify the selected stage and why the earlier stage was rejected.

### `UIAI-COCKPIT-009/BGC-008` — private session by default

Opening a browser session SHALL NOT create a public FPV bearer URL automatically.

Share creation SHALL be explicit. Defaults are:

- read-only;
- short bounded TTL;
- bounded view policy;
- revocable;
- no raw token in ordinary model-visible diagnostics; and
- preview of what data and controls will be exposed.

A control-enabled share requires explicit operator authorization and an audit receipt.

### `UIAI-COCKPIT-009/BGC-009` — secret-safe auth state

Normal tool output SHALL never return reusable cookies, storage tokens, credentials, or FPV bearer capabilities unless an explicitly authorized secret-transfer operation requires them.

Auth-state documentation SHALL use an opaque encrypted state reference rather than plaintext `/tmp` files.

### `UIAI-COCKPIT-009/BGC-010` — receipted automatic selection

`auto` SHALL select only among profiles and routes permitted by current authority. Selection SHALL be immutable for the resulting browser context and SHALL produce a receipt containing candidates, exclusions, selected profile, policy refs, reasons, and freshness.

### `UIAI-COCKPIT-009/BGC-011` — evidence propagation

The effective profile digest, engine, persistence class, route class, challenge posture, and coherence result SHALL propagate through:

- session summaries;
- observations and stale-state checks;
- actions and deltas;
- diagnostics and browser-read results;
- execution capsules;
- verification results;
- Focusa evidence linkage; and
- UIAI Engine Eval results.

### `UIAI-COCKPIT-009/BGC-012` — required regression suites

At minimum, release proof SHALL include:

1. transparent `detect` mode;
2. coherent `no_detect` mode;
3. operator persona reopen with preserved harmless test cookie;
4. concurrent operator-persona lock rejection;
5. Chromium versus Camoufox capability negotiation;
6. timezone/locale/route/WebRTC/DNS coherence;
7. impossible geometry rejection;
8. Turnstile and DataDome typed challenge outcomes;
9. rendered SPA extraction fallback;
10. session creation with no implicit FPV share;
11. explicit read-only and control-share lifecycle;
12. auth-state secret scan;
13. profile selection through Pi, MCP, HTTP, and Cockpit; and
14. profile digest propagation into evidence and receipts.

## 6. Required document updates

| Document | Required amendment |
|---|---|
| Cockpit 004 | Add low-information perception detection, rendered-SPA fallback, browser identity/challenge Eval fixtures, and independence requirements |
| Cockpit 005 | Add canonical session/tool integration, config-to-runtime coverage, complete operator persona lifecycle, and explicit implementation status per phase |
| Cockpit 005 C01 | Add `BGC-*` links, task IDs, implementation status, tests, and evidence refs |
| Cockpit 008 C04 | Add operator persona secret store, opaque auth-state reference, origin binding, revocation, and deletion receipts |
| `SESSION_API.md` | Replace plaintext auth-state examples and add mode/profile selection to canonical session open |
| `BROWSER_DIAGNOSTICS_SPEC.md` | Add profile/coherence/route/challenge diagnostics without raw identity or secrets |
| `CAPTCHA_SOLVER_SPEC.md` | Separate challenge detection from solver support and add Turnstile, DataDome, and operator-handoff outcomes |
| `FPV_REMAINING_GAPS_SPEC.md` | Make sharing explicit and private-by-default; prohibit automatic control shares during ordinary session creation |
| Cockpit document register | Register this amendment and its eventual machine-readable companion |

## 7. Delivery order

```text
G0  Reconcile this matrix with Cockpit 004, 005, 008 and their ledgers
G1  Create bounded implementation tasks for every BGC row
G2  Make canonical session/Pi/MCP profile selection work
G3  Make detect/no_detect Chromium behavior coherent and testable
G4  Secure operator persona and auth-state lifecycle
G5  Correct FPV private-by-default behavior
G6  Add extraction fallback and challenge event/handoff contracts
G7  Implement network identity and leak checks
G8  Implement Camoufox adapter behind capability negotiation
G9  Implement auto selection, receipts, Research Lab, and full Eval packs
G10 Close only after cross-client and evidence proof
```

Security and canonical path corrections (`G2`, `G4`, `G5`) precede experimental Camoufox work.

## 8. Closure policy

A matrix row may be marked complete only when all of the following exist:

- accepted normative requirement;
- durable task/issue reference;
- implementation reference;
- focused regression test;
- cross-client compatibility proof where applicable;
- bounded evidence reference;
- contradiction review; and
- release-gate result.

Green compilation, a visible Cockpit control, a configuration field, a passing registry test, or prose in a pull request is not sufficient completion evidence.

## 9. False-completion blockers

This amendment cannot close while:

- ordinary Pi/MCP sessions bypass browser-profile selection;
- any active profile setting is silently ignored;
- identity bundles retain contradictory UA, UA-CH, platform, timezone, screen, or network signals;
- operator personas are plaintext, unlocked across processes, or mixed across operators;
- Camoufox parity is claimed without a native protocol adapter and conformance proof;
- challenge detection is presented as successful solver support;
- rendered pages can return empty or irrelevant `browser_read` results without typed fallback;
- ordinary session creation emits a public control bearer URL;
- raw auth state or share tokens appear in normal logs, transcripts, diagnostics, or evidence;
- profile selection and coherence are absent from receipts and evidence;
- open phases lack durable task references; or
- required Eval and regression evidence is missing.

## 10. Cross-repository exclusions

The following observed defects are not owned by UIAI Engine and SHALL be tracked separately in their owning integration surface:

- Focusa browser-workflow validation rendering `[object Object]`;
- missing or undiscoverable Focusa result rehydration;
- unrelated trajectory injection into bounded side-quest browser output; and
- generic web-search adapter failure `onUpdate is not a function`.

They are listed here only to prevent accidental reassignment to Cockpit 005.

## 11. Final principle

A browser identity is a stable, coherent, governed execution context—not a collection of independent spoofing switches. UIAI Engine may claim a browser mode only when the canonical agent path, runtime behavior, security lifecycle, diagnostics, tests, and evidence all agree on what that mode actually did.
