#!/usr/bin/env python3
import json, re
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
DIR=ROOT/'docs/examples/focusa-packets'
SCHEMA='uiai.focusa_research_diagnostics_packet.v1'
expected={'research':'focusa_evidence_capture','diagnose':'focusa_browser_diagnostics_intake','proof':'focusa_evidence_capture'}
required_surfaces={'search','read','snapshot','diagnostics','error','screenshot','share'}
secret=re.compile(r'(?i)(authorization|bearer\s+|api[_-]?key|password|secret|token=|base64)')
missing=[]; surfaces=set(); sizes={}
for mode,tool in expected.items():
    path=DIR/f'{mode}-packet.example.json'
    if not path.exists(): missing.append(f'{path}: missing'); continue
    text=path.read_text(); data=json.loads(text); sizes[mode]=len(text.encode())
    if len(text.encode())>8192: missing.append(f'{path}: over 8KB')
    if data.get('schema')!=SCHEMA: missing.append(f'{path}: schema')
    if data.get('mode')!=mode: missing.append(f'{path}: mode')
    if data.get('recommended_focusa',{}).get('preferred_tool')!=tool: missing.append(f'{path}: preferred_tool')
    if not data.get('recommended_focusa',{}).get('args_preview'): missing.append(f'{path}: args_preview')
    if not data.get('cleanup'): missing.append(f'{path}: cleanup')
    for c in data.get('captures',[]): surfaces.add(c.get('type'))
    sanitized=text.replace('sensitive query values','redacted query values')
    if secret.search(sanitized): missing.append(f'{path}: possible secret/raw blob string')
miss=sorted(required_surfaces-surfaces)
if miss: missing.append('missing surfaces: '+', '.join(miss))
report={'ok':not missing,'sizes':sizes,'surfaces':sorted(surfaces),'missing':missing}
print(json.dumps(report, indent=2))
if missing: raise SystemExit('packet examples check failed')
