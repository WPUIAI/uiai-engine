<script lang="ts">
  import WorkspacePlaceholder from "$lib/ui/WorkspacePlaceholder.svelte";
</script>
<svelte:head><title>Nodes & Services · UIAI Engine Cockpit</title></svelte:head>
<WorkspacePlaceholder eyebrow="System" title="Nodes & Services" description="Connected nodes, protected workers, capacity, pairing, sync, and update posture belong in this system workspace." state="guarded" route="/nodes-services" subsections={["UIAI Engine", "Focusa Local", "Focusa Cloud", "Pairing & Devices", "Capacity", "Updates"]} nextAction="The engine health indicator remains available in the shell; detailed node state requires its owning adapter." />
