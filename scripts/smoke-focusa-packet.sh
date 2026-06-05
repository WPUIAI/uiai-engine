#!/usr/bin/env bash
set -euo pipefail
ENGINE_URL="${ENGINE_URL:-http://127.0.0.1:7456}"
OUT="${OUT:-/tmp/uiai-focusa-packet-smoke.json}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-15}"
SITE_DIR="$(mktemp -d)"
SITE_PORT="${SITE_PORT:-8765}"
SERVER_PID=""
cleanup(){
  if [[ -n "$SERVER_PID" ]]; then kill "$SERVER_PID" >/dev/null 2>&1 || true; fi
  rm -rf "$SITE_DIR"
}
trap cleanup EXIT
cat > "$SITE_DIR/index.html" <<'HTML'
<!doctype html><html><head><meta charset="utf-8"><title>UIAI Focusa packet smoke</title><meta name="description" content="Harmless local UIAI Focusa packet smoke page"></head><body><main><h1>UIAI Focusa packet smoke</h1><p>Bounded read and diagnostics proof page.</p><a href="/next?token=redacted-candidate#frag">redaction candidate</a><script>console.warn("packet smoke warning");</script></main></body></html>
HTML
python3 -m http.server "$SITE_PORT" --directory "$SITE_DIR" >/tmp/uiai-focusa-packet-site.log 2>&1 &
SERVER_PID=$!
export ENGINE_URL OUT TIMEOUT_SECONDS SITE_PORT
python3 - <<'PY'
import json, os, time, urllib.request
engine=os.environ['ENGINE_URL'].rstrip('/')
out=os.environ['OUT']
timeout=int(os.environ.get('TIMEOUT_SECONDS','15'))
site=f"http://127.0.0.1:{os.environ['SITE_PORT']}/index.html?ok=1&token=redacted-candidate#frag"

def req(method,path,body=None):
    data=None
    headers={'Content-Type':'application/json'}
    if body is not None:
        data=json.dumps(body).encode()
    r=urllib.request.Request(engine+path, data=data, method=method, headers=headers)
    with urllib.request.urlopen(r, timeout=timeout) as resp:
        return json.loads(resp.read().decode() or '{}')

def try_req(method,path,body=None):
    try:
        return req(method,path,body), None
    except Exception as e:
        return None, str(e)

def trunc(s,n):
    s=str(s or '').strip()
    return s if len(s)<=n else s[:n-1]+'…'

def capture_from_response(resp, typ):
    f=(resp or {}).get('focusa') or {}
    if not f: return None
    return {'type':typ,'evidence_ref':trunc(f.get('evidence_ref',''),240),'target_ref':trunc(f.get('target_ref',''),500),'title':trunc(resp.get('title') or ((resp.get('results') or [{}])[0].get('title') if resp.get('results') else ''),200),'summary':trunc(f.get('summary',''),500)}

for _ in range(40):
    try:
        urllib.request.urlopen(engine+'/health', timeout=2).read(); break
    except Exception: time.sleep(0.25)
else:
    raise SystemExit('engine health unavailable')

providers,_=try_req('GET','/api/search/providers')
search=None
if providers and (providers.get('providers') or [{}])[0].get('status') == 'ready':
    search,_=try_req('POST','/api/search',{'query':'UIAI Engine browser agents','limit':1})

sid=None
try:
    opened=req('POST','/api/session',{'url':site,'width':900,'height':700,'focusa_scope':{'project_root':'/home/wpuiai/uiai-engine','continuity_id':'focusa-cont-uiai-engine-82afe24f-90ce-4d6e-b5f2-1b431b7773fc','evidence_ref':'uiai-focusa-packet-smoke'}})
    sid=opened['session']['id']
    read=req('POST',f'/api/session/{sid}/read',{'max_chars':2000,'include_links':True})
    diag=req('GET',f'/api/session/{sid}/diagnostics?limit=50')
finally:
    if sid:
        try_req('DELETE',f'/api/session/{sid}')

captures=[]
if search:
    c=capture_from_response(search,'search')
    if c: captures.append(c)
for resp,typ in [(read,'read'),(diag,'diagnostics')]:
    c=capture_from_response(resp,typ)
    if c: captures.append(c)
refs=[c['evidence_ref'] for c in captures if c.get('evidence_ref')]
targets=[c['target_ref'] for c in captures if c.get('target_ref')]
preferred='focusa_browser_diagnostics_intake' if any(c['type']=='diagnostics' for c in captures) else 'focusa_evidence_capture'
packet={'schema':'uiai.focusa_research_diagnostics_packet.v1','mode':'proof','goal':'Harmless local UIAI Focusa packet smoke','scope_status':'present','focusa_scope':{'project_root':'/home/wpuiai/uiai-engine','continuity_id':'focusa-cont-uiai-engine-82afe24f-90ce-4d6e-b5f2-1b431b7773fc','evidence_ref':'uiai-focusa-packet-smoke'},'target_refs':targets[:16],'evidence_refs':refs[:32],'captures':captures[:8],'diagnostics_summary':{'console_errors':diag.get('summary',{}).get('console_errors',0),'failed_requests':diag.get('summary',{}).get('failed_requests',0),'top_findings':[]},'active_object_hints':[{'kind':'url','hint':t} for t in targets[:16]],'recommended_focusa':{'preferred_tool':preferred,'fallback_tool':'focusa_evidence_capture','args_preview':{'target_ref':targets[0] if targets else 'uiai:packet','result':captures[0]['summary'] if captures else 'UIAI packet smoke','evidence_ref':refs[0] if refs else 'uiai-packet:smoke','attach_to_workpoint':False},'next_tools':[preferred,'focusa_evidence_capture','focusa_active_object_resolve','focusa_predict_record']},'recommended_next_action':'Capture packet smoke evidence, then proceed to guided Pi workflows.','render':{'summary_line':f"UIAI packet evidence={len(refs)} target={(targets[0] if targets else 'none')} scope=present tool={preferred} next=Capture packet smoke evidence"},'headless_next_action':'Capture packet smoke evidence, then proceed to guided Pi workflows.','cleanup':{'session_id':sid or '', 'recommended_action':'close_when_done','tool':'uiai_browser_close'}}
encoded=json.dumps(packet,separators=(',',':'))
if len(encoded.encode())>8192: raise AssertionError(f"packet over 8KB: {len(encoded.encode())}")
for required in ['read','diagnostics']:
    if not any(c['type']==required for c in captures): raise AssertionError(f"missing {required} capture: {captures}")
for bad in ['redacted-candidate','cookie','authorization','#frag']:
    if bad in encoded.lower(): raise AssertionError(f"leaked {bad}")
report={'ok':True,'packet_bytes':len(encoded.encode()),'search_ran':bool(search),'session_closed':bool(sid),'packet':packet}
with open(out,'w') as f: json.dump(report,f,indent=2)
print(f"focusa packet smoke ok: out={out} packet_bytes={report['packet_bytes']} evidence_refs={len(refs)} search_ran={report['search_ran']}")
PY
