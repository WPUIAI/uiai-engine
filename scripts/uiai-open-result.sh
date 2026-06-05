#!/usr/bin/env bash
set -euo pipefail
ENGINE_URL="${UIAI_ENGINE_URL:-http://localhost:7456}"
TIMEOUT_SECONDS="${UIAI_CLI_TIMEOUT_SECONDS:-30}"
QUERY=""; SEARCH_JSON=""; INDEX=1; READ_PAGE=1; MAX_CHARS=2000; OUT=""
usage(){ cat <<'USAGE'
Usage: scripts/uiai-open-result.sh (--query Q | --search-json FILE) [--index N] [--no-read] [--max-chars N] [--out FILE]

Searches or reads an existing search response, opens the selected 1-based result URL in a browser session, optionally reads bounded page text, and returns JSON with session id, selected URL, evidence refs, Focusa metadata, and cleanup recommendation.
USAGE
}
while [[ $# -gt 0 ]]; do
  case "$1" in
    --query) QUERY="${2:?missing query}"; shift 2 ;;
    --search-json) SEARCH_JSON="${2:?missing file}"; shift 2 ;;
    --index) INDEX="${2:?missing index}"; shift 2 ;;
    --no-read) READ_PAGE=0; shift ;;
    --max-chars) MAX_CHARS="${2:?missing max chars}"; shift 2 ;;
    --out) OUT="${2:?missing out}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done
if [[ -z "$QUERY" && -z "$SEARCH_JSON" ]]; then echo "missing --query or --search-json" >&2; usage >&2; exit 2; fi
if [[ -n "$QUERY" && -n "$SEARCH_JSON" ]]; then echo "use --query or --search-json, not both" >&2; exit 2; fi
python3 - "$ENGINE_URL" "$TIMEOUT_SECONDS" "$QUERY" "$SEARCH_JSON" "$INDEX" "$READ_PAGE" "$MAX_CHARS" "$OUT" "${UIAI_API_KEY:-}" "${UIAI_BEARER_TOKEN:-}" <<'PY'
import json, sys, urllib.request
engine, timeout, query, search_json, index, read_page, max_chars, out, api_key, bearer = sys.argv[1:]
timeout=float(timeout); index=int(index); read_page=read_page=='1'; max_chars=int(max_chars)
headers={}
if api_key: headers['X-API-Key']=api_key
elif bearer: headers['Authorization']='Bearer '+bearer
def req(method,path,payload=None):
    data=None; h=dict(headers)
    if payload is not None:
        data=json.dumps(payload).encode(); h['Content-Type']='application/json'
    r=urllib.request.Request(engine+path, data=data, headers=h, method=method)
    with urllib.request.urlopen(r, timeout=timeout) as res: return json.load(res)
search=json.load(open(search_json)) if search_json else req('POST','/api/search',{'query':query,'limit':max(index,3)})
results=search.get('results') or []
if index < 1 or index > len(results):
    print(json.dumps({'ok':False,'error_class':'invalid_result_index','message':f'index {index} outside 1..{len(results)}','count':len(results)}, indent=2)); raise SystemExit(1)
selected=results[index-1]; url=selected.get('url') or ''
if not url:
    print(json.dumps({'ok':False,'error_class':'missing_result_url','index':index,'result':selected}, indent=2)); raise SystemExit(1)
evidence=selected.get('evidence_ref') or f'uiai-search:selected:{index}'
opened=req('POST','/api/session',{'url':url,'focusa_scope':{'project_root':'/home/wpuiai/uiai-engine','evidence_ref':evidence}})
session=(opened.get('session') or {}); sid=session.get('id') or opened.get('session_id') or opened.get('id')
read=None; read_evidence=None
if read_page and sid:
    read=req('POST',f'/api/session/{sid}/read',{'max_chars':max_chars,'include_links':True})
    read_evidence=(read.get('focusa_evidence') or {}).get('evidence_ref') or read.get('evidence_ref')
report={'ok':True,'query':query or None,'index':index,'selected_url':url,'selected_result':{'title':selected.get('title'),'url':url,'evidence_ref':evidence,'rank':selected.get('rank')},'session_id':sid,'read':read,'evidence_refs':[x for x in [evidence, read_evidence] if x],'focusa':{'target_ref':url,'evidence_ref':read_evidence or evidence,'result':'Opened selected search result and captured bounded read text.' if read else 'Opened selected search result in browser session.'},'cleanup':{'recommended_action':'close_when_done','tool':'uiai_browser_close','session_id':sid}}
text=json.dumps(report, indent=2)
if out: open(out,'w').write(text+'\n')
print(text)
PY
