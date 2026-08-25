<script lang="ts">
  import { onMount } from "svelte";
  import "$lib/ui/screen.css";
  import { entitlementFromHost, installEntitlementProjection, type CanonicalEntitlementProjection } from "$lib/contracts/entitlement";
  import { engineClient, engineUrl, type BrowserHealth, type EngineHealth } from "$lib/engine-client";
  import WorkspaceViewTabs from "$lib/ui/WorkspaceViewTabs.svelte";

  const views = [
    { id: "nodes", label: "Nodes" }, { id: "uiai-engine", label: "UIAI Engine" },
    { id: "focusa-local", label: "Focusa Local" }, { id: "focusa-cloud", label: "Focusa Cloud" },
    { id: "ai-api", label: "AI API" }, { id: "pairing", label: "Pairing & Devices" },
    { id: "capacity", label: "Capacity" }, { id: "sync", label: "Sync" },
    { id: "updates", label: "Updates & Compatibility" },
  ] as const;
  let activeView = "nodes";
  let health: EngineHealth | null = null;
  let browser: BrowserHealth | null = null;
  let loading = true;
  let error = "";
  let entitlement: CanonicalEntitlementProjection | null = null;

  async function refresh() {
    error = "";
    try { [health, browser] = await Promise.all([engineClient.health(), engineClient.browserHealth()]); }
    catch (cause) { error = cause instanceof Error ? cause.message : "Node health adapters are unavailable."; }
    finally { loading = false; }
  }

  onMount(() => {
    const requested = new URLSearchParams(window.location.search).get("view");
    if (requested && views.some((view) => view.id === requested)) activeView = requested;
    entitlement = entitlementFromHost();
    const onEntitlement = (event: Event) => { entitlement = installEntitlementProjection((event as CustomEvent<unknown>).detail); };
    window.addEventListener("uiai:entitlement-state", onEntitlement);
    void refresh();
    return () => window.removeEventListener("uiai:entitlement-state", onEntitlement);
  });
  $: activeLabel = views.find((view) => view.id === activeView)?.label || "Nodes";
  $: hasEngineData = activeView === "nodes" || activeView === "uiai-engine" || activeView === "capacity";
</script>

<svelte:head><title>Nodes & Services · UIAI Engine Cockpit</title></svelte:head>
<div class="screen nodes-screen">
  <div class="screen-header"><div><p class="screen-kicker">System</p><h1>Nodes & Services</h1><p class="screen-lede">Connected backends, protected workers, pairing, capacity, sync, and compatibility stay explicit by owner.</p></div><button class="screen-button" type="button" onclick={refresh} disabled={loading}>↻ Refresh</button></div>
  <WorkspaceViewTabs label="Nodes and services views" route="/nodes-services" {views} active={activeView} />
  {#if error}<div class="error-banner" role="alert"><strong>Node health degraded.</strong><span>{error}</span></div>{/if}

  {#if hasEngineData}
    <section class="node-grid" aria-label={activeLabel}>
      <article class="screen-card node-card"><div class="node-heading"><span class:online={health?.status === "healthy"} class="node-dot"></span><div><p class="screen-kicker">UIAI Engine</p><h2>{health?.status || (loading ? "checking" : "unavailable")}</h2></div></div><dl><div><dt>Endpoint</dt><dd>{engineUrl()}</dd></div><div><dt>Service</dt><dd>{health?.service || "not returned"}</dd></div><div><dt>Uptime</dt><dd>{health?.uptime ?? "not returned"}</dd></div></dl></article>
      <article class="screen-card node-card"><div class="node-heading"><span class:online={browser?.browser_alive} class="node-dot"></span><div><p class="screen-kicker">Browser runtime</p><h2>{browser?.browser_state || browser?.status || (loading ? "checking" : "unavailable")}</h2></div></div><dl><div><dt>Active pages</dt><dd>{browser?.active_pages ?? 0}</dd></div><div><dt>Capacity</dt><dd>{browser?.max_pages ?? "not returned"}</dd></div><div><dt>Available</dt><dd>{browser?.available_pages ?? "not returned"}</dd></div></dl></article>
      <article class="screen-card node-card"><div class="node-heading"><span class:online={entitlement?.protected_worker.worker_status === "ready"} class="node-dot"></span><div><p class="screen-kicker">Entitlement & protected worker</p><h2>{entitlement?.state || "recovery-only"}</h2></div></div>{#if entitlement}<dl><div><dt>Authority</dt><dd>{entitlement.source}</dd></div><div><dt>Worker</dt><dd>{entitlement.protected_worker.worker_status}</dd></div><div><dt>Capsule</dt><dd>{entitlement.protected_worker.capsule_status}</dd></div><div><dt>Compatibility</dt><dd>{entitlement.protected_worker.compatibility || "not returned"}</dd></div><div><dt>Local artifacts</dt><dd>{entitlement.local_artifacts.access}</dd></div></dl>{:else}<p class="node-empty">No canonical entitlement projection is connected. Execution remains blocked while health, recovery, local artifacts, and Evidence stay available.</p><a class="screen-button" href="/settings">Configure authority</a>{/if}</article>
    </section>
  {:else}
    <section class="empty-screen"><div class="empty-mark">⌘</div><h2>{activeLabel} adapter is not connected</h2><p>{activeView === "pairing" ? "No canonical pairing state was returned. Device trust, repair, and revocation remain owned by the Focusa pairing contracts." : activeView === "updates" ? "The shell exposes signed update posture, but no compatibility matrix adapter is registered for this view." : `No ${activeLabel} contract is connected. The Cockpit does not infer node health, entitlement, protected-worker, sync, or cloud state.`}</p><div class="screen-actions"><a class="screen-button primary" href="/settings">Configure connection</a><a class="screen-button" href="/capabilities">Inspect capability catalog</a></div></section>
  {/if}
</div>
<style>
  .nodes-screen { display: grid; gap: 20px; }
  .node-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
  .node-card { padding: 18px; }
  .node-heading { display: flex; align-items: center; gap: 10px; }
  .node-heading h2 { margin: 2px 0 0; font-size: 1.05rem; }
  .node-dot { width: 10px; height: 10px; border-radius: 50%; background: var(--color-text-muted); }
  .node-dot.online { background: var(--color-success); box-shadow: 0 0 0 4px color-mix(in srgb, var(--color-success) 15%, transparent); }
  dl { display: grid; gap: 9px; margin: 16px 0 0; }
  dl div { display: flex; justify-content: space-between; gap: 12px; border-top: 1px solid var(--color-border); padding-top: 9px; }
  dt { color: var(--color-text-muted); } dd { margin: 0; text-align: right; overflow-wrap: anywhere; }
  .error-banner { display: flex; gap: 10px; padding: 12px 14px; border: 1px solid var(--color-border); border-radius: 9px; color: var(--color-error); }
  .error-banner span { color: var(--color-text-muted); }
  .node-empty { color: var(--color-text-muted); line-height: 1.45; }
  @media (max-width: 720px) { .node-grid { grid-template-columns: 1fr; } }
</style>
