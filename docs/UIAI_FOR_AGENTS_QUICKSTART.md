# UIAI for Agents Quickstart

This is the shortest end-to-end path for using UIAI Engine as an agent browser, research, diagnostics, and Focusa evidence backend. For repeatable operator/agent recipes, see [Agent UX Cookbook](AGENT_UX_COOKBOOK.md).

## 1. Start with health and discovery

Use discovery before loading full schemas.

### CLI

```bash
cd /home/wpuiai/uiai-engine
scripts/uiai status
scripts/uiai health
scripts/uiai tools search diagnostics
scripts/uiai tools graph --json
```

### HTTP

```bash
curl -s http://127.0.0.1:7456/health | jq .
curl -s http://127.0.0.1:7456/api/tools/agent-card | jq .
curl -s http://127.0.0.1:7456/api/tools/docs | jq .
curl -s 'http://127.0.0.1:7456/api/tools/search?q=diagnostics' | jq .
curl -s http://127.0.0.1:7456/api/tools/graph | jq .
```

### Pi

From the UIAI Engine repo root, Pi discovers the project extension and skills:

```text
pi_uiai_agent_card
pi_uiai_tool_search q="diagnostics"
pi_uiai_tool_graph
uiai_health
```

Useful project skills:

```text
/skill:uiai-agent
/skill:uiai-focusa-packet
/skill:uiai-release
/skill:uiai-mcp
/skill:uiai-remote-auth
/skill:uiai-docs-maintenance
/skill:uiai-ci-debug
/skill:uiai-browser-debug
/skill:vision
```

### MCP

Install/merge the bridge config, then reconnect the MCP client after tool changes:

```bash
DRY_RUN=1 scripts/install-agent-integrations.sh
scripts/install-agent-integrations.sh
scripts/install-pi-skills.sh --dry-run
scripts/install-pi-skills.sh --apply
scripts/smoke-agent-integrations.sh
scripts/smoke-mcp-tool-routes.sh
```

MCP clients can call `uiai_agent_card`, `uiai_tool_search`, `uiai_tool_graph`, `browser_search`, `browser_open`, `browser_read`, `browser_snapshot`, `browser_diagnostics`, and `uiai_focusa_packet_compose`.

Draft roadmap note: Source-to-Markdown is specified in [`SOURCE_TO_MARKDOWN_AGENT_SPEC.md`](SOURCE_TO_MARKDOWN_AGENT_SPEC.md) for future `source_to_markdown` / `uiai_to_markdown` / CLI `uiai md` discovery across HTTP, Pi, MCP, CLI, Focusa, and WPUIAI surfaces.

## 2. Search, open, read, snapshot, diagnose

This workflow gives agents useful text, reliable selectors, and failure evidence without depending on screenshots alone.

### CLI/HTTP flow

```bash
# Search
curl -s -X POST http://127.0.0.1:7456/api/search \
  -H 'Content-Type: application/json' \
  -d '{"query":"UIAI Engine browser agents","limit":1}' \
  | tee /tmp/uiai-search.json | jq '{provider,count,results:[.results[] | {rank,title,url,evidence_ref}]}'

# Open a selected URL with optional Focusa scope
SID=$(curl -s -X POST http://127.0.0.1:7456/api/session \
  -H 'Content-Type: application/json' \
  -d '{
    "url":"https://example.com",
    "width":1280,
    "height":800,
    "focusa_scope":{
      "project_root":"/home/wpuiai/uiai-engine",
      "continuity_id":"focusa-cont-uiai-engine-82afe24f-90ce-4d6e-b5f2-1b431b7773fc",
      "evidence_ref":"uiai-agent-quickstart"
    }
  }' | jq -r '.session.id')

# Read page text
curl -s -X POST "http://127.0.0.1:7456/api/session/$SID/read" \
  -H 'Content-Type: application/json' \
  -d '{"max_chars":2000,"include_links":true}' \
  | tee /tmp/uiai-read.json | jq '{session_id,url,focusa,text:(.text[0:500])}'

# Snapshot for @ref selectors
curl -s -X POST "http://127.0.0.1:7456/api/session/$SID/snapshot" \
  -H 'Content-Type: application/json' -d '{}' \
  | tee /tmp/uiai-snapshot.json | jq '{session_id,stats,focusa}'

# Diagnostics for console, exceptions, failed requests, API/CORS clues
curl -s "http://127.0.0.1:7456/api/session/$SID/diagnostics?limit=50" \
  | tee /tmp/uiai-diagnostics.json | jq '{session_id,summary,focusa}'
```

