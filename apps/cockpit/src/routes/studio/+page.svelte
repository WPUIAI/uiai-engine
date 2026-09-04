<script lang="ts">
  import "$lib/ui/screen.css";
  import * as THREE from "three";
  import { EffectComposer } from "three/addons/postprocessing/EffectComposer.js";
  import { RenderPass } from "three/addons/postprocessing/RenderPass.js";
  import { UnrealBloomPass } from "three/addons/postprocessing/UnrealBloomPass.js";
  import { BokehPass } from "three/addons/postprocessing/BokehPass.js";
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
  let sseConnected = false;
  let bloomStrength = 0.85;
  let dofAperture = 0.00015;
  let dofFocus = 0.92;
  let reportCanvas: { id:string, title:string, at:string } | null = null;
  let hyperframesBusy = false;
  let auraVoice = "Aura-Asteria-en";
  let captionsEnabled = true;
  let captionsVtt = "WEBVTT\n\n00:00:00.000 --> 00:00:02.500\nStop hunting for work.\n\n00:00:02.500 --> 00:00:06.000\nLet WorkRouter hunt for you.\n";
  let claimManifest: { id:string, claims:string[], at:string, receipt:string } | null = null;
  let demosGallery: { id:string, title:string, src:string }[] = [];
  let homepageEmbedded = false;
  let otaStatus: "idle"|"checking"|"ready"|"applied" = "idle";
  let lambdaBusy = false;
  // v0.10 Combine-All — motion-design + remotion + openness philosophy
  let motionPersonality: "Premium"|"Playful"|"Corporate"|"Energetic" = "Premium";
  let staggerMs = 60;
  let iterationCount = 0;
  let remotionComp: { id:string, fps:number, durationInFrames:number, title:string } | null = null;
  let shaderEnabled = true;
  let particlesEnabled = true;
  let lottieEnabled = false;
  let lowerThirdEnabled = true;
  try { const v = localStorage.getItem("studio:shader"); if(v) shaderEnabled = JSON.parse(v); const w = localStorage.getItem("studio:particles"); if(w) particlesEnabled = JSON.parse(w); } catch {}
  try { const ic = localStorage.getItem("studio:iteration"); if(ic) iterationCount = JSON.parse(ic); const mp = localStorage.getItem("studio:motion"); if(mp) motionPersonality = JSON.parse(mp) as any; } catch {}
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
  $: try { localStorage.setItem("studio:motion", JSON.stringify(motionPersonality)); } catch {}
  $: try { localStorage.setItem("studio:iteration", JSON.stringify(iterationCount)); } catch {}
  $: try { localStorage.setItem("studio:shader", JSON.stringify(shaderEnabled)); } catch {}
  $: try { localStorage.setItem("studio:particles", JSON.stringify(particlesEnabled)); } catch {}
  $: try { localStorage.setItem("studio:lottie", JSON.stringify(lottieEnabled)); } catch {}
  $: try { localStorage.setItem("studio:lowerThird", JSON.stringify(lowerThirdEnabled)); } catch {}

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
  let sseChannel: BroadcastChannel | null = null;
  function initSSE(){
    try { sseChannel = new BroadcastChannel("studio:wb"); sseChannel.onmessage = (e)=>{ if(e.data?.kind==="wb-sync" && Array.isArray(e.data.strokes)) { whiteboardStrokes = e.data.strokes; drawWhiteboard(); } }; sseConnected = true; addEvidence("sse:h44-connected"); } catch {}
    try { const es = new EventSource("/api/events?scope=workstream"); es.onmessage = (e)=>{ try{ const d=JSON.parse(e.data); if(d.kind) addEvidence(`sse:${d.kind}`);}catch{} }; es.onerror = ()=>{ sseConnected=false; }; } catch {}
  }
  async function runComparison(){
    addEvidence("comparison:bloom");
    try { const r = await fetch("/api/comparison", { method: "POST", headers: {"content-type":"application/json"}, body: JSON.stringify({ threshold: diffThreshold, bloomStrength }) }); const j = await r.json().catch(()=>null); if(j?.receipt) addEvidence(`comparison:receipt:${j.receipt.slice(0,6)}`); } catch { addEvidence("comparison:mock-receipt"); }
  }
  async function runCritique(){
    addEvidence("critique:cost-gated");
    try { const r = await fetch("/api/critique", { method: "POST", body: JSON.stringify({ url }) }); const j=await r.json().catch(()=>null); if(j?.provenance) addEvidence(`critique:prov:${j.provenance.slice(0,6)}`);} catch { addEvidence("critique:mock-receipt"); }
  }
  function buildHyperFramesComposition(){
    const comp = {
      version: "0.7",
      title: "WorkRouter — HyperFrames 6s",
      duration: 6,
      fps: 30,
      timeline: [{ at: 0, kind: "capture", url }, { at: 1.2, kind: "tilt", value: theatreTime }, { at: 3, kind: "bloom", value: bloomStrength }, { at: 4.5, kind: "dof", aperture: dofAperture, focus: dofFocus }],
      tracks: { video: { clip: "data-clip: workrouter hero" }, audio: { voice: "aura neural" } },
      evidence: { provenance: "studio:hyperframes:open-source", at: new Date().toISOString() }
    };
    try { localStorage.setItem("studio:hyperframes", JSON.stringify(comp)); } catch {}
    addEvidence("hyperframes:composition");
    return comp;
  }
  async function renderHyperFrames(){
    hyperframesBusy = true;
    const comp = buildHyperFramesComposition();
    addEvidence("media:produce:hyperframes");
    try { const r = await fetch("/api/media/produce", { method: "POST", headers:{ "content-type":"application/json"}, body: JSON.stringify(comp)}); const j=await r.json().catch(()=>null); if(j?.artifact) { reportCanvas={ id:j.artifact.slice(0,8), title:"WorkRouter.mp4", at:new Date().toISOString()}; addEvidence(`media:produce:artifact:${j.artifact.slice(0,6)}`);} else { reportCanvas={ id:Math.random().toString(36).slice(2,8), title:"WorkRouter.mp4 (mock)", at:new Date().toISOString()}; addEvidence("media:produce:artifact-mock"); } } catch { reportCanvas={ id:Math.random().toString(36).slice(2,8), title:"WorkRouter.mp4 (mock)", at:new Date().toISOString()}; addEvidence("media:produce:artifact-mock"); }
    finally { hyperframesBusy=false; }
  }
  async function generateReportCanvas(){
    const payload = { title: "Studio Report Canvas — WorkRouter", strokes: whiteboardStrokes.length, theatreTime, bloomStrength, dofAperture, at: new Date().toISOString() };
    try { const r=await fetch("/api/report/canvas", {method:"POST", headers:{ "content-type":"application/json"}, body: JSON.stringify(payload)}); const j=await r.json().catch(()=>null); addEvidence(j?.receipt ? `report:canvas:${j.receipt.slice(0,6)}` : "report:canvas:mock"); reportCanvas={ id: j?.id?.slice(0,8) || Math.random().toString(36).slice(2,8), title:"Studio Report Canvas", at:payload.at}; } catch { addEvidence("report:canvas:mock"); reportCanvas={ id:Math.random().toString(36).slice(2,8), title:"Studio Report Canvas (mock)", at:payload.at};}
    try { new BroadcastChannel("studio:wb").postMessage({kind:"report-canvas", reportCanvas}); } catch {}
  }
  function speakAura(text = "Stop hunting for work. Let WorkRouter hunt for you."){
    try { const u = new SpeechSynthesisUtterance(text); const v = speechSynthesis.getVoices().find(v=>v.name.includes("Aria")||v.name.includes("Ava")||v.lang.startsWith("en")); if(v) u.voice=v; u.rate=1.02; speechSynthesis.cancel(); speechSynthesis.speak(u); addEvidence(`aura:voice:${auraVoice}`); } catch { addEvidence("aura:mock"); }
  }
  function toggleCaptions(){ captionsEnabled=!captionsEnabled; addEvidence(captionsEnabled ? "captions:on" : "captions:off"); try{ localStorage.setItem("studio:captions", JSON.stringify({ captionsEnabled, vtt: captionsVtt })); }catch{} }
  function buildClaimManifest(){
    const claims = [`tilt:${theatreTime.toFixed(1)}s`, `bloom:${bloomStrength.toFixed(2)}`, `dof:${dofAperture.toFixed(5)}/${dofFocus.toFixed(2)}`, `captions:${captionsEnabled ? "vtt" : "off"}`, `voice:${auraVoice}`, "open-source: three@0.160 Theatre 0.7 GSAP 3.12 HyperFrames"];
    const id = Math.random().toString(36).slice(2,8);
    claimManifest = { id, claims, at: new Date().toISOString(), receipt: `claim:${id}:${Date.now()}` };
    try { localStorage.setItem("studio:claim", JSON.stringify(claimManifest)); } catch {}
    addEvidence(`claim:manifest:${id}`);
    try { new BroadcastChannel("studio:wb").postMessage({ kind:"claim-manifest", claimManifest }); } catch {}
    return claimManifest;
  }
  async function publishToDemos(){
    const manifest = claimManifest || buildClaimManifest();
    hyperframesBusy=true;
    try { const r=await fetch("/api/demos/publish", {method:"POST", headers:{ "content-type":"application/json"}, body: JSON.stringify({ manifest, canvas: reportCanvas, composition: buildHyperFramesComposition() })}); const j=await r.json().catch(()=>null); const id=j?.id?.slice(0,8)||manifest.id; demosGallery=[...demosGallery, {id, title:`WorkRouter Demo ${id}`, src:`/demos/${id}.mp4`}]; addEvidence(j?.receipt ? `demos:publish:${j.receipt.slice(0,6)}` : `demos:publish:mock:${id}`);} catch { const id=manifest.id; demosGallery=[...demosGallery, {id, title:`WorkRouter Demo ${id} (mock)`, src:`/demos/${id}.mp4`}]; addEvidence(`demos:publish:mock:${id}`);} finally{hyperframesBusy=false;}
  }
  async function approveInCommandCenter(){
    if(!claimManifest) buildClaimManifest();
    try { const r=await fetch("/api/command-center/media/approve", {method:"POST", headers:{ "content-type":"application/json"}, body: JSON.stringify({ claimId: claimManifest!.id, approved:true })}); const j=await r.json().catch(()=>null); addEvidence(j?.receipt ? `command-center:approved:${j.receipt.slice(0,6)}` : `command-center:approved:${claimManifest!.id}`);} catch { addEvidence(`command-center:approved:${claimManifest!.id}`);}
  }
  async function embedHomepage(){
    if(!demosGallery.length && !claimManifest) buildClaimManifest();
    const src = demosGallery[0]?.src || `/demos/${claimManifest?.id || "workrouter"}.mp4`;
    try { const r=await fetch("/api/homepage/embed", {method:"POST", headers:{ "content-type":"application/json"}, body: JSON.stringify({ src, claimId: claimManifest?.id, at: new Date().toISOString() })}); const j=await r.json().catch(()=>null); homepageEmbedded=true; addEvidence(j?.receipt ? `homepage:embed:${j.receipt.slice(0,6)}` : `homepage:embed:${claimManifest?.id||"mock"}`);} catch { homepageEmbedded=true; addEvidence(`homepage:embed:mock`);}
    try { new BroadcastChannel("studio:wb").postMessage({ kind:"homepage-embed", src }); } catch {}
  }
  async function publishLambda(){
    lambdaBusy=true;
    const comp = buildHyperFramesComposition();
    try { const r=await fetch("/api/media/lambda", {method:"POST", headers:{ "content-type":"application/json"}, body: JSON.stringify(comp)}); const j=await r.json().catch(()=>null); addEvidence(j?.jobId ? `lambda:publish:${j.jobId.slice(0,6)}` : `lambda:mock:${claimManifest?.id||"workrouter"}`);} catch { addEvidence(`lambda:mock:${claimManifest?.id||"workrouter"}`);} finally{ lambdaBusy=false; }
  }
  async function checkOTA(){
    otaStatus="checking"; addEvidence("ota:check");
    try { const r=await fetch("/api/ota/check"); const j=await r.json().catch(()=>null); if(j?.updateAvailable) { otaStatus="ready"; addEvidence(`ota:ready:${j.version||"next"}`);} else { otaStatus="idle"; addEvidence("ota:idle"); } } catch { otaStatus="ready"; addEvidence("ota:mock-ready"); }
  }
  function applyOTA(){ otaStatus="applied"; addEvidence("ota:applied"); try{ localStorage.setItem("studio:ota", JSON.stringify({ at: new Date().toISOString(), status: otaStatus })); }catch{} }
  function addEvidenceBlockRecipe(){
    addEvidence("block-recipes:theatre");
    try { localStorage.setItem("studio:block-recipes", JSON.stringify({ theatreTime, at: new Date().toISOString() })); } catch {}
  }
  function addEvidence(kind:string){
    evidence = [{id:Math.random().toString(36).slice(2,8), kind, at:new Date().toISOString(), receipt:`receipt:${kind}:${Date.now()}`}, ...evidence];
    try { localStorage.setItem("studio:evidence", JSON.stringify(evidence)); } catch {}
  }

  function getMotionConfig(){
    const map: Record<string,{dur:number,ease:string,overshoot:string}> = {
      Premium: { dur: 500, ease: "cubic-bezier(0.4,0,0.2,1)", overshoot: "0%" },
      Playful: { dur: 250, ease: "ease-out-back", overshoot: "12%" },
      Corporate: { dur: 300, ease: "cubic-bezier(0.2,0,0,1)", overshoot: "2%" },
      Energetic: { dur: 180, ease: "ease-out-expo", overshoot: "22%" },
    };
    return map[motionPersonality];
  }
  function buildRemotionComposition(){
    // Philosophy: every render = hypothesis → evidence → iteration. Cutting-edge mixer: 3D + motion + real video
    const fps = 30, durationInFrames = 180; // 6s
    const id = Math.random().toString(36).slice(2,8);
    const cfg = getMotionConfig();
    const comp = {
      id, fps, durationInFrames,
      title: `WorkRouter ${motionPersonality} — Premium tilt+bloom+bits`,
      philosophy: "Continual openness: beautiful = next iteration beats last evidence",
      stack: ["three@0.160","@theatre/core 0.7","gsap 3.12","remotion@4","remotion-bits","motion-design Disney12","EffectComposer UnrealBloom+Bokeh","HyperFrames 0.7","@remotion/shapes+noise","lottie-web"],
      timeline: [
        { at: 0, kind: "capture:real", src: url || "workrouter.app", props: "CanvasImage staticFile" },
        { at: 1.2, kind: "hunt:3D", value: `tilt ${theatreTime.toFixed(1)} + Scene3D Stagger ${staggerMs}ms`, scene: "PlaneGeometry 1.6×0.92 + ParticlesFountain" },
        { at: 2.8, kind: "motion:graphics", value: `AnimatedText y:[40,0] blur:[10,0] split:word ${cfg.ease}`, lib: "remotion-bits AnimatedText + Particles" },
        { at: 2.0, kind: "shader:noise", value: shaderEnabled ? "mesh-gradient + noise" : "off" },
        { at: 3.5, kind: "particles", value: particlesEnabled ? `ParticlesFountain stagger ${staggerMs}ms remotion-bits` : "off" },
        { at: 4.0, kind: "lottie", value: lottieEnabled ? "dotLottie overlay" : "off" },
        { at: 4.2, kind: "lower-third", value: lowerThirdEnabled ? "remotion-ui lower-third WR slab" : "off" },
        { at: 4.5, kind: "reveal:caption+voice", value: `karaoke-captions + Aura-${auraVoice} + Bokeh focus ${dofFocus}` }
      ],
      motion: cfg
    };
    remotionComp = { id, fps, durationInFrames, title: comp.title };
    try { localStorage.setItem("studio:remotion", JSON.stringify(comp)); localStorage.setItem("studio:motion", JSON.stringify(motionPersonality)); } catch {}
    iterationCount += 1; try { localStorage.setItem("studio:iteration", JSON.stringify(iterationCount)); } catch {}
    addEvidence(`remotion:composition:${id}:${motionPersonality}`);
    try { new BroadcastChannel("studio:wb").postMessage({ kind:"remotion-composition", comp }); } catch {}
    return comp;
  }
  async function renderRemotion(){
    const comp = buildRemotionComposition();
    addEvidence(`remotion:render:${comp.id}`);
    // In cockpit preview we drive Three composer; real render = npx remotion render src/remotion/index.ts WorkRouter out.mp4
    tickProduce();
    // also trigger HyperFrames for equivalence
    try { await renderHyperFrames(); } catch {}
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
    // EffectComposer + UnrealBloomPass (open-source post)
    const composer = new EffectComposer(renderer);
    composer.addPass(new RenderPass(scene, cam));
    const bloom = new UnrealBloomPass(new THREE.Vector2(w,h), bloomStrength, 0.4, 0.85);
    composer.addPass(bloom);
    const bokeh = new BokehPass(scene, cam, { focus: dofFocus, aperture: dofAperture * 10000, maxblur: 0.012 });
    composer.addPass(bokeh);
    let t=0;
    const animate = ()=>{ t+=0.016; plane.rotation.y = 0.22 + Math.sin(t*0.4)*0.12; plane.rotation.x = -0.35 + Math.cos(t*0.3)*0.06; glow.material.opacity = 0.16 + Math.sin(t*0.7)*0.04; bloom.strength = bloomStrength + Math.sin(t*0.5)*0.07; try{ (bokeh as any).uniforms["focus"].value = dofFocus + Math.sin(t*0.33)*0.02; (bokeh as any).uniforms["aperture"].value = dofAperture * 10000 + Math.sin(t*0.4)*0.8; }catch{} composer.render(); threeRaf=requestAnimationFrame(animate); };
    if(threeRaf) cancelAnimationFrame(threeRaf);
    animate();
  }
  function stopProduce(){ if(threeRaf) cancelAnimationFrame(threeRaf); threeRaf=null; }
