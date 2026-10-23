<script lang="ts">
  import WorkspacePlaceholder from "$lib/ui/WorkspacePlaceholder.svelte";
</script>
<svelte:head><title>Automations · UIAI Engine Cockpit</title></svelte:head>
<WorkspacePlaceholder eyebrow="Create / Automations" title="Automations" description="Recipes, runs, intake, migration, templates, and schedules are registered as a guarded workspace." state="planned" route="/automations" subsections={["Recipes", "Runs", "Intake", "Migration", "Templates"]} nextAction="No automation adapter is connected in the current engine surface; no job or schedule has been invented." />
