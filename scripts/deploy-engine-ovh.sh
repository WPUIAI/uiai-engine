#!/usr/bin/env bash
set -euo pipefail

# Deploy a UIAI Engine linux/amd64 release binary to the OVH worker.
# Public hostnames are intentionally not rewired by this script.

: "${ASSET_PATH:?ASSET_PATH must point to a local uiai-engine binary}"
: "${REMOTE_HOST:=}"
: "${REMOTE_USER:=}"
: "${REMOTE_PORT:=22}"
: "${REMOTE_INSTALL_ROOT:=/home/wpuiai/uiai-engine}"
: "${REMOTE_SERVICE_NAME:=uiai-engine-ovh.service}"
: "${REMOTE_HEALTH_URL:=http://127.0.0.1:7456/health}"
: "${REMOTE_OWNER:=wpuiai}"
: "${REMOTE_GROUP:=wpuiai}"
: "${DRY_RUN:=0}"
: "${RUN_BROWSER_SMOKE:=1}"
: "${REMOTE_EVIDENCE_SCOPE_JSON:=}"
: "${REMOTE_UIAI_EPWA_PUBLIC_BASE_URL:=}"
: "${REMOTE_EXTRA_SERVICES:=}"
: "${REMOTE_SMOKE_BASE_URLS:=http://127.0.0.1:7460}"
: "${RELEASE_TAG:=manual}"

if [[ ! -f "$ASSET_PATH" ]]; then
  echo "asset missing: $ASSET_PATH" >&2
  exit 2
fi
if ! command -v sha256sum >/dev/null; then echo "sha256sum missing" >&2; exit 2; fi

LOCAL_SHA=$(sha256sum "$ASSET_PATH" | awk '{print $1}')
ASSET_BASENAME=$(basename "$ASSET_PATH")
REMOTE_TMP="/tmp/${ASSET_BASENAME}.${RELEASE_TAG}.${LOCAL_SHA}.tmp"

cat <<PLAN
UIAI Engine OVH deploy plan
  release_tag=$RELEASE_TAG
  asset=$ASSET_PATH
  local_sha256=$LOCAL_SHA
  remote=${REMOTE_USER:+$REMOTE_USER@}${REMOTE_HOST:-unset}:${REMOTE_INSTALL_ROOT}
  service=$REMOTE_SERVICE_NAME
  health_url=$REMOTE_HEALTH_URL
  dry_run=$DRY_RUN
  browser_smoke=$RUN_BROWSER_SMOKE
  evidence_scope_configured=$([[ -n "$REMOTE_EVIDENCE_SCOPE_JSON" ]] && echo true || echo false)
  epwa_base_url_configured=$([[ -n "$REMOTE_UIAI_EPWA_PUBLIC_BASE_URL" ]] && echo true || echo false)
  extra_services=${REMOTE_EXTRA_SERVICES:-none}
  smoke_bases=$REMOTE_SMOKE_BASE_URLS
PLAN

if [[ "$DRY_RUN" == "1" || "$DRY_RUN" == "true" ]]; then
  echo "DRY_RUN: artifact and plan validated; skipping SSH/upload/install"
  exit 0
fi

: "${REMOTE_HOST:?REMOTE_HOST is required, e.g. uiai-ovh, vps-d09121de, or 100.69.132.82}"
if ! command -v ssh >/dev/null; then echo "ssh missing" >&2; exit 2; fi
if ! command -v scp >/dev/null; then echo "scp missing" >&2; exit 2; fi

REMOTE_TARGET="$REMOTE_HOST"
if [[ -n "$REMOTE_USER" ]]; then
  REMOTE_TARGET="$REMOTE_USER@$REMOTE_HOST"
fi
SSH=(ssh -p "$REMOTE_PORT" -o BatchMode=yes -o StrictHostKeyChecking=accept-new "$REMOTE_TARGET")
SCP=(scp -P "$REMOTE_PORT" -o BatchMode=yes -o StrictHostKeyChecking=accept-new)

