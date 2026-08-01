# UIAI Cockpit Browser Profiles, Identity, Challenge, and Research Amendment

**Document number:** `UIAI-COCKPIT-005`  
**Parent document:** `UIAI-COCKPIT-000`  
**Preceding amendment:** `UIAI-COCKPIT-004`  
**Status:** Proposed normative amendment  
**Version:** 1.0  
**Date:** 2026-08-01  
**Primary implementation home:** `WPUIAI/uiai-engine` Go core  
**Cockpit implementation home:** `apps/cockpit/`  
**Machine-readable companion:** [`UIAI-COCKPIT-005-C01`](./contracts/UIAI_COCKPIT_005_C01_BROWSER_PROFILE_LEDGER_v1.yaml)

---

## 0. Amendment decision

UIAI Engine SHALL promote browser identity, fingerprint, network route, challenge handling, operator presence, and research instrumentation into one selectable **Browser Profile subsystem** in the Go core.

Cockpit SHALL expose the subsystem through Settings, Live session creation, Test Lab, Nodes & Services, Capabilities, and the universal inspector.

The existing CAPTCHA and anti-detection implementation is preserved and generalized. Existing capabilities include:

- text CAPTCHA solving with VLM, Tesseract, and ddddocr fallbacks;
- multi-model and multi-pass voting;
- reCAPTCHA v2 grid and dynamic-tile handling;
- audio fallback through Whisper;
- timing jitter;
- session-scoped and stateless solve APIs;
- ephemeral proxied browsers;
- local-IP and external-proxy pools;
- weighted, least-connections, round-robin, and random routing;
- per-IP health, active probes, cooldown, retry, and flag tracking;
- WebDriver, language, plugin, Chrome-object, and user-agent patches.

These SHALL no longer remain isolated as CAPTCHA-only browser behavior.

---

# 1. Browser Profile model

A profile SHALL resolve to one immutable launch contract:

```yaml
schema: uiai.browser_profile.v1
profile_id:
label:
mode: detect | no_detect | operator | research | auto
engine: chromium | system_chromium | camoufox
headless:
launch:
  executable_path:
  user_data_dir:
  persistent_context:
  extra_args: []
identity:
  fingerprint_profile_ref:
  locale:
  languages: []
  timezone:
  geolocation:
  user_agent:
  client_hints:
  platform:
  screen:
  device_scale_factor:
  hardware_concurrency:
  device_memory:
  fonts_profile:
  webgl_profile:
  canvas_profile:
  audio_profile:
  media_devices_profile:
  webdriver_exposure:
network:
  route: direct | named_proxy | local_ip_pool | tailscale_exit | operator_route
  route_ref:
  dns_mode:
  webrtc_mode:
  geo_consistency:
storage:
  isolation_key:
  cookie_mode:
  cache_mode:
  local_storage_mode:
  credential_profile_ref:
challenge:
  policy: disabled | detect | assist | solve | solve_and_retry
  text_solver:
  image_grid_solver:
  audio_solver:
  max_attempts:
  route_rotation:
  operator_escalation:
behavior:
  timing_profile:
  pointer_profile:
  scrolling_profile:
  navigation_profile:
observability:
  diagnostics_level:
  fingerprint_capture:
  challenge_capture:
  network_capture:
  evidence_grade:
```

The resolved profile digest SHALL be recorded on Session, Observation, Action Receipt, execution capsule, Evidence, and Eval result.

---

# 2. Required operating modes

## 2.1 `detect`

Purpose: deterministic automation, debugging, testing, and maximum observability.

Default posture:

- ordinary Chromium automation identity;
- direct network route unless overridden;
- explicit automation exposure permitted;
- stable viewport, locale, and timing;
- full developer diagnostics;
- challenge detection available;
- challenge solving configurable;
- no fingerprint randomization unless explicitly enabled.

## 2.2 `no_detect`

Purpose: anti-detect browser execution with internally consistent browser and network identity.

Required capabilities:

