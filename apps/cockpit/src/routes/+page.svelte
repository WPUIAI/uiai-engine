<script lang="ts">
  import { onMount } from "svelte";
  import "$lib/ui/screen.css";
  import { engineClient, engineUrl, type BrowserHealth, type BrowserSession, type EngineHealth } from "$lib/engine-client";
  import { parseCockpitWorkpointResume, workpointResumeFromHost, type CockpitWorkpointResume } from "$lib/contracts/workpoint-resume";

  let health: EngineHealth | null = null;
  let browserHealth: BrowserHealth | null = null;
  let sessions: BrowserSession[] = [];
  let resumeState: CockpitWorkpointResume | null = null;
  let loading = true;
  let error = "";

  async function refresh() {
    error = "";
    try {
      [health, browserHealth, sessions] = await Promise.all([
        engineClient.health(),
        engineClient.browserHealth(),
        engineClient.sessions(),
      ]);
    } catch (cause) {
      error = cause instanceof Error ? cause.message : "The engine could not be reached.";
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    resumeState = workpointResumeFromHost();
    const onResumeContract = (event: Event) => {
      try {
        resumeState = parseCockpitWorkpointResume((event as CustomEvent<unknown>).detail);
      } catch {
        resumeState = null;
      }
    };
    window.addEventListener("uiai:workpoint-resume", onResumeContract);
    void refresh();
    const timer = window.setInterval(() => void refresh(), 8000);
    return () => {
      window.removeEventListener("uiai:workpoint-resume", onResumeContract);
      window.clearInterval(timer);
    };
  });

  $: resumable = resumeState?.status === "resumable" ? resumeState : null;
  $: blockedResume = resumeState?.status === "blocked" ? resumeState : null;
  $: continueTitle = resumable
    ? resumable.label
    : sessions.length
      ? "Continue live browser work"
      : "No Workpoint is connected";
  $: continueText = resumable
    ? "Canonical Focusa resume state is available for this Workpoint."
    : sessions.length
      ? `${sessions.length} live session${sessions.length === 1 ? " is" : "s are"} returned by the engine.`
      : "A canonical Workpoint or live session will appear here when its owning adapter supplies real state.";
  $: continueHref = resumable?.target.href || (sessions.length ? "/runs" : "/settings?section=scope");
  $: continueAction = resumable ? "Resume Workpoint" : sessions.length ? "Open Live runs" : "Connect scope";
</script>

<svelte:head><title>Overview · UIAI Engine Cockpit</title></svelte:head>

