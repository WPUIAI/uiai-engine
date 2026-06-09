# UIAI Engine — Recommendation Conflicts Analysis

## Type 1: Same problem, two proposed fixes (pick one)

### Conflict A: #3 vs #47 — Diagnostics filter
- **#3** (Round 1): Add `level`, `category`, `since_seq` params + `format: "summary"`
- **#47** (Round 2): Split into 3 sub-calls (`uiai_browser_console`, `uiai_browser_network`, `uiai_browser_exceptions`)
- **Verdict**: **Pick #3.** #47 is a more invasive refactor that splits one tool into three. #3 keeps the same tool but adds proper filtering. Filters compose better with arbitrary new diagnostic categories (storage events, web vitals, etc.) than hard-splitting into 3.
- **Resolution**: Implement filtering first; if specific workflows still need a single-category-only call, add a thin wrapper later.

### Conflict B: #1 vs #109 — Screenshot output
- **#1** (Round 1): Add `output: "file"|"url"|"json"` to `uiai_browser_screenshot`
- **#109** (Round 2): Add a first-line note to the response saying "use jq to extract"
- **Verdict**: **Pick #1.** #109 is a band-aid; #1 fixes the root cause. The metadata-only response shape is a design smell, not a doc gap.

### Conflict C: #6 vs #7 vs #113 — Long-running operations
- **#6** (Round 1): Async tool calls with polling (`uiai_browser_click --async` returns request_id)
- **#7** (Round 1): "Session auto-restart after hang"
- **#113** (Round 2): "Auto-resume mode that re-opens session with same cookies"
- **Verdict**: **Pick #6.** #7 and #113 are *both* attempts to paper over the same root cause: blocking tool calls in an event-driven runtime. Async-by-default is the correct architectural choice. Auto-resume is a useful complement for *failures* but doesn't address the underlying UX of waiting.
- **Resolution**: Async-first, with a fallback "auto-resume on next call" only for crash recovery, not for wait time.

### Conflict D: #16 vs #22 vs #58 — Multi-action compositions
- **#16** (Round 1): "Scenario runner" — JSON array of actions
- **#22** (Round 1): "Batched input+submit+screenshot" — one composite primitive
- **#58** (Round 2): "Composite navigate+fill+click+screenshot"
- **Verdict**: **All three are the same idea.** #16 is the most general (N actions), #22 and #58 are special-cases. Don't ship all three.
- **Resolution**: Ship **#16 as the general primitive**, then add named convenience macros (`uiai_browser_scenario.form_submit`, `uiai_browser_scenario.visual_audit`) that expand to the underlying scenario runner.

### Conflict E: #63 vs #112 — Error envelopes
- **#63** (Round 1): Richer error envelopes with `code`, `retryable`, `last_tool`, `target_selector`
- **#112** (Round 2): Distinguish 504 vs 502 vs timeout in error envelopes
- **Verdict**: **They're the same fix at different levels of detail.** #112 is a subset of #63.
- **Resolution**: Implement #63 with `code` as an enum that includes the 504/502/timeout variants from #112. Don't ship two different error schema.

---

## Type 2: Overlapping but distinct (coexist with care)

### Conflict F: #71 vs #117 — Sessions as objects vs. IS the workpoint
- **#71** (Round 3): UIAI sessions as first-class Focusa objects with properties
- **#117** (Round 3): UIAI session *is* the workpoint's web context (the workpoint holds the session)
- **Verdict**: **They are not the same.** #71 is "sessions are linkable objects" (orthogonal, additive). #117 is "delete the distinction between session and workpoint context" (a fundamental redesign).
- **Resolution**: Ship #71 first as it stands on its own. #117 is the long-term north star that obsoletes #71. Once #117 lands, deprecate #71's session model.

### Conflict G: #73 vs #78 — Diagnostic correlation vs. session scope
- **#73** (Round 3): Diagnostic events tagged with `cause_action_id`, `workpoint_id`
- **#78** (Round 2): Diagnostics aren't session-scoped (current bug)
- **Verdict**: **Different problems.** #78 is a *correctness* fix (events leak across sessions). #73 is a *feature* (semantic correlation).
- **Resolution**: Both needed; do #78 first as a bug fix, then #73 as a feature.

### Conflict H: #70 (auto-attach) vs. agent control
- **#70** (Round 3): `uiai_browser_capture_evidence({auto_attach_to_workpoint: true})` auto-calls `focusa_evidence_capture`
- **Concern**: Auto-attach can fight explicit agent control. An agent may want to take a screenshot for visual inspection without binding it to the workpoint.
- **Resolution**: Default to `auto_attach: false`. Only set `auto_attach: true` when the agent explicitly opts in. Or invert: default to true but allow `auto_attach: false` opt-out.

---

