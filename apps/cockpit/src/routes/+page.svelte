<script lang="ts">
  import { phase0Cards } from "$lib/cards/phase0-card-manifest";
</script>

<h1>Focusa Cockpit</h1>
<p class="muted">Backend-first shell (Slice 0). Phase 0 card manifest loaded into the navigator.</p>

<section class="cards" aria-label="Phase 0 cards">
  {#each phase0Cards as card}
    <article class="card" data-card-id={card.card_id}>
      <header>
        <h3>{card.label}</h3>
        <span class="badge" data-product={card.product_surface}>
          {card.product_surface}
        </span>
      </header>
      <p class="muted">
        <code>{card.card_id}</code>
        {#if card.contract_ref}
          &nbsp;·&nbsp;contract&nbsp;
          <code>{card.contract_ref}</code>
        {:else}
          &nbsp;·&nbsp;<em>{card.notes ?? "adapter_only"}</em>
        {/if}
      </p>
    </article>
  {/each}
</section>

<style>
  h1 {
    font-size: 28px;
    line-height: 32px;
    font-weight: 700;
    margin: 0 0 var(--space-4);
  }
  .muted { color: var(--color-text-muted); }
  code {
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    font-size: 12px;
    padding: 1px 4px;
    background: var(--color-surface);
    border-radius: 4px;
  }
  .cards {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: var(--space-3);
    margin-top: var(--space-4);
  }
  .card {
    padding: var(--space-4);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-card);
    box-shadow: var(--shadow-card);
  }
  .card header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: var(--space-2);
  }
  .card h3 {
    margin: 0;
    font-size: 16px;
    line-height: 22px;
  }
  .badge {
    font-size: 11px;
    padding: 2px 8px;
    border-radius: var(--radius-pill);
    background: var(--color-surface-elevated);
    border: 1px solid var(--color-border);
  }
  .badge[data-product="focusa_local"] { border-color: var(--color-success); color: var(--color-success); }
  .badge[data-product="focusa_cloud"] { border-color: var(--color-warn); color: var(--color-warn); }
  .badge[data-product="ai_api"] { border-color: var(--color-accent); color: var(--color-accent); }
  .badge[data-product="uiai_engine"] { border-color: var(--color-text-muted); }
</style>