- selectable Chromium or Camoufox engine adapter;
- automation-control exposure suppression;
- coherent user agent, client hints, platform, screen, locale, timezone, fonts, WebGL, canvas, audio, device, and media-device profile;
- per-context or persistent identity bundles;
- realistic timing, pointer, scrolling, focus, and navigation behavior profiles;
- direct, named proxy, local IP pool, Tailscale exit, or other configured network route;
- DNS, WebRTC, locale, timezone, and geolocation consistency with the selected route;
- challenge detection, solving, retry, route rotation, and operator escalation;
- fingerprint and challenge regression evaluation.

Randomization without consistency is prohibited by contract because contradictory fingerprint fields increase detection. Profile generation SHALL produce a coherent bundle and stable identity for the configured lifetime.

## 2.3 `operator`

Purpose: run as the selected operator context when the task requires the operator's established login, cookies, device posture, network route, browser profile, or interactive presence.

Required capabilities:

- operator-owned persistent browser data directory or imported encrypted profile;
- selected system or Camoufox browser executable;
- operator locale, timezone, screen, fonts, and network route;
- explicit cookie/storage/account identity;
- passkey, MFA, native prompt, and manual takeover support;
- protected operator-entry state;
- session handoff and handback;
- profile lock preventing concurrent corruption of the same user-data directory;
- recording and screenshot controls for authentication intervals.

An operator profile SHALL never be silently converted into an ephemeral identity or mixed with another operator profile.

## 2.4 `research`

Purpose: instrumented challenge, fingerprint, compatibility, and security evaluation.

Required capabilities:

- all profile components independently controllable;
- network, request, response, navigation, console, storage, DOM, accessibility, visual, and challenge traces;
- fingerprint-surface capture before and after navigation;
- challenge type, trigger point, rounds, route, solver, model, retry, result, and evidence capture;
- A/B comparison between engines and profiles;
- deterministic replay where possible;
- Test Lab benchmark-pack integration;
- redacted exportable research packets.

## 2.5 `auto`

Purpose: policy-directed selection between `detect`, `no_detect`, `operator`, and `research` profiles.

Selection MAY use:

- domain and route rules;
- current Workpoint and operation requirements;
- existing operator account/session binding;
- challenge history;
- fingerprint and block-rate Eval results;
- required observability;
- cost and resource availability;
- operator preference.

Selection SHALL be receipted with candidate profiles, selected profile, policy refs, and reasons. A profile transition that changes browser identity or storage SHALL create a new browser context or process and a new Observation baseline.

---

# 3. Go-core architecture

The implementation SHALL add:

```text
internal/browserprofile/
  types.go          canonical profile types and enums
  registry.go       config loading, inheritance, resolution, digest
  launcher.go       engine-neutral launch plan
  chromium.go       Chromium/system Chromium adapter
  camoufox.go       Camoufox adapter and capability detection
  fingerprint.go    coherent fingerprint bundles
  network.go        route selection and consistency checks
  behavior.go       timing, pointer, scroll, and focus profiles
  challenge.go      challenge policy and existing solver integration
  manager.go        profile-scoped browser/context lifecycle
  eval.go           fingerprint and challenge probes
```

## 3.1 Engine adapter

```go
type EngineAdapter interface {
    Capabilities(ctx context.Context) EngineCapabilities
    Launch(ctx context.Context, profile ResolvedProfile) (*BrowserRuntime, error)
    Inspect(ctx context.Context, runtime *BrowserRuntime) (RuntimeFingerprint, error)
    Close(ctx context.Context, runtime *BrowserRuntime) error
}
```

Chromium and Camoufox SHALL share the same profile, session, observation, action, artifact, receipt, resource-lease, and Eval contracts.

## 3.2 Profile inheritance

Profiles MAY use `extends`, but resolution SHALL produce a complete immutable profile. Cycles, unknown keys, inconsistent identity fields, missing executable paths, invalid routes, and shared persistent-data conflicts SHALL fail during configuration validation.

## 3.3 Session selection

Session creation SHALL accept:

```json
{
  "url": "https://example.com",
  "browser_profile": "no_detect",
  "browser_mode": "no_detect",
  "profile_selection": "explicit",
  "operator_profile_ref": null,
  "network_route_ref": null,
  "challenge_policy": "solve_and_retry"
}
```

