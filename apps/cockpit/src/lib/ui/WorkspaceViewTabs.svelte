<script lang="ts">
  interface WorkspaceView {
    id: string;
    label: string;
  }

  export let label: string;
  export let route: string;
  export let views: readonly WorkspaceView[];
  export let active: string;
</script>

<nav class="workspace-views" aria-label={label}>
  {#each views as view}
    <a href={`${route}?view=${encodeURIComponent(view.id)}`} aria-current={active === view.id ? "page" : undefined}>{view.label}</a>
  {/each}
</nav>

<style>
  .workspace-views {
    display: flex;
    gap: 4px;
    overflow-x: auto;
    padding: 4px;
    border: 1px solid var(--line, #e4e7ec);
    border-radius: 11px;
    background: var(--color-surface);
  }
  .workspace-views a {
    flex: 0 0 auto;
    padding: 7px 10px;
    border-radius: 8px;
    color: var(--color-text-muted);
    font-size: 0.78rem;
    font-weight: 650;
    text-decoration: none;
  }
  .workspace-views a[aria-current="page"] {
    background: var(--color-surface-elevated);
    color: var(--text, #101828);
    box-shadow: 0 1px 2px rgb(16 24 40 / 8%);
  }
  .workspace-views a:focus-visible {
    outline: 3px solid var(--focus-ring, #4f46e5);
    outline-offset: 1px;
  }
</style>
