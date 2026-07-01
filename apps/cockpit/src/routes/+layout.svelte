<script lang="ts">
  import "../lib/ui/design-tokens.css";
</script>

<main class="cockpit-shell">
  <header class="scope-strip" role="region" aria-label="Scope strip">
    <span class="chip">Local Only</span>
    <span class="chip">No node selected</span>
    <span class="chip">ScopeRef: none</span>
    <span class="chip accent">UIAI: ok</span>
  </header>

  <div class="body">
    <nav class="nav" aria-label="Primitive navigator">
      <h2>Primitives</h2>
      <ul>
        <li>UIAI Engine</li>
        <li>Focusa Local</li>
        <li>Focusa Cloud</li>
        <li>AI API</li>
      </ul>
    </nav>

    <section class="viewport" role="main">
      <slot />
    </section>

    <aside class="inspector" role="region" aria-label="Inspector">
      <h2>Inspector</h2>
      <p class="muted">Select a primitive to inspect.</p>
    </aside>
  </div>

  <footer class="ribbon" role="region" aria-label="Process ribbon">
    <span class="ribbon-item">routines: 0</span>
    <span class="ribbon-item">decisions: 0</span>
    <span class="ribbon-item">predictions: 0</span>
    <span class="ribbon-item">proofs: 0</span>
  </footer>
</main>

<style>
  .cockpit-shell {
    display: grid;
    grid-template-rows: auto 1fr auto;
    min-height: 100vh;
    background: var(--color-bg);
    color: var(--color-text);
  }

  .scope-strip {
    display: flex;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-4);
    border-bottom: 1px solid var(--color-border);
    background: var(--color-surface);
  }

  .chip {
    padding: var(--space-1) var(--space-3);
    background: var(--color-surface-elevated);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-pill);
    font-size: 12px;
    line-height: 16px;
  }
  .chip.accent {
    border-color: var(--color-accent);
    color: var(--color-accent);
  }

  .body {
    display: grid;
    grid-template-columns: 240px 1fr 320px;
    gap: var(--space-2);
    padding: var(--space-2);
  }

  .nav, .viewport, .inspector {
    background: var(--color-surface-elevated);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-card);
    padding: var(--space-4);
    box-shadow: var(--shadow-card);
  }

  .nav h2, .inspector h2 {
    margin: 0 0 var(--space-3);
    font-size: 14px;
    line-height: 20px;
    font-weight: 600;
  }
  .nav ul { padding: 0; margin: 0; list-style: none; }
  .nav li {
    padding: var(--space-2) 0;
    color: var(--color-text-muted);
    cursor: pointer;
  }
  .nav li:hover { color: var(--color-text); }

  .muted { color: var(--color-text-muted); }

  .ribbon {
    display: flex;
    gap: var(--space-4);
    padding: var(--space-2) var(--space-4);
    border-top: 1px solid var(--color-border);
    background: var(--color-surface);
    font-size: 12px;
    line-height: 16px;
  }
  .ribbon-item { color: var(--color-text-muted); }
</style>