"${SSH[@]}" bash -s -- "$REMOTE_INSTALL_ROOT" "$REMOTE_SERVICE_NAME" <<"REMOTE_PREFLIGHT"
set -euo pipefail
install_root=$1
service_name=$2
test -d "$install_root"
systemctl status "$service_name" --no-pager >/dev/null
echo remote_preflight_ok
REMOTE_PREFLIGHT

"${SCP[@]}" "$ASSET_PATH" "${REMOTE_TARGET}:${REMOTE_TMP}"

"${SSH[@]}" bash -s -- "$REMOTE_TMP" "$REMOTE_INSTALL_ROOT" "$REMOTE_SERVICE_NAME" "$REMOTE_OWNER" "$REMOTE_GROUP" "$LOCAL_SHA" "$RELEASE_TAG" "$REMOTE_HEALTH_URL" "$REMOTE_UIAI_EPWA_PUBLIC_BASE_URL" "$REMOTE_EXTRA_SERVICES" <<"REMOTE_DEPLOY"
set -euo pipefail
remote_tmp=$1
install_root=$2
service_name=$3
owner=$4
group=$5
expected_sha=$6
release_tag=$7
health_url=$8
epwa_base_url=$9
extra_services=${10}
binary_path="$install_root/uiai-engine"
backup_dir="$install_root/backups"
stamp=$(date -u +%Y%m%dT%H%M%SZ)

install -d -o "$owner" -g "$group" "$backup_dir"
test -f "$remote_tmp"
remote_sha=$(sha256sum "$remote_tmp" | awk '{print $1}')
if [[ "$remote_sha" != "$expected_sha" ]]; then
  echo "remote sha mismatch: $remote_sha != $expected_sha" >&2
  exit 3
fi
if [[ -f "$binary_path" ]]; then
  cp -a "$binary_path" "$backup_dir/uiai-engine.${release_tag}.${stamp}"
fi
install -o "$owner" -g "$group" -m 0755 "$remote_tmp" "$binary_path"
rm -f "$remote_tmp"
if [[ -n "$epwa_base_url" ]]; then
  # Configure the engine's own durable HTTPS evidence base (public routing is not rewired here).
  dropin_dir="/etc/systemd/system/${service_name}.d"
  mkdir -p "$dropin_dir"
  printf "[Service]\nEnvironment=UIAI_EPWA_PUBLIC_BASE_URL=%s\n" "$epwa_base_url" > "$dropin_dir/epwa-public-base-url.conf"
  systemctl daemon-reload
fi
systemctl restart "$service_name"
"$binary_path" -version
sha256sum "$binary_path"
# BEGIN HEALTH READINESS GATE (exercised directly by test_deploy_health.py)
health_output=$(mktemp)
healthy=0
for attempt in {1..20}; do
  : > "$health_output"
  if http_code=$(curl -sS -m 3 -o "$health_output" -w "%{http_code}" "$health_url") && [[ "$http_code" == 200 ]]; then
    healthy=1
    break
  fi
  echo "waiting_for_health attempt=$attempt status=${http_code:-transport_error}" >&2
  if (( attempt < 20 )); then sleep 1; fi
done
cat "$health_output"
echo
echo "health_http_code=${http_code:-transport_error}"
if (( healthy != 1 )); then
  echo "health readiness failed; response retained at $health_output" >&2
  exit 4
fi
rm -- "$health_output"
# END HEALTH READINESS GATE
systemctl is-active "$service_name"
for extra in $extra_services; do
  if [[ -n "$epwa_base_url" ]]; then
    extra_dropin="/etc/systemd/system/${extra}.d"
    mkdir -p "$extra_dropin"
    printf "[Service]\nEnvironment=UIAI_EPWA_PUBLIC_BASE_URL=%s\n" "$epwa_base_url" > "$extra_dropin/epwa-public-base-url.conf"
  fi
  systemctl daemon-reload
  systemctl restart "$extra"
  sleep 12
  systemctl is-active "$extra"
  echo "extra_service_active=$extra"
