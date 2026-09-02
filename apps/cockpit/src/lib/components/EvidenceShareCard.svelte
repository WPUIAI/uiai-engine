<script lang="ts">
  import { humanBytes, sourceHost, type EvidenceShareManifest, type EvidenceSharePacket, type EvidenceShareVerification } from "$lib/evidence-share";
  export let packet: EvidenceSharePacket;
  export let manifest: EvidenceShareManifest | undefined = undefined;
  export let verification: EvidenceShareVerification | undefined = undefined;
  export let detailLoading = false;
  export let inspect: () => void;
  let expanded = false;
  function toggle() { expanded = !expanded; if (expanded && !manifest && !detailLoading) inspect(); }
</script>

<article class="share-card" data-status={manifest?.availability || "ready"}>
  <a class="preview" href={packet.artifact_url} target="_blank" rel="noopener noreferrer" aria-label={`View ${packet.descriptor}`}><img src={packet.thumbnail_url} alt="Screenshot evidence preview" loading="lazy" decoding="async" /></a>
  <div class="body">
    <div class="topline"><span class:valid={verification?.valid} class="status"><i></i>{verification ? (verification.valid ? "Verified bytes" : "Verification issue") : "Evidence ready"}</span><time datetime={packet.captured_at}>{new Date(packet.captured_at).toLocaleString()}</time></div>
    <h2>{packet.descriptor}</h2><p class="source">{sourceHost(packet.source_url)}</p>
    <div class="actions"><a class="primary" href={packet.artifact_url} target="_blank" rel="noopener noreferrer">View evidence <span>↗</span></a><button type="button" aria-expanded={expanded} on:click={toggle}>{expanded ? "Hide details" : "Inspect"}</button></div>
    {#if expanded}<div class="details" aria-live="polite">
      {#if detailLoading}<p>Loading packet details…</p>{:else if manifest}
        <dl><div><dt>Dimensions</dt><dd>{manifest.width} × {manifest.height}</dd></div><div><dt>Format</dt><dd>{manifest.format.toUpperCase()} · {humanBytes(manifest.bytes)}</dd></div><div><dt>Workpoint</dt><dd>{manifest.scope?.workpoint_ref || "Not bound"}</dd></div><div><dt>Packet SHA-256</dt><dd><code>{manifest.artifact_sha256}</code></dd></div></dl>
        <p class="truth">{manifest.truth_notice}</p><a class="json" href={`${packet.artifact_url}artifact.json`} target="_blank" rel="noopener noreferrer">Machine JSON ↗</a>
      {:else}<p>Packet details are unavailable. The visual link remains read-only.</p>{/if}
    </div>{/if}
  </div>
</article>

<style>
.share-card{display:grid;grid-template-columns:minmax(220px,38%) 1fr;overflow:hidden;border:1px solid var(--color-border);border-radius:16px;background:var(--color-surface);box-shadow:0 18px 50px color-mix(in srgb,var(--color-text) 8%,transparent)}.preview{display:block;min-height:220px;background:#141712;overflow:hidden}.preview img{width:100%;height:100%;object-fit:cover;object-position:top}.body{padding:clamp(18px,3vw,30px);min-width:0}.topline,.actions{display:flex;align-items:center;justify-content:space-between;gap:12px}.topline time,.source{font-size:12px;color:var(--color-text-muted)}.status{display:flex;align-items:center;gap:7px;font-size:11px;font-weight:750;text-transform:uppercase;letter-spacing:.08em}.status i{width:8px;height:8px;border-radius:50%;background:#d79d25}.status.valid i{background:#48a749}.body h2{margin:18px 0 5px;font-size:clamp(20px,3vw,30px);letter-spacing:-.04em}.source{margin:0 0 22px}.actions{justify-content:flex-start}.actions a,.actions button{border:1px solid var(--color-border);border-radius:9px;padding:9px 12px;background:transparent;color:var(--color-text);font:inherit;font-size:12px;text-decoration:none;cursor:pointer}.actions .primary{background:var(--color-text);color:var(--color-bg);font-weight:700}.details{margin-top:20px;padding-top:18px;border-top:1px solid var(--color-border)}dl{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:1px;background:var(--color-border)}dl div{padding:10px;background:var(--color-surface)}dt{font-size:10px;text-transform:uppercase;letter-spacing:.08em;color:var(--color-text-muted)}dd{margin:5px 0 0;font-size:12px;overflow-wrap:anywhere}.truth{font-size:12px;line-height:1.55;color:var(--color-text-muted)}.json{font-size:12px}@media(max-width:680px){.share-card{grid-template-columns:1fr}.preview{min-height:180px;max-height:58vh}.topline{align-items:flex-start;flex-direction:column}dl{grid-template-columns:1fr}}@media(prefers-reduced-motion:reduce){*{scroll-behavior:auto}}
</style>
