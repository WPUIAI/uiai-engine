#!/usr/bin/env python3
from __future__ import annotations
import json, re, sys
from pathlib import Path
SECRET=re.compile(r'(?i)(authorization|bearer\s+[A-Za-z0-9._-]+|api[_-]?key|password|secret|token=|cookie)')
def clean(v, limit=220):
    s=json.dumps(v, ensure_ascii=False) if not isinstance(v,str) else v
    s=SECRET.sub('REDACTED', s).replace('\n',' ')
    return s[:limit]+('…' if len(s)>limit else '')
def get(d,*path,default=None):
    cur=d
    for p in path:
        if not isinstance(cur,dict) or p not in cur: return default
        cur=cur[p]
    return cur

def render_packet(d):
    pkt=d.get('packet') if isinstance(d.get('packet'),dict) else d
    print(f"Packet: schema={pkt.get('schema','?')} mode={pkt.get('mode','?')} bytes={d.get('packet_bytes','?')}")
    print(f"Goal: {clean(pkt.get('goal',''))}")
    print(f"Evidence refs: {', '.join(map(str,pkt.get('evidence_refs',[])[:8])) or '(none)'}")
    caps=pkt.get('captures') or []
    print(f"Captures: {len(caps)}")
    for c in caps[:6]: print(f"- {c.get('type','?')} {c.get('evidence_ref','')} {clean(c.get('summary',''))}")
    rf=pkt.get('recommended_focusa') or {}
    print(f"Focusa preferred: {rf.get('preferred_tool','?')}")
    if rf.get('args_preview'): print(f"Args preview: {clean(rf.get('args_preview'), 500)}")
    print(f"Next: {clean(pkt.get('recommended_next_action') or pkt.get('headless_next_action') or '')}")
    cleanup=pkt.get('cleanup') or {}
    if cleanup: print(f"Cleanup: {clean(cleanup)}")

def render_diag(d):
    summary=d.get('summary') or d.get('diagnostics_summary') or get(d,'diagnostics','summary',default={}) or {}
    print('Diagnostics artifact')
    if summary: print('Summary: '+clean(summary, 500))
    focusa=d.get('focusa_evidence') or d.get('focusa') or get(d,'diagnostics','focusa',default={}) or {}
    if focusa: print('Focusa: '+clean(focusa, 500))
    for key,label in [('console','Console'),('console_errors','Console'),('exceptions','Exceptions'),('failed_requests','Failed requests'),('network','Network')]:
        vals=d.get(key) or get(d,'diagnostics',key,default=None) or []
        if isinstance(vals,dict): vals=list(vals.values())
        if vals:
            print(f"{label}: {len(vals)}")
            for item in vals[:5]: print(f"- {clean(item)}")
    events=d.get('events') or []
    if events:
        print(f"Errors/events: {len(events)}")
        for e in events[:5]: print(f"- {e.get('id','?')} class={e.get('class') or e.get('error_class')} status={e.get('status')} next={clean(e.get('suggested_next_action') or e.get('diagnostics') or '')}")

def main():
    if len(sys.argv)!=2 or sys.argv[1] in ('-h','--help'):
        print('Usage: scripts/render-diagnostics-artifact.py <diagnostics-or-packet-json>')
        raise SystemExit(0 if len(sys.argv)==2 else 2)
    p=Path(sys.argv[1]); d=json.loads(p.read_text())
    print(f"Artifact: {p}")
    if isinstance(d,dict) and (d.get('packet') or d.get('schema')=='uiai.focusa_research_diagnostics_packet.v1'):
        render_packet(d)
    else:
        render_diag(d)
if __name__=='__main__': main()
