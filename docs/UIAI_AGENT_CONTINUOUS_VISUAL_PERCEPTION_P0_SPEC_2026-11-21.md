# UIAI P0 — Agent-Owned Continuous Visual Perception
**Date:** 2026-11-21
**Repository:** WPUIAI/uiai-engine
**Priority:** P0, requested by the owner in the live product discussion
**Architecture status:** PROPOSED FOR EXACT OWNER REVIEW. P0 priority and intended outcome are owner-requested; the technical choices below are recommended, not automatically promoted to canonical architecture.
**Implementation status:** Not implemented by this specification PR. No installed runtime, benchmark, deployed acceptance, or successful user visual review is implied.
**Problem:** Agents inspect source, DOM, and a few static screenshots while humans perceive a continuous rendered interface. This creates severe failures in premium visual quality, responsive composition, dynamic states, and artwork fidelity.

## 0. Outcome and strict scope
Give each authorized agent a private, continuously maintained, **pixel-informed browser observation** so it can look, interpret, act, and look again. The first production proving ground is the responsive and editorial quality of The Entrepreneur's Path website. Make visual observation the primary verification channel for visual-design work, without suppressing semantic accessibility, DOM, diagnostics, tests, source specifications, or deterministic assertions.

This is **agent perception**; it is NOT the human FPV viewer, a human video stream, a duplicate browser, a new Focusa/Radar observer, a new memory store, or permission for independent autonomous action. Do not change existing scope, authority, or architecture silently. Propose any additions, removals, deferrals, or changed implementation ownership for explicit owner approval. All requirements stay open until evidenced.

## 1. Source architecture and reuse
- Reuse internal/vision/session_screencast.go: existing Session.CDPScreencast, its single page-owned hub, subscriber fanout, latest-only bounded channels, and screencast ack semantics. No second Chrome screencast pump for the same page. Start a private agent-observation subscriber under the current session lease; do NOT route private agent perception via public /m/ share tokens.
- Reuse existing browser Session API, document/navigation/frame identities, semantic @refs, DOM/accessibility snapshots, diagnostics, image artifacts, capture authorization, influence firewall, and EPWA delivery contracts. The event stream does not mint new canonical sources of truth.
- Preserve UIAI as browser observation/execution authority, Focusa as Workstream/task/authority/receipt owner, Veragensia as OS/DesktopObservation composition authority. Radar may consume bounded existing observation refs but never receives an extra screenshot poller. Preserve control leases, generations, fencing, re-observation after takeover, and fresh origin/target checks.
- Preserve the canonical workspace /home/wpuiai/uiai-engine and its existing .focusa genesis markers; the implementing agent must inspect local status, bd, active branch/worktrees and unrelated changes before editing. This PR does not authorize creation of a persistent duplicate checkout.

## 2. Recommended execution architecture (priority order)
Browser compositor/CDP screencast and browser semantic events
    -> session-bound private capture subscriber with timestamp and runtime-generation binding
    -> **cheap first-stage change triage** in the existing Go UIAI engine
    -> changed tiles/regions + semantic event hints + temporal coverage keyframes
    -> bounded private visual observation with full-scene scan and optional high-resolution focused crops
    -> provider-neutral multimodal analysis on selected observations
    -> compact findings, uncertainty, image refs and causal links to the same agent's next governed action
    -> fresh post-action observation and explicit visual + functional verification
    -> existing Evidence/EPWA and Focusa predicate/Receipt handoff where required

### 2.1 Cheap change triage, not continuous VLM calls
Compute bounded per-tile luminance difference, perceptual hashes, optional SSIM-like measures, dirty-region count, temporal novelty and layout movement. Start in Go using standard image decoding/math and already accepted dependencies. Avoid heavy learned embeddings for every incoming frame. Apply small region masks but never conclude a static-looking frame is semantically unchanged merely from a global hash.

Browser-native event hints (read-only, bounded, permission-checked) include navigation, resize, scroll, focus, modal/dialog changes, DOM/accessibility state deltas, image load/completion, layout shift, animation/transition, form validation and click aftermath. Select the smallest observation set that can establish what actually changed. Frame difference and semantic events are complementary; neither is the sole source of truth.

