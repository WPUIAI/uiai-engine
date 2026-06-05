# Focusa Packet Examples Gallery

These examples show redacted `uiai.focusa_research_diagnostics_packet.v1` packets for research, diagnose, and proof workflows. They are illustrative fixtures, not live secrets or raw browser artifacts.

## Files

| Mode | Fixture | Surfaces covered | Expected Focusa tool |
|---|---|---|---|
| research | [`examples/focusa-packets/research-packet.example.json`](examples/focusa-packets/research-packet.example.json) | search, source_markdown, read, snapshot | `focusa_evidence_capture` |
| diagnose | [`examples/focusa-packets/diagnose-packet.example.json`](examples/focusa-packets/diagnose-packet.example.json) | diagnostics, error | `focusa_browser_diagnostics_intake` |
| proof | [`examples/focusa-packets/proof-packet.example.json`](examples/focusa-packets/proof-packet.example.json) | screenshot, share, read | `focusa_evidence_capture` |

Together the fixtures cover search/read/snapshot/diagnostics/error/screenshot/share packet surfaces.

## How to read a packet

Key fields: `schema`, `mode`, `captures[]`, `recommended_focusa.preferred_tool`, `recommended_focusa.args_preview`, and `cleanup`.

## Validation

```bash
scripts/check-focusa-packet-examples.py
```

The validator checks schema, modes, required fields, packet budget, preferred tools, redaction, and combined surface coverage.

## Live generation commands

```bash
scripts/uiai research packet --url https://example.com --goal "Proof packet" --out /tmp/uiai-research-packet.json
scripts/smoke-focusa-packet-ci.sh
```

## Focusa handoff examples

Research/proof packets usually route to `focusa_evidence_capture` using `recommended_focusa.args_preview`; Source-to-Markdown captures use `uiai-source-markdown:sha256:<prefix>` evidence refs.

Diagnostics packets usually route to `focusa_browser_diagnostics_intake` using `recommended_focusa.args_preview.diagnostics`.

## Redaction rules

- No raw image payloads or transcript blobs.
- No cookies, Authorization headers, API keys, bearer tokens, webhook secrets, or sensitive query values.
- URL fragments are omitted.
- Evidence refs and bounded summaries replace raw logs/screenshots/page dumps.
