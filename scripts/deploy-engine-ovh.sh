#!/usr/bin/env bash
set -euo pipefail

# Deploy a UIAI Engine linux/amd64 release binary to the OVH worker.
# Public hostnames are intentionally not rewired by this script.

: "${ASSET_PATH:?ASSET_PATH must point to a local uiai-engine binary}"
: "${REMOTE_HOST:?REMOTE_HOST is required, e.g. vps-d09121de or 100.69.132.82}"
: "${REMOTE_USER:=root}"
: "${REMOTE_PORT:=22}"
: "${REMOTE_INSTALL_ROOT:=/home/wpuiai/uiai-engine}"
: "${REMOTE_SERVICE_NAME:=uiai-engine-ovh.service}"
: "${REMOTE_HEALTH_URL:=http://127.0.0.1:7456/v1/health}"
: "${REMOTE_OWNER:=wpuiai}"
: "${REMOTE_GROUP:=wpuiai}"
: "${DRY_RUN:=0}"
: "${RUN_BROWSER_SMOKE:=1}"
: "${RELEASE_TAG:=manual}"

if [[ ! -f "$ASSET_PATH" ]]; then
  echo "asset missing: $ASSET_PATH" >&2
  exit 2
fi

if ! command -v ssh >/dev/null; then echo "ssh missing" >&2; exit 2; fi
if ! command -v scp >/dev/null; then echo "scp missing" >&2; exit 2; fi
if ! command -v sha256sum >/dev/null; then echo "sha256sum missing" >&2; exit 2; fi

LOCAL_SHA=$(sha256sum "$ASSET_PATH" | awk "{print \$1}")
ASSET_BASENAME=$(basename "$ASSET_PATH")
REMOTE_TMP="/tmp/${ASSET_BASENAME}.${RELEASE_TAG}.${LOCAL_SHA}.tmp"
SSH=(ssh -p "$REMOTE_PORT" -o BatchMode=yes -o StrictHostKeyChecking=accept-new "${REMOTE_USER}@${REMOTE_HOST}")
SCP=(scp -P "$REMOTE_PORT" -o BatchMode=yes -o StrictHostKeyChecking=accept-new)

cat <<PLAN
UIAI Engine OVH deploy plan
  release_tag=$RELEASE_TAG
  asset=$ASSET_PATH
  local_sha256=$LOCAL_SHA
  remote=${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_INSTALL_ROOT}
  service=$REMOTE_SERVICE_NAME
  health_url=$REMOTE_HEALTH_URL
  dry_run=$DRY_RUN
  browser_smoke=$RUN_BROWSER_SMOKE
PLAN

"${SSH[@]}" "set -euo pipefail; test -d ; systemctl status  --no-pager >/dev/null; echo remote_preflight_ok"

if [[ "$DRY_RUN" == "1" || "$DRY_RUN" == "true" ]]; then
  echo "DRY_RUN: stopping before upload/install"
  exit 0
fi

"${SCP[@]}" "$ASSET_PATH" "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_TMP}"

"${SSH[@]}" "set -euo pipefail
  install -d -o  -g  /backups
  test -f 
  REMOTE_SHA=\$(sha256sum  | awk {print $1})
  if [[ \"\$REMOTE_SHA\" !=  ]]; then
    echo \"remote sha mismatch: \$REMOTE_SHA != $LOCAL_SHA\" >&2
    exit 3
  fi
  if [[ -f /uiai-engine ]]; then
    cp -a /uiai-engine /backups/uiai-engine..20260705T164522Z
  fi
  install -o  -g  -m 0755  /uiai-engine
  rm -f 
  systemctl restart 
  sleep 3
  systemctl is-active 
  /uiai-engine -version || true
  sha256sum /uiai-engine
  HTTP_CODE=\$(curl -sS -m 10 -o /tmp/uiai-engine-health.out -w %{http_code}  || true)
  cat /tmp/uiai-engine-health.out || true
  echo
  echo \"health_http_code=\$HTTP_CODE\"
  case \"\$HTTP_CODE\" in 200|401) ;; *) echo \"unexpected health status: \$HTTP_CODE\" >&2; exit 4 ;; esac
"

if [[ "$RUN_BROWSER_SMOKE" == "1" || "$RUN_BROWSER_SMOKE" == "true" ]]; then
  "${SSH[@]}" "python3 - <<PY
import json, subprocess, urllib.request, time
base=http://127.0.0.1:7456
body=json.dumps({url:https://example.com,width:800,height:600}).encode()
req=urllib.request.Request(base+/api/session, data=body, headers={Content-Type:application/json}, method=POST)
t0=time.perf_counter()
with urllib.request.urlopen(req, timeout=75) as resp:
    raw=resp.read().decode(errors=replace)
    ms=(time.perf_counter()-t0)*1000
    js=json.loads(raw)
    sid=js.get(session_id) or js.get(id) or (js.get(session) or {}).get(session_id) or (js.get(session) or {}).get(id)
    ss=js.get(screenshot)
    print(json.dumps({status: resp.status, ms: round(ms,2), session_id_present: bool(sid), screenshot_present: isinstance(ss,str) and len(ss)>0}, sort_keys=True))
if sid:
    subprocess.run([curl,-sS,-m,15,-X,DELETE,base+/api/session/+sid], check=False)
PY"
fi

echo "deploy_complete local_sha256=$LOCAL_SHA"
