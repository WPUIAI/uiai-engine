#!/usr/bin/env bash
# Trim WPUIAI/UIAI runtime logs/state without deleting service configuration.
# Defaults are conservative and can be overridden with env vars.

set -euo pipefail

ROOT_DIR="${UIAI_ROOT:-/home/wpuiai/uiai-engine}"
DATA_DIR="${UIAI_DATA_DIR:-$ROOT_DIR/data}"
USAGE_JSON="${UIAI_USAGE_JSON:-$DATA_DIR/usage.json}"
ENGINE_LOG="${UIAI_ENGINE_LOG:-/var/log/uiai/engine.log}"
CAPTCHA_STATS="${UIAI_CAPTCHA_STATS:-/var/log/uiai/captcha-stats.jsonl}"
IP_POOL_HEALTH="${UIAI_IP_POOL_HEALTH:-/var/log/uiai/ip-pool-health.json}"

KEEP_USAGE_DAYS="${KEEP_USAGE_DAYS:-30}"
KEEP_USAGE_RECORDS="${KEEP_USAGE_RECORDS:-5000}"
KEEP_JSONL_LINES="${KEEP_JSONL_LINES:-10000}"
KEEP_LOG_LINES="${KEEP_LOG_LINES:-5000}"
MAX_ENGINE_LOG_BYTES="${MAX_ENGINE_LOG_BYTES:-10485760}" # 10 MiB
DRY_RUN="${DRY_RUN:-0}"

log() { printf "%s\n" "$*"; }

trim_usage_json() {
  local path="$1"
  [[ -f "$path" ]] || { log "skip usage: missing $path"; return 0; }

  python3 - "$path" "$KEEP_USAGE_DAYS" "$KEEP_USAGE_RECORDS" "$DRY_RUN" <<'PY'
import json, os, sys, tempfile
from datetime import datetime, timezone, timedelta

path, keep_days_s, keep_records_s, dry_s = sys.argv[1:5]
keep_days = int(keep_days_s)
keep_records = int(keep_records_s)
dry = dry_s == "1"

with open(path, "r", encoding="utf-8") as f:
    data = json.load(f)
if not isinstance(data, list):
    raise SystemExit(f"refuse usage trim: {path} is not a JSON array")

cutoff = datetime.now(timezone.utc) - timedelta(days=keep_days)
kept = []
for rec in data:
    ts = rec.get("createdAt") if isinstance(rec, dict) else None
    keep = False
    if isinstance(ts, str) and ts:
        try:
            dt = datetime.fromisoformat(ts.replace("Z", "+00:00"))
            keep = dt >= cutoff
        except ValueError:
            keep = True  # malformed timestamp: keep rather than risk data loss
    else:
        keep = True
    if keep:
        kept.append(rec)

if len(kept) > keep_records:
    kept = kept[-keep_records:]

print(f"usage {path}: {len(data)} -> {len(kept)} records")
if dry or len(kept) == len(data):
    raise SystemExit(0)

dirname = os.path.dirname(path) or "."
fd, tmp = tempfile.mkstemp(prefix=".usage-trim-", suffix=".json", dir=dirname)
try:
    with os.fdopen(fd, "w", encoding="utf-8") as f:
        json.dump(kept, f, indent=2)
        f.write("\n")
    os.chmod(tmp, 0o644)
    os.replace(tmp, path)
finally:
    if os.path.exists(tmp):
        os.unlink(tmp)
PY
}

trim_tail_lines() {
  local path="$1" keep_lines="$2" label="$3"
  [[ -f "$path" ]] || { log "skip $label: missing $path"; return 0; }
  local total
  total=$(wc -l < "$path" | tr -d " ")
  log "$label $path: $total -> max $keep_lines lines"
  [[ "$DRY_RUN" == "1" || "$total" -le "$keep_lines" ]] && return 0
  local tmp
  tmp=$(mktemp "$(dirname "$path")/.${label}.trim.XXXXXX")
  tail -n "$keep_lines" "$path" > "$tmp"
  chmod --reference="$path" "$tmp" 2>/dev/null || chmod 0644 "$tmp"
  mv "$tmp" "$path"
}

trim_engine_log() {
  local path="$1"
  [[ -f "$path" ]] || { log "skip engine log: missing $path"; return 0; }
  local size
  size=$(stat -c %s "$path")
  log "engine log $path: ${size}B, max ${MAX_ENGINE_LOG_BYTES}B"
  [[ "$size" -le "$MAX_ENGINE_LOG_BYTES" ]] && return 0
  trim_tail_lines "$path" "$KEEP_LOG_LINES" "engine-log"
}

trim_usage_json "$USAGE_JSON"
trim_tail_lines "$CAPTCHA_STATS" "$KEEP_JSONL_LINES" "captcha-stats"
trim_engine_log "$ENGINE_LOG"
[[ -f "$IP_POOL_HEALTH" ]] && log "keep ip-pool health snapshot: $IP_POOL_HEALTH"
