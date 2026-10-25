#!/usr/bin/env bash
# Bounded startup warm-up; cleanup ownership comes only from this exact POST.
set -euo pipefail
base="${UIAI_WARM_BASE:-http://127.0.0.1:7456}"
count="${UIAI_WARM_COUNT:-2}"
for _ in $(seq 1 30); do
  if curl -fsS --max-time 2 "$base/api/session" >/dev/null; then break; fi
  sleep 1
done
workdir=$(mktemp -d)
cleanup(){ rm -rf -- "$workdir"; }
trap cleanup EXIT
open_one(){
  idx="$1"; json="$workdir/$idx.json"; headers="$workdir/$idx.headers"
  curl_status=0
  code=$(curl -sS --max-time 60 -D "$headers" -o "$json" -w "%{http_code}" \
    -H "Content-Type: application/json" \
    -d '{"url":"https://example.com/","width":800,"height":600,"actor_ref":"system:uiai-warmup"}' \
    "$base/api/session/") || curl_status=$?
  sid=$(python3 - "$json" "$headers" <<'PY'
import json,re,sys
from pathlib import Path
try:
    data=json.loads(Path(sys.argv[1]).read_text())
except (OSError,ValueError):
    data={}
session=data.get('session') if isinstance(data,dict) else None
body=session.get('id','') if isinstance(session,dict) else ''
if not isinstance(body,str): body=''
header=''
try:
    for line in Path(sys.argv[2]).read_text().splitlines():
        key,separator,value=line.partition(':')
        if separator and key.lower()=='x-uiai-created-session-id': header=value.strip()
except OSError:
    pass
if header and body and header!=body: raise SystemExit('warmup cleanup identity mismatch')
sid=header or body
if sid and not re.fullmatch(r'[A-Za-z0-9_-]{1,128}',sid): raise SystemExit('warmup cleanup identity invalid')
print(sid)
PY
  )
  if [[ -z "$sid" ]]; then
    echo "warm_session_${idx}_unsettled status=$code transport=$curl_status cleanup=unknown" >&2
    return 0
  fi
  cleanup_status=0
  closed=$(curl -sS --max-time 15 -o "$workdir/$idx.closed" -w "%{http_code}" -X DELETE "$base/api/session/$sid") || cleanup_status=$?
  if [[ "$cleanup_status" != 0 || "$closed" != 200 ]]; then
    echo "warm_session_${idx}_cleanup_failed status=$closed transport=$cleanup_status" >&2
    return 0
  fi
  if [[ "$curl_status" == 0 && ( "$code" == 201 || "$code" == 202 ) ]]; then
    echo "warm_session_${idx}_ok sid_closed=yes"
  else
    echo "warm_session_${idx}_unsettled status=$code transport=$curl_status owned_cleanup=closed" >&2
  fi
}
for i in $(seq 1 "$count"); do open_one "$i" & done
wait
