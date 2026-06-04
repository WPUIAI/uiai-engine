# Full API Parity Evaluation and Retirement Inventory — 2026-03-07

Scope: evaluate **all** Go API surfaces before retiring any old AI API dependency code built on Bun / standalone PHP API / old npm-era API services.

Rule:
- Do **not** remove local plugin functionality.
- Do **not** remove old AI API dependency code unless Go parity is verified end-to-end.
- "Route exists" is not enough.
- Required proof: caller compatibility, behavior parity, auth parity, response-shape parity, fallback/repair parity where applicable.

---

## Source anchors

### Go route inventory
- `/home/wpuiai/uiai-engine/internal/server/server.go:127-253`

### Current plugin/admin caller inventory
- `/home/wpuiai/public_html/wp-content/plugins/wpuiai`
- `/home/wpuiai/public_html/wp-content/plugins/wpuiai-ai-cloud-admin`

### Existing audit reference
- `/home/wpuiai/public_html/wp-content/plugins/wpuiai/docs/CLOUD_READINESS_AUDIT.md:159-170`

### Current UI Reverse gap inventory
- `/home/wpuiai/uiai-engine/docs/UI_REVERSE_GO_PARITY_GAP_INVENTORY_2026-03-07.md`

---

## Route surface map (Go)

Core + AI:
- `/api/health`
- `/api/status`
- `/api/critique`
- `/api/ui-reverse`
- `/api/section-detect`
- `/api/layout-compare`
- `/api/style-enhance`
- `/api/copilot`
- `/api/intake`
- `/api/workflow`
- `/api/usage`
- `/api/extension`
- `/api/memory`
- `/api/admin`
- `/api/reference`
- `/api/intelligence`
- `/api/training`

Media + browser:
- `/api/media`
- `/api/screenshot`
- `/api/share`
- `/api/session`
- `/api/tools`
- `/vision`
- `/v/{token}`

Pipeline/orchestration:
- `/api/design-system`
- `/api/content-map`
- `/api/block-recipes`
- `/api/comparison`
- `/api/migration`
- `/api/events`

---

## Current certification status by domain

### 1. Health / status / auth / core infra
**Status:** NOT FULLY EVALUATED

Known:
- health/status routes exist in Go
- auth middleware exists and includes many compatibility exceptions
- old docs still contain Bun/PHP-era assumptions

Need:
- verify `/api/health`, `/health`, `/api/status` contracts vs plugin/admin expectations
- verify auth parity for license/API key/webhook/extension token callers
- verify old compatibility exceptions are still necessary

---

### 2. Screenshot / share / viewer
**Status:** PARTIAL — NOT YET SAFE TO RETIRE

Known:
- screenshot capture route is real in Go and works for current Go-style callers
- share upload semantics are not fully ported
- multi-share viewport upload semantics are not fully ported
- viewer parity is not proven

Evidence:
- Go screenshot: `/home/wpuiai/uiai-engine/internal/routes/screenshot.go`
- Go share: `/home/wpuiai/uiai-engine/internal/routes/share.go`
- Go viewer: `/home/wpuiai/uiai-engine/internal/routes/stubs.go`
- plugin callers: `wpuiai/includes/class-temp-screenshot.php`, `class-screenshot-provider-vision.php`

Retirement decision:
- keep old API dependency code for share flows until parity is proven

---

### 3. Critique
**Status:** MOSTLY VERIFIED, but final retirement still needs inventory-level signoff

Known:
- Go critique handler is real and structured
- plugin caller compatibility is much better than UI Reverse
- existing readiness audit historically marked mismatch, so fresh certification should still be written explicitly

Evidence:
- Go: `/home/wpuiai/uiai-engine/internal/routes/critique.go`
- plugin caller: `wpuiai/includes/class-capability-router.php`

Need:
- contract confirmation doc
- explicit retirement decision on old critique dependency assumptions

---

### 4. UI Reverse / reference analysis
**Status:** PARTIAL — NOT YET ACCEPTED AS FULL PORT

Known:
- main parent-plugin cloud path now points to `reference/analyze`
- Go analyzer is implemented
- function-level gaps remain, especially spacing fallback/default behavior and validator parity

