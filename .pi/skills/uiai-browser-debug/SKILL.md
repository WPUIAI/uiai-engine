---
name: uiai-browser-debug
description: UIAI browser/session failure debugging: diagnostics-first triage for blank pages, failed selectors, JS/network errors, screenshots, flakiness, and Focusa evidence intake.
---
# UIAI Browser Debug Skill

Use this skill when debugging browser/session failures, blank or broken pages, failed clicks/waits/evals, network/CORS/API clues, screenshots that do not match expectations, or Browser Reliability flakes.

## Core rule

Diagnostics before screenshots-only guessing. Use `browser_read` for content, `browser_snapshot` for @ref action targets, and `browser_diagnostics` for failure evidence.

## Source docs

- `docs/BROWSER_DIAGNOSTICS_SPEC.md`
- `docs/BROWSER_RELIABILITY_RUNBOOK.md`
- `docs/SESSION_API.md`
- `docs/AGENT_UX_COOKBOOK.md`
- `.pi/skills/vision/SKILL.md`

## Quick triage flow

```text
uiai_browser_open url="<url>" width=1280 height=800 focusa_scope={project_root,continuity_id,evidence_ref}
uiai_browser_read session_id="<sid>" max_chars=2000 include_links=true
uiai_browser_snapshot session_id="<sid>" interactive=true
uiai_browser_diagnostics session_id="<sid>" limit=100
```

If page/action is wrong:

```text
uiai_browser_diagnostics session_id="<sid>" limit=100
uiai_errors limit=20 source="browser_session"
```

Only then retry with better selectors/actions.

Cleanup:

```text
uiai_browser_close session_id="<sid>"
```

## Failure playbook

| Symptom | First tool | Likely class | Next action |
|---|---|---|---|
| Blank page | `browser_diagnostics` | console/network/navigation failure | Inspect console/exceptions/failed_requests; capture diagnostics intake. |
| Click failed | `browser_snapshot` + `browser_diagnostics` | selector_not_found/stale selector | Use @ref from fresh snapshot; avoid guessed CSS. |
| Wait timed out | `browser_diagnostics` | wait_timeout/API not loaded | Check failed requests/console; shorten repro; verify selector. |
| Eval failed | `browser_diagnostics` | eval_failed/JS exception | Capture exception summary; prefer direct UIAI actions over long eval. |
| Screenshot wrong | `browser_read` + diagnostics | late load/layout/CSS issue | Wait for stable selector, read content, then screenshot. |
| Session stale | `uiai_errors` | not_found/stale session | Reopen session; cite stale-session error handle if relevant. |
| Network/API failure | diagnostics failed_requests | CORS/API/HTTP status | Capture URL/status/error class; no raw secret query/header values. |
| Flaky CI browser | artifact logs + soak JSON | pool starvation/startup path | Use `/skill:uiai-ci-debug`; reproduce stress/soak locally. |

## @ref action loop

```text
uiai_browser_snapshot session_id="<sid>" interactive=true
uiai_browser_click session_id="<sid>" selector="@e3"
uiai_browser_fill session_id="<sid>" selector="@e4" text="..."
uiai_browser_press session_id="<sid>" key="Enter"
uiai_browser_read session_id="<sid>" max_chars=2000
uiai_browser_diagnostics session_id="<sid>" limit=100
```

Rules:

- Snapshot before action when selectors are uncertain.
- Prefer `fill` over `type` when replacing input values.
- Use `press` for Enter/Tab/Escape instead of JS event hacks.
- Use `eval_async` only for bounded awaits; keep `timeout_ms` small.

## Diagnostics intake to Focusa

For actionable failure evidence:

```text
focusa_browser_diagnostics_intake(
  diagnostics=<diagnostics-json>,
  target_ref="<page/endpoint/object>",
  result="browser diagnostics show <bounded failure summary>",
  project_root="/home/wpuiai/uiai-engine",
  continuity_id="focusa-cont-uiai-engine-82afe24f-90ce-4d6e-b5f2-1b431b7773fc"
)
```

For non-failure proof after a successful read/snapshot/screenshot:

```text
focusa_evidence_capture(
  target_ref="<page/endpoint/object>",
  result="read/snapshot/screenshot verified <bounded outcome>",
  evidence_ref="uiai-browser:session=<sid>:read:<seq>",
  project_root="/home/wpuiai/uiai-engine",
  continuity_id="focusa-cont-uiai-engine-82afe24f-90ce-4d6e-b5f2-1b431b7773fc"
)
```

## HTTP/CLI helpers

Diagnostics:

```bash
curl -s "$UIAI_ENGINE_URL/api/session/$SID/diagnostics?limit=100" | jq '{summary,console,exceptions,failed_requests,focusa}'
curl -s "$UIAI_ENGINE_URL/api/errors?source=browser_session&limit=20" | jq .
```

CLI core loop:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai session open https://example.com'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai session read <sid>'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai session diagnostics <sid>'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai session close <sid>'
```

## Local regression gates

Run after browser/session/action/error changes:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-browser-error-regressions.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-failed-network-diagnostics.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && SESSIONS=4 ROUNDS=10 OUT=/tmp/uiai-local-diagnostics-4x10.json scripts/stress-browser-diagnostics.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && DURATION_SECONDS=30 CONCURRENCY=2 OUT=/tmp/uiai-local-soak.json scripts/soak-browser-flakiness.sh'
```

Full reliability:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && make browser-reliability'
```

## Evidence rules

- Keep raw screenshots/base64 out of transcripts and Focusa state.
- Cite stable handles: `uiai-diagnostics:*`, `uiai-error:*`, `uiai-browser:*`, `uiai-screenshot:*`, `uiai-share:*`.
- Redact URL fragments and secret-like query keys.
- Store bounded summaries, not raw logs.
- Close sessions after proof or cleanup recommendation.

## Final browser-debug report

Include:

- URL/session id or route under test.
- Failure class or success proof.
- Diagnostics/error evidence handles.
- Action taken and selectors used.
- Gates run.
- Cleanup status.
