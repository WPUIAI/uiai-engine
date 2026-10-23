> Parent authority: https://github.com/WPUIAI/uiai-engine/issues/106
> Canonical source: https://github.com/WPUIAI/uiai-engine/issues/106

## Severity
P0 operator-evidence delivery gap

## Problem
A completed UIAI responsive/browser proof run currently returns terminal-oriented handles such as `uiai-diagnostics:session=<id>:seq=<n>` and short-lived live-session FPV URLs. Neither is an operator-usable QA artifact:

- `browser_fpv_share` is session-bound, expires in at most 240 minutes, and stops being useful after browser cleanup.
- `browser_screenshot output=url` exposes individual images, not a proof report.
- Focusa research/diagnostics packets are machine envelopes, not a human artifact PWA.
- Closing the browser—as required by UIAI cleanup guidance—removes the only interactive presentation surface.

This caused a real handoff failure after a Focusa Homepage V2.1 run at 1440/768/390/320: the operator received an internal evidence reference rather than a shareable report link.

## Reproduction
1. `browser_open https://focusa.dev/`
2. Resize and capture assertions/screenshots at 1440, 768, 390, and 320.
3. Capture `browser_diagnostics`.
4. Close the session.
5. Attempt to give the operator one durable human-viewable artifact link containing the run.

**Actual:** no registered UIAI tool or default workflow produces one.

**Expected:** UIAI publishes a durable, read-only, templated Browser Proof PWA and returns its URL by default for proof/QA workflows.

## Required fix
Add a canonical publish surface (name illustrative: `browser_proof_publish`) that accepts bounded existing UIAI responses/artifact refs and emits one immutable report artifact.

### Report contents
- Target URL, title, run timestamp, engine/version, viewport matrix.
- Per-breakpoint screenshot cards.
- Structured assertions with pass/fail and measured values.
- Interaction trace for reproduced defects (for example, menu open → scrollY changed).
- Console, exception, failed-request, HTTP 4xx/5xx summary.
- Source/evidence references for machine consumers.
- Clear generated/non-live evidence labeling.

### Delivery contract
- Return `artifact_id`, `artifact_url`, `created_at`, `expires_at` or retention class, content hash, and redaction status.
- Artifact must remain viewable after `browser_close` and engine/browser restart.
- Default read-only URL; explicit access class (`private`, `unlisted`, `public`) with safe default.
- No credentials, cookies, authorization headers, form secrets, or private browser state in report payloads.
- Idempotent publication for the same proof manifest.
- Responsive, mobile-first PWA template suitable for direct operator/customer review.
- Downloadable bounded JSON manifest for audits.

### Workflow integration
- `browser_open`/`browser_resize`/`browser_screenshot`/`browser_diagnostics` responses remain composable.
- Proof-mode packet composition should recommend or invoke publication—not return only an internal handle.
- Pi/MCP/OpenAI schemas and agent card must discover the publisher.
- Cleanup may close the browser only after durable artifact publication succeeds.

## Acceptance
1. Automated API/tool test publishes a multi-breakpoint report and verifies it after session closure.
2. PWA displays desktop/tablet/mobile screenshots, assertions, and diagnostics without terminal access.
3. Redaction tests prove secrets cannot appear.
4. Restart-persistence test passes.
5. Agent card/tool search exposes the feature for `artifact`, `proof`, `share`, `report`, and `PWA` queries.
6. A Focusa Homepage V2.1 dogfood run returns one operator-clickable artifact URL.

## Fix plan
1. Define versioned proof-manifest and artifact-delivery contracts.
2. Add immutable artifact storage plus bounded retention/access policy.
3. Implement report renderer/PWA route and publish tool/API.
4. Wire tool registry, MCP/OpenAI schemas, CLI, and agent guidance.
5. Add unit, redaction, persistence, and end-to-end dogfood tests.