### Pi equivalent

```text
uiai_search query="UIAI Engine browser agents" limit=1
uiai_browser_open url="https://example.com" focusa_scope={...}
uiai_browser_read session_id="<sid>" max_chars=2000 include_links=true
uiai_browser_snapshot session_id="<sid>"
uiai_browser_diagnostics session_id="<sid>" limit=50
```

### MCP equivalent

Use the same MCP sequence with `browser_search`, `browser_open`, `browser_read`, `browser_snapshot`, and `browser_diagnostics`. CLI shortcut: `scripts/uiai-open-result.sh --query "UIAI Engine browser agents" --index 1 --out /tmp/uiai-open-result.json`.

## 3. Compose a Focusa research diagnostics packet

UIAI packets are bounded, redacted evidence proposals. They do **not** write Focusa memory by themselves. Focusa remains the cognitive authority.

Schema:

```text
uiai.focusa_research_diagnostics_packet.v1
```

### One-command CLI workflow

```bash
scripts/uiai research packet --url https://example.com \
  --goal "Quickstart proof packet" \
  --out /tmp/uiai-research-packet.json

jq '{ok,selected_url,session_closed,packet:{schema:.packet.schema,mode:.packet.mode,evidence_refs:(.packet.evidence_refs|length),preferred_tool:.packet.recommended_focusa.preferred_tool}}' \
  /tmp/uiai-research-packet.json
```

With provider search first:

```bash
scripts/uiai research packet --query "UIAI Engine browser agents" \
  --goal "Quickstart research packet" \
  --out /tmp/uiai-research-packet.json
```

The command searches when `--query` is supplied, opens the selected URL, reads bounded text, captures a snapshot and diagnostics, composes a packet, closes the session, and writes the report artifact.

### HTTP composer

```bash
jq -n \
  --slurpfile search /tmp/uiai-search.json \
  --slurpfile read /tmp/uiai-read.json \
  --slurpfile snapshot /tmp/uiai-snapshot.json \
  --slurpfile diagnostics /tmp/uiai-diagnostics.json \
  '{
    mode:"proof",
    goal:"Quickstart proof packet for UIAI agent workflow",
    responses:[ $search[0], $read[0], $snapshot[0], $diagnostics[0] ],
    focusa_scope:{
      project_root:"/home/wpuiai/uiai-engine",
      continuity_id:"focusa-cont-uiai-engine-82afe24f-90ce-4d6e-b5f2-1b431b7773fc",
      evidence_ref:"uiai-agent-quickstart"
    },
    recommended_next_action:"Capture packet evidence with Focusa, then close the UIAI browser session.",
    cleanup_session_id:env.SID
  }' \
  | scripts/uiai --json packet compose - \
  | tee /tmp/uiai-packet.json \
  | jq '{schema,mode,scope_status,evidence_refs,captures,recommended_focusa,cleanup}'
```

Direct HTTP:

```bash
curl -s -X POST http://127.0.0.1:7456/api/agent/research-packet \
  -H 'Content-Type: application/json' \
  --data @/tmp/uiai-packet-request.json | jq .
```

### Pi composer

```text
uiai_focusa_packet_compose goal="Quickstart proof packet" mode="proof" responses=[search,read,snapshot,diagnostics] focusa_scope={...} cleanup_session_id="<sid>"
```

Use `uiai_focusa_packet_build` only when you need local Pi composition without the HTTP composer.

Executable Pi command shortcuts:

```text
/uiai research <query>
/uiai proof <url>
/uiai diagnose <session_id>
```

These commands compose the packet and insert the recommended Focusa args preview into the editor; `research` and `proof` also close the browser session after packet composition.

