# UI Reverse Go Parity Gap Inventory — 2026-03-07

Scope: strict function-by-function parity audit between the local PHP `WPUIAI_UI_Reference_Analyzer` and the Go `reference/analyze` path now used by `WPUIAI_Capability_Router::call_cloud_ui_reverse()`.

Rule: do **not** remove old AI API dependency code until Go parity is proven complete. Local plugin functionality is out of scope for removal.

## Current conclusion

**UI Reverse is NOT yet certified as fully ported to Go.**

What is proven:
- Parent plugin cloud path now targets Go `reference/analyze`, not Bun `/api/ui-reverse`.
- Go reference analyzer is implemented and non-stubbed.

What is NOT yet proven:
- function-by-function parity for all UI Reverse operations
- exact response/validation/fallback parity
- safe retirement of old API dependency code tied to UI Reverse

---

## Caller of record

### Parent plugin cloud path
- `/home/wpuiai/public_html/wp-content/plugins/wpuiai/includes/class-capability-router.php:687-753`
- maps UI Reverse operations to Go `reference/analyze`
- explicitly notes at `:730-731` that the Go endpoint is `reference/analyze`, not `ui-reverse`

### Local fallback path
- `/home/wpuiai/public_html/wp-content/plugins/wpuiai/includes/class-capability-router.php:765-778`
- still uses local `WPUIAI_UI_Reference_Analyzer`
- local functionality must remain intact regardless of API retirement

---

## Function parity matrix

### 1. analyze_reference

#### Local
- `/home/wpuiai/public_html/wp-content/plugins/wpuiai/includes/class-ui-reference-analyzer.php:75-134`
- Parses JSON and requires:
  - `page`
  - `sections`
- Adds:
  - `warnings`
  - `metadata` with screenshot path/hash, analyzed_at, provider, model

#### Go
- `/home/wpuiai/uiai-engine/internal/reference/analyzer.go:97-145`
- `/home/wpuiai/uiai-engine/internal/routes/reference.go:82-89`
- Returns:
  - `success`
  - `data`

#### Gap
- Go route does not return local-style `metadata`
- Go validator is weaker than local validator:
  - Local checks `type`, `aspect_ratio`, `background`, `primary_function`, and section `components`
  - Go only checks a subset
- Source:
  - local validator: `class-ui-reference-analyzer.php:720-755`
  - Go validator: `analyzer.go:360-379`

#### Verdict
- Near parity, not exact parity

---

### 2. extract_components

#### Local
- `/home/wpuiai/public_html/wp-content/plugins/wpuiai/includes/class-ui-reference-analyzer.php:157-199`
- Adds `validation` and `metadata`
- Enforces per-component required fields

#### Go
- `/home/wpuiai/uiai-engine/internal/reference/analyzer.go:166-213`
- `/home/wpuiai/uiai-engine/internal/routes/reference.go:91-99`

#### Gap
- Go route does not return local-style `validation` / `metadata`
- Go validator only checks count + hero count, not full per-field structure
- Source:
  - local validator: `class-ui-reference-analyzer.php:757-790`
  - Go validator: `analyzer.go:380-394`

#### Verdict
- Mostly ported, validator parity incomplete

---

### 3. extract_tokens

#### Local
- `/home/wpuiai/public_html/wp-content/plugins/wpuiai/includes/class-ui-reference-analyzer.php:202-239`
- Parse requires all token groups:
  - `colors`
  - `typography`
  - `spacing`
  - `shapes`
- Validation checks many required tokens
- Source: `class-ui-reference-analyzer.php:661-683`, `:793-839`

#### Go
- `/home/wpuiai/uiai-engine/internal/reference/analyzer.go:215-241`
- Validation only checks a minimal subset
- Source: `analyzer.go:395-406`

#### Gap
- Core model port exists
- Quality-gate parity is incomplete
- Missing stronger required-token enforcement present in local analyzer

#### Verdict
- Data model ported, validation parity incomplete

---

### 4. extract_spacing

#### Local
- `/home/wpuiai/public_html/wp-content/plugins/wpuiai/includes/class-ui-reference-analyzer.php:247-312`
- If parse fails or spacing is effectively empty:
  - falls back to a full default spacing system
- Sources:
  - parse: `:685-706`
  - default spacing: `:841-879`
  - empty detection: `:882-898`
  - validation: `:900-918`

#### Go
- `/home/wpuiai/uiai-engine/internal/reference/analyzer.go:243-282`
- Only defaults:
  - `BaseUnit = 8`
  - `Scale = [8,16,24,32,48,64,96]`

#### Gap
- Go does **not** implement the local full fallback spacing object:
  - `detected`
  - `vertical_rhythm`
  - `horizontal_layout`
  - `container_padding`
  - `alignment_patterns`
  - `source = default_fallback`
- This is the strongest function-level parity failure

#### Verdict
- NOT fully ported

---

### 5. Full pipeline / operation mapping

#### Local operation names
- `analyze-reference`
- `extract-components`
- `extract-tokens`
- `extract-spacing`
- local switch: `/home/wpuiai/public_html/wp-content/plugins/wpuiai/includes/class-capability-router.php:765-778`

#### Cloud mapping names
- `analyze`
- `extract_components`
- `extract_tokens`
- `extract_spacing`
- mapping: `/home/wpuiai/public_html/wp-content/plugins/wpuiai/includes/class-capability-router.php:705-710`

#### Gap
- operation naming conventions differ between local and cloud paths
- not yet proven all callers normalize these consistently

#### Verdict
- operation-name parity still needs explicit proof/tests

---

## Removal decision

### NOT safe to remove yet
- old UI Reverse API dependency code/assumptions tied to feature completeness
- any migration cleanup that assumes UI Reverse is fully ported

### Safe to say today
- Bun runtime dependency for the main parent-plugin cloud path is gone
- full feature parity is not yet certified

---

## Required closure work before retirement

1. Implement local-equivalent spacing fallback behavior in Go `ExtractSpacing`
2. Decide and implement validator parity strategy for:
   - analyze_reference
   - extract_components
   - extract_tokens
3. Add explicit operation normalization / compatibility tests for UI Reverse caller names
4. Add end-to-end parity tests comparing local analyzer output contracts vs Go output contracts for all passes
5. Re-run parity audit and only then mark old UI Reverse dependency cleanup safe
