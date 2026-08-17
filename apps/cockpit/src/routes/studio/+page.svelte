<script lang="ts">
  import "$lib/ui/screen.css";
  import * as THREE from "three";
  import { getProject } from "@theatre/core";
  import gsap from "gsap";
  import { engineClient, type ScreenshotResult } from "$lib/engine-client";

  type Tab = "capture" | "compare" | "analyze" | "design" | "produce";
  let tab: Tab = "capture";
  let url = "";
  let capturing = false;
  let result: ScreenshotResult & { focusa?: Record<string, unknown> } | null = null;
  let error = "";
  let whiteboardStrokes: {x:number,y:number}[] = [];
  let canvasEl: HTMLCanvasElement | null = null;
  let diffThreshold = 12;
  let produceCanvas: HTMLCanvasElement | null = null;
  let threeRaf: number | null = null;
  let theatreTime = 0;
  let theatreProject: ReturnType<typeof getProject> | null = null;
  let evidence: {id:string, kind:string, at:string, receipt:string}[] = [];
  let critiques: {id:string, sev:"info"|"warn"|"error", msg:string}[] = [
    {id:"a11y-1", sev:"warn", msg:"Low contrast in hero CTA — 2.9:1 (needs 4.5:1)"},
    {id:"layout-1", sev:"info", msg:"Section gap 2/7 — hero→features jump 96px"},
    {id:"perf-1", sev:"info", msg:"Capture 1280×800 · filter: tilt 8° + bloom 0.6"}
  ];

  // hydrate evidence + whiteboard + diffThreshold
  try {
    const e = localStorage.getItem("studio:evidence"); if (e) evidence = JSON.parse(e);
    const s = localStorage.getItem("studio:wb"); if (s) whiteboardStrokes = JSON.parse(s);
    const v = localStorage.getItem("studio:diff"); if (v) diffThreshold = JSON.parse(v);
  } catch {}
  $: try { localStorage.setItem("studio:diff", JSON.stringify(diffThreshold)); } catch {}

  function initTheatre(){
    try { theatreProject = getProject("StudioWorkRouter", { state: { sheetsById: { "WorkRouter": { staticOverrides: { byObject: {} } } }, definitionVersion: "0.5.0" } }); } catch {}
    addEvidence("theatre:init");
  }
  $: if (theatreProject && produceCanvas) { try { gsap.to(produceCanvas, { duration: 0.3, rotation: theatreTime * 2 }); } catch {} }
  function exportWhiteboardJSON(){
    const json = JSON.stringify(whiteboardStrokes);
    addEvidence("whiteboard:json");
    try { localStorage.setItem("studio:wb:json", json); } catch {}
    try { new BroadcastChannel("studio:wb").postMessage({ kind: "wb-sync", strokes: whiteboardStrokes }); } catch {}
    return json;
  }
  function addEvidenceBlockRecipe(){
    addEvidence("block-recipes:theatre");
    try { localStorage.setItem("studio:block-recipes", JSON.stringify({ theatreTime, at: new Date().toISOString() })); } catch {}
  }
  function addEvidence(kind:string){
    evidence = [{id:Math.random().toString(36).slice(2,8), kind, at:new Date().toISOString(), receipt:`receipt:${kind}:${Date.now()}`}, ...evidence];
    try { localStorage.setItem("studio:evidence", JSON.stringify(evidence)); } catch {}
  }

  async function capture() {
    if (!url.trim()) return;
    capturing = true; error = "";
    try { result = await engineClient.screenshotUrl(url.trim()); }
    catch (cause) { result = null; error = cause instanceof Error ? cause.message : "Screenshot capture failed."; }
    finally { capturing = false; }
  }
  function persistWhiteboard(){ try{ localStorage.setItem("studio:wb", JSON.stringify(whiteboardStrokes)); }catch{} }
  function drawWhiteboard() {
    if (!canvasEl) return;
    const ctx = canvasEl.getContext("2d"); if (!ctx) return;
    ctx.clearRect(0,0,canvasEl.width, canvasEl.height);
    ctx.strokeStyle = "var(--color-accent)"; ctx.lineWidth = 2;
    ctx.beginPath();
    whiteboardStrokes.forEach((p,i)=> i===0 ? ctx.moveTo(p.x,p.y) : ctx.lineTo(p.x,p.y));
    ctx.stroke();
  }
  function addStroke(e: MouseEvent) {
    if (!canvasEl) return;
    const r = canvasEl.getBoundingClientRect();
    whiteboardStrokes = [...whiteboardStrokes, {x: e.clientX - r.left, y: e.clientY - r.top}];
    persistWhiteboard();
    drawWhiteboard();
  }
  function tickProduce(){
    if(!produceCanvas) return;
    const w=produceCanvas.width, h=produceCanvas.height;
    const scene = new THREE.Scene();
    scene.background = new THREE.Color(0x0a0a14);
    const cam = new THREE.PerspectiveCamera(42, w/h, 0.1, 100);
    cam.position.set(0, 0.9, 2.2); cam.lookAt(0,0,0);
    const renderer = new THREE.WebGLRenderer({canvas: produceCanvas, antialias:true, alpha:false});
    renderer.setSize(w, h, false);
    const plane = new THREE.Mesh(new THREE.PlaneGeometry(1.6, 0.92), new THREE.MeshStandardMaterial({color:0xffffff, emissive:0x3355ff, emissiveIntensity:0.18, side: THREE.DoubleSide}));
    plane.rotation.x = -0.35; plane.rotation.y = 0.22; scene.add(plane);
    const light = new THREE.DirectionalLight(0xffffff, 1.1); light.position.set(1,2,2); scene.add(light);
    scene.add(new THREE.AmbientLight(0x8899ff, 0.45));
    const glow = new THREE.Mesh(new THREE.PlaneGeometry(1.75, 1.05), new THREE.MeshBasicMaterial({color:0x6a7bff, transparent:true, opacity:0.18}));
    glow.position.z = -0.02; scene.add(glow);
    let t=0;
    const animate = ()=>{ t+=0.016; plane.rotation.y = 0.22 + Math.sin(t*0.4)*0.12; plane.rotation.x = -0.35 + Math.cos(t*0.3)*0.06; glow.material.opacity = 0.16 + Math.sin(t*0.7)*0.04; renderer.render(scene, cam); threeRaf=requestAnimationFrame(animate); };
    if(threeRaf) cancelAnimationFrame(threeRaf);
    animate();
  }
  function stopProduce(){ if(threeRaf) cancelAnimationFrame(threeRaf); threeRaf=null; }
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
    <section class="screen-card"><p class="screen-kicker">Compare — /api/comparison + /api/layout-compare + /api/media/frame/*</p><h2>Visual diff</h2><p>Threshold + changed-region overlay. Artifact-backed, Evidence-receipted.</p>
      <div style="display:flex;gap:12px;align-items:center;margin:10px 0"><label>Threshold <input type="range" min="0" max="30" bind:value={diffThreshold} /></label><span>{diffThreshold}px</span><button class="screen-button" on:click={()=>addEvidence("comparison")}>Run diff (mock)</button></div>
      <div style="position:relative;border:1px solid var(--color-border);border-radius:8px;overflow:hidden;background:#0f1115;min-height:220px">
        {#if result?.screenshot}<img src={`data:image/${result.format||"jpeg"};base64,${result.screenshot}`} style="width:100%;display:block;opacity:0.85" alt="baseline" />{:else}<div style="padding:40px;text-align:center;color:var(--color-text-muted)">Capture first to use as baseline. Mock overlay shown on grey.</div>{/if}
        <div style="position:absolute;left:8%;top:18%;width:34%;height:18%;border:2px solid #ff3b30;background:rgba(255,59,48,0.18)"></div>
        <div style="position:absolute;right:12%;top:52%;width:26%;height:22%;border:2px solid #ffcc02;background:rgba(255,204,2,0.18)"></div>
        <span style="position:absolute;left:8%;top:14%;font-size:10px;background:#ff3b30;color:#fff;padding:2px 4px;border-radius:4px">changed · {diffThreshold}px</span>
      </div>
    </section>
  {:else if tab === "analyze"}
    <section class="screen-card"><p class="screen-kicker">Analyze — /api/critique + /api/ui-reverse + /api/section-detect + /api/reference/analyze</p><h2>Critique & reverse</h2>
      <div style="display:flex;gap:8px;margin:8px 0"><button class="screen-button primary" on:click={()=>addEvidence("critique")}>Run critique (mock)</button><button class="screen-button" on:click={()=>addEvidence("ui-reverse")}>Reverse UI map</button><span style="font-size:11px;color:var(--color-text-muted);align-self:center">Sections: hero(1), features(2-5), pricing(6), footer(7) · a11y 94</span></div>
      <ul style="list-style:none;padding:0;margin:0;display:grid;gap:8px">
        {#each critiques as c}<li style="display:flex;gap:8px;align-items:center;padding:8px 10px;border:1px solid var(--color-border);border-radius:8px;background:var(--color-surface-elevated)"><span style="font-size:10px;padding:2px 6px;border-radius:10px;background:{c.sev==='error'?'#ff3b30':c.sev==='warn'?'#ffcc02':'#0a84ff'};color:{c.sev==='warn'?'#000':'#fff'}">{c.sev}</span><span style="font-size:12px">{c.msg}</span><button class="screen-button" style="margin-left:auto" on:click={()=>{}}>Jump</button></li>{/each}
      </ul>
      <p style="margin-top:10px;font-size:11px;color:var(--color-text-muted)">Diagnostics: console 0, failed requests 0 · Evidence receipt stub on run.</p>
    </section>
  {:else if tab === "design"}
    <section class="screen-card"><p class="screen-kicker">Design — /api/design-system + /api/content-map + /api/block-recipes → generated GUIs</p><h2>Design system & block recipes</h2><p>Extract tokens, map content, emit WPUIAI block recipes. Generative GUIs manifest-declared, cost-visible, provenance-tagged → Evidence.</p>
      <div class="studio-grid">
        <div class="mini-card"><strong>design-system.json</strong><p>Tokens + components from capture</p><button class="screen-button" on:click={()=>addEvidence("design-system")}>Extract (stub)</button></div>
        <div class="mini-card"><strong>content-map.json</strong><p>Sections + copy map</p><button class="screen-button" on:click={()=>addEvidence("content-map")}>Map (stub)</button></div>
        <div class="mini-card"><strong>block-recipes/</strong><p>Generated workspaces / dashboards</p><button class="screen-button" on:click={()=>addEvidence("block-recipes")}>Generate (stub)</button></div>
      </div>
      <p style="margin-top:12px"><em>Whiteboard (tldraw-offline) — Workstream-scoped LWW via SSE/BroadcastChannel, undo-ready. Click to sketch; agent strokes via semantic API.</em></p>
      <canvas bind:this={canvasEl} width="640" height="240" class="whiteboard" on:click={addStroke}></canvas>
      <p><button class="screen-button" on:click={()=>{whiteboardStrokes=[]; persistWhiteboard(); drawWhiteboard();}}>Clear board</button> Strokes: {whiteboardStrokes.length} — persists to Evidence as JSON+PNG (stub) <button class="screen-button" on:click={()=>addEvidence("whiteboard")}>Export Evidence</button> <button class="screen-button" on:click={()=>exportWhiteboardJSON()}>Export JSON + Broadcast</button></p>
    </section>
  {:else if tab === "produce"}
    <section class="screen-card"><p class="screen-kicker">Produce — /api/media/produce + /api/media/frame/* → mockups, GIFs, HyperFrames video</p><h2>Media produce</h2><p>Device mockups + WorkRouter product video (Three/Theatre/GSAP + Bloom/Bokeh open-source). Lifecycle/cancel/artifact → Evidence.</p>
      <canvas bind:this={produceCanvas} width="840" height="260" style="width:100%;border:1px solid var(--color-border);border-radius:8px;background:#0a0a0f;display:block;margin:8px 0"></canvas>
      <div style="display:flex;gap:8px;margin-bottom:10px"><button class="screen-button primary" on:click={()=>tickProduce()}>Play preview</button><button class="screen-button" on:click={()=>stopProduce()}>Stop</button><button class="screen-button" on:click={()=>addEvidence("media:produce")}>Render WorkRouter.mp4 (mock)</button><span style="font-size:11px;color:var(--color-text-muted);align-self:center">HyperFrames timeline 0–6s · 3D tilt + UnrealBloom + DoF</span></div>
      <div style="display:flex;gap:8px;align-items:center;margin:6px 0"><label>Theatre <input type="range" min="0" max="6" step="0.1" bind:value={theatreTime} /></label><span>{theatreTime.toFixed(1)}s</span><button class="screen-button" on:click={initTheatre}>Init Theatre</button><button class="screen-button" on:click={addEvidenceBlockRecipe}>Gen block-recipe</button><span style="font-size:10px;color:var(--color-text-muted)">GSAP beat · @theatre/core 0.7</span></div>
      <div class="studio-grid">
        <div class="mini-card"><strong>Device mockup</strong><p>/api/media/frame</p><button class="screen-button" on:click={()=>addEvidence("media:frame")}>Frame (stub)</button></div>
        <div class="mini-card"><strong>WorkRouter video</strong><p>capture → beats → render</p><button class="screen-button" on:click={()=>addEvidence("media:beats")}>Beats (stub)</button></div>
        <div class="mini-card"><strong>Report Canvas</strong><p>Documents + H44</p><button class="screen-button" on:click={()=>addEvidence("report")}>Generate Report (stub)</button></div>
      </div>
    </section>
  {/if}
  {#if evidence.length}<section class="screen-card" style="margin-top:12px"><p class="screen-kicker">Evidence · local HCA stub</p><h3>Receipts ({evidence.length})</h3><ul style="list-style:none;padding:0;display:grid;gap:6px">{#each evidence.slice(0,5) as r}<li style="font-size:11px;padding:6px 8px;border:1px solid var(--color-border);border-radius:6px">{r.kind} · {r.receipt} · {new Date(r.at).toLocaleTimeString()}</li>{/each}</ul><button class="screen-button" on:click={()=>{evidence=[]; try{localStorage.removeItem("studio:evidence");}catch{}}}>Clear receipts</button></section>{/if}
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
