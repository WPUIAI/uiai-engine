import type { FocusaDaemonCandidateV1 } from "$lib/contracts/focusa-pairing";
import type { FocusaDiscoveryContext, FocusaDiscoverySourceAdapter } from "./focusa-discovery-source";
import { fetchFocusaHealth, focusaCandidateId, type FocusaHealthProbe } from "./focusa-direct-sources";

const MAGIC_DNS_NAME = /^(?=.{1,253}$)(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/;

export class TailscaleDiscoverySource implements FocusaDiscoverySourceAdapter {
  readonly source = "tailscale" as const;
  constructor(
    private readonly approvedNames: readonly string[],
    private readonly port = 8787,
    private readonly probe: FocusaHealthProbe = fetchFocusaHealth,
  ) {}

  async discover(context: Readonly<FocusaDiscoveryContext>): Promise<FocusaDaemonCandidateV1[]> {
    if (context.localOnly || context.signal?.aborted) return [];
    if (!Number.isInteger(this.port) || this.port < 1 || this.port > 65_535) return [];
    const names = [...new Set(this.approvedNames.map((name) => name.trim().toLowerCase()))]
      .filter((name) => MAGIC_DNS_NAME.test(name) && !name.endsWith(".local"))
      .slice(0, 16);
    const candidates: FocusaDaemonCandidateV1[] = [];
    for (const name of names) {
      if (context.signal?.aborted) break;
      const baseUrl = `https://${name}:${this.port}`;
      const controller = new AbortController();
      const abort = () => controller.abort();
      context.signal?.addEventListener("abort", abort, { once: true });
      const health = await this.probe(baseUrl, controller.signal);
      context.signal?.removeEventListener("abort", abort);
      candidates.push({
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
      });
    }
    return candidates;
  }
}
