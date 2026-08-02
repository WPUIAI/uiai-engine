<script lang="ts">
  export let eyebrow = "Workspace";
  export let title: string;
  export let description: string;
  export let state: "planned" | "guarded" | "current" = "planned";
  export let nextAction = "";
  export let route = "";
  export let subsections: string[] = [];
</script>

<div class="screen"><div class="screen-header"><div><p class="screen-kicker">{eyebrow}</p><h1>{title}</h1><p class="screen-lede">{description}</p></div><span class="badge" data-state={state}>{state === "current" ? "Current route" : state === "guarded" ? "Capability guarded" : "Adapter pending"}</span></div>{#if subsections.length}<nav class="workspace-secondary" aria-label={`${title} secondary navigation`}>{#each subsections as subsection}{#if state === "current"}<a href={`${route}?view=${encodeURIComponent(subsection.toLowerCase().replaceAll(" ", "-"))}`}>{subsection}</a>{:else}<span>{subsection}<small>guarded</small></span>{/if}{/each}</nav>{/if}<section class="empty-screen"><div class="empty-mark">⌁</div><h2>{state === "current" ? "Surface ready for integration" : "No surface data returned"}</h2><p>{nextAction || "This route is registered by the Cockpit manifest, but it will remain truthful until its owning adapter returns data."}</p></section></div>
<style>.workspace-secondary { display: flex; flex-wrap: wrap; gap: 7px; margin: -12px auto 28px; max-width: 760px; } .workspace-secondary a, .workspace-secondary span { display: inline-flex; align-items: center; gap: 6px; padding: 7px 10px; border: 1px solid var(--color-border); border-radius: var(--radius-button); color: var(--color-text-muted); background: var(--color-surface); font-size: 11px; text-decoration: none; } .workspace-secondary a:hover { color: var(--color-text); border-color: var(--color-accent); } .workspace-secondary small { color: var(--color-warn); font-size: 9px; text-transform: uppercase; }</style>