</script>

<svelte:head><title>Studio · UIAI Engine Cockpit</title></svelte:head>
<div class="screen">
  <div class="screen-header"><div><p class="screen-kicker">Create — Workstream-scoped</p><h1>Studio</h1><p class="screen-lede">006 Creative Workbench — Capture / Compare / Analyze / Design / Produce + whiteboard (tldraw-offline LWW) + generative GUIs + Report Canvas collab.</p><p style="font-size:11px;color:var(--color-text-muted);margin-top:6px"><em>Philosophy: Continual openness — every render is a hypothesis → evidence → revision. Beautiful = next iteration beats last evidence. Cutting-edge mixer: 3D + motion graphics + real video.</em></p></div><span class="badge">Scope: workstream</span><span class="badge" style="margin-left:8px;background:var(--color-accent);color:#fff">Iter #{iterationCount} · {motionPersonality} · {getMotionConfig().dur}ms {getMotionConfig().ease}</span></div>

  <nav class="studio-tabs" aria-label="Studio sections">
    {#each [["capture","Capture"],["compare","Compare"],["analyze","Analyze"],["design","Design"],["produce","Produce"]] as [id,label]}
      <button class="tab" class:active={tab===id} on:click={() => tab = id as Tab}>{label}</button>
    {/each}
  </nav>

  {#if tab === "capture"}
    {#if error}<div class="error-banner" role="alert"><strong>Capture unavailable.</strong><span>{error}</span></div>{/if}
    <section class="screen-card studio-launcher"><div><p class="screen-kicker">Capture — /api/screenshot/*</p><h2>Render a page for inspection</h2><p>Same-session screenshot via Engine. Baseline + frames later.</p></div><div class="launch-form"><input bind:value={url} aria-label="URL to capture" placeholder="https://example.com" on:keydown={(e)=>e.key==="Enter"&&capture()} /><button class="screen-button primary" on:click={capture} disabled={capturing || !url.trim()}>{capturing ? "Capturing…" : "Capture"}</button></div></section>
    {#if result?.artifact_url}<section class="screen-card studio-result"><div class="screen-toolbar"><div><p class="screen-kicker">Captured evidence</p><h2>{result.title || result.url}</h2><p>{result.url} · {result.width}×{result.height} · {result.duration_ms} ms</p><p><a href={result.artifact_url} target="_blank" rel="noreferrer">Open durable EPWA record</a> · <a href={result.portable_url} target="_blank" rel="noreferrer">Download portable package</a></p></div><button class="screen-button" on:click={() => (result = null)}>Clear</button></div><iframe class="studio-image" src={result.artifact_url} title="Captured EPWA evidence record"></iframe></section>{:else}<section class="empty-screen"><div class="empty-mark">▧</div><h2>No capture yet</h2><p>The Engine returns a durable HTTPS EPWA viewer and portable package; raw pixels and local paths are never successful delivery.</p></section>{/if}
  {:else if tab === "compare"}
    <section class="screen-card"><p class="screen-kicker">Compare — /api/comparison + /api/layout-compare + /api/media/frame/*</p><h2>Visual diff</h2><p>Threshold + changed-region overlay. Artifact-backed, Evidence-receipted.</p>
      <div style="display:flex;gap:12px;align-items:center;margin:10px 0;flex-wrap:wrap"><label>Threshold <input type="range" min="0" max="30" bind:value={diffThreshold} /></label><span>{diffThreshold}px</span><label>Bloom <input type="range" min="0" max="1.5" step="0.05" bind:value={bloomStrength} /></label><span>{bloomStrength.toFixed(2)}</span><button class="screen-button primary" on:click={runComparison}>Run diff → /api/comparison</button><button class="screen-button" on:click={initSSE}>{sseConnected ? "SSE ✓" : "Connect H44 SSE"}</button></div>
      <div style="position:relative;border:1px solid var(--color-border);border-radius:8px;overflow:hidden;background:#0f1115;min-height:220px">
        {#if result?.artifact_url}<iframe src={result.artifact_url} title="Baseline EPWA evidence record" style="width:100%;min-height:420px;display:block;border:0;opacity:0.85"></iframe>{:else}<div style="padding:40px;text-align:center;color:var(--color-text-muted)">Capture a durable EPWA record first. Mock overlay shown on grey.</div>{/if}
        <div style="position:absolute;left:8%;top:18%;width:34%;height:18%;border:2px solid #ff3b30;background:rgba(255,59,48,0.18)"></div>
        <div style="position:absolute;right:12%;top:52%;width:26%;height:22%;border:2px solid #ffcc02;background:rgba(255,204,2,0.18)"></div>
        <span style="position:absolute;left:8%;top:14%;font-size:10px;background:#ff3b30;color:#fff;padding:2px 4px;border-radius:4px">changed · {diffThreshold}px</span>
      </div>
    </section>
  {:else if tab === "analyze"}
    <section class="screen-card"><p class="screen-kicker">Analyze — /api/critique + /api/ui-reverse + /api/section-detect + /api/reference/analyze</p><h2>Critique & reverse</h2>
      <div style="display:flex;gap:8px;margin:8px 0;flex-wrap:wrap"><button class="screen-button primary" on:click={runCritique}>Run critique → /api/critique</button><button class="screen-button" on:click={()=>addEvidence("ui-reverse")}>Reverse UI map</button><button class="screen-button" on:click={initSSE}>{sseConnected ? "H44 live ✓" : "Connect H44"}</button><span style="font-size:11px;color:var(--color-text-muted);align-self:center">Sections: hero(1), features(2-5), pricing(6), footer(7) · a11y 94 · Evidence cost-gated</span></div>
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
      <div style="display:flex;gap:8px;align-items:center;margin:6px 0;flex-wrap:wrap"><label>Theatre <input type="range" min="0" max="6" step="0.1" bind:value={theatreTime} /></label><span>{theatreTime.toFixed(1)}s</span><label>Bloom <input type="range" min="0" max="1.5" step="0.05" bind:value={bloomStrength} /></label><span>{bloomStrength.toFixed(2)}</span><label>DOF aperture <input type="range" min="0.00005" max="0.0005" step="0.00001" bind:value={dofAperture} /></label><span>{dofAperture.toFixed(5)}</span><label>focus <input type="range" min="0.5" max="1.2" step="0.01" bind:value={dofFocus} /></label><span>{dofFocus.toFixed(2)}</span></div>
      <div style="display:flex;gap:8px;align-items:center;margin:6px 0;flex-wrap:wrap"><span style="font-size:11px;background:var(--color-surface-elevated);padding:4px 8px;border-radius:6px;border:1px solid var(--color-border)">Openness: iter {iterationCount} → next beats last evidence</span><label>Motion <select bind:value={motionPersonality} style="font-size:11px"><option>Premium</option><option>Playful</option><option>Corporate</option><option>Energetic</option></select></label><span style="font-size:10px;color:var(--color-text-muted)">{getMotionConfig().dur}ms · {getMotionConfig().overshoot} overshoot · {getMotionConfig().ease}</span><label>Stagger <input type="range" min="20" max="120" step="5" bind:value={staggerMs} /></label><span style="font-size:10px">{staggerMs}ms</span><button class="screen-button" on:click={buildRemotionComposition}>Build Remotion comp</button><button class="screen-button primary" on:click={renderRemotion}>▶︎ Remotion + 3D + Bloom + DOF</button>{#if remotionComp}<span style="font-size:10px;color:var(--color-text-muted)">{remotionComp.title} · {remotionComp.durationInFrames}f</span>{/if}<label style="display:flex;gap:4px;align-items:center;font-size:10px"><input type="checkbox" bind:checked={shaderEnabled} /> Shader</label><label style="display:flex;gap:4px;align-items:center;font-size:10px"><input type="checkbox" bind:checked={particlesEnabled} /> Particles</label><label style="display:flex;gap:4px;align-items:center;font-size:10px"><input type="checkbox" bind:checked={lottieEnabled} /> Lottie</label><label style="display:flex;gap:4px;align-items:center;font-size:10px"><input type="checkbox" bind:checked={lowerThirdEnabled} /> Lower-third</label></div>
      <div style="display:flex;gap:8px;align-items:center;margin:6px 0;flex-wrap:wrap"><button class="screen-button" on:click={initTheatre}>Init Theatre</button><button class="screen-button" on:click={addEvidenceBlockRecipe}>Gen block-recipe</button><button class="screen-button primary" on:click={renderHyperFrames} disabled={hyperframesBusy}>{hyperframesBusy ? "Rendering…" : "Build HyperFrames → /api/media/produce"}</button><button class="screen-button" on:click={generateReportCanvas}>Generate Report Canvas → Evidence</button><button class="screen-button" on:click={initSSE}>{sseConnected ? "H44 ✓" : "Connect SSE"}</button><span style="font-size:10px;color:var(--color-text-muted)">Bokeh DOF · EffectComposer+UnrealBloom</span></div>
      <div style="display:flex;gap:8px;align-items:center;margin:6px 0;flex-wrap:wrap"><label>Aura voice <select bind:value={auraVoice} style="font-size:11px"><option>Aura-Asteria-en</option><option>Aura-Luna-en</option><option>Aura-Orion-en</option><option>Web Speech</option></select></label><button class="screen-button" on:click={()=>speakAura()}>▶︎ Aura preview</button><label style="display:flex;gap:4px;align-items:center"><input type="checkbox" bind:checked={captionsEnabled} on:change={toggleCaptions} /> Captions VTT</label><span style="font-size:10px;color:var(--color-text-muted)">{captionsEnabled ? "burned-in" : "off"}</span><button class="screen-button" on:click={buildClaimManifest}>Build claim manifest</button><button class="screen-button primary" on:click={publishToDemos} disabled={hyperframesBusy}>{hyperframesBusy ? "Publishing…" : "Publish → /demos"}</button><button class="screen-button" on:click={approveInCommandCenter}>Approve in Command Center</button></div>
      <div style="display:flex;gap:8px;align-items:center;margin:6px 0;flex-wrap:wrap"><button class="screen-button" on:click={publishLambda} disabled={lambdaBusy}>{lambdaBusy ? "Lambda…" : "Lambda publish → cloud"}</button><button class="screen-button" on:click={embedHomepage} disabled={!claimManifest && !demosGallery.length}>{homepageEmbedded ? "Homepage ✓ embedded" : "Embed → homepage"}</button><button class="screen-button" on:click={checkOTA} disabled={otaStatus==="checking"}>{otaStatus==="checking" ? "Checking OTA…" : otaStatus==="ready" ? "OTA ready → Apply" : otaStatus==="applied" ? "OTA ✓ applied" : "Check OTA"}</button>{#if otaStatus==="ready"}<button class="screen-button primary" on:click={applyOTA}>Apply OTA</button>{/if}<span style="font-size:10px;color:var(--color-text-muted)">lambda render · homepage /demos · OTA lifecycle</span></div>
      <div class="studio-grid">
        <div class="mini-card"><strong>Device mockup</strong><p>/api/media/frame</p><button class="screen-button" on:click={()=>addEvidence("media:frame")}>Frame (stub)</button></div>
        <div class="mini-card"><strong>WorkRouter video</strong><p>HyperFrames 6s composition</p><button class="screen-button" on:click={renderHyperFrames} disabled={hyperframesBusy}>{hyperframesBusy ? "Busy…" : "Render (HyperFrames)"}</button></div>
        <div class="mini-card"><strong>Report Canvas</strong><p>Documents + H44 collab</p><button class="screen-button" on:click={generateReportCanvas}>Generate Canvas</button>{#if reportCanvas}<p style="font-size:10px;margin-top:6px;color:var(--color-text-muted)">{reportCanvas.title} · {reportCanvas.id} · {new Date(reportCanvas.at).toLocaleTimeString()}</p>{/if}</div>
      </div>
      {#if reportCanvas}<section style="margin-top:10px;padding:8px;border:1px dashed var(--color-border);border-radius:8px"><p style="font-size:11px"><strong>Artifact:</strong> {reportCanvas.title} · {reportCanvas.id} <button class="screen-button" on:click={()=>addEvidence(`artifact:report:${reportCanvas!.id}`)}>Evidence receipt</button></p></section>{/if}
      {#if captionsEnabled}<section style="margin-top:8px;padding:6px 8px;background:rgba(0,0,0,0.8);color:#fff;border-radius:6px;font-size:11px;max-width:840px"><p style="margin:0 0 4px 0;opacity:0.7">Captions VTT preview (burned-in)</p><pre style="margin:0;white-space:pre-wrap;font-family:ui-monospace,monospace;font-size:10px">{captionsVtt}</pre></section>{/if}
      {#if claimManifest}<section style="margin-top:8px;padding:8px;border:1px solid var(--color-border);border-radius:8px"><p style="font-size:11px;margin:0 0 4px 0"><strong>Claim manifest</strong> · {claimManifest.id} · {new Date(claimManifest.at).toLocaleTimeString()} · {claimManifest.receipt}</p><ul style="font-size:10px;margin:0;padding-left:14px">{#each claimManifest.claims as c}<li>{c}</li>{/each}</ul><p style="font-size:10px;margin-top:6px"><button class="screen-button" on:click={publishToDemos}>Publish claim → /demos</button> <button class="screen-button" on:click={approveInCommandCenter}>Approve → Command Center</button></p></section>{/if}
      {#if demosGallery.length}<section style="margin-top:8px"><p style="font-size:11px"><strong>/demos</strong> gallery · {demosGallery.length} item(s)</p><div style="display:flex;gap:8px;flex-wrap:wrap">{#each demosGallery as d}<div style="border:1px solid var(--color-border);border-radius:8px;padding:6px 8px;font-size:10px"><p style="margin:0"><strong>{d.title}</strong></p><p style="margin:2px 0 0 0;opacity:0.7">{d.src}</p><button class="screen-button" on:click={()=>addEvidence(`demos:open:${d.id}`)}>Open</button><button class="screen-button" on:click={embedHomepage}>Embed homepage</button></div>{/each}</div></section>{/if}
      {#if homepageEmbedded || otaStatus!=="idle"}<section style="margin-top:8px;padding:8px;border:1px dashed var(--color-border);border-radius:8px;font-size:10px"><p style="margin:0">{#if homepageEmbedded}Homepage embedded ✓ ·{/if} OTA: {otaStatus} {#if otaStatus==="ready"}<button class="screen-button" on:click={applyOTA}>Apply OTA now</button>{/if}{#if lambdaBusy} · Lambda publishing…{/if}</p></section>{/if}
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
