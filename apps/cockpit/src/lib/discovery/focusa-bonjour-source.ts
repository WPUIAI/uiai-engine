import type { FocusaDaemonCandidateV1 } from "$lib/contracts/focusa-pairing";
import type { FocusaDiscoveryContext, FocusaDiscoverySourceAdapter } from "./focusa-discovery-source";
import { fetchFocusaHealth, focusaCandidateId, type FocusaHealthProbe } from "./focusa-direct-sources";

export interface BonjourFocusaService { url: string; host: string; port: number }
export type BonjourDiscover = (timeoutSecs: number) => Promise<BonjourFocusaService | null>;

function validateService(service: BonjourFocusaService): string {
  if (!Number.isInteger(service.port) || service.port < 1 || service.port > 65_535) throw new Error("Bonjour port invalid");
  const host = service.host.replace(/\.$/, "").toLowerCase();
  if (!/^[a-z0-9][a-z0-9.-]*\.local$/.test(host)) throw new Error("Bonjour host invalid");
  const url = new URL(service.url);
  if (url.protocol !== "http:" || url.hostname !== host || Number(url.port) !== service.port || !["", "/"].includes(url.pathname) || url.username || url.password || url.search || url.hash) throw new Error("Bonjour endpoint mismatch");
  return url.origin;
}

export class BonjourDiscoverySource implements FocusaDiscoverySourceAdapter {
  readonly source = "bonjour" as const;
  constructor(private readonly discoverBonjour: BonjourDiscover, private readonly probe: FocusaHealthProbe = fetchFocusaHealth) {}

  async discover(context: Readonly<FocusaDiscoveryContext>): Promise<FocusaDaemonCandidateV1[]> {
    if (context.localOnly || context.signal?.aborted) return [];
    const service = await this.discoverBonjour(2);
    if (!service) return [];
    let baseUrl: string;
    try { baseUrl = validateService(service); } catch { return []; }
    const controller = new AbortController();
    const abort = () => controller.abort();
    context.signal?.addEventListener("abort", abort, { once: true });
    const health = await this.probe(baseUrl, controller.signal);
    context.signal?.removeEventListener("abort", abort);
    return [{
      schema: "focusa.daemon_candidate.v1",
      candidate_id: focusaCandidateId(this.source, baseUrl),
      base_url: baseUrl,
      source: this.source,
      location: "remote",
      observed_at: context.observedAt,
      health_status: health.healthy ? "healthy" : "unavailable",
      latency_ms: Math.max(0, health.latencyMs),
      daemon_id: health.daemonId,
      machine_id: health.machineId,
      version: health.version,
      capabilities: health.capabilities,
    }];
  }
}