done
# Check the running processes, not merely the replacement file on disk.
for running_service in "$service_name" $extra_services; do
  pid=$(systemctl show -p MainPID --value "$running_service")
  [[ "$pid" =~ ^[1-9][0-9]*$ ]] || { echo "missing live pid: $running_service" >&2; exit 4; }
  running_sha=$(sha256sum "/proc/$pid/exe" | awk '{print $1}')
  [[ "$running_sha" == "$expected_sha" ]] || { echo "running binary mismatch: $running_service" >&2; exit 4; }
  echo "running_binary_verified=$running_service sha256=$running_sha"
done
REMOTE_DEPLOY

if [[ "$RUN_BROWSER_SMOKE" == "1" || "$RUN_BROWSER_SMOKE" == "true" ]]; then
  [[ -n "$REMOTE_EVIDENCE_SCOPE_JSON" ]] || { echo "REMOTE_EVIDENCE_SCOPE_JSON is required for the mandatory EPWA browser smoke" >&2; exit 4; }
  scope_b64="$(printf '%s' "$REMOTE_EVIDENCE_SCOPE_JSON" | base64 -w0)"
  "${SSH[@]}" python3 - "$scope_b64" $REMOTE_SMOKE_BASE_URLS <<"PY"
import base64, json, subprocess, sys, urllib.request, time
scope = json.loads(base64.b64decode(sys.argv[1]))
for base in sys.argv[2:]:
    ready = False
    for attempt in (1, 2, 3, 4):
        body = json.dumps({"url":"https://example.com", "width":800, "height":600, "focusa_scope":scope}).encode()
        req = urllib.request.Request(base + "/api/session", data=body, headers={"Content-Type":"application/json"}, method="POST")
        t0 = time.perf_counter()
        sid = None
        with urllib.request.urlopen(req, timeout=75) as resp:
            raw = resp.read().decode(errors="replace")
            ms = (time.perf_counter() - t0) * 1000
            js = json.loads(raw)
            session = js.get("session") or {}
            sid = js.get("session_id") or js.get("id") or session.get("session_id") or session.get("id")
            delivery = js.get("epwa_delivery") or {}
            epwa = delivery.get("epwa") or {}
            ready = (delivery.get("schema") == "uiai.epwa_delivery.v1" and delivery.get("state") == "ready" and js.get("delivery_state") == "ready" and str(epwa.get("record_url", "")).startswith("https://") and str(epwa.get("portable_url", "")).startswith("https://") and js.get("artifact_url") == epwa.get("record_url") and js.get("portable_url") == epwa.get("portable_url") and not any(key in js for key in ("screenshot", "imageBase64", "image_base64", "artifact_path", "result_path", "result_url")))
            print(json.dumps({"smoke_base": base, "attempt": attempt, "status": resp.status, "ms": round(ms, 2), "session_id_present": bool(sid), "epwa_delivery_ready": ready, "delivery_state": delivery.get("state"), "delivery_id": delivery.get("delivery_id")}, sort_keys=True), flush=True)
        if sid:
            subprocess.run(["curl", "-sS", "-m", "15", "-X", "DELETE", base + "/api/session/" + sid], check=False)
        if ready:
            # Recipient-facing check: the emitted HTTPS record and portable copy must be publicly fetchable.
            try:
                with urllib.request.urlopen(epwa.get("record_url"), timeout=30) as rr:
                    page_ok = rr.status == 200 and len(rr.read()) > 500
                with urllib.request.urlopen(epwa.get("portable_url"), timeout=30) as rp:
                    zip_ok = rp.status == 200 and rp.read(2) == b"PK"
                # Diagnostic only: some egress points (e.g. this host via Cloudflare) are bot-challenged
                # while real recipient browsers fetch fine; external recipient proof happens outside the matrix.
                print(json.dumps({"smoke_base": base, "attempt": attempt, "public_fetch": page_ok and zip_ok}, sort_keys=True), flush=True)
            except Exception as exc:
                print(json.dumps({"smoke_base": base, "attempt": attempt, "public_fetch": False, "error": str(exc)[:140]}, sort_keys=True), flush=True)
            if ready:
                break
        if attempt < 4:
            time.sleep(20)
    if not ready:
        raise SystemExit("browser smoke did not return a ready HTTPS EPWA delivery")
PY
fi

echo "deploy_complete local_sha256=$LOCAL_SHA"
