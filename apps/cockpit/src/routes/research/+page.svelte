<script lang="ts">
  import WorkspacePlaceholder from "$lib/ui/WorkspacePlaceholder.svelte";
</script>
<svelte:head><title>Research · UIAI Engine Cockpit</title></svelte:head>
<WorkspacePlaceholder eyebrow="Work / Research" title="Research" description="Search, captures, collections, and packets are registered in the Cockpit manifest." state="current" route="/research" subsections={["Search", "Captures", "Collections", "Packets"]} nextAction="Use Documents for the current provider-neutral search and Source-to-Markdown capture surface." />
<p class="route-link"><a href="/documents">Open current research capture →</a></p>
<style>.route-link { max-width: 520px; margin: -28px auto 0; text-align: center; font-size: 12px; } .route-link a { color: var(--color-accent); }</style>