Mandatory keyframes: first settled page, page/target/navigation replacement, every consequential action aftermath, viewport or device-scale change, reference-comparison request, detected high-impact visual state change, periodic bounded heartbeat, before marking a visually reviewed task complete, and explicit model request. Capture short transient episodes without allowing unlimited history growth. Preserve enough temporal context to recognize animations and toasts, plus a timeout when content never settles.

### 2.2 Agent-oriented observation modes
One logical native capability, not unrelated tool families:
- SCAN: low-cost complete viewport, sufficient to establish overall composition, hierarchy, imagery, whitespace and major visual differences; include changed-region annotations.
- FOCUS: high-resolution full-color crop(s) grounded in the current frame, coordinate system and exact viewport/scale; apply to typography, CTA, artwork/crop, tiny labels, icons, and suspected defects. Retain a full-scene anchor alongside crops so context is not silently lost.
- READ: existing DOM/native accessibility and text content as complementary semantic input, not as a substitute for rendered pixels.
- COMPARE: actual TWO visual inputs to an explicitly multi-image-capable model, or a deterministic image diff plus bounded honest explanation if provider lacks multi-image support. Never claim that the model inspected both when only one is supplied.
- TIMELINE: bounded set of change events/keyframes with ordering and selection reasons; no indefinite screenshot transcript dump.

The agent's model must receive supported image content parts, or a strongly typed authorized image handle it can resolve. Returning a URL or SHA alone without a path that actually presents pixels is a **blocked** perceptual outcome, not success. Tool adapters (native Pi, Focusa Pi, MCP, CLI) must truthfully advertise their capabilities.

### 2.3 Memory and model-side optimization
Keep a per-session bounded keyframe/observation ring keyed by document and runtime generation; no competing durable memory database. Keep compact state summaries and refs in agent context; re-resolve high-resolution crops on demand. Prefer event batching, keyframe diversity, and question-aware selection under budgets. Do not train a new VLM or implement proprietary internal model KV-cache alterations in P0. Learned patch/token pruning may be evaluated later behind provider capability detection and exact benchmarks. Caches must be invalidated on navigation/frame/runtime changes.

### 2.4 Runtime/transport choice
- **PRIMARY:** Existing Chrome DevTools Protocol Page.startScreencast plus in-process Go channels, avoiding network streaming for agent and engine co-resident on the VPS. Existing single producer, latest-frame subscriber semantics, bounded per-session budgets and backpressure. CDP is observation; it does not itself reason.
- **REMOTE AGENT:** Existing authenticated private engine transport and stable image handles; WebSocket is optional for push/control or transport only where supported and proven necessary. No public viewer token or generic wildcard SSE channel for private perception.
- **WebRTC:** Do not introduce it for the internal perception loop. Reserve for separately approved low-latency remote human media paths if the FPV transport owner establishes a need.
- **WebAssembly SIMD / WebGPU:** Optional measured accelerators, not architectural prerequisites. Go native path remains the portable default across Linux, macOS and Windows. WebGPU availability, server GPUs, page trust boundaries and extra memory copies must be benchmarked before introduction.
- Preserve compatibility with existing CPU-only hosts, multi-session leases and browser cleanup. Do not add a second browser automation framework or put Playwright into Focusa.

## 3. Provider-neutral real visual reasoning
A higher-level model assesses intended aesthetic outcome, layout quality, reference fidelity, optical balance and content correctness *only on genuinely observed frames*. It must emit typed defect: issue category, severity, observation + crop refs, location/coordinate binding, confidence, evidence, recommended permitted repair, and recheck requirement. Model critique is judgment/evidence candidate, not authoritative correctness by assertion.

For editorial sites, follow approved brand/art/typography specifications. No universal rubric that penalizes a white background merely for being white; design cohesion is measured against the brand's approved direction, not generic decorative density. Maintain representative full-page and full-viewport context so crop selection does not reward local correctness while the overall composition is poor.