Evidence:
- inventory doc: `/home/wpuiai/uiai-engine/docs/UI_REVERSE_GO_PARITY_GAP_INVENTORY_2026-03-07.md`

Retirement decision:
- do not retire old UI Reverse dependency code yet

---

### 5. Section detect / layout compare / style enhance / copilot
**Status:** NOT FULLY EVALUATED

Known:
- Go handlers exist in `internal/routes/ai_routes.go`
- previous readiness audit flagged these as not ready / mismatch
- plugin has cloud-caller functions for these capabilities

Need:
- function-by-function caller/contract audit like UI Reverse
- policy verification (cloud_first/local_only drift)
- response normalization verification

---

### 6. Intake / workflow
**Status:** NOT FULLY EVALUATED

Known:
- Go routes exist
- plugin docs/test plans still reference these surfaces heavily
- prior audits implied contract drift risk

Need:
- evaluate actual plugin/admin callers vs Go request/response shapes
- verify job/run/status semantics

---

### 7. Intelligence / training
**Status:** NOT FULLY EVALUATED

Known:
- Go routes exist
- plugin still has live `WPUIAI_Cloud_API::request()` callers for intelligence index operations
- historic beads/docs reference Bun implementations and route assumptions

Need:
- verify plugin callers against Go endpoints one-by-one
- verify service-token/auth semantics and route completeness

---

### 8. Extension / memory / admin / usage
**Status:** NOT FULLY EVALUATED

Known:
- Go routes exist
- old docs and helper UIs still reference legacy assumptions in some places

Need:
- verify each domain’s live callers and response contracts
- certify whether old AI API dependency code can be retired or must stay

---

### 9. Media / session / tools / vision interactive
**Status:** IMPLEMENTED, BUT RETIREMENT RELEVANCE NOT YET CLASSIFIED

Known:
- routes exist in Go
- media route includes TODO/production hardening notes
- session/tools are newer Go-native surfaces, not obvious Bun/PHP retirement targets

Need:
- classify which of these are legacy-replacement surfaces vs net-new Go surfaces
- avoid mixing “new API” work with “safe retirement” decisions

---

### 10. Design-system / content-map / block-recipes / comparison / migration / events
**Status:** NOT YET CLASSIFIED FOR RETIREMENT PURPOSES

Known:
- Go routes exist
- unclear whether these replace old Bun/PHP dependencies or are additive/new

Need:
- classify by retirement relevance:
  - direct replacement
  - additive Go-only feature
  - no retirement implication

---

## Required work categories

### A. Domain-by-domain parity audits
Each domain needs:
1. caller inventory
2. Go route/handler inventory
3. contract diff
4. fallback/repair diff
5. auth diff
6. retirement decision

### B. Verified docs
Need source-backed docs that say, for each domain:
- SAFE TO RETIRE old dependency code
- KEEP FOR NOW
- LOCAL FUNCTIONALITY — KEEP

### C. Bead gating
No dependency retirement should proceed until the relevant parity bead closes with explicit acceptance evidence.

---

## Immediate priority order

1. Screenshot/share/viewer
2. Critique
3. UI Reverse/reference (already started)
4. Section/layout/style/copilot cluster
5. Intake/workflow
6. Intelligence/training
7. Extension/memory/admin/usage
8. Remaining route classification

---

## Current bottom line

As of 2026-03-07:
- **full API parity has not yet been certified**
- only portions of the API have been evaluated deeply
- **no broad retirement of old AI API dependency code should happen yet**
- the correct next step is documentation + bead-driven evaluation across every API domain

---

## Related documentation

- Project overview and successor relationship to `WPUIAI/ai-api`: [README](../README.md)
- UI Reverse function-level parity details: [UI Reverse Go Parity Gap Inventory](UI_REVERSE_GO_PARITY_GAP_INVENTORY_2026-03-07.md)
- Workflow/cloud caller mapping: [Workflow API Orchestration](WORKFLOW_API_ORCHESTRATION.md)
- Browser/session API surfaces added beyond the old API: [Session API](SESSION_API.md)
- Screenshot diagnostics and evidence refs: [Browser Diagnostics Spec](BROWSER_DIAGNOSTICS_SPEC.md)
- Device-frame and media integration: [Device Frame Integration](DEVICE_FRAME_INTEGRATION.md)
