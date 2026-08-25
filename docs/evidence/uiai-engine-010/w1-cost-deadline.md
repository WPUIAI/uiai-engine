# W1b evidence — 2026-08-25
- go test ./internal/routes -run 'Deadline|Cost': ok (4/4)
  * stalled handler → 503 deadline envelope <120ms, retry guidance present
  * fast handler untouched; cost object + headers stamped; pages_touched=2 propagated
  * struct payloads and non-instrumented writers safe no-op
- go build ./... clean
