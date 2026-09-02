#!/usr/bin/env python3
import json, sys
from pathlib import Path
root=Path(__file__).resolve().parents[1]
old=json.loads((root/'docs/106q-uiai-evidence-pwa-completion-callgraph.json').read_text())
new=json.loads((root/'docs/106u-uiai-evidence-pwa-focusa-callgraph-v1.json').read_text())
errors=[]
if new.get('schema')!='focusa.callgraph.v1': errors.append('wrong schema')
old_ids=[n['id'] for n in old['nodes']]
new_ids=[f['frame_id'] for f in new['frames']]
if old_ids!=new_ids: errors.append('frame order/identity drift')
old_edges={(d,n['id']) for n in old['nodes'] for d in n['depends']}
new_edges={(e['from_frame_id'],e['to_frame_id']) for e in new['edges']}
if old_edges!=new_edges: errors.append('dependency edge drift')
if len(new_ids)!=32 or len(set(new_ids))!=32: errors.append('expected 32 unique frames')
if len(new_edges)!=55: errors.append('expected 55 unique edges')
if new.get('entry_frame_ids')!=['CG-01']: errors.append('CG-01 must be sole entry')
for f in new['frames']:
    if not f.get('acceptance',{}).get('acceptance_atoms'): errors.append(f'{f["frame_id"]}: missing acceptance')
    if f['frame_id'].startswith('CG-') and f['frame_id']!='CG-01':
        expected={d for d,n in old_edges if n==f['frame_id']}
        actual={d for d,n in new_edges if n==f['frame_id']}
        if expected!=actual: errors.append(f'{f["frame_id"]}: predecessor drift')
if errors:
    print('\n'.join(errors),file=sys.stderr); sys.exit(1)
print('106u projection audit: PASS (32 frames, 55 preserved dependencies, CG-01 entry)')