## 4. Hard correctness and safety rules
1. Every frame/region binds to exact session, browser context, target, document/navigation generation, viewport dimensions, DPR, scroll offset, capture time and observation sequence. Stale frames must not ground actions.
2. A current control lease and approved Focusa operation are mandatory for actions; perception is read-only and grants no additional actuation, origin access, credentials, private data egress, or authority. A control takeover requires reconciliation and fresh observation before agent resume.
3. Browser text, visual text, image/OCR, model captions and page-generated audio are untrusted input through existing influence firewall; never become operator instructions.
4. Redaction/isolation apply to capture, thumbnail, crop, model request, EPWA, logs and artifacts. Public FPV shares and private model perception are separate access classes. No raw secrets or screenshot/base64 bytes in generic text logs.
5. Keep precise bytes for the local authorized model adapter as permitted by privacy policy; never silently weaken EPWA project-evidence requirements. Ordinary authorized ad hoc observation must use the approved distinct private unsealed path, once #299/#308 is resolved, not impersonate formal project evidence.
6. On stream loss, document replacement, permission/entitlement mismatch, pixel-delivery failure, over-budget, or provider unavailability, emit typed degraded/blocked status and recover with authorized resync; never hallucinate vision or silently fall back to code-only for a task that requires visual verification.
7. Bound FPS, capture quality, ROI count, frame-in-flight, bytes/min, image tokens, inference calls, session lifetime, memory and per-agent CPU use. Fairness across concurrent agents and cancel/cleanup are mandatory.
8. Existing evidence, source and receipts remain discoverable and their verification classes truthful.

## 5. Known code defects to address as dependencies
- internal/routes/screenshot_compare.go currently passes ONLY the first image into the VLM while representing the second by its byte length. Replace the invalid comparator with a true two-image model path, or honestly refuse unsupported multimodal comparison; cover with a fixture in which both images are visually essential.
- integrations/pi/uiai-engine.ts uses withoutScreenshot for browser open/screenshot tool output; existing EPWA artifacts are not necessarily consumer-visible pixels. Integrate with ongoing #299, #238, #224 and PR #308 instead of weakening capture/privacy guards.
- Existing internal/routes/critique.go has aesthetic default caps that treat plain white surfaces as poor, inappropriate for approved premium editorial/light-first design. Make evaluation spec-aware, calibratable and evidence-grounded; stop claiming visual review without a real image.
- #68 tracks human FPV transport reliability; do not conflate it with this internal observer.
- #294 separately tracks understanding <video> and webpage audio; a rendered GUI observation loop does not automatically implement media decoding/transcription.

## 6. P0 implementation packets and artifacts
**A — Pixel truth / adapter contract.** Make one supported model/agent path receive an actual page image, with exact scope/security, typed error on failure, adapter parity and tests. Work with existing #299/#308, no duplicated evidence bypass. No continuous-vision success claim without this gate.

**B — Private session observation sampler.** Subscribe to existing screencast hub; add bounded local frame triage, change masks, event reasons, forced keyframes, backpressure, runtime/viewport binding, timeouts and resync. Use session lifecycle; no unrelated poller.

**C — Perception routing + model integration.** Deliver SCAN/FOCUS/READ/TIMELINE/COMPARE through current agent exchange profile, accurate image content/handles, provider support checks, actual two-image comparison, cost accounting, and structured findings. Preserve browser action controls.

**D — Visual QA harness and design gate.** Produce reproducible real-browser receipts for responsive width sweeps, scroll, interactions, light/dark, zoom, focus, animation and visual regression. Test reference screenshots with matching dimensions/fonts/stable rendering; compare pixel/layout and human-reviewed aesthetic judgments separately. Attach evidence to existing EPWA, not a second report store.

**E — Operator/deployment validation.** Pinned source-to-installed binary, macOS/Linux/Windows where applicable, native Pi/MCP/CLI parity, soak/leak/authorization tests, real headless and headed browsers, independent visual review of the first proving-ground site, rollback. Never close on docs, PR creation, a pretty screenshot alone, or unexecuted tests.

Agents can implement A–E with dependency-ordered atomic PRs under this one P0 contract. No packet may silently defer, delete or enlarge another requirement. Each packet records exact changed files, build/tests run, installations tested, observations inspected, performance measurements and limitations.

## 7. Objective efficiency accounting — not marketing estimates
Define baselines on identical hardware, browser, site route, viewport, action trace and selected model:
- Baseline-1: every sampled screencast frame submitted to a VLM (naive theoretical upper-cost reference).
- Baseline-2: screenshots after each UI action (typical practical agent reference).
- Candidate: continuous cheap triage plus selected visual calls, same source pixels and task.

