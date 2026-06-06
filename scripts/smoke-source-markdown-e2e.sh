#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${ROOT_DIR:-/home/wpuiai/uiai-engine}"
ENGINE_BIN="${ENGINE_BIN:-/tmp/uiai-engine-source-markdown-e2e}"
ENGINE_PORT="${ENGINE_PORT:-17656}"
SITE_PORT="${SITE_PORT:-17657}"
TMPDIR="${TMPDIR:-$(mktemp -d /tmp/uiai-source-md-e2e.XXXXXX)}"
REPORT="${REPORT:-/tmp/uiai-source-markdown-e2e-report.json}"
KEEP="${KEEP:-0}"

cleanup() {
  if [[ "$KEEP" != "1" ]]; then
    [[ -f "$TMPDIR/engine.pid" ]] && kill "$(cat "$TMPDIR/engine.pid")" 2>/dev/null || true
    [[ -f "$TMPDIR/site.pid" ]] && kill "$(cat "$TMPDIR/site.pid")" 2>/dev/null || true
    pkill -P "$(cat "$TMPDIR/engine.pid" 2>/dev/null || echo 0)" 2>/dev/null || true
  fi
}
trap cleanup EXIT

cd "$ROOT_DIR"
go build -o "$ENGINE_BIN" ./cmd/uiai-engine
mkdir -p "$TMPDIR/data" "$TMPDIR/share" "$TMPDIR/site"
cat > "$TMPDIR/config.yaml" <<YAML
server:
  host: 127.0.0.1
  port: $ENGINE_PORT
  vision_pool_size: 1
vision:
  pool_size: 1
  max_pool: 2
  idle_timeout: 60s
  screenshot_quality: 65
  allow_private_urls: true
  share_dir: $TMPDIR/share
storage:
  data_dir: $TMPDIR/data
  usage_file: $TMPDIR/data/usage.json
logging:
  level: info
cors:
  origins: ["*"]
  methods: ["GET", "POST", "DELETE", "OPTIONS"]
  headers: ["*"]
YAML
cat > "$TMPDIR/site/index.html" <<HTML
<!doctype html><html><head><title>E2E Source Markdown</title><meta name="description" content="E2E description"><link rel="canonical" href="http://127.0.0.1:$SITE_PORT/index.html"></head><body><header>Skip nav</header><main><h1>E2E Source Markdown</h1><p>This is a real browser-rendered paragraph for Source-to-Markdown.</p><ul><li>alpha</li><li>beta</li></ul><a href="/next.html">Next page</a></main><footer>Footer</footer></body></html>
HTML
( cd "$TMPDIR/site" && python3 -m http.server "$SITE_PORT" --bind 127.0.0.1 > "$TMPDIR/site.log" 2>&1 & echo $! > "$TMPDIR/site.pid" )
"$ENGINE_BIN" -config "$TMPDIR/config.yaml" > "$TMPDIR/engine.log" 2>&1 & echo $! > "$TMPDIR/engine.pid"
for _ in $(seq 1 90); do
  if curl -fsS "http://127.0.0.1:$ENGINE_PORT/api/health" >/dev/null; then break; fi
  sleep 1
done
curl -fsS "http://127.0.0.1:$ENGINE_PORT/api/health" >/dev/null

python3 - "$ENGINE_PORT" "$SITE_PORT" "$REPORT" <<'PY'
import json, sys, urllib.request
engine_port, site_port, report_path = sys.argv[1:4]
BASE=f"http://127.0.0.1:{engine_port}"
LOCAL=f"http://127.0.0.1:{site_port}/index.html"
tests=[]
def req(method,path,body=None,timeout=150):
    headers={}; data=None
    if body is not None:
        headers["Content-Type"]="application/json"; data=json.dumps(body).encode()
    with urllib.request.urlopen(urllib.request.Request(BASE+path,data=data,headers=headers,method=method),timeout=timeout) as res:
        return json.loads(res.read().decode("utf-8","replace"))
def add(name, ok, detail="", data=None): tests.append({"name":name,"ok":bool(ok),"detail":detail,"data":data})
def no_secret(value):
    text=json.dumps(value).lower()
    return "#frag" not in text and "token=secret" not in text and "token%3dsecret" not in text and "?token=secret" not in text
try:
    h=req("GET","/api/health"); add("health", h.get("status") in ("healthy","ok"), h.get("status"), h)
except Exception as e: add("health", False, repr(e))
try:
    names=[t.get("name") for t in req("GET","/api/tools/search?q=markdown").get("tools",[])]
    add("tool_search_markdown", "source_to_markdown" in names and "browser_read" in names, str(names[:6]), {"names":names[:10]})
except Exception as e: add("tool_search_markdown", False, repr(e))
try:
    graph=req("GET","/api/tools/graph"); workflows=[w.get("name") for w in graph.get("workflows",[])]; refs=graph.get("focusa_integration",{}).get("evidence_refs",[])
    add("tool_graph_focusa_source_markdown", "source_to_markdown" in workflows and "uiai-source-markdown:sha256:<prefix>" in refs, str(workflows), {"refs":refs})
