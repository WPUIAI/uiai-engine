# WordPress Plugin ↔ Go Engine Route Parity Matrix

Last updated: 2026-06-04

Scope: inventory existing WordPress plugin calls from `/home/wpuiai/public_html/wp-content/plugins/wpuiai` against Go engine contracts in this repo. This doc does **not** create a new plugin; it maps the existing plugin route usage.

Auth references:
- Engine auth modes: [`ENDPOINT_AUTH_MATRIX.md`](ENDPOINT_AUTH_MATRIX.md)
- Structured error contract: [`UIAI_ENGINE_INTEROPERABILITY_QUALITY_SPEC.md`](UIAI_ENGINE_INTEROPERABILITY_QUALITY_SPEC.md#41-structured-error-envelope)
- Workflow endpoint map: [`WORKFLOW_API_ORCHESTRATION.md`](WORKFLOW_API_ORCHESTRATION.md)

## Summary table

| Plugin caller / flow | Existing plugin source | Engine route | Method | Auth mode | Request contract | Expected response | Error/UI behavior today | Parity gap / follow-up |
|---|---|---|---|---|---|---|---|---|
| Cloud design critique | `includes/class-capability-router.php:602` | `/api/critique` | POST | authenticated | `websiteUrl`, `model`, `provider`, `pageType`, `critiqueMode`, optional `referenceUrl`, `designTokens`, `imageBase64`, `imageType` | Plugin accepts `{success}` or direct `scores`, `critique`, `priority_fixes`, `summary`, `aggregates`, `usage` | Parse-error repair exists; failures reduce to string `error` and local fallback. | Preserve engine `error_id`, `error_class`, `suggested_next_action`, `diagnostics` in plugin result/log before fallback. |
| UI reverse / reference analysis | `includes/class-capability-router.php:743` | `/api/reference/analyze` | POST | authenticated | `imageBase64`, `imageType`, `pass`, `model`, `provider`, optional `sections`, `components` | Plugin unwraps `{success,data}` and returns local analyzer-compatible `data` | Failures reduce to `error`; operation names mapped to `pass`. | Keep mapping; surface structured engine error fields for failed analysis. |
| Section detection | `includes/class-capability-router.php:835` | `/api/section-detect` | POST | authenticated | Screenshot/image payload plus model/provider/options | Section list/data compatible with local section detector | Cloud failure falls back local with string error. | Verify route still mounted as AI route; record structured errors. |
| Layout comparison | `includes/class-capability-router.php:907` | `/api/layout-compare` | POST | authenticated | Built/reference image payloads plus model/provider/options | Comparison result compatible with local layout comparator | Cloud failure falls back local with string error. | Verify schema against Go route and retain structured errors. |
| Style enhancement | `includes/class-capability-router.php:980` | `/api/style-enhance` | POST | authenticated | Built/reference image payloads, model/provider/options | Enhancement/fix suggestions compatible with local style enhancer | Cloud failure falls back local with string error. | Verify schema and structured error display. |
| Copilot chat | `includes/class-capability-router.php:1056` | `/api/copilot/chat` | POST | authenticated | Message/options/context payload | Chat response compatible with local copilot | Cloud failure falls back local with string error. | Preserve engine `error_id/class/action` in Copilot UI/log. |
| Design system generation | `includes/class-capability-router.php:1098` | `/api/design-system` | POST | authenticated | `sources`, `model`, `provider` | `design_system` response; plugin returns `{success:true}+response` | Failure recorded, local fallback if generator exists. | Confirm response shape and show structured errors when no local fallback. |
| Content map | `includes/class-capability-router.php:1122` | `/api/content-map` | POST | authenticated | `pages`, `blueprint`, `model`, `provider` | `content_map` response; plugin returns `{success:true}+response` | Failure recorded, local fallback if mapper exists. | Confirm response shape and structured errors. |
| Block recipes | `includes/class-capability-router.php:1147` | `/api/block-recipes` | POST | authenticated | `section`, `design_system`, `content_map`, `model`, `provider` | `blocks` response; plugin returns `{success:true}+response` | Failure recorded, local fallback if recipes exist. | Confirm response shape and structured errors. |
| Five-way comparison | `includes/class-capability-router.php:1173` | `/api/comparison` | POST | authenticated | `built_page_url`, `design_tokens`, `reference_sections`, `reference_components`, `model`, `provider` | `comparison` response; plugin returns `{success:true}+response` | Failure recorded, local fallback if class exists. | Confirm response shape and structured errors. |
| Media production submit | `includes/class-media-producer.php:99` | `/api/media/produce` | POST | authenticated | `type`, `url`/`urls`, device/GIF options | `job_id` | Missing job id becomes `WP_Error(media_submit_failed, error)` | Include structured engine fields in `WP_Error` data. |
| Media production poll | `includes/class-media-producer.php:126` | `/api/media/status/{jobID}` | GET | public read | job id path | `status=complete|failed`, result URL/file metadata | Failed job becomes `WP_Error(media_job_failed, error)`; timeout local `media_timeout`. | Public-read matches matrix; include `error_id/class/action` if status endpoint supplies them. |
| One-shot screenshot | `includes/class-screenshot-provider-vision.php:97` | `/api/screenshot` | POST | loopback-public remote-auth | `url`, `width`, `height`, `fullPage`, `format`, `quality`, `delay` | `screenshot`, optional `dom_report`, size/duration | Non-200 falls back to session; message is `Go engine HTTP N: error`. | Preserve `error_class`, `error_id`, `suggested_next_action`, `diagnostics` in fallback reason/result metadata. |
| Session fallback open | `includes/class-screenshot-provider-vision.php:180` | `/api/session` | POST | loopback-public remote-auth | `url`, `width`, `height` | `session.id`, initial `screenshot` | Open failure concatenates raw body into failure message. | Parse structured error envelope before displaying raw JSON. |
| Session fallback resnap | `includes/class-screenshot-provider-vision.php:203` | `/api/session/{id}/screenshot` | POST | loopback-public remote-auth | `format`, `quality`, `fullPage` | `screenshot` | Failed resnap ignored if initial screenshot exists. | Log structured resnap failure fields when present. |
| Session fallback close | `includes/class-screenshot-provider-vision.php:215` | `/api/session/{id}` | DELETE | loopback-public remote-auth | session id path | close status | Fire-and-forget. | OK; no UI requirement unless close fails repeatedly. |
| Screenshot health | `includes/class-screenshot-provider-vision.php:255` | `/api/screenshot/health` | GET | loopback-public remote-auth | none | health/status | Reports engine reachable/healthy. | Engine also exposes `/api/health/browser`; consider aligning health route naming. |
| Cloud screenshot provider | `includes/class-screenshot-provider-cloud.php:121` | `/api/screenshot` via `WPUIAI_Cloud_API::request("screenshot")` | POST | loopback-public remote-auth locally; remote-auth via Cloud_API | screenshot args | screenshot payload | Cloud_API handles retry/402 but not structured fields. | Same structured error preservation as vision provider. |
| Cloud health | `includes/class-screenshot-provider-cloud.php:210` | `/api/health` via `WPUIAI_Cloud_API::request("health")` | GET | public | none | health payload | Fallback host retry. | OK; no structured error required beyond transport failure. |
| Training datasets from DevConsole | `includes/devconsole/tabs/class-tab-ai-calls.php:423` | `/api/training/datasets` | POST | service-token | dataset payload | dataset/job response | Uses `WPUIAI_Cloud_API::request`; route requires training service token per matrix. | Verify Cloud_API supplies correct service token or mark unsupported. |

## Shared plugin auth behavior

`WPUIAI_Capability_Router::post_cloud_json()` (`includes/class-capability-router.php:394-418`) sends JSON with either:

- Dev mode: `X-Webhook-Secret`
- Production: `X-License-Key`

`WPUIAI_Media_Producer` uses `X-Webhook-Secret` when available, otherwise `X-License-Key` + `X-Domain`.

## Current highest-priority gaps

1. **Structured error propagation:** Most plugin cloud paths collapse engine failures to a string `error`. Product UI/logs should retain `error_id`, `error_class`, `suggested_next_action`, and `diagnostics` when present.
2. **Screenshot/session fallback visibility:** Session fallback hides or concatenates structured JSON. Parse and display actionable class/action text.
3. **Training service-token mismatch risk:** `WPUIAI_Cloud_API::request("training/datasets")` needs verification against `/api/training/*` service-token auth.
4. **Legacy route naming:** Screenshot health uses `/api/screenshot/health`; engine status also has `/api/health/browser`. Keep both documented or converge callers.

## Update rules

- Any new plugin cloud call must add a row here with method, auth mode, request/response shape, and error behavior.
- Any engine route response change must update the corresponding plugin expected response row.
- Plugin UI changes for errors should use the structured engine fields, not raw response bodies or opaque strings.
