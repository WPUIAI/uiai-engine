# UIAI screenshot mode and Focusa session alignment
**November 14, 2026 · implementation contract for UIAI #299, Focusa #672 and both Pi extensions**

## Decision
**Ad hoc is the default browser mode until an owner-authorized session is formalized as a project.** Normal authorized exploratory browser operations, including visual screenshots, must work without constructing a Focusa Project, Workstream, Workset, CallGraph or Workpoint. A useful current objective remains attached even in ad hoc mode. The owner may direct this work, or the agent may initiate it within its existing grants.

**Formal project evidence cannot be bypassed.** Once the owning Focusa session is project-bound, authoritative policy requires its original project context and evidence admission. An agent cannot downgrade it by dropping `focusa_scope`, setting a local `ad_hoc` flag, changing cwd, restarting a browser or using a different tool adapter. Only Focusa's authorized formalization/release mechanism can change that binding.

## Cross-product architecture

- **Focusa session and Focusa Pi extension** (`Startempire-Wire/focusa` issue #672, Spec187, `apps/pi-extension/`): owning session route (`ad_hoc` or `project`), objective, authorized principal, permitted target/effects, durable promotion and immutable attachment lineage. The initial full ad hoc primitive is *not yet verified as implemented*; the missing-frontier = reconcile fix alone is not completion.
- **UIAI Engine browser** (`internal/routes/session.go`, `screenshot.go`, `epwa_delivery.go`): consumes the **effective** Focusa session route and evidence policy, never treats the absence of request metadata as proof that a known project is ad hoc. For an ad hoc authorized session, deliver an ordinary screenshot through a private session-bound preview with bounded lifetime, no public artifact URL, no raw image field in generic JSON output, no made-up formal evidence status. For an active formal project, publish through the existing canonical EPWA path, or return a typed missing-scope error.
- **UIAI Pi extension** (`integrations/pi/uiai-engine.ts`), **UIAI MCP** (`mcp/browser-session-mcp.mjs`) and **Focusa Pi extension** (`apps/pi-extension/`): expose the same mode and next action; forward exact authoritative scope only when it exists. Honor browser session authority after promotion and in every screenshot call; don't create competing session-mode or consent stores.
- **Scope diagnostics:** distinguish no project, a known project with scope missing, project scope incomplete, a revoked grant, EPWA publication failure, and unavailable image transport. Every error should say **what the agent can do next**, with no fabricated identity or permission.

## Required transitions and tests
1. Fresh authorized session: `ad_hoc`; agent can browse, read, navigate and get an image without full project scope. No formal EPWA proof is implied.
2. A declared project gains an authoritative project binding and associated evidence rules. New project-bound captures require full existing scope/EPWA delivery; dropping metadata cannot change the mode.
3. Ad hoc history can be referenced during formalization after authorization, but earlier unsealed screenshots do not retrospectively become sealed/verified evidence.
4. Session close, reload, reconnection, and separate Pi/MCP invocations preserve mode and cannot silently turn a project into an unscoped session.
5. Untrusted sites, cross-origin sharing, private credentials and authenticating sessions keep their existing safety boundary. Do not expose public images or arbitrary unbounded screenshot base64. Existing provider/permission constraints apply even in ad hoc.
6. Regressions across Go API, UIAI Pi adapter, MCP adapter, CLI and Focusa Pi extension. Verify real agent-visible pixels, not only JSON `ok` or a test image URL.

## Increment in this PR
This PR delivers **explicit, typed recovery guidance** when formal EPWA evidence scope is missing, and keeps the existing evidence fail-closed semantics. It **does not yet enable** unscoped ad hoc screenshot pixels. That requires a separate secure image-delivery + canonical Focusa session-binding implementation and installed acceptance. See UIAI #299 and Focusa #672.
