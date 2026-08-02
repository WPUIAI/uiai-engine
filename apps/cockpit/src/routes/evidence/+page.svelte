<script lang="ts">
  import { onMount } from "svelte";
  import "$lib/ui/screen.css";
  import { engineClient, type EngineHealth } from "$lib/engine-client";

  let health: EngineHealth | null = null;
  let error = "";
  let loading = true;

  onMount(async () => {
    try { health = await engineClient.health(); } catch (cause) { error = cause instanceof Error ? cause.message : "The engine could not be reached."; } finally { loading = false; }
  });
</script>

<svelte:head><title>Evidence · UIAI Engine Cockpit</title></svelte:head>
<div class="screen"><div class="screen-header"><div><p class="screen-kicker">Prove</p><h1>Evidence</h1><p class="screen-lede">Evidence is captured by Focusa operations and attached to the active Workpoint.</p></div><span class:success={health?.status === "healthy"} class="badge">{loading ? "Checking engine" : health?.status || "Unavailable"}</span></div>
  {#if error}<div class="error-banner" role="alert"><strong>Engine unavailable.</strong><span>{error}</span></div>{/if}
  <section class="empty-screen"><div class="empty-mark">◇</div><h2>No evidence is in the current scope</h2><p>The Cockpit will not fabricate evidence records. Connect a project and Workpoint, then capture proof through the governed Focusa evidence flow.</p><div class="screen-actions"><a class="screen-button primary" href="/settings">Review connection</a><a class="screen-button" href="/runs">Open live runs</a></div></section>
  <section class="screen-card pad evidence-contract"><p class="screen-kicker">Evidence contract</p><h2>What appears here</h2><div class="contract-list"><span><strong>Stable handle</strong> — an evidence ref or artifact ref returned by the operation.</span><span><strong>Target</strong> — the browser object, document, session, or test it proves.</span><span><strong>Scope</strong> — project, Workpoint, continuity, and authority context.</span><span><strong>Result</strong> — bounded verification output, never raw transcript history.</span></div></section>
</div>
<style>.error-banner { display: flex; align-items: center; gap: 10px; margin: -8px 0 18px; padding: 12px 14px; border: 1px solid color-mix(in srgb, var(--color-error) 25%, var(--color-border)); border-radius: 9px; color: var(--color-error); background: color-mix(in srgb, var(--color-error) 7%, transparent); font-size: 12px; } .error-banner span { color: var(--color-text-muted); } .evidence-contract { margin-top: 18px; } .evidence-contract h2 { margin: 0; font-size: 17px; } .contract-list { display: grid; gap: 0; margin-top: 14px; border-top: 1px solid var(--color-border); } .contract-list span { padding: 12px 0; border-bottom: 1px solid var(--color-border); color: var(--color-text-muted); font-size: 13px; } .contract-list strong { color: var(--color-text); }</style>
