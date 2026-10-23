# W1b evidence — 2026-08-25
- go test ./internal/routes -run 'Deadline|Cost': ok (4/4)
  * stalled handler → 503 deadline envelope <120ms, retry guidance present
  * fast handler untouched; cost object + headers stamped; pages_touched=2 propagated
  * struct payloads and non-instrumented writers safe no-op
- go build ./... clean

## LIVE PROOF (production W2, 2026-08-25)
GET /api/session body: {"cost":{"duration_ms":0,"pages_touched":0},...}
GET /api/health/browser trailers: X-UIAI-Cost-Bytes: 7220, X-UIAI-Cost-Ms, X-UIAI-Cost-Pages
Root cause of prior gap: install/restart desync + wrapper layers; fixed via resolveCost Unwrap walk + inner group mounts.
