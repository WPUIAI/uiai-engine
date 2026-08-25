<script lang="ts">
  import { onMount } from "svelte";
  import "$lib/ui/screen.css";
  import { engineClient, type BrowserSession } from "$lib/engine-client";
  import WorkspaceViewTabs from "$lib/ui/WorkspaceViewTabs.svelte";

  const views = [
    { id: "now", label: "Now" }, { id: "approvals", label: "Approvals" },
    { id: "history", label: "History" }, { id: "jobs", label: "Jobs" },
    { id: "notifications", label: "Notifications" }, { id: "audit", label: "Audit" },
  ] as const;
  let activeView = "now";
  let sessions: BrowserSession[] = [];
  let loading = true;
  let error = "";

  onMount(async () => {
    const requested = new URLSearchParams(window.location.search).get("view");
    if (requested && views.some((view) => view.id === requested)) activeView = requested;
    try { sessions = await engineClient.sessions(); }
    catch (cause) { error = cause instanceof Error ? cause.message : "Activity adapters are unavailable."; }
    finally { loading = false; }
  });
  $: activeLabel = views.find((view) => view.id === activeView)?.label || "Now";
</script>

<svelte:head><title>Activity · UIAI Engine Cockpit</title></svelte:head>
<div class="screen activity-screen">
  <div class="screen-header"><div><p class="screen-kicker">Prove</p><h1>Activity</h1><p class="screen-lede">Now, approvals, history, jobs, notifications, and audit signals share one operator-attention surface.</p></div><span class="badge">{loading ? "Checking activity" : `${sessions.length} live`}</span></div>
  <WorkspaceViewTabs label="Activity segments" route="/activity" {views} active={activeView} />
  {#if error}<div class="error-banner" role="alert"><strong>Activity degraded.</strong><span>{error}</span></div>{/if}

  {#if activeView === "now" && sessions.length}
    <section class="screen-section" aria-labelledby="activity-now-heading">
      <div class="section-heading"><div><p class="screen-kicker">Now</p><h2 id="activity-now-heading">Live browser work</h2></div><a href="/runs">Open Live →</a></div>
      <div class="data-list">{#each sessions as session}<div class="data-row"><span class="session-orb">◉</span><div class="data-row-copy"><strong>{session.title || "Untitled browser session"}</strong><span>{session.url || "No URL"}</span></div><span class="badge">Live</span><a class="data-row-action" href="/runs">Inspect →</a></div>{/each}</div>
    </section>
  {:else}
    <section class="empty-screen"><div class="empty-mark">≋</div><h2>No {activeLabel.toLowerCase()} signals</h2><p>{activeView === "approvals" ? "No owning adapter returned a pending approval. The Cockpit never implies consent or entitlement from absence." : activeView === "audit" ? "No bounded audit records were returned. Raw event streams are not projected into ordinary navigation." : `No ${activeLabel.toLowerCase()} records were returned by the registered activity adapters.`}</p><div class="screen-actions"><a class="screen-button" href="/nodes-services">Inspect services</a><a class="screen-button" href="/evidence">Review evidence</a></div></section>
  {/if}
</div>
<style>
  .activity-screen { display: grid; gap: 20px; }
  .error-banner { display: flex; gap: 10px; padding: 12px 14px; border: 1px solid var(--color-border); border-radius: 9px; color: var(--color-error); }
  .error-banner span { color: var(--color-text-muted); }
  .session-orb { color: var(--color-success); }
</style>
