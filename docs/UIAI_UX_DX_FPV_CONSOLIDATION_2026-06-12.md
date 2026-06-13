# UIAI UX/DX/FPV Consolidation — 2026-06-12

Source docs created on 2026-06-09:

- `docs/UIAI_BROWSER_UX_DX_FEEDBACK_2026-06-09.md`
- `docs/UIAI_BROWSER_RECOMMENDATION_CONFLICTS_2026-06-09.md`
- `docs/UIAI_AGENT_FPV_COPILOT_SPEC_2026-06-09.md`
- `docs/UIAI_AGENT_FPV_PWA_SPEC_2026-06-09.md`

## Consolidated decision

Treat the 6/9 docs as one product path:

1. **DX reliability primitives**: make current agent browser tools easier to use safely and debug quickly.
2. **UX evidence surfaces**: make screenshots, diagnostics, and sessions easier for humans to inspect and share.
3. **FPV PWA MVP**: expose a shareable read-only live operator view before adding steering controls.
4. **FPV co-pilot steering**: add typed/click/draw operator controls after read-only FPV is proven.

## Conflict resolutions carried forward

From `UIAI_BROWSER_RECOMMENDATION_CONFLICTS_2026-06-09.md`:

- Diagnostics: implement composable filters on `browser_diagnostics`; do not split into many tools first.
- Screenshots: add first-class file/url/json output rather than teaching agents to unwrap base64 manually.
- Waits: make navigation/read actions auto-wait, while preserving explicit wait tools for complex cases.
- Agent UI: prioritize practical MCP/Pi tool improvements before building a full playground.
- Focusa integration: keep UIAI as evidence/proposal execution surface, not Focusa authority.

## Implementation breakdown

### Bead A — Screenshot evidence output

Status: implemented in current slice for session screenshots.

Scope:
- Add `output: "json" | "file" | "url"` to browser screenshot tools/routes where applicable.
- Ensure file/url outputs return a stable artifact path/ref and omit base64 by default.
- Update Pi + MCP schemas and docs.

Why first:
- Top DX pain (#1) and enables FPV/evidence sharing without huge JSON blobs.

Smoke:
- Session screenshot with `output=file` creates readable artifact.
- Existing screenshot JSON behavior remains backward compatible.
- Pi/MCP registration and route parity pass.

### Bead B — Diagnostics filtering + summary

Status: implemented in current slice.

Scope:
- Add server-side filters for diagnostics: `level`, `category`, `since_seq`, `failed_only`, `format=summary|full`.
- Keep one diagnostics tool; wrappers can come later.
- Include bounded counts and latest seq in summary.

Why:
- Resolves conflict A and reduces agent/tool-output overload.

Smoke:
- Browser error regression tests pass.
- Diagnostics route filters console/network/exceptions without returning unrelated entries.

### Bead C — Auto-wait and text selector ergonomics

Scope:
- Add optional auto-wait behavior to navigate/read/click/fill where safe.
- Add text/role selector support or a safe helper endpoint that resolves text to @ref/CSS.
- Preserve explicit wait for complex workflows.

Why:
- Top-5 DX items #4/#5 and makes current browser sessions more reliable before FPV.

Smoke:
- Navigation/read waits for ready content.
- Text selector resolves a simple visible button/link in a fixture page.

### Bead D — FPV PWA read-only MVP

Scope:
- Add a session share endpoint that returns a short-lived URL/token for read-only live session view.
- Add minimal PWA page: current screenshot stream/poll, URL/title, session status, diagnostics summary.
- No operator steering in MVP; only observe + copy evidence/session refs.

Why:
- Matches `UIAI_AGENT_FPV_PWA_SPEC_2026-06-09.md`: zero-install operator oversight first.

Smoke:
- Open a browser session, create share link, fetch PWA page, verify status/screenshot metadata updates.
- Link expires or rejects unknown token.

### Bead E — FPV co-pilot steering controls

Scope:
- Add typed operator commands for click/type/press/draw annotations through the share session.
- Gate controls behind explicit token permission and audit log.
- Keep read-only PWA MVP stable before enabling steering.

Why:
- Implements second phase of FPV co-pilot spec after read-only operator view is proven.

Smoke:
- Steering action records audit entry and maps to existing browser action endpoint.
- Unauthorized token cannot mutate session.

## Current status after consolidation

- 2FA support is done and shipped separately (`6d27e10`, config cleanup `0e495ba`).
- The 6/9 UX/DX/FPV docs are not yet decomposed in beads before this consolidation.
- Recommended next implementation slice: **Bead A — Screenshot evidence output**.