## Type 3: Subtle contradictions (design attention needed)

### Conflict I: #24 vs responsive audit pattern
- **#24** (Round 2): Viewport state sticky across navigations
- **Pattern**: A responsive visual audit *requires* multiple viewport resizes per page
- **Verdict**: **Sticky viewport is correct** because the dominant case is "I'm staying on one URL, scrolling through it". For the audit pattern, agents explicitly `resize` between viewports, so the resize is intentional. But the resize tool's *side effect* of always returning a screenshot (Round 2 #15) is wasteful.
- **Resolution**: Make resize `return_screenshot: false` by default. The 1280 default should also be configurable per session.

### Conflict J: #38 (set state) vs. #45 (disable cache) vs. #94 (baseline diff)
- **#38**: Inject localStorage/cookies to set up state
- **#45**: Disable cache to test fresh loads
- **#94**: Compare against a saved baseline
- **Concern**: If state changes between baseline and current run, the diff is false-positive. If cache is disabled, the diff is true-positive but slow.
- **Resolution**: Bundle the three into one `uiai_browser_audit({baseline: "id", cache: "bypass", setup: {...}, viewport: 1280})` composite. The composite owns the state-vs-cache-vs-baseline bookkeeping.

### Conflict K: #42 (intercept route) — security/auditability
- **#42**: Stub network responses with `route.fulfill()`
- **Concern**: A naive implementation lets an agent silently mutate API behavior and miss bugs. Or worse, an agent could stub authentication and produce "passed" tests that don't reflect production.
- **Resolution**: The intercept primitive must record every stub applied to a `focusa_evidence` record so the test is auditable. The workpoint's "not_done_if" should include "intercept was used".

### Conflict L: #43 (throttle) can affect other primitives
- **#43**: Throttle CPU/network for performance testing
- **Concern**: If the agent throttles then tries to do a normal interaction, the interaction may time out and be reported as a hang. The session's behavior changes under throttle.
- **Resolution**: Throttle mode should be a per-session flag (`uiai_browser_open({perf_mode: "slow_3g"})`), not a global toggle. Clear visual indication that the session is in perf-mode.

### Conflict M: #118 (learn from past audits) — false confidence
- **#118**: Suggest "you ran this scenario yesterday; replay?"
- **Concern**: If the underlying state has changed (DB, plugin version, third-party API), replaying produces "passed" tests that don't reflect reality. This is the "false-positive regression" risk.
- **Resolution**: When suggesting replay, always include a "validity check" step: re-verify the underlying state matches the original audit (e.g. compare to a state-hash from the previous run). If different, surface "stale" and prompt a fresh audit.

---

## Type 4: Big-picture philosophical conflict (must resolve)

### Conflict N: #116 ("do this on the web") vs. browser-primitive philosophy
- **#116**: Higher-level intent-based primitives (`uiai_do({intent: "log in"})`)
- **Implicit current direction**: Atomic browser primitives (`click`, `fill`, `screenshot`)
- **Verdict**: **Not strictly conflicting, but they compete for the same surface area.** If both exist, agents don't know when to use which.
- **Resolution**: Position intent-primitives as the "default" for new agents. Keep atomic primitives as "advanced" for power users and edge cases. A "level" parameter on session-open could choose: `level: "intent"` (high-level) vs `level: "primitive"` (low-level).

---

## Type 5: Gaps that look like recommendations but are pre-requisites

### Conflict O: Tooling changes need docs updates
- Every primitive change (e.g. #1, #6, #16) needs to update:
  - `uiai-engine/README.md`
  - `uiai-engine/docs/AGENT_UX_COOKBOOK.md`
  - `uiai-engine/docs/UIAI_FOR_AGENTS_QUICKSTART.md`
  - `uiai-engine/docs/SESSION_API.md`
  - `/root/.pi/skills/wpuiai-workflow/SKILL.md`
- **Without coordinated doc updates, the recommendations make the engine *worse* for new agents**, because they'll have stale docs and try old patterns.

---

## Summary

Of 120 items:
- **5 hard conflicts** (Type 1) — same problem, two proposed fixes
- **3 philosophical overlaps** (Type 2) — coexist but need design care
- **4 subtle contradictions** (Type 3) — work but need cross-references
- **1 big-picture tension** (Type 4) — primitive vs. intent philosophy
- **Implicit: doc-update pre-requisite** (Type 5) — every code change needs doc updates

The most important takeaway: **#16 (scenario runner) subsumes #22 and #58, and #63 (richer errors) subsumes #112**. Don't ship both. Pick the most general and let special cases compose.

The second most important: **#117 (UIAI IS workpoint) is a 6-month redesign**. Don't conflate it with #71 (sessions as objects). Ship #71, plan #117, but don't try to do both at once.
