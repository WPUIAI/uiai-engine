<script lang="ts">
  import { onMount } from "svelte";
  import "$lib/ui/screen.css";
  import { engineClient, savedScope, type EngineHealth } from "$lib/engine-client";
  import WorkspaceViewTabs from "$lib/ui/WorkspaceViewTabs.svelte";

  const views = [
    { id: "current", label: "Current Workpoint" }, { id: "recent", label: "Recent" },
    { id: "needs-capture", label: "Needs capture" }, { id: "needs-review", label: "Needs review" },
    { id: "verified", label: "Verified" }, { id: "provisional", label: "Provisional / Surrogate" },
    { id: "public-safe", label: "Public-safe" }, { id: "receipts", label: "Receipts" },
    { id: "reports", label: "Reports" },
  ] as const;

  let health: EngineHealth | null = null;
  let error = "";
  let loading = true;
  let activeView = "current";
  let scope: ReturnType<typeof savedScope> = {};

  onMount(async () => {
    scope = savedScope();
    const requested = new URLSearchParams(window.location.search).get("view");
    if (requested && views.some((view) => view.id === requested)) activeView = requested;
    try { health = await engineClient.health(); }
    catch (cause) { error = cause instanceof Error ? cause.message : "The engine could not be reached."; }
    finally { loading = false; }
  });

  $: activeLabel = views.find((view) => view.id === activeView)?.label || "Current Workpoint";
</script>

<svelte:head><title>Evidence · UIAI Engine Cockpit</title></svelte:head>
<div class="screen evidence-screen">
  <div class="screen-header"><div><p class="screen-kicker">Prove</p><h1>Evidence</h1><p class="screen-lede">Evidence, receipts, and Review Reports remain governed Workpoint outputs.</p></div><span class:success={health?.status === "healthy"} class="badge">{loading ? "Checking engine" : health?.status || "Unavailable"}</span></div>
  <WorkspaceViewTabs label="Evidence saved views" route="/evidence" {views} active={activeView} />
  {#if error}<div class="error-banner" role="alert"><strong>Engine unavailable.</strong><span>{error}</span></div>{/if}

  <section class="empty-screen">
    <div class="empty-mark">{activeView === "reports" ? "▤" : "◇"}</div>
    <h2>No {activeLabel.toLowerCase()} evidence is in scope</h2>
    {#if scope.project_root && scope.continuity_id}
      <p>Scope is connected, but no adapter returned records for this saved view. The Cockpit does not infer evidence, receipts, review state, or Reports.</p>
      <div class="screen-actions"><a class="screen-button primary" href="/settings?section=scope">Inspect Workpoint scope</a><a class="screen-button" href="/runs">Open live runs</a></div>
    {:else}
      <p>Connect a project and Workpoint before governed Focusa evidence can appear. Review Reports remain Evidence work objects, not a separate workspace.</p>
      <div class="screen-actions"><a class="screen-button primary" href="/settings?section=scope">Review connection</a><a class="screen-button" href="/runs">Open live runs</a></div>
    {/if}
  </section>

  <section class="screen-card pad evidence-contract"><p class="screen-kicker">Evidence contract</p><h2>What appears here</h2><div class="contract-list"><span><strong>Stable handle</strong> — an evidence ref, artifact ref, receipt ref, or Report ref returned by its owner.</span><span><strong>Target</strong> — the browser object, document, session, or test it proves.</span><span><strong>Scope</strong> — project, Workpoint, continuity, and authority context.</span><span><strong>Result</strong> — bounded verification output, never raw transcript history.</span></div></section>
</div>
<style>
  .evidence-screen { display: grid; gap: 20px; }
  .error-banner { display: flex; align-items: center; gap: 10px; padding: 12px 14px; border: 1px solid color-mix(in srgb, var(--color-error) 25%, var(--color-border)); border-radius: 9px; color: var(--color-error); background: color-mix(in srgb, var(--color-error) 7%, transparent); font-size: 12px; }
  .error-banner span { color: var(--color-text-muted); }
  .evidence-contract h2 { margin: 0; font-size: 17px; }
  .contract-list { display: grid; margin-top: 14px; border-top: 1px solid var(--color-border); }
  .contract-list span { padding: 12px 0; border-bottom: 1px solid var(--color-border); color: var(--color-text-muted); font-size: 13px; }
  .contract-list strong { color: var(--color-text); }
</style>
