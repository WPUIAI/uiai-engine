# C-010-11/12 + 26a evidence — 2026-08-25
- pressure.go: treeRSSKB procfs walk; recycle at RSS budget (UIAI_RECYCLE_RSS_MB, default 1536); tests TreeRSS/ShouldRecycle/EnvOverride ok
- priority queue: batchShouldYield + GetPageBatch defers to interactive within 5s starvation window; tests Batch/Prio ok
- caddy active checks (v2.6.2): health_uri /health interval 5s timeout 2s fail_duration 10s
  LIVE: normal 200/2.4ms · W2 stopped → 200/2.3ms ×2 via W1 · W2 back 200/2.4ms
- full suites green: routes/license/captcha/vision
