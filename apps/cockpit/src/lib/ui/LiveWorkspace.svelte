<script lang="ts">
  import { onMount } from "svelte";
  import "$lib/ui/screen.css";
  import { attachFpvSession, revokeFpvSession, type FpvLiveAttachment } from "$lib/adapters/fpv-live-adapter";
  import { engineClient, engineUrl, savedScope, type BrowserHealth, type BrowserSession, type EngineHealth, type ScreenshotResult } from "$lib/engine-client";
  import { requireScopedMutation } from "$lib/navigation/scope-guard";

  let sessions: BrowserSession[] = [];
  let health: EngineHealth | null = null;
  let browserHealth: BrowserHealth | null = null;
  let loading = true;
  let creating = false;
  let error = "";
  let url = "";
  let selected: BrowserSession | null = null;
  let screenshot: ScreenshotResult | null = null;
  let navigationUrl = "";
  let diagnostics: Record<string, unknown> | null = null;
  let requestedSessionAttached = false;
  let fpvAttachment: FpvLiveAttachment | null = null;
  let fpvLoading = false;
  let fpvError = "";

  async function load() {
    loading = true;
    error = "";
    try {
      const [nextHealth, nextBrowserHealth, nextSessions] = await Promise.all([engineClient.health(), engineClient.browserHealth(), engineClient.sessions()]);
      health = nextHealth;
      browserHealth = nextBrowserHealth;
      sessions = nextSessions;
      const requestedSessionId = new URL(window.location.href).searchParams.get("session") || window.localStorage.getItem("uiai.cockpit.requested_session_id");
      if (requestedSessionId && !requestedSessionAttached) {
        const requested = sessions.find((session) => session.id === requestedSessionId);
        if (requested) {
          requestedSessionAttached = true;
          selected = requested;
          const [shot, detail] = await Promise.all([engineClient.screenshot(requested.id), engineClient.diagnostics(requested.id), attachLiveSession(requested, false)]);
          screenshot = shot;
          diagnostics = detail;
        } else {
          error = `Requested session ${requestedSessionId} is not available from the connected engine.`;
        }
      }
    } catch (cause) {
      error = cause instanceof Error ? cause.message : "The engine could not be reached.";
    } finally {
      loading = false;
    }
  }

  async function createSession() {
    if (!url.trim()) return;
    creating = true;
    error = "";
    try {
      requireScopedMutation(savedScope(), "Open browser session");
      const result = await engineClient.openSession(url.trim());
      sessions = [result.session, ...sessions.filter((session) => session.id !== result.session.id)];
      selected = result.session;
      screenshot = { screenshot: result.screenshot, format: "jpeg", url: result.session.url, title: result.session.title };
      url = "";
    } catch (cause) {
      error = cause instanceof Error ? cause.message : "The browser session could not be opened.";
    } finally {
      creating = false;
    }
  }

  async function attachLiveSession(session: BrowserSession, controls = false) {
    fpvLoading = true;
    fpvError = "";
    try {
      const previous = fpvAttachment;
      const next = await attachFpvSession(session.id, controls);
      fpvAttachment = next;
      if (previous?.token !== next.token) void revokeFpvSession(previous).catch(() => undefined);
    } catch (cause) {
      fpvError = cause instanceof Error ? cause.message : "The live FPV attachment could not be created.";
    } finally {
      fpvLoading = false;
    }
  }

  async function refreshSession(session: BrowserSession) {
    selected = session;
    error = "";
    try { const [shot, detail] = await Promise.all([engineClient.screenshot(session.id), engineClient.diagnostics(session.id), attachLiveSession(session, false)]); screenshot = shot; diagnostics = detail; } catch (cause) { error = cause instanceof Error ? cause.message : "The session inspection request failed."; }
  }

  async function setFpvControls(controls: boolean) {
    if (!selected) return;
    try {
      if (controls) requireScopedMutation(savedScope(), "Enable audited FPV controls");
      await attachLiveSession(selected, controls);
    } catch (cause) {
      fpvError = cause instanceof Error ? cause.message : "FPV control authority is unavailable.";
    }
  }

  async function navigateSession() {
    if (!selected || !navigationUrl.trim()) return;
    try { requireScopedMutation(savedScope(), "Navigate browser session"); screenshot = await engineClient.navigate(selected.id, navigationUrl.trim()); navigationUrl = ""; await load(); } catch (cause) { error = cause instanceof Error ? cause.message : "Navigation failed."; }
  }

  async function closeSession(session: BrowserSession) {
    try { requireScopedMutation(savedScope(), "Close browser session"); await engineClient.closeSession(session.id); sessions = sessions.filter((item) => item.id !== session.id); if (selected?.id === session.id) { void revokeFpvSession(fpvAttachment).catch(() => undefined); fpvAttachment = null; selected = null; screenshot = null; diagnostics = null; } } catch (cause) { error = cause instanceof Error ? cause.message : "The session could not be closed."; }
  }

  onMount(() => { void load(); const timer = window.setInterval(() => void load(), 5000); return () => { window.clearInterval(timer); void revokeFpvSession(fpvAttachment).catch(() => undefined); }; });
