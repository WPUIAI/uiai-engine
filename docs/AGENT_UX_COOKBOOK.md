# UIAI Agent UX Cookbook

This cookbook gives agents and operators repeatable UIAI workflows with clear intent, tools, evidence handles, Focusa handoff, and cleanup.

## Recipe 1: Search → open → read

Use when an agent needs web content and should avoid raw SERP blobs.

### Pi

```text
uiai_search query="<topic>" limit=3
uiai_browser_open url="<selected result url>" focusa_scope={project_root,continuity_id,evidence_ref}
uiai_browser_read session_id="<sid>" max_chars=2000 include_links=true
uiai_browser_close session_id="<sid>"
```

### MCP

```text
browser_search query="<topic>" limit=3
browser_open url="<selected result url>" focusa_scope={...}
browser_read session_id="<sid>" max_chars=2000 include_links=true
browser_close session_id="<sid>"
```

### CLI/HTTP

```bash
scripts/uiai --json tools search search
curl -s -X POST "$UIAI_ENGINE_URL/api/search" -H 'Content-Type: application/json' -d '{"query":"<topic>","limit":3}'
```

Evidence handles:

- `uiai-search:<provider>:<query-hash>:<rank>`
- `uiai-browser:session=<id>:read:<seq>`

Cleanup:

- Always close the browser session when the page is no longer needed.

## Recipe 2: Persistent browser loop with @refs

Use when the agent must interact with a web UI repeatedly.

```text
uiai_browser_open url="<url>"
uiai_browser_snapshot session_id="<sid>" interactive=true
uiai_browser_click session_id="<sid>" selector="@e3"
uiai_browser_fill session_id="<sid>" selector="@e4" text="..."
uiai_browser_press session_id="<sid>" key="Enter"
uiai_browser_read session_id="<sid>" max_chars=2000
uiai_browser_close session_id="<sid>"
```

Rules:

- Prefer snapshot `@ref` selectors over guessed CSS.
- Read page text when content matters more than pixels.
- On any failed action or unexpected navigation, run diagnostics before patching or retrying.

Evidence handles:

- snapshot/read/session metadata in response `focusa` fields
- browser error events as `uiai-error:<error_id>` when failures occur

## Recipe 3: Diagnostics-first debugging

Use when the page is blank, broken, flaky, or action output is surprising.

```text
uiai_browser_diagnostics session_id="<sid>" limit=100
uiai_errors limit=20 source="browser_session"
```

HTTP:

```bash
curl -s "$UIAI_ENGINE_URL/api/session/$SID/diagnostics?limit=100" | jq '{summary,console,exceptions,failed_requests,focusa}'
curl -s "$UIAI_ENGINE_URL/api/errors?source=browser_session&limit=20" | jq .
```

If diagnostics show relevant console/errors/network failures, compose a packet or intake directly:

```text
focusa_browser_diagnostics_intake diagnostics=<diagnostics-json> target_ref=<target> project_root=<root> continuity_id=<continuity>
```

Rules:

- Do not rely on screenshots alone for broken pages.
- Preserve bounded diagnostic summaries and evidence refs, not raw full logs.
- Clear diagnostics only after evidence is captured or when starting a clean repro.

## Recipe 4: Proof packet capture

Use when an agent needs a bounded, redacted proof bundle for Focusa or a release report.

Steps:

1. Run search/read/snapshot/diagnostics or screenshot/share actions.
2. Compose a packet.
3. Capture/link evidence in Focusa.
4. Close the session.

Pi:

```text
uiai_focusa_packet_compose goal="<proof goal>" mode="proof" responses=[search,read,snapshot,diagnostics] focusa_scope={...} cleanup_session_id="<sid>"
```

One-command CLI:

```bash
scripts/uiai research packet --url https://example.com --goal "Proof packet" --out /tmp/uiai-research-packet.json
```

Manual CLI composer:

```bash
scripts/uiai --json packet compose /tmp/uiai-packet-request.json | tee /tmp/uiai-packet.json
```

HTTP:

```bash
curl -s -X POST "$UIAI_ENGINE_URL/api/agent/research-packet" \
  -H 'Content-Type: application/json' \
  --data @/tmp/uiai-packet-request.json | jq .
```

Focusa evidence capture:

```text
focusa_evidence_capture(
  target_ref=packet.recommended_focusa.args_preview.target_ref,
  result=packet.recommended_focusa.args_preview.result,
  evidence_ref=packet.recommended_focusa.args_preview.evidence_ref,
  project_root=<root>,
  continuity_id=<continuity>
)
```