The response SHALL include effective profile, profile digest, engine, network route label, storage isolation, challenge policy, runtime ref, and any policy override.

## 3.4 Profile-scoped pools

Browser processes, contexts, pages, and sessions SHALL be pooled only when their resolved engine, launch, identity, storage, network, and observability contracts are compatible.

Persistent operator profiles receive exclusive locks. No-detect identities receive explicit lifetime and reuse policy. Research profiles remain isolated from production operator profiles.

---

# 4. Fingerprint consistency system

UIAI SHALL treat fingerprinting as one multi-layer identity rather than independent toggles.

The profile generator and validator SHALL cover:

- browser version and user-agent/client-hint agreement;
- operating-system and platform agreement;
- viewport, screen, scale factor, and window metrics;
- locale, languages, timezone, geolocation, and route geography;
- fonts, font metrics, and text rendering;
- canvas, WebGL renderer/vendor, audio, codecs, and media capabilities;
- hardware concurrency, device memory, touch points, battery, and sensor posture;
- plugins, MIME types, permissions, notification, clipboard, and storage behavior;
- WebRTC candidate exposure;
- DNS and proxy route consistency;
- automation-control properties and browser-object surfaces;
- TLS/HTTP/browser-engine consistency where the selected adapter supports it.

The inspector SHALL report contradictions, missing surfaces, unstable surfaces, and differences from the expected profile bundle.

---

# 5. Network Identity Manager

Network route SHALL be a first-class profile component.

Supported route adapters SHALL include:

- direct node egress;
- existing local-IP SOCKS5 bindings;
- existing external HTTP/SOCKS proxy endpoints;
- named proxy pools;
- Tailscale exit nodes;
- operator-local route relays;
- future managed route providers.

The manager SHALL provide:

- route health and active probes;
- weighted, least-connections, round-robin, random, sticky, and policy selection;
- per-route concurrency and cooldown;
- success, block, challenge, latency, bandwidth, and cost metrics;
- DNS and WebRTC leak checks;
- route/profile geo consistency;
- retry on a different route according to challenge policy;
- route identity on Receipts without exposing secrets.

The existing CAPTCHA IP pool becomes the first implementation adapter rather than a solver-private subsystem.

---

# 6. Challenge subsystem

The challenge subsystem SHALL expose one shared API for detection, classification, solving, retry, and escalation.

```yaml
challenge_event:
  challenge_id:
  session_ref:
  observation_ref:
  profile_ref:
  type: text | recaptcha_v2 | image_grid | dynamic_grid | audio | turnstile | hcaptcha | waf | login_risk | unknown
  provider:
  detected_by: []
  trigger_stage:
  route_ref:
  fingerprint_ref:
  solver_attempts: []
  result: detected | solved | failed | escalated | bypassed_by_operator | disappeared
  evidence_refs: []
```

## 6.1 Existing solver integration

The current text, reCAPTCHA grid, audio, voting, preprocessing, proxy, health, retry, and statistics implementation SHALL be wrapped by the shared challenge interface without feature removal.

## 6.2 Detection

Detection SHALL combine DOM, accessibility, visual, iframe, script, network, navigation, and page-text signals. It SHALL distinguish:

- a challenge visibly present;
- a challenge script loaded but not active;
- an interstitial or WAF block;
- a login-risk or account-verification prompt;
- an ordinary application validation error.

## 6.3 Solver routing

Solver selection MAY use:

- challenge type and provider;
- profile and operator preference;
- configured model/provider chain;
- historical accuracy and cost;
- route/fingerprint health;
- remaining attempts and time budget;
- operator availability.

Every attempt SHALL record solver, model, prompt/profile version, preprocessing, selected tiles or transcription, timing, route, result, and evidence refs.

## 6.4 Extensibility

Challenge providers and solvers SHALL register through typed manifests. New providers SHALL not require duplicating session, profile, route, retry, stats, evidence, or Cockpit contracts.

---

# 7. Challenge and browser research lab

Test Lab SHALL provide a dedicated Browser Identity & Challenge suite with:

- local/synthetic challenge fixtures;
- authorized target packs;
- engine/profile/route/model matrices;
- fingerprint probes and consistency checks;
- CAPTCHA and challenge accuracy evaluation;
- block/challenge-rate evaluation;
- operator-profile continuity tests;
- proxy/IP route comparison;
- session persistence and restart tests;
- stale-state and challenge-transition tests;
- replayable failure anatomy;
- signed benchmark manifests.

Research SHALL produce:

- challenge trigger taxonomy;
- profile leakage report;
- fingerprint consistency score;
- route reputation and health report;
- solve success and false-positive rates;
- cost and latency by challenge/provider/model;
- recommended profile and route policy updates.

---

# 8. Cockpit Settings

Cockpit SHALL add **Settings → Browser Profiles** with these sections.

## 8.1 General

- default profile and mode;
- `auto` selection enabled/disabled;
- default engine;
- headless/headful preference;
- default challenge policy;
- default observability level.

## 8.2 Profiles

- create, clone, rename, validate, export, and delete profiles;
- mode and inheritance;
- engine and executable;
- user-data directory and persistence;
- fingerprint bundle;
- behavior profile;
- storage isolation;
- per-domain rules.

## 8.3 Network Identity

- direct route;
- local IPs and proxy endpoints;
- pool strategy and concurrency;
- Tailscale/operator routes;
- health, cooldown, block, challenge, and latency status;
- DNS, WebRTC, and geo-consistency test.

## 8.4 Challenge Handling

- solver enablement and provider/model chain;
- text, image-grid, dynamic-grid, and audio options;
- attempt, retry, timing, and route-rotation policy;
- operator escalation;
- status, accuracy, latency, and cost.

## 8.5 Operator Profiles

- operator profile reference and data directory;
- account/session bindings;
- route binding;
- profile lock state;
- takeover and recording behavior;
- verify operator fingerprint and login continuity.

## 8.6 Research & Eval

- run fingerprint probe;
- run profile comparison;
- run challenge benchmark pack;
- inspect captures and failure anatomy;
- compare engines, profiles, routes, and solvers;
- promote a tested profile revision.

Every settings change SHALL show whether it applies immediately, requires a new context, requires a new browser process, or requires engine restart.

---

# 9. Cockpit runtime surfaces

Live session creation SHALL offer a compact profile selector with the effective mode and route.

The universal inspector SHALL expose:

- requested and effective profile;
- engine and browser version;
- profile digest;
- persistent/ephemeral storage identity;
- network route and leak-check state;
- fingerprint consistency and contradictions;
- active challenge and solver state;
- profile selection rationale;
- related Eval results.

Activity and Evidence SHALL record profile changes, route changes, challenge events, operator takeover, solver attempts, and benchmark results.

---

# 10. Implementation order

```text
Phase 0  Canonical profile types, registry, validation, digest, and config migration
Phase 1  Chromium launch adapter using detect/no_detect/operator/research profiles
Phase 2  Generalize existing CAPTCHA IP pool and stealth patches into shared adapters
Phase 3  Profile-scoped pools, session selection, receipts, and inspector metadata
Phase 4  Cockpit Browser Profiles settings and Live selector
Phase 5  Camoufox adapter, capability negotiation, and profile parity
Phase 6  Fingerprint consistency probes, network leak checks, and challenge detection
Phase 7  Challenge Research Lab and reproducible Eval packs
Phase 8  Auto profile selection and Focusa autonomy calibration
```

# 11. Acceptance conditions

1. The Go core can resolve and launch `detect`, `no_detect`, `operator`, and `research` profiles.
2. Session creation can explicitly select a profile or use `auto` policy.
3. Existing CAPTCHA and IP-pool capabilities remain operational through the shared challenge and network adapters.
4. Profile identity is internally consistent and has a stable digest.
5. Operator profiles preserve their intended browser data, network route, and exclusive lock.
6. Network-route health, challenge rate, block rate, and leaks are observable.
7. Camoufox and Chromium conform to one engine-adapter contract.
8. Cockpit can edit, validate, test, select, and inspect profiles.
9. Test Lab can compare profiles, engines, routes, and challenge solvers reproducibly.
10. Every Receipt and evidence bundle identifies the effective browser profile and route without exposing credentials.