except Exception as e: add("tool_graph_focusa_source_markdown", False, repr(e))
try:
    names=[t.get("name") for t in req("GET","/api/tools/mcp").get("tools",[])]
    add("mcp_source_to_markdown", "source_to_markdown" in names, "present" if "source_to_markdown" in names else str(names[:10]))
except Exception as e: add("mcp_source_to_markdown", False, repr(e))
local=None
try:
    local=req("POST","/api/markdown",{"url":LOCAL,"max_chars":6000,"include_links":True})
    markdown=local.get("markdown",""); card=local.get("wpuiai",{}).get("research_card",{})
    add("api_markdown_local_webpage", local.get("schema")=="uiai.source_markdown.v1" and "# E2E Source Markdown" in markdown and local.get("focusa",{}).get("evidence_ref","").startswith("uiai-source-markdown:sha256:") and card.get("schema")=="wpui.source_markdown_research_card.v1" and local.get("cleanup",{}).get("closed") is True and "Footer" not in markdown and no_secret(local), local.get("focusa",{}).get("evidence_ref",""), {"title":local.get("title"),"chars":local.get("chars"),"adapter":local.get("metadata",{}).get("adapter")})
except Exception as e: add("api_markdown_local_webpage", False, repr(e))
try:
    opened=req("POST","/api/session",{"url":LOCAL,"width":1000,"height":700}); sid=opened.get("session",{}).get("id")
    read=req("POST",f"/api/session/{sid}/read",{"format":"markdown","max_chars":4000,"include_links":True})
    close=req("DELETE",f"/api/session/{sid}")
    add("browser_session_read_markdown", sid and read.get("schema")=="uiai.browser_read.v2" and read.get("format")=="markdown" and "# E2E Source Markdown" in read.get("text","") and read.get("focusa",{}).get("evidence_ref","").startswith("uiai-browser:session="), sid or "", {"chars":read.get("chars"),"close":close})
except Exception as e: add("browser_session_read_markdown", False, repr(e))
try:
    packet=req("POST","/api/agent/research-packet",{"goal":"E2E source markdown packet","responses":[local or {}],"recommended_next_action":"Capture Source-to-Markdown E2E evidence."})
    caps=packet.get("captures",[])
    add("focusa_packet_from_source_markdown", packet.get("schema")=="uiai.focusa_research_diagnostics_packet.v1" and caps and caps[0].get("type")=="source_markdown" and packet.get("recommended_focusa",{}).get("preferred_tool")=="focusa_evidence_capture" and no_secret(packet), caps[0].get("evidence_ref") if caps else "no capture", {"capture_type":caps[0].get("type") if caps else None})
except Exception as e: add("focusa_packet_from_source_markdown", False, repr(e))
for name,url,adapter in [("github_adapter_live","https://github.com/WPUIAI/uiai-engine/issues/1?token=secret#frag","github_public"),("reddit_adapter_live","https://old.reddit.com/r/golang/comments/1/example_title/?token=secret#frag","reddit_public"),("hackernews_adapter_live","https://news.ycombinator.com/item?id=8863&token=secret#frag","hackernews_public"),("youtube_adapter_live","https://www.youtube.com/watch?v=dQw4w9WgXcQ&token=secret#frag","youtube_public"),("x_adapter_live","https://x.com/example/status/12345?token=secret#frag","x_public")]:
    last_error=""; last_data=None; passed=False; detail=""
    for _attempt in range(2):
        try:
            out=req("POST","/api/markdown",{"url":url,"max_chars":2500})
            meta=out.get("metadata",{})
            detail=meta.get("failure_class") or meta.get("source_type") or meta.get("adapter") or ""
            last_data={"adapter":meta.get("adapter"),"source_type":meta.get("source_type"),"blocked":meta.get("blocked"),"failure_class":meta.get("failure_class"),"records":len(out.get("records",[])),"chars":out.get("chars")}
            passed=bool(out.get("schema")=="uiai.source_markdown.v1" and meta.get("adapter")==adapter and out.get("records") and out.get("focusa",{}).get("evidence_ref","").startswith("uiai-source-markdown:sha256:") and no_secret(out))
            if passed:
                break
        except Exception as e:
            last_error=repr(e)
    add(name, passed, detail or last_error, last_data)
report={"ok":all(t["ok"] for t in tests),"base":BASE,"tests":tests}
open(report_path,"w").write(json.dumps(report,indent=2))
print(json.dumps({"ok":report["ok"],"passed":sum(t["ok"] for t in tests),"total":len(tests),"report":report_path,"failed":[t["name"] for t in tests if not t["ok"]]},indent=2))
if not report["ok"]: raise SystemExit(1)
PY
