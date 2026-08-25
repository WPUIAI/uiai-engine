<script lang="ts">
  export let name: string;
</script>

<svg class="sidebar-icon" aria-hidden="true" viewBox="0 0 24 24">
  {#if name === "overview"}
    <path d="M4 11 12 4l8 7v8a1 1 0 0 1-1 1h-5v-6h-4v6H5a1 1 0 0 1-1-1Z" />
  {:else if name === "live"}
    <circle cx="12" cy="12" r="3" /><path d="M7.8 7.8a6 6 0 0 0 0 8.4M16.2 7.8a6 6 0 0 1 0 8.4M4.6 4.6a10.5 10.5 0 0 0 0 14.8M19.4 4.6a10.5 10.5 0 0 1 0 14.8" />
  {:else if name === "test_lab"}
    <path d="m9 3 6 0M10 3v5l-5 9a2 2 0 0 0 1.7 3h10.6a2 2 0 0 0 1.7-3l-5-9V3M8 14h8" />
  {:else if name === "documents"}
    <path d="M6 3h8l4 4v14H6zM14 3v5h5M9 12h6M9 16h6" />
  {:else if name === "research"}
    <circle cx="10.5" cy="10.5" r="5.5" /><path d="m15 15 5 5" />
  {:else if name === "studio"}
    <path d="M4 7h16v12H4zM8 7l2-3h4l2 3M12 11v4M10 13h4" />
  {:else if name === "automations"}
    <path d="M12 3v3M12 18v3M3 12h3M18 12h3M5.6 5.6l2.1 2.1M16.3 16.3l2.1 2.1M18.4 5.6l-2.1 2.1M7.7 16.3l-2.1 2.1" /><circle cx="12" cy="12" r="3" />
  {:else if name === "evidence"}
    <path d="m12 3 8 4v5c0 5-3.3 8-8 9-4.7-1-8-4-8-9V7Z" /><path d="m8.5 12 2.2 2.2 4.8-5" />
  {:else if name === "activity"}
    <path d="M4 12h3l2-5 4 10 2-5h5" />
  {:else if name === "nodes_services"}
    <rect x="3" y="4" width="7" height="6" rx="1" /><rect x="14" y="14" width="7" height="6" rx="1" /><path d="M10 7h4a3 3 0 0 1 3 3v4M14 17h-4a3 3 0 0 1-3-3v-4" />
  {:else if name === "capabilities"}
    <path d="M12 3 4 7v10l8 4 8-4V7Z" /><path d="m4 7 8 4 8-4M12 11v10" />
  {:else if name === "settings"}
    <circle cx="12" cy="12" r="3" /><path d="M19 13.5v-3l-2-.7-.7-1.7.9-1.9-2.1-2.1-1.9.9-1.7-.7L10.5 2h-3l-.7 2-1.7.7-1.9-.9-2.1 2.1.9 1.9-.7 1.7-2 .7v3l2 .7.7 1.7-.9 1.9 2.1 2.1 1.9-.9 1.7.7.7 2h3l.7-2 1.7-.7 1.9.9 2.1-2.1-.9-1.9.7-1.7Z" transform="scale(.75) translate(4 4)" />
  {:else if name === "help"}
    <circle cx="12" cy="12" r="9" /><path d="M9.8 9a2.4 2.4 0 1 1 3.4 2.2c-.8.4-1.2.9-1.2 1.8M12 17h.01" />
  {:else if name === "context"}
    <circle cx="12" cy="12" r="8" /><circle cx="12" cy="12" r="2" /><path d="M12 2v3M12 19v3M2 12h3M19 12h3" />
  {:else if name === "search"}
    <circle cx="10.5" cy="10.5" r="5.5" /><path d="m15 15 5 5" />
  {:else if name === "user"}
    <circle cx="12" cy="8" r="3.5" /><path d="M5 20a7 7 0 0 1 14 0" />
  {:else}
    <circle cx="12" cy="12" r="7" />
  {/if}
</svg>

<style>
  .sidebar-icon { width: 100%; height: 100%; display: block; fill: none; stroke: currentColor; stroke-linecap: round; stroke-linejoin: round; stroke-width: 1.65; vector-effect: non-scaling-stroke; }
</style>
