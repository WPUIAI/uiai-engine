<script lang="ts">
  import "$lib/ui/screen.css";
  import { engineClient, type ScreenshotResult } from "$lib/engine-client";

  type Tab = "capture" | "compare" | "analyze" | "design" | "produce";
  let tab: Tab = "capture";
  let url = "";
  let capturing = false;
  let result: ScreenshotResult & { focusa?: Record<string, unknown> } | null = null;
  let error = "";
  // 006 stubs: whiteboard LWW, generative pipeline state
  let whiteboardStrokes: {x:number,y:number}[] = [];
  let canvasEl: HTMLCanvasElement | null = null;

  async function capture() {
    if (!url.trim()) return;
    capturing = true; error = "";
    try { result = await engineClient.screenshotUrl(url.trim()); }
    catch (cause) { result = null; error = cause instanceof Error ? cause.message : "Screenshot capture failed."; }
    finally { capturing = false; }
  }
  function addStroke(e: MouseEvent) {
    if (!canvasEl) return;
    const r = canvasEl.getBoundingClientRect();
    whiteboardStrokes = [...whiteboardStrokes, {x: e.clientX - r.left, y: e.clientY - r.top}];
    drawWhiteboard();
  }
  function drawWhiteboard() {
    if (!canvasEl) return;
    const ctx = canvasEl.getContext("2d"); if (!ctx) return;
    ctx.clearRect(0,0,canvasEl.width, canvasEl.height);
    ctx.strokeStyle = "var(--color-accent)"; ctx.lineWidth = 2;
    ctx.beginPath();
    whiteboardStrokes.forEach((p,i)=> i===0 ? ctx.moveTo(p.x,p.y) : ctx.lineTo(p.x,p.y));
    ctx.stroke();
  }
</script>