</script>

<svelte:head><title>Live runs · UIAI Engine Cockpit</title></svelte:head>
<div class="screen">
  <div class="screen-header"><div><p class="screen-kicker">Operate</p><h1>Live runs</h1><p class="screen-lede">Real browser sessions from the connected UIAI Engine.</p></div><div class="screen-actions"><span class:success={health?.status === "healthy"} class="badge">{health ? health.status : "Checking engine"}</span><button class="screen-button primary" type="button" onclick={load} disabled={loading}>↻ Refresh</button></div></div>

  {#if error}<div class="error-banner" role="alert"><strong>Engine request unavailable.</strong><span>{error}</span><button class="data-row-action" type="button" onclick={load}>Retry</button></div>{/if}
  <div class="screen-grid" style="margin-bottom: 26px"><article class="screen-card screen-stat"><strong>{sessions.length}</strong><span>sessions returned by engine</span></article><article class="screen-card screen-stat"><strong>{browserHealth?.active_pages ?? "—"}</strong><span>active browser pages</span></article><article class="screen-card screen-stat"><strong>{browserHealth?.max_pages ?? "—"}</strong><span>available page capacity</span></article></div>

  <section class="screen-card session-launcher"><div><p class="screen-kicker">Open a real session</p><h2>Inspect a page in the browser engine</h2><p>Opening a URL creates a persistent session, captures its first screenshot, and keeps the session available for navigation.</p></div><div class="launch-form"><input bind:value={url} aria-label="URL to open" placeholder="https://example.com" onkeydown={(event) => event.key === "Enter" && createSession()} /><button class="screen-button primary" type="button" onclick={createSession} disabled={creating || !url.trim()}>{creating ? "Opening…" : "Open session"}</button></div></section>

  {#if loading && sessions.length === 0}<section class="empty-screen"><div class="empty-mark loading-mark">◌</div><h2>Reading the engine</h2><p>Loading live browser sessions and capacity from {engineUrl()}.</p></section>{:else if sessions.length === 0}<section class="empty-screen"><div class="empty-mark">◉</div><h2>No live sessions</h2><p>The engine is connected but has no browser sessions. Open a URL above to begin real work.</p></section>{:else}<section class="screen-card"><div class="screen-toolbar section-pad"><div><h2>Sessions from the engine</h2><p>These rows are server state, not local placeholders.</p></div><span class="badge success">● {browserHealth?.browser_state ?? "Connected"}</span></div><div class="data-list">{#each sessions as session}<div class:active-row={selected?.id === session.id} class="data-row"><span class="activity-icon" data-tone="green">◉</span><div class="data-row-main"><strong>{session.title || session.url}</strong><span>{session.id} · {session.url}</span></div><button class="data-row-action" type="button" onclick={() => refreshSession(session)}>Inspect →</button><button class="data-row-action muted-action" type="button" onclick={() => closeSession(session)}>Close</button></div>{/each}</div></section>{/if}

  {#if selected}<aside class="screen-card pad inspector" aria-live="polite"><div class="screen-toolbar"><div><p class="screen-kicker">Inspect session</p><h2>{selected.title || selected.url}</h2><p>{selected.url}</p></div><button class="screen-button" type="button" onclick={() => { void revokeFpvSession(fpvAttachment).catch(() => undefined); fpvAttachment = null; selected = null; diagnostics = null; screenshot = null; }}>Close inspector</button></div><div class="inspector-grid"><span><small>Session</small><strong>{selected.id}</strong></span><span><small>Engine</small><strong>{engineUrl()}</strong></span><span><small>State</small><strong>Live</strong></span></div><div class="inspector-detail"><p class="screen-kicker">Details</p><span>{diagnostics ? `${String(diagnostics.url || selected.url)} · ${String(diagnostics.title || "Untitled")}` : "Capture or inspect to load session diagnostics."}</span></div><div class="fpv-toolbar"><div><p class="screen-kicker">Existing session stream</p><span class:success={Boolean(fpvAttachment)} class="badge">{fpvLoading ? "Connecting" : fpvAttachment ? fpvAttachment.controls ? "Audited control" : "Read-only live" : "Screenshot fallback"}</span></div><div>{#if fpvAttachment}<button class="screen-button" type="button" disabled={fpvLoading} onclick={() => setFpvControls(!fpvAttachment?.controls)}>{fpvAttachment.controls ? "Return to read-only" : "Enable audited controls"}</button>{/if}<button class="screen-button" type="button" disabled={fpvLoading} onclick={() => attachLiveSession(selected!, fpvAttachment?.controls || false)}>Reconnect</button></div></div>{#if fpvError}<div class="fpv-warning" role="status">{fpvError} Screenshot inspection remains available.</div>{/if}{#if fpvAttachment}<iframe class="session-fpv" src={fpvAttachment.viewer_url} title={`Live FPV view of ${selected.title || selected.url}`} sandbox="allow-scripts allow-same-origin" onerror={() => (fpvError = "The FPV stream degraded or became unreachable.")}></iframe>{:else if fpvLoading}<div class="empty-inline">Attaching the existing Engine session stream…</div>{/if}<div class="session-controls"><input bind:value={navigationUrl} aria-label="Navigate session to URL" placeholder="Navigate to a URL" onkeydown={(event) => event.key === "Enter" && navigateSession()} /><button class="screen-button primary" type="button" onclick={navigateSession} disabled={!navigationUrl.trim()}>Navigate</button><button class="screen-button" type="button" onclick={() => refreshSession(selected!)}>Capture screenshot</button></div>{#if !fpvAttachment}{#if screenshot?.screenshot}<img class="session-screenshot" src={`data:image/${screenshot.format || "jpeg"};base64,${screenshot.screenshot}`} alt={`Current browser view of ${screenshot.url || selected.url}`} />{:else}<div class="empty-inline">Capture a screenshot to inspect the current browser surface.</div>{/if}{/if}</aside>{/if}
</div>

<style>
  .session-launcher { display: grid; grid-template-columns: 1fr minmax(330px, .85fr); gap: 26px; align-items: end; padding: 20px; margin-bottom: 18px; border-color: color-mix(in srgb, var(--color-accent) 24%, var(--color-border)); background: linear-gradient(135deg, color-mix(in srgb, var(--color-accent) 7%, var(--color-surface-elevated)), var(--color-surface-elevated)); }
  .session-launcher h2 { margin: 0; font-size: 18px; } .session-launcher p { margin: 7px 0 0; }
  .launch-form, .session-controls { display: flex; gap: 8px; } .launch-form input, .session-controls input { min-width: 0; flex: 1; padding: 10px 11px; border: 1px solid var(--color-border); border-radius: 8px; outline: none; color: var(--color-text); background: var(--color-surface); font: inherit; font-size: 12px; } .launch-form input:focus, .session-controls input:focus { border-color: var(--color-accent); box-shadow: 0 0 0 3px var(--color-focus-ring); }
  .section-pad { padding: 17px 16px 13px; margin: 0; } .inspector-detail { margin: 16px 0; padding-top: 14px; border-top: 1px solid var(--color-border); } .inspector-detail .screen-kicker { margin-bottom: 5px; } .inspector-detail span { color: var(--color-text-muted); font-size: 12px; }
 .section-pad p { margin: 5px 0 0; } .active-row { background: color-mix(in srgb, var(--color-accent) 6%, transparent); } .muted-action { color: var(--color-text-muted); } .error-banner { display: flex; align-items: center; gap: 10px; margin: -12px 0 18px; padding: 12px 14px; border: 1px solid color-mix(in srgb, var(--color-error) 25%, var(--color-border)); border-radius: 9px; color: var(--color-error); background: color-mix(in srgb, var(--color-error) 7%, transparent); font-size: 12px; } .error-banner span { color: var(--color-text-muted); } .error-banner .data-row-action { margin-left: auto; } .empty-inline { padding: 24px; border: 1px dashed var(--color-border); color: var(--color-text-muted); text-align: center; } .session-screenshot { display: block; width: 100%; max-height: 520px; margin-top: 18px; object-fit: contain; border: 1px solid var(--color-border); border-radius: 8px; background: #111; } .fpv-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 14px; margin: 16px 0 10px; padding-top: 14px; border-top: 1px solid var(--color-border); } .fpv-toolbar > div { display: flex; align-items: center; gap: 8px; } .fpv-toolbar .screen-kicker { margin: 0; } .session-fpv { display: block; width: 100%; min-height: 520px; margin-bottom: 12px; border: 1px solid var(--color-border); border-radius: 8px; background: #111; } .fpv-warning { margin-bottom: 10px; padding: 9px 11px; border: 1px solid color-mix(in srgb, var(--color-warn) 30%, var(--color-border)); border-radius: 7px; color: var(--color-warn); background: color-mix(in srgb, var(--color-warn) 7%, transparent); font-size: 11px; } .loading-mark { animation: loading-breathe 1.2s ease-in-out infinite; } @keyframes loading-breathe { 50% { opacity: .35; transform: rotate(90deg); } }
  @media (prefers-reduced-motion: reduce) { .loading-mark { animation: none; } } @media (max-width: 760px) { .session-launcher { grid-template-columns: 1fr; } .launch-form, .session-controls { flex-wrap: wrap; } }
</style>
