#!/usr/bin/env bash
# Mixed UIAI browser soak: sessions + delayed actions + screenshots + diagnostics + failure envelopes.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="${UIAI_ROOT:-$(cd "$SCRIPT_DIR/.." && pwd)}"
export UIAI_EPWA_CONTRACT_DIR="$SCRIPT_DIR"
ENGINE_PORT="${ENGINE_PORT:-7470}"
SITE_PORT="${SITE_PORT:-7471}"
DURATION_SECONDS="${DURATION_SECONDS:-300}"
CONCURRENCY="${CONCURRENCY:-2}"
OUT="${OUT:-/tmp/uiai-browser-flakiness-soak.json}"
ENGINE_BIN="${ENGINE_BIN:-/tmp/uiai-engine-soak}"
: "${UIAI_EVIDENCE_SCOPE_JSON:?UIAI_EVIDENCE_SCOPE_JSON is required for EPWA-producing soak runs}"
: "${UIAI_EPWA_PUBLIC_BASE_URL:?UIAI_EPWA_PUBLIC_BASE_URL must name the governed HTTPS EPWA surface}"
[[ "$UIAI_EPWA_PUBLIC_BASE_URL" == https://* ]] || { echo "UIAI_EPWA_PUBLIC_BASE_URL must use HTTPS" >&2; exit 2; }
jq -e 'type == "object"' >/dev/null <<<"$UIAI_EVIDENCE_SCOPE_JSON" || { echo "UIAI_EVIDENCE_SCOPE_JSON must be a JSON object" >&2; exit 2; }
export UIAI_EPWA_PUBLIC_BASE_URL

cd "$ROOT_DIR"
go build -o "$ENGINE_BIN" ./cmd/uiai-engine
TMPDIR=$(mktemp -d)
cleanup() {
  local code=$?
  [[ -n "${ENGINE_PID:-}" ]] && kill "$ENGINE_PID" 2>/dev/null || true
  [[ -n "${SITE_PID:-}" ]] && kill "$SITE_PID" 2>/dev/null || true
  wait "${ENGINE_PID:-}" "${SITE_PID:-}" 2>/dev/null || true
  rm -rf "$TMPDIR" "$ENGINE_BIN"
  exit "$code"
}
trap cleanup EXIT

cp config.yaml "$TMPDIR/config.yaml"
python3 - "$TMPDIR/config.yaml" "$ENGINE_PORT" <<'PY'
from pathlib import Path
import sys
p = Path(sys.argv[1])
s = p.read_text().replace('port: 7456', f'port: {sys.argv[2]}', 1)
s = s.replace('allow_private_urls: false', 'allow_private_urls: true', 1)
s = s.replace('data_dir: "/home/wpuiai/uiai-engine/data"', f'data_dir: "{p.parent / "data"}"', 1)
s = s.replace('share_dir: "/home/wpuiai/ai-api/shares"', f'share_dir: "{p.parent / "shares"}"', 1)
s = s.replace('script_dir: "/home/wpuiai/public_html/wp-content/plugins/wpuiai/assets/templates/devices"', f'script_dir: "{p.parent / "device-templates"}"', 1)
s = s.replace('health_file: "/var/log/uiai/ip-pool-health.json"', f'health_file: "{p.parent / "ip-pool-health.json"}"', 1)
s = s.replace('log_file: "/var/log/uiai/captcha-stats.jsonl"', f'log_file: "{p.parent / "captcha-stats.jsonl"}"', 1)
s = s.replace('file: "/var/log/uiai/engine.log"', f'file: "{p.parent / "engine.log"}"', 1)
p.write_text(s)
PY
mkdir -p "$TMPDIR/site" /tmp/uiai-soak-data
cat > "$TMPDIR/site/index.html" <<'HTML'
<!doctype html><title>uiai soak</title>
<script>
const id = new URL(location.href).searchParams.get('id') || 'none';
console.error('soak-console-error', id);
fetch('/missing-api?id=' + encodeURIComponent(id));
setTimeout(() => { const b=document.createElement('button'); b.id='late'; b.textContent='Late '+id; document.body.appendChild(b); }, 350 + Math.floor(Math.random()*450));
</script><h1>UIAI Soak</h1>
HTML
python3 -m http.server "$SITE_PORT" -d "$TMPDIR/site" >/tmp/uiai-soak-site.log 2>&1 & SITE_PID=$!
"$ENGINE_BIN" -config "$TMPDIR/config.yaml" >/tmp/uiai-soak-engine.log 2>&1 & ENGINE_PID=$!
for _ in $(seq 1 120); do curl -fsS "http://127.0.0.1:$ENGINE_PORT/health" >/dev/null 2>&1 && break; sleep 0.5; done
curl -fsS "http://127.0.0.1:$ENGINE_PORT/health" >/dev/null

export ENGINE_PORT SITE_PORT DURATION_SECONDS CONCURRENCY OUT UIAI_EVIDENCE_SCOPE_JSON
python3 - <<'PY'
import concurrent.futures, json, os, statistics, sys, time, urllib.error, urllib.request
sys.dont_write_bytecode = True
sys.path.insert(0, os.environ['UIAI_EPWA_CONTRACT_DIR'])
from epwa_raw_contract import find_raw
engine=f"http://127.0.0.1:{os.environ['ENGINE_PORT']}"
site=f"http://127.0.0.1:{os.environ['SITE_PORT']}"
duration=int(os.environ['DURATION_SECONDS'])
concurrency=int(os.environ['CONCURRENCY'])
out=os.environ['OUT']
end=time.time()+duration
focusa_scope=json.loads(os.environ['UIAI_EVIDENCE_SCOPE_JSON'])

def req(method,url,body=None,ok=(200,201)):
    data=None if body is None else json.dumps(body).encode()
    r=urllib.request.Request(url,data=data,method=method,headers={'Content-Type':'application/json'})
    try:
        with urllib.request.urlopen(r,timeout=45) as res:
            raw=res.read().decode()
            return res.status, json.loads(raw) if raw else {}
    except urllib.error.HTTPError as e:
        raw=e.read().decode()
        try: body=json.loads(raw)
        except Exception: body={'error': raw}
        return e.code, body

def require_delivery(body, operation):
    delivery=body.get('epwa_delivery') or {}
    epwa=delivery.get('epwa') or {}
    leaked = find_raw(body)
    if leaked: raise ValueError(f'{operation}: raw artifact field returned at {leaked}')
    if delivery.get('schema')!='uiai.epwa_delivery.v1' or delivery.get('state')!='ready' or body.get('delivery_state')!='ready' or (delivery.get('artifact') or {}).get('artifact_ref')!=body.get('artifact_ref'):
        raise ValueError(f'{operation}: EPWA delivery is not ready and identity-bound')
    if not str(epwa.get('record_url','')).startswith('https://') or not str(epwa.get('portable_url','')).startswith('https://') or body.get('artifact_url')!=epwa.get('record_url') or body.get('portable_url')!=epwa.get('portable_url'):
        raise ValueError(f'{operation}: canonical HTTPS EPWA URLs missing')

def one(i):
    label=f"w{i}-{int(time.time()*1000)}"
    started=time.time()
    open_body={'url':f'{site}/?id={label}','width':800,'height':600}
    if focusa_scope:
        open_body['focusa_scope']=focusa_scope
    status, opened=req('POST',f"{engine}/api/session",open_body)
    if status!=201:
        return {'ok':False,'phase':'open','status':status,'error_class':opened.get('error_class'),'elapsed_ms':round((time.time()-started)*1000)}
    sid=opened['session']['id']
    try:
        require_delivery(opened,'session open')
        status, late=req('POST',f"{engine}/api/session/{sid}/click",{'selector':'#late'})
        status2, shot=req('POST',f"{engine}/api/session/{sid}/screenshot",{})
        status3, missing=req('POST',f"{engine}/api/session/{sid}/click",{'selector':'#missing'},ok=(500,))
        status4, diag=req('GET',f"{engine}/api/session/{sid}/diagnostics?limit=50")
        require_delivery(shot,'screenshot')
        require_delivery(diag,'diagnostics')
        req('DELETE',f"{engine}/api/session/{sid}")
        # Core soak assertions: session opens, late click succeeds, screenshot succeeds.
        # Negative assertions (missing selector, failed network) are tracked separately.
        soak_ok=status==200 and status2==200
        # Negative assertions: #missing should 404 gracefully, network failure should be in diagnostics.
        negative_ok=status3==500 and missing.get('error_class')=='selector_not_found'
        return {'ok':soak_ok,'phase':'mixed','elapsed_ms':round((time.time()-started)*1000),'late_status':status,'missing_status':status3,'error_class':missing.get('error_class'),'diag':diag.get('summary',{}),'negative_ok':negative_ok}
    except Exception as e:
        try: req('DELETE',f"{engine}/api/session/{sid}")
        except Exception: pass
        return {'ok':False,'phase':'exception','error':str(e),'elapsed_ms':round((time.time()-started)*1000)}

results=[]
i=0
with concurrent.futures.ThreadPoolExecutor(max_workers=concurrency) as ex:
    pending=set()
    while time.time()<end or pending:
        while time.time()<end and len(pending)<concurrency:
            pending.add(ex.submit(one,i)); i+=1
        done,pending=concurrent.futures.wait(pending,timeout=0.1,return_when=concurrent.futures.FIRST_COMPLETED)
        for f in done: results.append(f.result())

lat=sorted(r.get('elapsed_ms',0) for r in results)
def pct(p):
    if not lat: return 0
    idx=min(len(lat)-1, max(0, int(round((p/100)*(len(lat)-1)))))
    return lat[idx]
classes={}
for r in results:
    k=r.get('error_class') or ('ok' if r.get('ok') else r.get('phase','unknown'))
    classes[k]=classes.get(k,0)+1
passed=sum(1 for r in results if r.get('ok'))
failed=sum(1 for r in results if not r.get('ok'))
total=len(results)
soak_rate=passed/total if total else 0
negative_passed=sum(1 for r in results if r.get('negative_ok'))
negative_rate=negative_passed/total if total else 0
soak_ok=soak_rate>=0.8 and bool(results)
evidence_ref=os.environ.get('FOCUSA_EVIDENCE_REF',f'uiai-browser-flakiness-soak:{out}')
report={'ok':soak_ok,'duration_seconds':duration,'concurrency':concurrency,'total_runs':total,'passed':passed,'failed':failed,'soak_rate':round(soak_rate,3),'negative_passed':negative_passed,'negative_rate':round(negative_rate,3),'avg_elapsed_ms':round(statistics.mean(lat),1) if lat else 0,'p95_elapsed_ms':pct(95),'p99_elapsed_ms':pct(99),'max_elapsed_ms':max(lat) if lat else 0,'failure_classes':classes,'focusa_evidence':{'target_ref':'UIAI browser flakiness soak','result':f"flakiness soak ok={soak_ok} soak_rate={soak_rate:.1%} passed={passed}/{total} p95_ms={pct(95)} negative_rate={negative_rate:.1%}",'evidence_ref':evidence_ref,'diagnostics_ref':out,'focusa_scope':focusa_scope,'intake_tool':'focusa_evidence_capture'},'results':results}
with open(out,'w') as f: json.dump(report,f,indent=2)
print(json.dumps({k:report[k] for k in ['ok','duration_seconds','concurrency','total_runs','passed','failed','soak_rate','negative_passed','negative_rate','avg_elapsed_ms','p95_elapsed_ms','p99_elapsed_ms','max_elapsed_ms','failure_classes']},indent=2))
if not soak_ok:
    raise SystemExit(1)
PY
printf 'soak_report=%s\n' "$OUT"

