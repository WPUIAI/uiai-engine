<script lang="ts">
  import { onMount } from "svelte";
  import "$lib/ui/screen.css";
  import { phase0Cards } from "$lib/cards/phase0-card-manifest";
  import { entitlementFromHost, installEntitlementProjection, type CanonicalEntitlementProjection } from "$lib/contracts/entitlement";
  import { phase0CardPlacements } from "$lib/cards/phase0-card-placement";
  import { buildCapabilityCatalog, filterCapabilityCatalog } from "$lib/cards/capability-catalog";

  const catalog = buildCapabilityCatalog(phase0Cards, phase0CardPlacements);
  const unique = (values: string[]) => [...new Set(values.filter(Boolean))].sort();
  const options = {
    workspace: unique(catalog.flatMap((entry) => entry.workspaces)), status: unique(catalog.map((entry) => entry.status)),
    source: unique(catalog.flatMap((entry) => entry.source_planes)), sideEffect: unique(catalog.flatMap((entry) => entry.side_effects)),
    scope: unique(catalog.flatMap((entry) => entry.required_scopes)), locality: unique(catalog.map((entry) => entry.locality)),
    license: unique(catalog.map((entry) => entry.license)), experimental: unique(catalog.map((entry) => String(entry.experimental))),
    artifact: unique(catalog.flatMap((entry) => entry.artifact_types)),
  };
  let query = "";
  let workspace = ""; let status = ""; let sourcePlane = ""; let sideEffect = ""; let requiredScope = "";
  let locality = ""; let license = ""; let experimental = ""; let artifactType = "";
  let entitlement: CanonicalEntitlementProjection | null = null;

  onMount(() => {
    const params = new URLSearchParams(window.location.search);
    query = params.get("capability") || params.get("object") || "";
    entitlement = entitlementFromHost();
    const onEntitlement = (event: Event) => { entitlement = installEntitlementProjection((event as CustomEvent<unknown>).detail); };
    window.addEventListener("uiai:entitlement-state", onEntitlement);
    return () => window.removeEventListener("uiai:entitlement-state", onEntitlement);
  });

  const entitlementFor = (capabilityId: string) => entitlement?.capabilities.find((feature) => feature.capability_id === capabilityId);

  $: filtered = filterCapabilityCatalog(catalog, {
    query, workspace, status, source_plane: sourcePlane, side_effect: sideEffect,
    required_scope: requiredScope, locality, license, experimental, artifact_type: artifactType,
  });
</script>

