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
import json, os, time, urllib.error, urllib.request
engine=os.environ['ENGINE_URL'].rstrip('/')
out=os.environ['OUT']
timeout=int(os.environ.get('TIMEOUT_SECONDS','15'))
site=os.environ['TARGET_URL']
scope={
    'project_ref':'project:uiai-engine-ci',
    'project_root':'/home/wpuiai/uiai-engine',
    'workstream_ref':'workstream:browser-reliability',
    'workset_ref':'workset:browser-reliability',
    'callgraph_ref':'callgraph:browser-reliability',
    'workpoint_id':'workpoint:focusa-packet-smoke',
    'work_item_ref':'work-item:focusa-packet-smoke',
    'continuity_id':'continuity:focusa-packet-smoke',
    'evidence_ref':'uiai-focusa-packet-smoke',
    'work_items':[{
        'provider_surface':'github-actions',
        'work_item_ref':'work-item:focusa-packet-smoke',
        'item_id':'focusa-packet-smoke',
        'item_type':'ci_fixture',
        'title':'Focusa packet EPWA contract fixture',
        'description_state':'unavailable',
        'revision':'1',
        'digest':'e887d8dbd6c14b7cdb9b81e274c786623e007568211e973f50358c752f44ba1b',
        'revision_state':'current',
        'status_at_capture':'in_progress',
        'closure_posture':'open',
        'authority':{
            'acceptance_atom_refs':['acceptance:ready-epwa'],
            'evidence_requirement_refs':['evidence:focusa-packet-smoke'],
            'completion_contract_ref':'contract:focusa-packet-smoke',
            'settlement_posture':'unsettled',
        },
    }],
}

def req(method,path,body=None):
    data=None
    headers={'Content-Type':'application/json'}
    if body is not None:
        data=json.dumps(body).encode()
    r=urllib.request.Request(engine+path, data=data, method=method, headers=headers)
    try:
        with urllib.request.urlopen(r, timeout=timeout) as resp:
            return json.loads(resp.read().decode() or '{}')
    except urllib.error.HTTPError as exc:
        response_body=exc.read().decode(errors='replace')[:2000]
        raise RuntimeError(f'{method} {path} returned HTTP {exc.code}: {response_body}') from exc

def try_req(method,path,body=None):
    try:
        return req(method,path,body), None
    except Exception as e:
        return None, str(e)

def require_delivery(body, operation):
    delivery=body.get('epwa_delivery') or {}
    epwa=delivery.get('epwa') or {}
    if delivery.get('schema')!='uiai.epwa_delivery.v1' or delivery.get('state')!='ready' or body.get('delivery_state')!='ready' or (delivery.get('artifact') or {}).get('artifact_ref')!=body.get('artifact_ref'):
        raise AssertionError(f'{operation}: EPWA delivery not ready and identity-bound: {body}')
    if not str(epwa.get('record_url','')).startswith('https://') or not str(epwa.get('portable_url','')).startswith('https://') or body.get('artifact_url')!=epwa.get('record_url') or body.get('portable_url')!=epwa.get('portable_url'):
        raise AssertionError(f'{operation}: canonical HTTPS EPWA URLs missing: {body}')
    if any(key in body for key in ('screenshot','imageBase64','image_base64','artifact_path','result_path','result_url')):
        raise AssertionError(f'{operation}: raw artifact field returned')

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
    if search:
        require_delivery(search,'search')
        responses.append(search)

sid=None
try:
    opened=req('POST','/api/session',{'url':site,'width':900,'height':700,'focusa_scope':scope})
    require_delivery(opened,'session open')
    sid=opened['session']['id']
    read=req('POST',f'/api/session/{sid}/read',{'max_chars':2000,'include_links':True})
    diag=req('GET',f'/api/session/{sid}/diagnostics?limit=50')
    require_delivery(read,'source read')
    require_delivery(diag,'diagnostics')
    responses.extend([read,diag])
finally:
    if sid:
        try_req('DELETE',f'/api/session/{sid}')

packet=req('POST','/api/agent/research-packet',{
    'mode':'proof',
    'goal':'Harmless local UIAI Focusa packet smoke',
    'responses':responses,
    'focusa_scope':scope,
    'recommended_next_action':'Capture packet smoke evidence, then proceed to guided Pi workflows.',
    'cleanup_session_id':sid or '',
})
require_delivery(packet,'research packet')
transport_encoded=json.dumps(packet,separators=(',',':'))
if packet.get('artifact_schema') != 'uiai.focusa_research_diagnostics_packet.v1': raise AssertionError(packet)
payload_exclusions={'schema','artifact_schema','artifact_ref','artifact_payload_posture','delivery_state','epwa_delivery','raw_output_posture','artifact_url','portable_url'}
payload={key:value for key,value in packet.items() if key not in payload_exclusions}
payload['schema']=packet['artifact_schema']
payload_encoded=json.dumps(payload,separators=(',',':'))
if packet.get('recommended_focusa',{}).get('preferred_tool') != 'focusa_browser_diagnostics_intake': raise AssertionError(packet.get('recommended_focusa'))
if len(payload_encoded.encode())>8192: raise AssertionError(f"packet payload over 8KB: {len(payload_encoded.encode())}")
for required in ['read','diagnostics']:
    if not any(c.get('type')==required for c in packet.get('captures',[])): raise AssertionError(f"missing {required} capture: {packet.get('captures')}")
for bad in ['redacted-candidate','cookie','authorization','#frag']:
    if bad in transport_encoded.lower(): raise AssertionError(f"leaked {bad}")
report={'ok':True,'packet_bytes':len(payload_encoded.encode()),'delivery_bytes':len(transport_encoded.encode()),'search_ran':bool(search),'session_closed':bool(sid),'packet':packet}
with open(out,'w') as f: json.dump(report,f,indent=2)
print(f"focusa packet smoke ok: out={out} packet_bytes={report['packet_bytes']} evidence_refs={len(packet.get('evidence_refs',[]))} search_ran={report['search_ran']}")
PY
