<script lang="ts">
  import { onMount } from "svelte";
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
  let scope: ReturnType<typeof savedScope> = {}, packets: EvidenceSharePacket[] = [];
  let manifests: Record<string, EvidenceShareManifest> = {}, verifications: Record<string, EvidenceShareVerification> = {}, detailLoading: Record<string, boolean> = {};

  onMount(async () => {
    scope = savedScope(); const requested = new URLSearchParams(window.location.search).get("view");
    if (requested && views.some((view) => view.id === requested)) activeView = requested;
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
  $: activeLabel = views.find((view) => view.id === activeView)?.label || "Current Workpoint";
  $: visiblePackets = activeView === "recent" ? packets : activeView === "current" ? packets.filter((packet) => !!scope.workpoint_id && packet.workpoint_ref === scope.workpoint_id) : activeView === "verified" ? packets.filter((packet) => verifications[packet.packet_id]?.valid) : activeView === "public-safe" ? packets.filter((packet) => manifests[packet.packet_id]?.access === "public_safe") : [];
</script>

<svelte:head><title>Evidence · UIAI Engine Cockpit</title></svelte:head>
<div class="screen evidence-screen">
  <div class="screen-header"><div><p class="screen-kicker">Prove</p><h1>Evidence</h1><p class="screen-lede">Beautiful, portable Screenshot Evidence Share Packets—visual first, provenance on demand.</p></div><span class:success={health?.status === "healthy"} class="badge">{loading ? "Loading evidence" : `${packets.length} packet${packets.length === 1 ? "" : "s"}`}</span></div>
  <WorkspaceViewTabs label="Evidence saved views" route="/evidence" {views} active={activeView} />
  {#if error}<div class="error-banner" role="alert"><strong>Evidence unavailable.</strong><span>{error}</span></div>{/if}
  {#if loading}<section class="loading-grid" aria-label="Loading evidence"><div></div><div></div></section>
  {:else if visiblePackets.length}<section class="packet-grid" aria-label={`${activeLabel} evidence packets`}>{#each visiblePackets as packet (packet.packet_id)}<EvidenceShareCard {packet} manifest={manifests[packet.packet_id]} verification={verifications[packet.packet_id]} detailLoading={detailLoading[packet.packet_id]} inspect={() => inspect(packet.packet_id)} />{/each}</section>
  {:else}<section class="empty-screen"><div class="empty-mark">◇</div><h2>No {activeLabel.toLowerCase()} share packets</h2><p>{scope.project_root && scope.continuity_id ? "This view remains empty until canonical packet scope or verification matches it." : "Connect a project and Workpoint to filter packet scope. Recent remains available without inferred bindings."}</p><div class="screen-actions"><a class="screen-button primary" href="/evidence?view=recent">View recent packets</a><a class="screen-button" href="/settings?section=scope">Review connection</a></div></section>{/if}
  <section class="screen-card pad evidence-contract"><p class="screen-kicker">Truth boundary</p><h2>Evidence is not completion</h2><p>Packets prove bounded captured states. Review, provider closure, and settlement remain separate governed decisions.</p></section>
</div>
<style>
.evidence-screen{display:grid;gap:20px}.packet-grid{display:grid;gap:20px}.error-banner{display:flex;align-items:center;gap:10px;padding:12px 14px;border:1px solid color-mix(in srgb,var(--color-error) 25%,var(--color-border));border-radius:9px;color:var(--color-error);background:color-mix(in srgb,var(--color-error) 7%,transparent);font-size:12px}.error-banner span,.evidence-contract p{color:var(--color-text-muted)}.loading-grid{display:grid;grid-template-columns:1fr 1fr;gap:20px}.loading-grid div{min-height:280px;border-radius:16px;background:linear-gradient(110deg,var(--color-surface),color-mix(in srgb,var(--color-surface) 70%,var(--color-border)),var(--color-surface));background-size:200% 100%;animation:loading 1.3s linear infinite}@keyframes loading{to{background-position:-200% 0}}.evidence-contract h2{margin:0;font-size:17px}.evidence-contract p{font-size:13px;line-height:1.55}@media(max-width:680px){.loading-grid{grid-template-columns:1fr}}@media(prefers-reduced-motion:reduce){.loading-grid div{animation:none}}
</style>
