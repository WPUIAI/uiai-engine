---
name: uiai-focusa-packet
description: UIAI Focusa research diagnostics packet playbook: compose bounded proof packets, route args_preview to Focusa tools, validate redaction, cleanup sessions, and run packet drift/smoke gates.
---
# UIAI Focusa Packet Skill

Use this skill when creating, validating, documenting, or debugging `uiai.focusa_research_diagnostics_packet.v1` packets from UIAI search/read/snapshot/diagnostics/error/screenshot/share responses.

## Core rule

UIAI packet output is a bounded proposal/evidence bundle. It does not write durable Focusa memory. Durable Workpoint/evidence/prediction/metacog writes happen through Focusa tools after scope is canonical.

## Canonical surfaces

- Executable Pi shortcuts: `/uiai research <query>`, `/uiai proof <url>`, `/uiai diagnose <session_id>`.
- Pi local builder: `uiai_focusa_packet_build`.
- Pi/HTTP composer: `uiai_focusa_packet_compose` → `POST /api/agent/research-packet`.
- MCP composer: `uiai_focusa_packet_compose`.
- CLI composer: `scripts/uiai packet compose <json-file|->`.
- Smoke: `scripts/smoke-focusa-packet.sh`, `scripts/uiai smoke packet`.
- Schema source: `internal/focusapacket/packet.go`.
- HTTP route: `internal/routes/agent_packet.go`.
- Drift check: `scripts/check-focusa-packet-drift.sh`.

## Packet modes

- `proof`: evidence for a completed check or release report; usually routes to `focusa_evidence_capture`.
- `diagnose`: browser/session or page failure evidence; usually routes to `focusa_browser_diagnostics_intake`.
- `research`: search/read/snapshot summary for a target; usually routes to `focusa_evidence_capture`.

## Recommended workflow

1. Verify scope when Focusa handoff matters:

```text
focusa_workpoint_resume
focusa_project_identity project_root="/home/wpuiai/uiai-engine"
```

2. Collect bounded UIAI responses:

```text
uiai_search query="<query>" limit=3
uiai_browser_open url="<selected url>" focusa_scope={project_root,continuity_id,evidence_ref}
uiai_browser_read session_id="<sid>" max_chars=2000 include_links=true
uiai_browser_snapshot session_id="<sid>" interactive=true
uiai_browser_diagnostics session_id="<sid>" limit=100
```

3. Compose packet:

```text
uiai_focusa_packet_compose goal="<goal>" mode="proof" responses=[search,read,snapshot,diagnostics] focusa_scope={project_root,continuity_id,evidence_ref} cleanup_session_id="<sid>"
```

4. Route `recommended_focusa.args_preview`:

```text
focusa_evidence_capture(
  target_ref=packet.recommended_focusa.args_preview.target_ref,
  result=packet.recommended_focusa.args_preview.result,
  evidence_ref=packet.recommended_focusa.args_preview.evidence_ref,
  project_root="/home/wpuiai/uiai-engine",
  continuity_id="focusa-cont-uiai-engine-82afe24f-90ce-4d6e-b5f2-1b431b7773fc"
)
```

For diagnostics/failure packets:

```text
focusa_browser_diagnostics_intake(
  diagnostics=packet.recommended_focusa.args_preview.diagnostics,
  target_ref=packet.recommended_focusa.args_preview.target_ref,
  result=packet.recommended_focusa.args_preview.result,
  project_root="/home/wpuiai/uiai-engine",
  continuity_id="focusa-cont-uiai-engine-82afe24f-90ce-4d6e-b5f2-1b431b7773fc"
)
```

5. Cleanup:

- If packet `cleanup` recommends closing a browser session, close it.
- Do not clear diagnostics until evidence has been captured or the repro is intentionally reset.

## CLI/HTTP examples

One-command workflow:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai research packet --url https://example.com --goal "Proof packet" --out /tmp/uiai-research-packet.json'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai research packet --query "UIAI Engine browser agents" --goal "Research packet" --out /tmp/uiai-research-packet.json'
```

Compose from a JSON request:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai --json packet compose /tmp/uiai-packet-request.json | tee /tmp/uiai-packet.json'
```

Compose over HTTP:

```bash
curl -s -X POST "$UIAI_ENGINE_URL/api/agent/research-packet" \
  -H 'Content-Type: application/json' \
  --data @/tmp/uiai-packet-request.json | jq .
```

Run the local smoke:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/smoke-focusa-packet.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/uiai smoke packet'
```

Expected smoke proof:

- Output includes `focusa packet smoke ok`.
- Artifact exists at `/tmp/uiai-focusa-packet-smoke.json`.
- Packet schema is `uiai.focusa_research_diagnostics_packet.v1`.
- `recommended_focusa.args_preview` is present.
- Search/read/snapshot/diagnostics captures are represented.

## Redaction and budget checks

Packet content must remain bounded and secret-safe:

- No raw base64 blobs.
- No cookies, Authorization headers, API keys, bearer tokens, webhook secrets, fragments, password/token query values, or request bodies containing secrets.
- Capture summaries are bounded; page text should be read with `max_chars<=2000` before packet composition.
- Default packet JSON should stay below the documented budget; oversize inputs are truncated.
- `args_preview` is a Focusa call preview, not proof of a durable Focusa write.

Use these gates:

```bash
as-user wpuiai 'cd /home/wpuiai/uiai-engine && scripts/check-focusa-packet-drift.sh'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && bun test ./.pi/extensions/uiai-engine.packet-builder.test.ts'
as-user wpuiai 'cd /home/wpuiai/uiai-engine && go test ./internal/focusapacket ./internal/routes'
```

## Troubleshooting

| Symptom | Likely cause | Next action |
|---|---|---|
| Missing `args_preview` | packet budget overflow or insufficient input fields | Narrow captures, provide `target_ref`, `result`, `evidence_ref`, rerun composer. |
| Diagnostics packet routes to evidence instead of diagnostics intake | mode/capture type does not include diagnostics/failure context | Use `mode=diagnose` and include `browser_diagnostics` or structured error response. |
| Packet drift check fails | docs/tool/schema surface changed without updating paired surfaces | Update source/docs/smoke together; run drift check again. |
| Focusa scope mismatch | project_root/continuity_id not canonical | Stop before durable Focusa write; run `focusa_project_verify` and `focusa_workpoint_resume`. |
| Raw blob in packet | screenshot/share response not normalized | Use packet builder/composer, not manual transcript paste; cite artifact/evidence refs only. |

## Docs to update when packet behavior changes

- `docs/UIAI_FOCUSA_PI_HAND_IN_GLOVE_SPEC.md`
- `docs/SESSION_API.md`
- `docs/UIAI_FOR_AGENTS_QUICKSTART.md`
- `docs/FOCUSA_PACKET_EXAMPLES_GALLERY.md`
- `docs/AGENT_UX_COOKBOOK.md`
- `docs/PUBLIC_API_PARITY_MATRIX.md`
- `README.md`
- `.pi/skills/uiai-focusa-packet/SKILL.md`
- `scripts/check-focusa-packet-drift.sh` only when canonical drift expectations change

## Final packet report

Include:

- Mode and goal.
- Packet schema.
- Target refs and evidence refs.
- Recommended Focusa tool and bounded `args_preview` summary.
- Cleanup action taken.
- Proof commands run.