Evidence handles:

- packet `evidence_refs[]`
- packet `recommended_focusa.args_preview.evidence_ref`
- optional artifact path, e.g. `/tmp/uiai-focusa-packet-smoke.json`

## Recipe 5: Visual QA with screenshot/share artifacts

Use when pixels matter: layout, visual regressions, frame renders, or human review.

One-shot screenshot:

```text
uiai_screenshot url="<url>" width=1280 height=800 format="jpeg" quality=70
```

Session screenshot:

```text
uiai_browser_screenshot session_id="<sid>" fullPage=true
```

Share artifact:

```bash
curl -s -X POST "$UIAI_ENGINE_URL/api/share" -H 'Content-Type: application/json' -d @/tmp/share-request.json
```

Frame render:

```text
uiai_frame_catalog
uiai_frame_render frameId="<frame>" imageBase64="<screenshot>"
```

Evidence handles:

- `uiai-screenshot:sha256:<prefix>`
- `uiai-share:<share_id>`

Rules:

- Keep raw base64 out of transcript and Focusa state.
- Store or cite stable artifact refs and bounded summaries.
- Run diagnostics if the screenshot is blank or visually broken.

## Recipe 6: Release proof loop

Use when changes are ready to ship.

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-focusa-packet-drift.sh && go test ./...'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && bun test ./.pi/extensions/uiai-engine.packet-builder.test.ts'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-pi-extension-registration.sh && scripts/smoke-agent-integrations.sh && scripts/smoke-mcp-tool-routes.sh'
```

Rebuild/restart and live smoke:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && cp ./uiai-engine ./uiai-engine.bak.$(date +%Y%m%d-%H%M%S) && go build -o ./uiai-engine ./cmd/uiai-engine'
systemctl restart uiai-engine.service
curl -fsS http://127.0.0.1:7456/health | jq .
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-focusa-packet.sh && scripts/smoke-agent-integrations.sh'
```

Push and CI:

```bash
cd /home/wpuiai/uiai-engine
gh auth setup-git
git push origin HEAD:main
gh run watch <run-id> --exit-status
```

If CI fails, use [CI Failure Diagnostics Guide](CI_FAILURE_DIAGNOSTICS_GUIDE.md).

Evidence handle format for final report:

```text
gh-run:<run-id>:green+git:<sha>+service:uiai-engine:healthy
```

## Recipe 7: Remote authenticated agent use

Use when the agent is outside loopback.

```bash
export UIAI_ENGINE_URL="https://<authenticated-tunnel>"
export UIAI_API_KEY="..."       # or UIAI_BEARER_TOKEN
scripts/uiai health
scripts/uiai tools graph
```

Rules:

- Discovery endpoints are public; browser/session/search/packet route families require auth remotely.
- Never paste token values into docs, logs, or reports.
- Prefer environment variables and approved secret storage.

## Recipe 8: MCP tool freshness after changes

Use after adding/removing/renaming tools or changing MCP call routes.

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && node --check mcp/browser-session-mcp.mjs && scripts/smoke-mcp-tool-routes.sh'
```

Then restart/reconnect MCP clients and reload Pi sessions that use the MCP adapter.

Symptoms of stale MCP metadata:

- a new tool works in HTTP/Pi but not MCP
- `tools/list` misses a new tool
- a removed tool is still visible
- `tools/call` fails for an advertised tool

## Cross-cutting rules

- Prefer `browser_read` for content and `browser_snapshot` for action targets.
- Prefer `browser_diagnostics` before patching broken browser behavior.
- Prefer stable evidence handles over raw blobs.
- Close sessions when done.
- Use Focusa tools for durable evidence, Workpoint, prediction, and metacog writes; UIAI packets are proposals, not Focusa memory.

## Related docs

- [UIAI for Agents Quickstart](UIAI_FOR_AGENTS_QUICKSTART.md)
- [Session API](SESSION_API.md)
- [Browser Diagnostics Spec](BROWSER_DIAGNOSTICS_SPEC.md)
- [Release Deploy Runbook](RELEASE_DEPLOY_RUNBOOK.md)
- [CI Failure Diagnostics Guide](CI_FAILURE_DIAGNOSTICS_GUIDE.md)
- [Endpoint Auth Matrix](ENDPOINT_AUTH_MATRIX.md)