<svelte:head><title>Studio · UIAI Engine Cockpit</title></svelte:head>
<div class="screen">
  <div class="screen-header"><div><p class="screen-kicker">Create — Workstream-scoped</p><h1>Studio</h1><p class="screen-lede">006 Creative Workbench — Capture / Compare / Analyze / Design / Produce + whiteboard (tldraw-offline LWW) + generative GUIs + Report Canvas collab.</p></div><span class="badge">Scope: workstream</span></div>

  <nav class="studio-tabs" aria-label="Studio sections">
    {#each [["capture","Capture"],["compare","Compare"],["analyze","Analyze"],["design","Design"],["produce","Produce"]] as [id,label]}
      <button class="tab" class:active={tab===id} on:click={() => tab = id as Tab}>{label}</button>
    {/each}
  </nav>

  {#if tab === "capture"}
    {#if error}<div class="error-banner" role="alert"><strong>Capture unavailable.</strong><span>{error}</span></div>{/if}
    <section class="screen-card studio-launcher"><div><p class="screen-kicker">Capture — /api/screenshot/*</p><h2>Render a page for inspection</h2><p>Same-session screenshot via Engine. Baseline + frames later.</p></div><div class="launch-form"><input bind:value={url} aria-label="URL to capture" placeholder="https://example.com" on:keydown={(e)=>e.key==="Enter"&&capture()} /><button class="screen-button primary" on:click={capture} disabled={capturing || !url.trim()}>{capturing ? "Capturing…" : "Capture"}</button></div></section>
    {#if result?.screenshot}<section class="screen-card studio-result"><div class="screen-toolbar"><div><p class="screen-kicker">Captured surface</p><h2>{result.title || result.url}</h2><p>{result.url} · {result.width}×{result.height} · {result.duration_ms} ms</p></div><button class="screen-button" on:click={() => (result = null)}>Clear</button></div><img class="studio-image" src={`data:image/${result.format || "jpeg"};base64,${result.screenshot}`} alt="capture" /></section>{:else}<section class="empty-screen"><div class="empty-mark">▧</div><h2>No capture yet</h2><p>Engine returns image + bounded Focusa metadata. Stub for Compare/Analyze consumes this artifact.</p></section>{/if}
  {:else if tab === "compare"}
    <section class="screen-card"><p class="screen-kicker">Compare — /api/comparison + /api/layout-compare + /api/media/frame/*</p><h2>Visual diff</h2><p>Compare captures against baseline. Threshold + changed-region overlay. Gated pipeline — artifact-backed, Evidence-receipted.</p><div class="empty-screen"><p>Drop two captures or a baseline to diff. (Stub — wires to /api/comparison in 006-T01)</p></div></section>
  {:else if tab === "analyze"}
    <section class="screen-card"><p class="screen-kicker">Analyze — /api/critique + /api/ui-reverse + /api/section-detect + /api/reference/analyze</p><h2>Critique & reverse</h2><p>Section-detect, UI reverse, a11y/contrast, diagnostics-assisted investigation. Consent-gated paid actions.</p><div class="empty-screen"><p>Run critique on last capture → Report Canvas draft. (Stub — 006-T03)</p></div></section>
  {:else if tab === "design"}
    <section class="screen-card"><p class="screen-kicker">Design — /api/design-system + /api/content-map + /api/block-recipes → generated GUIs</p><h2>Design system & block recipes</h2><p>Extract tokens, map content, emit WPUIAI block recipes. Generative GUIs manifest-declared, cost-visible, provenance-tagged → Evidence.</p>
      <div class="studio-grid">
        <div class="mini-card"><strong>design-system.json</strong><p>Tokens + components from capture</p><button class="screen-button">Extract (stub)</button></div>
        <div class="mini-card"><strong>content-map.json</strong><p>Sections + copy map</p><button class="screen-button">Map (stub)</button></div>
        <div class="mini-card"><strong>block-recipes/</strong><p>Generated workspaces / dashboards</p><button class="screen-button">Generate (stub)</button></div>
      </div>
      <p style="margin-top:12px"><em>Whiteboard (tldraw-offline) — Workstream-scoped LWW via SSE/BroadcastChannel, undo-ready. Click to sketch; agent strokes via semantic API.</em></p>
      <canvas bind:this={canvasEl} width="640" height="240" class="whiteboard" on:click={addStroke}></canvas>
      <p><button class="screen-button" on:click={()=>{whiteboardStrokes=[]; drawWhiteboard();}}>Clear board</button> Strokes: {whiteboardStrokes.length} — persists to Evidence as JSON+PNG (stub)</p>
    </section>
  {:else if tab === "produce"}
    <section class="screen-card"><p class="screen-kicker">Produce — /api/media/produce + /api/media/frame/* → mockups, GIFs, HyperFrames video</p><h2>Media produce</h2><p>Device mockups + WorkRouter product video via HyperFrames (Three/Theatre/GSAP + Bloom/Bokeh). Lifecycle/cancel/artifact proof gated.</p>
      <div class="studio-grid">
        <div class="mini-card"><strong>Device mockup</strong><p>Frame capture into device</p><button class="screen-button">Frame (stub)</button></div>
        <div class="mini-card"><strong>WorkRouter video</strong><p>HyperFrames 3D tilt/dolly/bloom</p><button class="screen-button">Produce (stub)</button></div>
        <div class="mini-card"><strong>Report Canvas</strong><p>Google-Docs-like collab via Documents + H44 live sync</p><button class="screen-button">Generate Report (stub)</button></div>
      </div>
      <p style="margin-top:12px">All Produce jobs show lifecycle in Activity, cancelable, artifact → Evidence with provenance.</p>
    </section>
  {/if}
</div>
<style>
  .studio-tabs{display:flex;gap:8px;margin:12px 0 16px;flex-wrap:wrap}
  .tab{padding:8px 12px;border:1px solid var(--color-border);border-radius:8px;background:var(--color-surface);cursor:pointer;font:inherit;font-size:12px}
  .tab.active{background:var(--color-accent);color:#fff;border-color:var(--color-accent)}
  .studio-launcher{display:grid;grid-template-columns:1fr minmax(330px,.85fr);gap:26px;align-items:end;padding:20px;margin-bottom:18px;border-color:color-mix(in srgb,var(--color-accent) 24%,var(--color-border));background:linear-gradient(135deg,color-mix(in srgb,var(--color-accent) 7%,var(--color-surface-elevated)),var(--color-surface-elevated))}
  .launch-form{display:flex;gap:8px}
  .launch-form input{flex:1;min-width:0;padding:10px 11px;border:1px solid var(--color-border);border-radius:8px;background:var(--color-surface);font:inherit;font-size:12px}
  .studio-result{padding:20px}
  .studio-image{display:block;width:100%;max-height:650px;margin-top:18px;object-fit:contain;border:1px solid var(--color-border);border-radius:8px;background:#111}
  .studio-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:12px;margin-top:12px}
  .mini-card{padding:12px;border:1px solid var(--color-border);border-radius:8px;background:var(--color-surface-elevated)}
  .whiteboard{width:100%;border:1px solid var(--color-border);border-radius:8px;background:#fff;margin-top:8px;cursor:crosshair}
  .error-banner{display:flex;gap:10px;margin:-8px 0 18px;padding:12px 14px;border:1px solid color-mix(in srgb,var(--color-error) 25%,var(--color-border));border-radius:9px;color:var(--color-error);background:color-mix(in srgb,var(--color-error) 7%,transparent);font-size:12px}
  @media(max-width:760px){.studio-launcher{grid-template-columns:1fr}.studio-grid{grid-template-columns:1fr}}
</style>