<svelte:head><title>Capabilities · UIAI Engine Cockpit</title></svelte:head>
<div class="screen capability-screen">
  <div class="screen-header"><div><p class="screen-kicker">System</p><h1>Capabilities</h1><p class="screen-lede">A read-only catalog projected from registered card contracts, not a list of implied grants.</p></div><span class="badge">{filtered.length} / {catalog.length} registered</span></div>

  <section class="screen-card filter-card" aria-labelledby="capability-filter-heading">
    <div><p class="screen-kicker">Catalog filters</p><h2 id="capability-filter-heading">Find by task and posture</h2></div>
    <label class="search-filter"><span>Task or capability</span><input bind:value={query} type="search" placeholder="Search capability, card, or task label" /></label>
    <div class="filter-grid">
      <label><span>Workspace</span><select bind:value={workspace}><option value="">All</option>{#each options.workspace as value}<option>{value}</option>{/each}</select></label>
      <label><span>Status</span><select bind:value={status}><option value="">All</option>{#each options.status as value}<option>{value}</option>{/each}</select></label>
      <label><span>Source plane</span><select bind:value={sourcePlane}><option value="">All</option>{#each options.source as value}<option>{value}</option>{/each}</select></label>
      <label><span>Side effect</span><select bind:value={sideEffect}><option value="">All</option>{#each options.sideEffect as value}<option>{value}</option>{/each}</select></label>
      <label><span>Required scope</span><select bind:value={requiredScope}><option value="">All</option>{#each options.scope as value}<option>{value}</option>{/each}</select></label>
      <label><span>Local / cloud</span><select bind:value={locality}><option value="">All</option>{#each options.locality as value}<option>{value}</option>{/each}</select></label>
      <label><span>License</span><select bind:value={license}><option value="">All</option>{#each options.license as value}<option>{value}</option>{/each}</select></label>
      <label><span>Experimental</span><select bind:value={experimental}><option value="">All</option>{#each options.experimental as value}<option>{value}</option>{/each}</select></label>
      <label><span>Artifact type</span><select bind:value={artifactType}><option value="">All</option>{#each options.artifact as value}<option>{value}</option>{/each}</select></label>
    </div>
  </section>

  {#if filtered.length}
    <section class="catalog-list" aria-label="Registered capabilities">
      {#each filtered as capability}
        <article class="screen-card capability-row">
          <div class="capability-main"><strong>{capability.capability_id}</strong><span>{capability.labels.join(" · ")}</span><small>Cards: {capability.card_ids.join(", ")}</small></div>
          <div class="tag-list"><span>{entitlementFor(capability.capability_id)?.status || capability.status}</span><span>{capability.locality}</span>{#each capability.side_effects as effect}<span>{effect}</span>{/each}{#each capability.required_scopes as scope}<span>{scope} scope</span>{/each}</div>
          <dl><div><dt>Workspace</dt><dd>{capability.workspaces.join(", ") || "not placed"}</dd></div><div><dt>Source</dt><dd>{capability.source_planes.join(", ")}</dd></div><div><dt>License</dt><dd>{entitlement?.state || capability.license}</dd></div><div><dt>Remaining</dt><dd>{entitlementFor(capability.capability_id)?.remaining ?? "not returned"}</dd></div><div><dt>Protected worker</dt><dd>{entitlement?.protected_worker.worker_status || "not connected"}</dd></div><div><dt>Capsule</dt><dd>{entitlement?.protected_worker.capsule_status || "not connected"}</dd></div></dl>
        </article>
      {/each}
    </section>
  {:else}
    <section class="empty-screen"><div class="empty-mark">✦</div><h2>No capability matches these filters</h2><p>Clear filters to return to the registered catalog. Absence never implies a grant, license, or hidden executable action.</p><button class="screen-button" type="button" onclick={() => { query = ""; workspace = ""; status = ""; sourcePlane = ""; sideEffect = ""; requiredScope = ""; locality = ""; license = ""; experimental = ""; artifactType = ""; }}>Clear filters</button></section>
  {/if}
</div>
<style>
  .capability-screen { display: grid; gap: 20px; }
  .filter-card { display: grid; gap: 14px; padding: 18px; }
  .filter-card h2 { margin: 2px 0 0; font-size: 1rem; }
  .search-filter, .filter-grid label { display: grid; gap: 5px; color: var(--color-text-muted); font-size: 0.72rem; font-weight: 650; }
  input, select { box-sizing: border-box; width: 100%; padding: 8px 9px; border: 1px solid var(--color-border); border-radius: 8px; background: var(--color-surface); color: var(--color-text); font: inherit; }
  input:focus, select:focus { outline: 3px solid var(--color-focus-ring); border-color: var(--color-accent); }
  .filter-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 10px; }
  .catalog-list { display: grid; gap: 10px; }
  .capability-row { display: grid; grid-template-columns: minmax(220px, 1.4fr) minmax(180px, 1fr) minmax(200px, 1fr); gap: 16px; padding: 16px; }
  .capability-main { display: grid; gap: 4px; min-width: 0; }
  .capability-main strong { overflow-wrap: anywhere; }
  .capability-main span, .capability-main small { color: var(--color-text-muted); }
  .tag-list { display: flex; flex-wrap: wrap; align-content: flex-start; gap: 5px; }
  .tag-list span { padding: 3px 7px; border-radius: 999px; background: var(--color-surface); color: var(--color-text-muted); font-size: 0.68rem; }
  dl { display: grid; gap: 5px; margin: 0; font-size: 0.72rem; } dl div { display: flex; justify-content: space-between; gap: 8px; } dt { color: var(--color-text-muted); } dd { margin: 0; text-align: right; overflow-wrap: anywhere; }
  @media (max-width: 900px) { .filter-grid { grid-template-columns: repeat(2, 1fr); } .capability-row { grid-template-columns: 1fr; } }
  @media (max-width: 600px) { .filter-grid { grid-template-columns: 1fr; } }
</style>