### MCP composer

```text
uiai_focusa_packet_compose
```

Pass the same `goal`, `mode`, `responses`, `focusa_scope`, and optional `cleanup_session_id` fields.

## 4. Capture or intake the packet in Focusa

Use the packet's `recommended_focusa` block. For redacted research/diagnose/proof fixtures, see [Focusa Packet Examples Gallery](FOCUSA_PACKET_EXAMPLES_GALLERY.md).

Typical evidence capture:

```text
focusa_evidence_capture(
  target_ref=packet.recommended_focusa.args_preview.target_ref,
  result=packet.recommended_focusa.args_preview.result,
  evidence_ref=packet.recommended_focusa.args_preview.evidence_ref,
  project_root="/home/wpuiai/uiai-engine",
  continuity_id="focusa-cont-uiai-engine-82afe24f-90ce-4d6e-b5f2-1b431b7773fc"
)
```

When the packet is diagnostics/error-heavy, prefer:

```text
focusa_browser_diagnostics_intake(diagnostics=<diagnostics-json>, target_ref=<target>, project_root=<root>, continuity_id=<continuity>)
```

## 5. Cleanup

Close sessions when the packet says cleanup is recommended.

```bash
curl -s -X DELETE "http://127.0.0.1:7456/api/session/$SID"
```

Pi/MCP equivalent:

```text
uiai_browser_close session_id="<sid>"
browser_close session_id="<sid>"
```

## 6. Auth boundary

Local loopback calls are frictionless for agent tools. Remote callers must authenticate for browser/session, screenshot, search, media/frame, errors, and `/api/agent/research-packet` route families unless the route is public discovery.

Public discovery:

```text
/api/tools/*
/health
```

Loopback-public, remote-auth:

```text
/api/session/*
/api/screenshot/*
/api/search/*
/api/errors/*
/api/media/frame/*
/api/agent/research-packet
```

Remote examples use environment variables, never literal secrets:

```bash
export UIAI_ENGINE_URL="https://your-authenticated-tunnel.example"
export UIAI_API_KEY="..."          # sent as X-API-Key by Pi/CLI helpers
export UIAI_BEARER_TOKEN="..."     # alternative Authorization: Bearer token
# See docs/REMOTE_AUTH_EXAMPLES.md for curl, scripts/uiai, Pi, and MCP examples.
scripts/uiai health
```

## 7. Verification smokes

Run these before claiming an agent-surface change is ready:

```bash
scripts/check-focusa-packet-drift.sh
go test ./...
bun test ./.pi/extensions/uiai-engine.packet-builder.test.ts
scripts/smoke-pi-extension-registration.sh
scripts/smoke-agent-integrations.sh
scripts/smoke-mcp-tool-routes.sh
scripts/smoke-focusa-packet.sh
```

For release-quality browser reliability:

```bash
make browser-reliability
```

## 8. Troubleshooting

- If an action fails or a page is blank/broken, call diagnostics before patching.
- If MCP tools look stale, reconnect/restart the MCP client/bridge and reload Pi sessions using the MCP adapter; see [MCP Cache and Reconnect Troubleshooting](MCP_CACHE_RECONNECT_TROUBLESHOOTING.md).
- If CI fails, inspect both failed step logs and uploaded artifacts with `gh run view --log-failed` and `gh run download`.
- If a remote call returns `401`, confirm the route auth mode in `docs/ENDPOINT_AUTH_MATRIX.md` and set `UIAI_API_KEY` or `UIAI_BEARER_TOKEN`.

Related docs:

- [Session API](SESSION_API.md)
- [Browser Diagnostics Spec](BROWSER_DIAGNOSTICS_SPEC.md)
- [Endpoint Auth Matrix](ENDPOINT_AUTH_MATRIX.md)
- [UIAI × Focusa × Pi Hand-in-Glove Spec](UIAI_FOCUSA_PI_HAND_IN_GLOVE_SPEC.md)
- [Agent Surface Release Proof Checklist](AGENT_SURFACE_RELEASE_PROOF_CHECKLIST.md)
