<script lang="ts">import "$lib/ui/screen.css";</script>
<svelte:head><title>Help · UIAI Engine Cockpit</title></svelte:head>
<div class="screen help-screen">
  <div class="screen-header"><div><p class="screen-kicker">System</p><h1>Help</h1><p class="screen-lede">Current shell shortcuts, recovery routes, and contract boundaries.</p></div><a class="screen-button" href="/settings">Open Settings</a></div>
  <section class="help-grid">
    <article class="screen-card"><p class="screen-kicker">Keyboard</p><h2>Navigate without drag or hover</h2><dl><div><dt>Find or do</dt><dd>⌘/Ctrl K</dd></div><div><dt>Settings</dt><dd>⌘/Ctrl ,</dd></div><div><dt>Help</dt><dd>?</dd></div><div><dt>Sidebar mode</dt><dd>[</dd></div><div><dt>Rows / groups</dt><dd>↑ ↓ / ← →</dd></div><div><dt>Row commands</dt><dd>Shift F10</dd></div><div><dt>Reorder</dt><dd>Alt ↑ / ↓</dd></div><div><dt>Favorites</dt><dd>⌘/Ctrl 1–9</dd></div></dl></article>
    <article class="screen-card"><p class="screen-kicker">Recovery</p><h2>Keep local work accessible</h2><p>When entitlement, scope, nodes, or protected workers are unavailable, the shell remains recovery-only. Local artifacts and Evidence remain accessible; execution stays blocked.</p><div class="help-actions"><a href="/nodes-services?view=uiai-engine">Entitlement & worker status →</a><a href="/settings?section=scope">Scope configuration →</a><a href="/evidence">Local Evidence →</a></div></article>
    <article class="screen-card"><p class="screen-kicker">Contracts</p><h2>Inspect registered truth</h2><p>Capabilities show manifest and signed entitlement metadata without implying grants. Overview resumes only canonical Focusa Workpoint state.</p><div class="help-actions"><a href="/capabilities">Capability catalog →</a><a href="/?inspector=scope">Context Inspector →</a><a href="/activity?view=audit">Bounded audit view →</a></div></article>
  </section>
</div>
<style>
  .help-screen { display: grid; gap: 20px; }
  .help-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 14px; }
  article { padding: 18px; } article h2 { margin: 3px 0 12px; font-size: 1rem; } article p { color: var(--color-text-muted); line-height: 1.5; }
  dl { display: grid; gap: 7px; margin: 0; } dl div { display: flex; justify-content: space-between; gap: 10px; border-top: 1px solid var(--color-border); padding-top: 7px; } dt { color: var(--color-text-muted); } dd { margin: 0; font-weight: 700; text-align: end; }
  .help-actions { display: grid; gap: 8px; } .help-actions a { color: inherit; font-weight: 700; } .help-actions a:focus-visible { outline: 3px solid var(--color-focus-ring); outline-offset: 2px; }
  @media (max-width: 900px) { .help-grid { grid-template-columns: 1fr; } }
</style>
