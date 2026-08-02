<script lang="ts">
  import WorkspacePlaceholder from "$lib/ui/WorkspacePlaceholder.svelte";
</script>
<svelte:head><title>Activity · UIAI Engine Cockpit</title></svelte:head>
<WorkspacePlaceholder eyebrow="Prove / Activity" title="Activity" description="Now, approvals, and history are the shared activity surface for jobs and operator attention." state="planned" route="/activity" subsections={["Now", "Approvals", "History"]} nextAction="Activity remains quiet until the owning job, approval, or evidence adapter returns a real signal." />
