# C-010-13 evidence — 2026-08-25
- go test ./internal/captcha -run 'Circuit|Healthy': ok (3/3)
  * breaker opens at 0 healthy, Pick fails fast with ErrEgressUnavailable (<50ms)
  * closes on recovery; pick succeeds
  * HealthyIPs counts correctly
- go build ./... clean
- Config: proxy.direct_egress_fallback (default false) engages direct route when circuit open
