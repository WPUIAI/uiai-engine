<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import { page } from "$app/stores";
  import "$lib/ui/screen.css";
  import { engineClient, savedScope, type EngineHealth } from "$lib/engine-client";
  import type { EvidenceShareManifest, EvidenceSharePacket, EvidenceShareVerification } from "$lib/evidence-share";
  import EvidenceShareCard from "$lib/components/EvidenceShareCard.svelte";
  import WorkspaceViewTabs from "$lib/ui/WorkspaceViewTabs.svelte";

  const views = [
    { id: "current", label: "Current Workpoint" }, { id: "recent", label: "Recent" },
    { id: "needs-capture", label: "Needs capture" }, { id: "needs-review", label: "Needs review" },
    { id: "verified", label: "Verified" }, { id: "provisional", label: "Provisional / Surrogate" },
    { id: "public-safe", label: "Public-safe" }, { id: "receipts", label: "Receipts" }, { id: "reports", label: "Reports" },
  ] as const;
  let health: EngineHealth | null = null, error = "", loading = true, activeView = "current";
  let metadataLoading = false, metadataLoaded = false, destroyed = false;
  $: requestedView = $page.url.searchParams.get("view") || "current";
  $: activeView = views.some((view) => view.id === requestedView) ? requestedView : "current";
  $: unsupportedView = activeView === "needs-capture" || activeView === "needs-review";
  $: if (!loading && !unsupportedView && !["current", "recent"].includes(activeView) && !metadataLoading && !metadataLoaded) void loadFilterMetadata();
  onDestroy(() => { destroyed = true; });
  let scope: ReturnType<typeof savedScope> = {}, packets: EvidenceSharePacket[] = [];
  let manifests: Record<string, EvidenceShareManifest> = {}, verifications: Record<string, EvidenceShareVerification> = {}, detailLoading: Record<string, boolean> = {};

  onMount(async () => {
    scope = savedScope();
    try { [health, { packets }] = await Promise.all([engineClient.health(), engineClient.evidenceShares()]); }
    catch (cause) { error = cause instanceof Error ? cause.message : "The engine could not be reached."; }
    finally { loading = false; }
  });
  async function inspect(packetId: string) {
    detailLoading = { ...detailLoading, [packetId]: true };
    try { const [manifest, verification] = await Promise.all([engineClient.evidenceShare(packetId), engineClient.verifyEvidenceShare(packetId)]); manifests = { ...manifests, [packetId]: manifest }; verifications = { ...verifications, [packetId]: verification }; }
    catch (cause) { error = cause instanceof Error ? cause.message : "Packet details are unavailable."; }
    finally { detailLoading = { ...detailLoading, [packetId]: false }; }
  }
  async function loadFilterMetadata() {
    metadataLoading = true;
    const pending = packets.filter((packet) => !manifests[packet.packet_id] || !verifications[packet.packet_id]);
    await Promise.all(Array.from({ length: Math.min(3, pending.length) }, async () => {
      while (!destroyed && pending.length) {
        const packet = pending.shift();
        if (packet) await inspect(packet.packet_id);
      }
    }));
    metadataLoaded = true;
    metadataLoading = false;
  }
  $: activeLabel = views.find((view) => view.id === activeView)?.label || "Current Workpoint";
  $: currentWorkpoint = scope.workpoint_ref || scope.workpoint_id;
  $: visiblePackets = packets.filter((packet) => {
    const manifest = manifests[packet.packet_id];
    switch (activeView) {
      case "recent": return true;
      case "current": return !!currentWorkpoint && packet.workpoint_ref === currentWorkpoint;
      case "verified": return verifications[packet.packet_id]?.valid === true;
      case "public-safe": return manifest?.access === "public_safe";
      case "provisional": return ["provisional", "surrogate"].includes(manifest?.availability || "");
      case "receipts": return manifest?.kind === "receipt";
      case "reports": return manifest?.kind === "report";
      default: return false;
    }
  });
</script>

<svelte:head><title>Evidence · UIAI Engine Cockpit</title></svelte:head>
<div class="screen evidence-screen">
  <div class="screen-header"><div><p class="screen-kicker">Prove</p><h1>Evidence</h1><p class="screen-lede">Read-only evidence packages, portable downloads, and provenance on demand. The index shows up to 100 packages.</p></div><span class:success={health?.status === "healthy"} class="badge">{loading ? "Loading evidence" : `${packets.length} packet${packets.length === 1 ? "" : "s"}`}</span></div>
  <WorkspaceViewTabs label="Evidence saved views" route="/evidence" {views} active={activeView} />
  {#if error}<div class="error-banner" role="alert"><strong>Evidence unavailable.</strong><span>{error}</span></div>{/if}
  {#if unsupportedView}<section class="empty-screen"><h2>{activeLabel} requirements are not in the package index</h2><p>Use the project Work Items and review requirements in Focusa. Package presence and integrity checks do not establish missing captures or review acceptance.</p></section>
  {:else if loading || metadataLoading}<section class="loading-grid" aria-label="Loading evidence"><div></div><div></div></section>
  {:else if visiblePackets.length}<section class="packet-grid" aria-label={`${activeLabel} evidence packets`}>{#each visiblePackets as packet (packet.packet_id)}<EvidenceShareCard {packet} manifest={manifests[packet.packet_id]} verification={verifications[packet.packet_id]} detailLoading={detailLoading[packet.packet_id]} inspect={() => inspect(packet.packet_id)} />{/each}</section>
  {:else}<section class="empty-screen"><div class="empty-mark">◇</div><h2>No {activeLabel.toLowerCase()} share packets</h2><p>{scope.project_root && scope.continuity_id ? "This view remains empty until canonical packet scope or verification matches it." : "Connect a project and Workpoint to filter packet scope. Recent remains available without inferred bindings."}</p><div class="screen-actions"><a class="screen-button primary" href="/evidence?view=recent">View recent packets</a><a class="screen-button" href="/settings?section=scope">Review connection</a></div></section>{/if}
  <section class="screen-card pad evidence-contract"><p class="screen-kicker">Truth boundary</p><h2>Evidence is not completion</h2><p>Packets prove bounded captured states. Integrity checks cover package bytes, not independent review acceptance. Review, provider closure, and settlement remain separate governed decisions.</p></section>
</div>
<style>
.evidence-screen{display:grid;gap:20px}.packet-grid{display:grid;gap:20px}.error-banner{display:flex;align-items:center;gap:10px;padding:12px 14px;border:1px solid color-mix(in srgb,var(--color-error) 25%,var(--color-border));border-radius:9px;color:var(--color-error);background:color-mix(in srgb,var(--color-error) 7%,transparent);font-size:12px}.error-banner span,.evidence-contract p{color:var(--color-text-muted)}.loading-grid{display:grid;grid-template-columns:1fr 1fr;gap:20px}.loading-grid div{min-height:280px;border-radius:16px;background:linear-gradient(110deg,var(--color-surface),color-mix(in srgb,var(--color-surface) 70%,var(--color-border)),var(--color-surface));background-size:200% 100%;animation:loading 1.3s linear infinite}@keyframes loading{to{background-position:-200% 0}}.evidence-contract h2{margin:0;font-size:17px}.evidence-contract p{font-size:13px;line-height:1.55}@media(max-width:680px){.loading-grid{grid-template-columns:1fr}}@media(prefers-reduced-motion:reduce){.loading-grid div{animation:none}}
</style>
