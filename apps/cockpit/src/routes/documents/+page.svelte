<script lang="ts">
  import "$lib/ui/screen.css";
  import { engineClient, type SearchResult } from "$lib/engine-client";

  let query = "";
  let sourceUrl = "";
  let searching = false;
  let capturing = false;
  let error = "";
  let results: SearchResult[] = [];
  let captured: { url: string; title?: string; markdown?: string; focusa?: Record<string, unknown> } | null = null;

  async function search() {
    if (!query.trim()) return;
    searching = true;
    error = "";
    try { results = (await engineClient.search(query.trim())).results || []; }
    catch (cause) { error = cause instanceof Error ? cause.message : "Search is unavailable."; results = []; }
    finally { searching = false; }
  }

  async function captureSource(url = sourceUrl) {
    if (!url.trim()) return;
    capturing = true;
    error = "";
    try { const result = await engineClient.sourceMarkdown(url.trim()); captured = { url: url.trim(), title: result.title, markdown: result.markdown, focusa: result.focusa }; }
    catch (cause) { error = cause instanceof Error ? cause.message : "The source could not be captured."; }
    finally { capturing = false; }
  }
</script>

<svelte:head><title>Documents · UIAI Engine Cockpit</title></svelte:head>
<div class="screen"><div class="screen-header"><div><p class="screen-kicker">Research</p><h1>Documents</h1><p class="screen-lede">Search the web, open a source, and capture bounded Markdown through the UIAI Engine.</p></div><span class="badge">Provider-neutral</span></div>
  {#if error}<div class="error-banner" role="alert"><strong>Research request unavailable.</strong><span>{error}</span><button class="data-row-action" type="button" onclick={() => (error = "")}>Dismiss</button></div>{/if}
  <section class="screen-card research-launcher"><div><p class="screen-kicker">Find</p><h2>Search within the readable source surface</h2><p>Results and evidence metadata come from the engine. Nothing is seeded locally.</p></div><div class="launch-form"><input bind:value={query} aria-label="Search query" placeholder="Search for a source…" onkeydown={(event) => event.key === "Enter" && search()} /><button class="screen-button primary" type="button" onclick={search} disabled={searching || !query.trim()}>{searching ? "Searching…" : "Search"}</button></div></section>
  {#if results.length}<section class="screen-card result-section"><div class="screen-toolbar"><div><p class="screen-kicker">Search results</p><h2>{results.length} results</h2></div><span class="badge">Open or capture a result</span></div><div class="data-list">{#each results as result}<div class="data-row"><span class="activity-icon" data-tone="blue">⌕</span><div class="data-row-main"><strong>{result.title}</strong><span>{result.description || result.url}</span></div><button class="data-row-action" type="button" onclick={() => (sourceUrl = result.url)}>Capture →</button><a class="data-row-action" href={result.url} target="_blank" rel="noreferrer">Open ↗</a></div>{/each}</div></section>{:else if query && !searching}<section class="empty-screen"><div class="empty-mark">⌕</div><h2>No results returned</h2><p>The provider returned no readable sources for this query.</p></section>{/if}
  <section class="screen-card source-launcher"><div><p class="screen-kicker">Capture</p><h2>Capture a source directly</h2><p>Open the page through the browser engine and return Markdown, links, diagnostics, and Focusa metadata.</p></div><div class="launch-form"><input bind:value={sourceUrl} aria-label="Source URL" placeholder="https://example.com/article" onkeydown={(event) => event.key === "Enter" && captureSource()} /><button class="screen-button primary" type="button" onclick={() => captureSource()} disabled={capturing || !sourceUrl.trim()}>{capturing ? "Capturing…" : "Capture source"}</button></div></section>
  {#if captured}<section class="screen-card document-result"><div class="screen-toolbar"><div><p class="screen-kicker">Captured source</p><h2>{captured.title || captured.url}</h2><p>{captured.url}</p></div><button class="screen-button" type="button" onclick={() => (captured = null)}>Clear</button></div><div class="capture-meta"><span><small>Evidence</small><strong>{String(captured.focusa?.evidence_ref || "Not returned")}</strong></span><span><small>Next tool</small><strong>{String(captured.focusa?.preferred_tool || "Not returned")}</strong></span></div><div class="document-markdown">{captured.markdown || "The engine returned no Markdown content."}</div></section>{:else}<section class="empty-screen"><div class="empty-mark">▤</div><h2>No captured source</h2><p>Capture a source above and the returned artifact will appear here for inspection.</p></section>{/if}
</div>

<style>
  .research-launcher, .source-launcher { display: grid; grid-template-columns: 1fr minmax(330px, .85fr); gap: 26px; align-items: end; padding: 20px; margin-bottom: 18px; border-color: color-mix(in srgb, var(--color-accent) 24%, var(--color-border)); background: linear-gradient(135deg, color-mix(in srgb, var(--color-accent) 7%, var(--color-surface-elevated)), var(--color-surface-elevated)); } .research-launcher h2, .source-launcher h2 { margin: 0; font-size: 18px; } .research-launcher p, .source-launcher p { margin: 7px 0 0; } .launch-form { display: flex; gap: 8px; } .launch-form input { min-width: 0; flex: 1; padding: 10px 11px; border: 1px solid var(--color-border); border-radius: 8px; outline: none; color: var(--color-text); background: var(--color-surface); font: inherit; font-size: 12px; } .launch-form input:focus { border-color: var(--color-accent); box-shadow: 0 0 0 3px var(--color-focus-ring); } .result-section, .document-result { margin-bottom: 18px; } .document-result { padding: 20px; } .capture-meta { display: grid; grid-template-columns: 1fr 1fr; gap: 1px; margin-top: 18px; border: 1px solid var(--color-border); background: var(--color-border); } .capture-meta span { display: grid; gap: 4px; padding: 11px; background: var(--color-surface); } .capture-meta small { color: var(--color-text-muted); font-size: 10px; } .capture-meta strong { overflow: hidden; color: var(--color-text); font-size: 11px; text-overflow: ellipsis; white-space: nowrap; } .document-markdown { max-height: 480px; overflow: auto; margin-top: 18px; padding-top: 18px; border-top: 1px solid var(--color-border); color: var(--color-text); white-space: pre-wrap; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; line-height: 19px; } .error-banner { display: flex; align-items: center; gap: 10px; margin: -8px 0 18px; padding: 12px 14px; border: 1px solid color-mix(in srgb, var(--color-error) 25%, var(--color-border)); border-radius: 9px; color: var(--color-error); background: color-mix(in srgb, var(--color-error) 7%, transparent); font-size: 12px; } .error-banner span { color: var(--color-text-muted); } .error-banner .data-row-action { margin-left: auto; } @media (max-width: 760px) { .research-launcher, .source-launcher { grid-template-columns: 1fr; } .launch-form { flex-wrap: wrap; } }
</style>
