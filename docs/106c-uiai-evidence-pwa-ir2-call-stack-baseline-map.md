> Parent authority: https://github.com/WPUIAI/uiai-engine/issues/106
> Canonical source: https://github.com/WPUIAI/uiai-engine/issues/106#issuecomment-5462558477

## IR2 baseline / exact current call-stack map

Read-only baseline: `653eb32d6f59981cf3bddb6304d369a1e96b7baa` on `main` (`go 1.25.5`). Target route/tool/FPV files are clean. The worktree has unrelated `.beads` and untracked Cockpit/contracts/docs/release material; implementation must not touch, normalize, stage, or delete it. The currently untracked `contracts/` tree cannot safely become this feature's schema home until ownership is reconciled.

### Existing screenshot evidence path

```text
Pi/MCP browser_screenshot
→ mcp/browser-session-mcp.mjs toolsCall(browser_screenshot)
→ POST /api/session/{session_id}/screenshot
→ internal/routes/session.go MountSessionReal screenshot handler
→ writeScreenshotOutput
→ saveScreenshotArtifact
→ {config.vision.share_dir}/session-screenshots/{session}-{timestamp}.{png|jpeg}
→ GET /api/session/{session_id}/screenshot/artifacts/{name}
```

Current output exposes `artifact_path`, `artifact_url`, format, dimensions, size, source URL/title, duration, and output mode. Path traversal has a basename guard and tests in `internal/routes/session_screenshot_output_test.go`. Focusa metadata/evidence behavior is covered by `internal/routes/artifact_evidence_test.go`.

> **Supersession note (issue #196):** The preceding paragraph records the IR2 baseline at commit `653eb32`; it is not current consumer guidance. Current successful artifact delivery requires a ready `uiai.epwa_delivery.v1`, complete evidence scope, durable HTTPS viewer, and portable package. Raw path/base64 routes are retired or fail closed.

Gaps at the recorded baseline: timestamp identity was not content-addressed; output exposed a raw local path; assets were session-namespaced; there was no immutable bundle, media lineage, manifest, redaction/access/retention contract, video/interaction segment, or automatic pre-cleanup publication.

### Existing persisted share path

```text
POST /api/share or /api/share/session
→ internal/routes/share.go MountShareReal
→ shareEntry + sync.Map shareStore
→ persistShare
→ {config.vision.share_dir}/persisted-shares/{id}.json
→ package init reloads non-expired entries after restart
→ GET /v/{token}
→ internal/routes/stubs.go HandleShareViewer
```

Current persistence and restart reload are useful primitives. `shareEntry.Data map[string]any` is not a sufficient canonical evidence schema. Current `/v/{token}` is a live iframe share viewer, not an immutable proof renderer; it depends on the source URL remaining usable and does not copy referenced assets into a bundle.

Security finding: `HandleShareViewer` currently concatenates stored title/URL into HTML without template escaping. The Evidence PWA must use escaped templates/strict CSP and the legacy viewer requires separate hardening; this cannot be copied into the new renderer.

### Existing public/tool/discovery path

```text
HTTP routes: internal/server/server.go
Tool registry/search/agent-card/graph/OpenAI/MCP schemas: internal/routes/tools.go
MCP HTTP bridge: mcp/browser-session-mcp.mjs
Current viewer: GET /v/{token}
FPV visual/PWA authority: web/fpv/{index.html,fpv.css,fpv.js}
```

Any implementation must add one capability through all discovery/bridge surfaces, not a route-only feature. The shell should reuse FPV tokens/layout conventions without importing live control semantics.

### Existing Focusa/UIAI handoff

- Session/screenshots carry Focusa-ready evidence metadata and optional `focusa_scope`.
- Diagnostics/research packets live under `internal/focusapacket` and UIAI route responses.
- Focusa remains the authority for Project/Workstream/Workpoint scope acceptance, canonical Evidence linkage, Acceptance Atom verification, Completion Decision, and settlement.
- UIAI artifact publication must preserve opaque Focusa refs and authority posture; it must not call itself canonical when Focusa capture/link failed.

### Existing Cockpit/Desktop consumers

- Cockpit contract already defines `uaiengine.cockpit.artifact_ref.v1` with artifact ref, kind, title, media type, hash, byte size, source/parent refs, scope/node, safe content URL, redaction state, verification class, creation time, and retention policy.
- Cockpit adapter result already exposes evidence ref, receipt ref, and redaction state.
- Focusa Desktop currently has `ProofPeek`, `WorkpointPeek`, `MissionCanvasView`, `ContextCognitionPeek`, and API/store types that can consume a generated artifact projection; no implementation should create a desktop-only truth DTO.

### Required call-stack layers for the new slice

```text
entry
  automatic evidence-result hook + explicit publish API/tool
handlers
  publish / inspect / manifest / asset / PWA / export
services
  scope validation → normalization → redaction → media intake
  → hash/dedup → immutable assembly → policy/access → rendering
adapters
  screenshot/diagnostics/interaction/video/test/release/receipt inputs
storage
  configured share/artifact root, content-addressed immutable bundle,
  atomic manifest write, index/retention metadata, no raw-path output
output
  compact cross-harness artifact projection + relative PWA/manifest refs
  + published|blocked|failed|not_applicable status
consumers
  tool discovery, MCP, Pi, CLI/OpenAPI, Cockpit, Focusa Desktop,
  Completion Verification Case, public-safe share/export
```

### IR2 gaps blocking source mutation

Before IR5, the packet must name and freeze:

1. exact schema owner/IDs and generated-vs-local boundary;
2. exact HTTP paths, tool operation names, side-effect/access/entitlement classes, and error codes;
3. exact storage root/index/atomicity/retention/GC and cross-version migration;
4. media limits/codecs/derivative policy and shell/manifest/transfer/bundle budgets;
5. trusted-proxy origin policy and private/local/public access model;
6. normalization/redaction rules and secret-classifier behavior;
7. completion-envelope compatibility and legacy caller behavior;
8. tests/fixtures, rollback, release proof, and installed-artifact acceptance;
9. ownership resolution for the untracked `contracts/` and `internal/routes/schemas_embed/` trees;
10. safe resolution of the current degraded Focusa project binding before canonical state/evidence writes.

This advances the UIAI producer layer to a verifiable current-baseline map without inventing canonical Focusa contracts or mutating source early.
