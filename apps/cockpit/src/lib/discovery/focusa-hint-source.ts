import { parseFocusaPlatformHint, type FocusaDaemonCandidateV1 } from "$lib/contracts/focusa-pairing";
import type { FocusaDiscoveryContext, FocusaDiscoverySourceAdapter } from "./focusa-discovery-source";
import {
  fetchFocusaHealth,
  focusaCandidateId,
  normalizeFocusaDaemonBaseUrl,
  type FocusaHealthProbe,
} from "./focusa-direct-sources";

export type FocusaHintProvider = () => Promise<unknown[]> | unknown[];

/** Combines non-secret sibling-platform and saved hints as advisory discovery only. */
export class SiblingAndSavedHintDiscoverySource implements FocusaDiscoverySourceAdapter {
  readonly source = "saved_hint" as const;

  constructor(
    private readonly siblingHints: FocusaHintProvider,
    private readonly savedHints: FocusaHintProvider,
    private readonly probe: FocusaHealthProbe = fetchFocusaHealth,
  ) {}

  async discover(context: Readonly<FocusaDiscoveryContext>): Promise<FocusaDaemonCandidateV1[]> {
    const raw = [...await this.siblingHints(), ...await this.savedHints()];
    const endpoints = new Map<string, ReturnType<typeof parseFocusaPlatformHint>>();
    for (const value of raw) {
      try {
        const hint = parseFocusaPlatformHint(value);
        const endpoint = normalizeFocusaDaemonBaseUrl(hint.daemon_url, false);
        if (context.localOnly && endpoint.location !== "local") continue;
        endpoints.set(endpoint.baseUrl, hint);
      } catch {
        // Invalid, stale, and secret-bearing hints are ignored; they grant no authority.
      }
    }

    const candidates: FocusaDaemonCandidateV1[] = [];
    for (const [baseUrl, hint] of endpoints) {
      if (context.signal?.aborted) break;
      const endpoint = normalizeFocusaDaemonBaseUrl(baseUrl, false);
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
        location: endpoint.location,
        observed_at: context.observedAt,
        health_status: health.healthy ? "healthy" : "unavailable",
        latency_ms: Math.max(0, health.latencyMs),
        daemon_id: health.daemonId ?? hint.daemon_id,
        machine_id: health.machineId,
        version: health.version,
        capabilities: health.capabilities,
      });
    }
    return candidates;
  }
}
