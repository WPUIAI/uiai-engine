# UIAI Engine OpenClaw tool plugin

Development package for issue #249; source/contract proof is not production activation,
provider authentication or Discord-delivery proof. The installed SDK exposes tool
plugins, not a third-party browser-provider registration hook. `provider: uiai` and
`fallback: none` below are this plugin's configuration, not invented core settings.

## Package and activate

From the repository's `integrations/` directory, `npm pack` packages the real ESM
entry, manifest and shared native/MCP delivery helpers. No additional runtime
package is installed by the plugin; OpenClaw is a peer dependency. Use the host's
supported `openclaw plugins install npm-pack:<absolute-tarball>` route, then inspect
`openclaw plugins list` and `openclaw plugins inspect uiai-engine --runtime --json`.
Installation/reload changes only an explicitly authorized OpenClaw profile.

```json
{
 "browser":{"enabled":false},
 "tools":{"allow":["uiai-engine"],"deny":["browser"]},
 "plugins":{"entries":{"uiai-engine":{"enabled":true,"config":{
  "provider":"uiai","fallback":"none","baseUrl":"http://127.0.0.1:7456"
 }}}}
}
```

Keep other approved communication tools in the host allowlist as needed. The plugin
refuses execution unless the core browser is explicitly disabled and denied. It
never launches a browser, invokes a node/desktop route, or retries failed operations.
Host-wide shell/network restrictions remain the host's responsibility.

Remote engine URLs require HTTPS. Configure `apiKey` or `bearerToken` using an
OpenClaw SecretRef; selected refs resolve through its official SDK for each request.
API-key configuration takes precedence. Loopback needs no unnecessary credential.
Secrets, upstream error bodies, credentials and evidence URLs are not logged or
placed in tool text. Engine content redaction remains a separate required boundary.

## Tools and lifecycle

`uiai_health` reports capacity/health, never authenticated-page or business success.
`uiai_browser` exposes `open`, `navigate`, `read`, `snapshot`, `click`, `type`, `fill`,
`scroll`, `screenshot`, `diagnostics` and `close` through its `action` parameter.
Pass the returned `session_id`/`session.id` and UIAI snapshot refs unchanged. An
OpenClaw agent/session UUID binding prevents another conversation adopting those
sessions. `label` with explicit `reuse:true` reuses only the same original target.
Close bounded QA sessions; abandoned sessions remain subject to UIAI's own TTL.
No second session authority, control lease or authentication store is created.
Explicit `focusa_scope` references may be forwarded on open; they are metadata,
not grants, and the plugin never invents missing Workset/CallGraph identifiers.

## Screenshot boundary

Enable `privateAttachments:true` only for an authorized conversation, and explicitly
allow its HTTPS evidence origin in `evidenceOrigins`. Screenshot responses still
require the canonical ready EPWA contract. Only an allowlisted record's screenshot
asset is fetched, without forwarding engine credentials, without redirects, with a
15-second/8-MiB bound and checked image signature. The result contains an OpenClaw
image block; evidence refs/states remain while URLs are withheld from model text.
This does not bypass blocked evidence publication or guarantee that a channel has
actually sent an attachment. Real Discord delivery and requesting-recipient proof
remain required before #249 may close.

Queued, cancelled or uncertain responses require inspection/reconciliation before
an authorized retry; cancellation of an await is not rollback of browser effects.
An unavailable UIAI endpoint returns a blocker, never fallback success.
