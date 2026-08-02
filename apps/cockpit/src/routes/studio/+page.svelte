<script lang="ts">
  import "$lib/ui/screen.css";
  import { engineClient, type ScreenshotResult } from "$lib/engine-client";

  let url = "";
  let capturing = false;
  let result: ScreenshotResult & { focusa?: Record<string, unknown> } | null = null;
  let error = "";

  async function capture() {
    if (!url.trim()) return;
    capturing = true;
    error = "";
    try { result = await engineClient.screenshotUrl(url.trim()); }
    catch (cause) { result = null; error = cause instanceof Error ? cause.message : "Screenshot capture failed."; }
    finally { capturing = false; }
  }
</script>

<svelte:head><title>Studio · UIAI Engine Cockpit</title></svelte:head>
<div class="screen"><div class="screen-header"><div><p class="screen-kicker">Inspect</p><h1>Studio</h1><p class="screen-lede">Capture and inspect a real browser surface through the UIAI Engine.</p></div><span class="badge">Screenshot capture</span></div>
  {#if error}<div class="error-banner" role="alert"><strong>Capture unavailable.</strong><span>{error}</span></div>{/if}
  <section class="screen-card studio-launcher"><div><p class="screen-kicker">Capture</p><h2>Render a page for inspection</h2><p>This calls the engine screenshot route and returns the image plus bounded Focusa metadata.</p></div><div class="launch-form"><input bind:value={url} aria-label="URL to capture" placeholder="https://example.com" onkeydown={(event) => event.key === "Enter" && capture()} /><button class="screen-button primary" type="button" onclick={capture} disabled={capturing || !url.trim()}>{capturing ? "Capturing…" : "Capture"}</button></div></section>
  {#if result?.screenshot}<section class="screen-card studio-result"><div class="screen-toolbar"><div><p class="screen-kicker">Captured surface</p><h2>{result.title || result.url}</h2><p>{result.url} · {result.width}×{result.height} · {result.duration_ms} ms</p></div><button class="screen-button" type="button" onclick={() => (result = null)}>Clear</button></div><img class="studio-image" src={`data:image/${result.format || "jpeg"};base64,${result.screenshot}`} alt={`Captured browser view of ${result.url || "the requested page"}`} /></section>{:else}<section class="empty-screen"><div class="empty-mark">▧</div><h2>No capture yet</h2><p>Use the capture action above. Studio stays empty until the engine returns an actual image.</p></section>{/if}
</div>
<style>.studio-launcher { display: grid; grid-template-columns: 1fr minmax(330px, .85fr); gap: 26px; align-items: end; padding: 20px; margin-bottom: 18px; border-color: color-mix(in srgb, var(--color-accent) 24%, var(--color-border)); background: linear-gradient(135deg, color-mix(in srgb, var(--color-accent) 7%, var(--color-surface-elevated)), var(--color-surface-elevated)); } .studio-launcher h2 { margin: 0; font-size: 18px; } .studio-launcher p { margin: 7px 0 0; } .launch-form { display: flex; gap: 8px; } .launch-form input { min-width: 0; flex: 1; padding: 10px 11px; border: 1px solid var(--color-border); border-radius: 8px; outline: none; color: var(--color-text); background: var(--color-surface); font: inherit; font-size: 12px; } .launch-form input:focus { border-color: var(--color-accent); box-shadow: 0 0 0 3px var(--color-focus-ring); } .studio-result { padding: 20px; } .studio-image { display: block; width: 100%; max-height: 650px; margin-top: 18px; object-fit: contain; border: 1px solid var(--color-border); border-radius: 8px; background: #111; } .error-banner { display: flex; align-items: center; gap: 10px; margin: -8px 0 18px; padding: 12px 14px; border: 1px solid color-mix(in srgb, var(--color-error) 25%, var(--color-border)); border-radius: 9px; color: var(--color-error); background: color-mix(in srgb, var(--color-error) 7%, transparent); font-size: 12px; } .error-banner span { color: var(--color-text-muted); } @media (max-width: 760px) { .studio-launcher { grid-template-columns: 1fr; } .launch-form { flex-wrap: wrap; } }</style>
