#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
python3 - <<'PY'
from pathlib import Path
schema='uiai.focusa_research_diagnostics_packet.v1'
checks={
 '.pi/extensions/uiai-engine.ts':[schema,'uiai_focusa_packet_build','buildResearchDiagnosticsPacket','args_preview','headless_next_action'],
 'internal/focusapacket/packet.go':[schema,'DefaultMaxPacketBytes','MaxCaptureSummaryChars','SanitizeURL'],
 'internal/routes/screenshot.go':['screenshotFocusaMetadata','uiai-screenshot:sha256:','focusa_evidence_capture'],
 'internal/routes/share.go':['shareEvidence','uiai-share:','focusa_evidence_capture'],
 'internal/routes/search.go':['searchFocusaMetadata','FocusaScopeStatus','focusa_evidence_capture'],
 'internal/vision/session.go':['ReadFocusaMetadata','uiai-browser:session=','focusa_evidence_capture'],
 'internal/vision/snapshot.go':['SnapshotFocusaMetadata','uiai-browser:session=','snapshot:','focusa_evidence_capture'],
 'internal/vision/diagnostics.go':['DiagnosticsFocusaMetadata','uiai-diagnostics:session=','focusa_browser_diagnostics_intake'],
 'internal/observability/errors.go':['ErrorFocusaMetadata','uiai-error:','focusa_browser_diagnostics_intake'],
 'internal/routes/tools.go':['focusa_research_packet','browser_snapshot','screenshot','share','packet_metadata_surfaces','packet_args_preview'],
 'docs/UIAI_FOCUSA_PI_HAND_IN_GLOVE_SPEC.md':[schema,'Implementation bead decomposition','Focusa packet intake friction gate'],
 'docs/SESSION_API.md':['Packet parity status','agent_pressure',schema],
 'README.md':['uiai_focusa_packet_build',schema],
 'scripts/smoke-focusa-packet.sh':[schema,'packet_bytes','focusa_browser_diagnostics_intake'],
}
missing=[]
for rel, needles in checks.items():
    path=Path(rel)
    if not path.exists():
        missing.append(f'{rel}: file missing')
        continue
    text=path.read_text()
    for needle in needles:
        if needle not in text:
            missing.append(f'{rel}: missing {needle}')
if missing:
    raise SystemExit('Focusa packet drift check failed:\n'+'\n'.join(missing))
print(f'focusa packet drift check ok: files={len(checks)} schema={schema}')
PY