Measure source frames/second, compositor/frame decoding CPU, selection latency p50/p95, browser RAM, dropped frames, consumer-visible image delivery, total vision-model frames and visual tokens, inference time/cost, network bytes, false-negative critical defects, visual severity recall, resolved defects, interaction task success, and **total cost per externally verified correct outcome**. Report false-negative and lost-frame cases, including deliberate short-lived toasts and one-character visual changes; no generalized '99% cheaper' claim from a single demo.

ILLUSTRATION ONLY: 8 source frames/second for 60 seconds = 480 locally inspected candidate frames. Sending 24 selected frames instead of all 480 is 95% fewer model-bound frames, but this is NOT a measured 95% reduction in end-to-end cost, model tokens or error rate. Selection, image crops, observation delivery, model latency and missed events must be measured separately. Comparing against a useful screenshot-after-action baseline is mandatory.

Acceptance requires an actual benchmark on responsive website traces and at least one unrelated interactive site; no severity-1/critical issue suppressed by change filters in required fixtures; no stale/unauthorized action; no secret leakage; no silent unavailable image; bounded resource/latency behavior; positive visual fidelity and functional evidence at agreed site states; agent can inspect images without requesting a human FPV feed.

## 8. The Entrepreneur's Path P0 proving ground
Test at **365, 768, 1024, 1440 CSS pixels**, plus 320/390 and width sweeps between major breakpoints, each with meaningful page sections, interactive states, scroll paths, dynamic content and viewport height variants. Exact website approved artwork, locked font/typography, image focal points, margins, breakpoint header/nav behavior, fluid/clamp spacing, art direction, reader flow and sale/checkout states must be independently visually checked. Do not equate four screenshots with an hour of human navigation.

On every visual repair: source/spec/reference -> installed/staging rendering -> agent's real observed pixels -> visual finding -> authorized edit -> new render -> evidence-backed recheck. Do not declare premium-ready because CSS compiles. Customer checkout remains separately gated by real EDD/Stripe verification and authorized test purchases.

## 9. Research and technology references (evidence, not adopted dependencies)
- Microsoft/EMNLP Findings 2026, ReVision — redundant GUI screenshot patches; 46% fewer visual tokens and +3% task success in the reported Qwen2.5-VL-7B setup. https://aclanthology.org/2026.findings-emnlp.2180/
- ICML 2026, VideoBrain — adaptive selection on long video, 30–40% fewer input frames with +3.5% to +9% reported benchmark gains; not a drop-in browser agent. https://proceedings.mlr.press/v306/zou26h.html
- CVPR 2026, StreamingTOM — token/cache compression, 15.7x KV compression and 2x TTFT in its evaluated streaming setup; model-internal methods not equivalent to browser transport. https://arxiv.org/abs/2510.18269
- 2026 preprint, Coverage-Driven Adaptive Keyframe Selection — 4–13x fewer candidate-frame scores and 3.1–5.4x speed in selected frame-selection benchmarks; warns that a heavy selector can be slower than inference. https://arxiv.org/abs/2608.00714
- 2025 neuromorphic event-camera survey — asynchronous sensor brightness-change events; the browser does not provide a physical DVS event sensor. The transferable idea is *sparse changed-region information*, not hardware substitution. https://pmc.ncbi.nlm.nih.gov/articles/PMC12526923/
- Chrome DevTools Protocol Page.startScreencast — existing supported Chrome browser instrumentation; transport/sampling, not reasoning. https://chromedevtools.github.io/devtools-protocol/tot/Page/

## 10. Existing authority and interface references
- AGENTS.md, docs/ARCHITECTURE_AUTHORITY_POLICY.md.
- docs/UIAI_COCKPIT_002_AGENT_FIRST_BROWSER_AMENDMENT_2026-07-19_v1.0.md and machine-readable C01 ledger.
- docs/contracts/UIAI_COCKPIT_008_C03_OPERATOR_CONTROL_LEASE_TAKEOVER_RECONCILIATION_v1.yaml.
- docs/UIAI_VERAGENSIA_COMPUTER_CONTROL_AND_VOICE_BINDING_2026-09-04.md; Focusa 181–184 / Veragensia 193–199 as required.
- Related UIAI issues #68 #238 #299 #294 and open PR #308.

**Definition of done:** an authorized working agent continuously receives meaningful, correctly bound visual perception of the live rendered UI, uses it in its own reasoning/action loop, catches visible defects humans notice, rechecks edits visually, and produces independent reproducible cost/accuracy/latency/safety proof. A document alone never closes this P0.