<div class="screen overview-screen">
  <div class="screen-header">
    <div>
      <p class="screen-kicker">Workspace</p>
      <h1>Overview</h1>
      <p class="screen-lede">Continue current work, inspect live activity, and recover from truthful system state.</p>
    </div>
    <div class="screen-actions">
      <span class:success={health?.status === "healthy"} class="badge">{health ? health.status : "Checking engine"}</span>
      <button class="screen-button" type="button" onclick={refresh} disabled={loading}>↻ Refresh</button>
    </div>
  </div>

  {#if error}
    <div class="error-banner" role="alert">
      <strong>Engine unavailable.</strong>
      <span>{error}</span>
      <button class="data-row-action" type="button" onclick={refresh}>Retry</button>
    </div>
  {/if}

  <section class="continue-card screen-card" aria-labelledby="continue-heading">
    <div class="continue-mark" aria-hidden="true">→</div>
    <div class="continue-copy">
      <p class="screen-kicker">Continue</p>
      <h2 id="continue-heading">{continueTitle}</h2>
      <p>{continueText}</p>
      {#if resumable}
        <dl class="mission-fields" aria-label="Mission Deck Workpoint fields">
          <div><dt>Workpoint</dt><dd>{resumable.workpoint_id}</dd></div>
          <div><dt>Authority</dt><dd>Canonical Focusa resume</dd></div>
          <div><dt>Destination</dt><dd>{resumable.target.workspace_id}</dd></div>
        </dl>
      {/if}
    </div>
    <a class="screen-button primary" href={continueHref}>{continueAction} →</a>
  </section>

  {#if blockedResume}
    <section class="screen-card recovery-card" aria-labelledby="resume-recovery-heading">
      <div>
        <p class="screen-kicker">Resume recovery</p>
        <h2 id="resume-recovery-heading">Workpoint cannot be resumed yet</h2>
        <p>{blockedResume.recovery.message}</p>
      </div>
      <a class="screen-button" href={blockedResume.recovery.href}>{blockedResume.recovery.action_label} →</a>
    </section>
  {/if}

  <section class="screen-section" aria-labelledby="active-now-heading">
    <div class="section-heading">
      <div><p class="screen-kicker">Active now</p><h2 id="active-now-heading">Live engine state</h2></div>
      <a href="/runs">View live runs →</a>
    </div>
    <div class="data-list">
      {#if loading}
        <div class="empty-state compact"><span>◌</span><div><strong>Reading live state</strong><p>Connecting to {engineUrl()}.</p></div></div>
      {:else if sessions.length === 0}
        <div class="empty-state compact"><span>◉</span><div><strong>All clear</strong><p>No browser sessions or running jobs are currently returned by the engine.</p></div></div>
      {:else}
        {#each sessions.slice(0, 4) as session}
          <div class="data-row">
            <span class="session-orb">◉</span>
            <div class="data-row-copy"><strong>{session.title || "Untitled browser session"}</strong><span>{session.url || "No URL"} · {session.width || "?"}×{session.height || "?"}</span></div>
            <span class="badge">Live</span><a class="data-row-action" href="/runs">Open →</a>
          </div>
        {/each}
      {/if}
    </div>
  </section>

  <section class="screen-section" aria-labelledby="recent-work-heading">
    <div class="section-heading"><div><p class="screen-kicker">Recent work</p><h2 id="recent-work-heading">Durable history</h2></div></div>
    <div class="empty-state compact"><span>↺</span><div><strong>No recent work in scope</strong><p>The engine does not expose durable Workpoint history here, so Overview leaves this surface empty instead of inventing records.</p></div></div>
  </section>

  <section class="screen-section" aria-labelledby="system-posture-heading">
    <div class="section-heading">
      <div><p class="screen-kicker">System posture</p><h2 id="system-posture-heading">Quiet status</h2></div>
      <a href="/nodes-services">Inspect services →</a>
    </div>
    <div class="posture-grid">
      <article class="posture-card"><span>Engine</span><strong>{health?.status || "unknown"}</strong><small>{health?.service || "UIAI Engine"}</small></article>
      <article class="posture-card"><span>Browser</span><strong>{browserHealth?.browser_state || browserHealth?.status || "unknown"}</strong><small>{browserHealth?.active_pages ?? 0} / {browserHealth?.max_pages ?? "?"} pages</small></article>
      <article class="posture-card"><span>Scope</span><strong>{resumable ? "Workpoint ready" : blockedResume ? "Recovery required" : "Not connected"}</strong><small>{resumable ? resumable.target.workspace_id : "No inferred mission data"}</small></article>
      <article class="posture-card"><span>Contracts</span><strong>Manifest-backed</strong><small><a href="/capabilities">Inspect registered capabilities</a></small></article>
    </div>
  </section>

  <section class="screen-section" aria-labelledby="suggested-actions-heading">
    <div class="section-heading"><div><p class="screen-kicker">Suggested next actions</p><h2 id="suggested-actions-heading">From current state</h2></div></div>
    <div class="suggestion-list">
      {#if blockedResume}
        <a href={blockedResume.recovery.href}><strong>{blockedResume.recovery.action_label}</strong><span>{blockedResume.recovery.message}</span></a>
      {:else if resumable}
        <a href={resumable.target.href}><strong>Resume {resumable.label}</strong><span>Continue from canonical Workpoint state.</span></a>
      {:else}
        <a href="/settings?section=scope"><strong>Connect project scope</strong><span>Resume actions remain hidden until canonical state is available.</span></a>
      {/if}
      <a href="/live"><strong>Inspect Live</strong><span>Review browser sessions and active UIAI work.</span></a>
      <a href="/evidence"><strong>Review Evidence</strong><span>Open saved proof and Workpoint-linked artifacts.</span></a>
    </div>
  </section>
</div>

<style>
  .overview-screen { display: grid; gap: 24px; }
  .continue-card, .recovery-card { display: flex; align-items: center; gap: 16px; padding: 20px; }
  .continue-mark { display: grid; place-items: center; width: 40px; height: 40px; flex: 0 0 auto; border-radius: 11px; background: color-mix(in srgb, var(--color-accent) 14%, var(--color-surface-elevated)); color: var(--color-accent); font-size: 1.2rem; }
  .continue-copy { flex: 1; min-width: 0; }
  .continue-copy h2, .recovery-card h2 { margin: 2px 0 5px; font-size: 1.08rem; }
  .continue-copy p, .recovery-card p { margin: 0; color: var(--color-text-muted); }
  .mission-fields { display: flex; flex-wrap: wrap; gap: 8px 18px; margin: 12px 0 0; }
  .mission-fields div { display: grid; gap: 2px; }
  .mission-fields dt { color: var(--color-text-muted); font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.06em; }
  .mission-fields dd { margin: 0; font-size: 0.78rem; overflow-wrap: anywhere; }
  .recovery-card { justify-content: space-between; border-color: var(--warning-line, #f2c94c); }
  .posture-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; }
  .posture-card { display: grid; gap: 4px; padding: 13px; border: 1px solid var(--color-border); border-radius: 10px; background: var(--color-surface-elevated); }
  .posture-card span, .posture-card small { color: var(--color-text-muted); }
  .posture-card a { color: inherit; }
  .suggestion-list { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 10px; }
  .suggestion-list a { display: grid; gap: 5px; padding: 13px; border: 1px solid var(--color-border); border-radius: 10px; background: var(--color-surface-elevated); color: inherit; text-decoration: none; }
  .suggestion-list a:hover { border-color: var(--color-border-strong); background: color-mix(in srgb, var(--color-surface-elevated) 88%, var(--color-accent)); }
  .suggestion-list span { color: var(--color-text-muted); font-size: 0.8rem; }
  .suggestion-list a:focus-visible, .posture-card a:focus-visible { outline: 3px solid var(--focus-ring, #4f46e5); outline-offset: 2px; }
  .session-orb { color: var(--success, #12b76a); }
  @media (max-width: 900px) { .posture-grid { grid-template-columns: repeat(2, 1fr); } .suggestion-list { grid-template-columns: 1fr; } }
  @media (max-width: 640px) { .continue-card, .recovery-card { align-items: flex-start; flex-wrap: wrap; } .posture-grid { grid-template-columns: 1fr; } }
</style>
