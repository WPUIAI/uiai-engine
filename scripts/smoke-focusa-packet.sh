#!/usr/bin/env bash
set -euo pipefail
ENGINE_URL="${ENGINE_URL:-http://127.0.0.1:7456}"
OUT="${OUT:-/tmp/uiai-focusa-packet-smoke.json}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-60}"
SITE_DIR=""
SITE_PORT="${SITE_PORT:-8765}"
SERVER_PID=""
TARGET_URL="${SMOKE_TARGET_URL:-https://example.com/?ok=1&token=redacted-candidate#frag}"
cleanup(){
  if [[ -n "$SERVER_PID" ]]; then kill "$SERVER_PID" >/dev/null 2>&1 || true; fi
  if [[ -n "$SITE_DIR" ]]; then rm -rf "$SITE_DIR"; fi
}
trap cleanup EXIT
if [[ "${UIAI_ALLOW_PRIVATE_SMOKES:-0}" == "1" ]]; then
  SITE_DIR="$(mktemp -d)"
  cat > "$SITE_DIR/index.html" <<'HTML'
<!doctype html><html><head><meta charset="utf-8"><title>UIAI Focusa packet smoke</title><meta name="description" content="Harmless local UIAI Focusa packet smoke page"></head><body><main><h1>UIAI Focusa packet smoke</h1><p>Bounded read and diagnostics proof page.</p><a href="/next?token=redacted-candidate#frag">redaction candidate</a><script>console.warn("packet smoke warning");</script></main></body></html>
HTML
  python3 -m http.server "$SITE_PORT" --directory "$SITE_DIR" >/tmp/uiai-focusa-packet-site.log 2>&1 &
  SERVER_PID=$!
  TARGET_URL="http://127.0.0.1:${SITE_PORT}/index.html?ok=1&token=redacted-candidate#frag"
fi
export ENGINE_URL OUT TIMEOUT_SECONDS TARGET_URL
python3 - <<'PY'
import json, os, time, urllib.request
engine=os.environ['ENGINE_URL'].rstrip('/')
out=os.environ['OUT']
timeout=int(os.environ.get('TIMEOUT_SECONDS','15'))
site=os.environ['TARGET_URL']

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

for _ in range(40):
    try:
        urllib.request.urlopen(engine+'/health', timeout=2).read(); break
    except Exception: time.sleep(0.25)
else:
    raise SystemExit('engine health unavailable')

providers,_=try_req('GET','/api/search/providers')
responses=[]
search=None
if providers and (providers.get('providers') or [{}])[0].get('status') == 'ready':
    search,_=try_req('POST','/api/search',{'query':'UIAI Engine browser agents','limit':1})
    if search: responses.append(search)

sid=None
try:
    opened=req('POST','/api/session',{'url':site,'width':900,'height':700,'focusa_scope':{'project_root':'/home/wpuiai/uiai-engine','continuity_id':'focusa-cont-uiai-engine-82afe24f-90ce-4d6e-b5f2-1b431b7773fc','evidence_ref':'uiai-focusa-packet-smoke'}})
    sid=opened['session']['id']
    read=req('POST',f'/api/session/{sid}/read',{'max_chars':2000,'include_links':True})
    diag=req('GET',f'/api/session/{sid}/diagnostics?limit=50')
    responses.extend([read,diag])
finally:
    if sid:
        try_req('DELETE',f'/api/session/{sid}')

packet=req('POST','/api/agent/research-packet',{
    'mode':'proof',
    'goal':'Harmless local UIAI Focusa packet smoke',
    'responses':responses,
    'focusa_scope':{'project_root':'/home/wpuiai/uiai-engine','continuity_id':'focusa-cont-uiai-engine-82afe24f-90ce-4d6e-b5f2-1b431b7773fc','evidence_ref':'uiai-focusa-packet-smoke'},
    'recommended_next_action':'Capture packet smoke evidence, then proceed to guided Pi workflows.',
    'cleanup_session_id':sid or '',
})
encoded=json.dumps(packet,separators=(',',':'))
if packet.get('schema') != 'uiai.focusa_research_diagnostics_packet.v1': raise AssertionError(packet)
if packet.get('recommended_focusa',{}).get('preferred_tool') != 'focusa_browser_diagnostics_intake': raise AssertionError(packet.get('recommended_focusa'))
if len(encoded.encode())>8192: raise AssertionError(f"packet over 8KB: {len(encoded.encode())}")
for required in ['read','diagnostics']:
    if not any(c.get('type')==required for c in packet.get('captures',[])): raise AssertionError(f"missing {required} capture: {packet.get('captures')}")
for bad in ['redacted-candidate','cookie','authorization','#frag']:
    if bad in encoded.lower(): raise AssertionError(f"leaked {bad}")
report={'ok':True,'packet_bytes':len(encoded.encode()),'search_ran':bool(search),'session_closed':bool(sid),'packet':packet}
with open(out,'w') as f: json.dump(report,f,indent=2)
print(f"focusa packet smoke ok: out={out} packet_bytes={report['packet_bytes']} evidence_refs={len(packet.get('evidence_refs',[]))} search_ran={report['search_ran']}")
PY
