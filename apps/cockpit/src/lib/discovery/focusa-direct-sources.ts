import type { FocusaDaemonCandidateV1 } from "$lib/contracts/focusa-pairing";
import type { FocusaDiscoveryContext, FocusaDiscoverySourceAdapter } from "./focusa-discovery-source";

const PROBE_TIMEOUT_MS = 1_500;

export interface FocusaHealthProbeResult {
  healthy: boolean;
  latencyMs: number;
  daemonId?: string;
  machineId?: string;
  version?: string;
  capabilities?: string[];
}
export type FocusaHealthProbe = (baseUrl: string, signal: AbortSignal) => Promise<FocusaHealthProbeResult>;

function normalizeBaseUrl(input: string): { baseUrl: string; location: "local" | "remote" } {
  if (input.length > 2048) throw new Error("daemon URL is oversized");
  const authority = input.match(/^[a-z]+:\/\/([^/?#]+)/i)?.[1] ?? "";
  if (!/:\d+$/.test(authority)) throw new Error("daemon URL requires an explicit port");
  const url = new URL(input);
  if (url.username || url.password || url.search || url.hash || !["", "/"].includes(url.pathname)) throw new Error("daemon URL contains forbidden components");
  const local = ["127.0.0.1", "localhost", "[::1]"].includes(url.hostname);
  if ((local && url.protocol !== "http:") || (!local && url.protocol !== "https:")) throw new Error("daemon URL transport is invalid");
  return { baseUrl: url.origin, location: local ? "local" : "remote" };
}

function candidateId(source: string, baseUrl: string): string {
  let hash = 5381;
  for (const char of `${source}:${baseUrl}`) hash = ((hash << 5) + hash) ^ char.charCodeAt(0);
  return `candidate_${source}_${(hash >>> 0).toString(16)}`;
}

export const fetchFocusaHealth: FocusaHealthProbe = async (baseUrl, parentSignal) => {
  const timeout = AbortSignal.timeout(PROBE_TIMEOUT_MS);
  const signal = AbortSignal.any([parentSignal, timeout]);
  const started = performance.now();
  try {
    const response = await fetch(`${baseUrl}/v1/health`, { method: "GET", cache: "no-store", redirect: "error", signal });
    if (!response.ok) return { healthy: false, latencyMs: Math.round(performance.now() - started) };
    const raw: unknown = await response.json();
    const body = raw && typeof raw === "object" && !Array.isArray(raw) ? raw as Record<string, unknown> : {};
    return {
      healthy: true,
      latencyMs: Math.round(performance.now() - started),
      daemonId: typeof body.daemon_id === "string" ? body.daemon_id : undefined,
      machineId: typeof body.machine_id === "string" ? body.machine_id : undefined,
      version: typeof body.version === "string" ? body.version : undefined,
      capabilities: Array.isArray(body.capabilities) && body.capabilities.every((value) => typeof value === "string") ? body.capabilities : undefined,
    };
  } catch {
    return { healthy: false, latencyMs: Math.round(performance.now() - started) };
  }
};

abstract class DirectSource implements FocusaDiscoverySourceAdapter {
  abstract readonly source: "loopback" | "environment" | "manual";
  constructor(private readonly values: readonly string[], private readonly probe: FocusaHealthProbe = fetchFocusaHealth) {}

  async discover(context: Readonly<FocusaDiscoveryContext>): Promise<FocusaDaemonCandidateV1[]> {
    const candidates: FocusaDaemonCandidateV1[] = [];
    for (const value of this.values) {
      if (context.signal?.aborted) break;
      let endpoint: ReturnType<typeof normalizeBaseUrl>;
      try { endpoint = normalizeBaseUrl(value); } catch { continue; }
      if (context.localOnly && endpoint.location !== "local") continue;
      const controller = new AbortController();
      const abort = () => controller.abort();
      context.signal?.addEventListener("abort", abort, { once: true });
      const health = await this.probe(endpoint.baseUrl, controller.signal);
      context.signal?.removeEventListener("abort", abort);
      candidates.push({
        schema: "focusa.daemon_candidate.v1",
        candidate_id: candidateId(this.source, endpoint.baseUrl),
        base_url: endpoint.baseUrl,
        source: this.source,
        location: endpoint.location,
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

export class LoopbackDiscoverySource extends DirectSource {
  readonly source = "loopback" as const;
  constructor(probe?: FocusaHealthProbe) { super(["http://127.0.0.1:8787"], probe); }
}
export class EnvironmentDiscoverySource extends DirectSource {
  readonly source = "environment" as const;
  constructor(values: readonly string[], probe?: FocusaHealthProbe) { super(values, probe); }
}
export class ManualDiscoverySource extends DirectSource {
  readonly source = "manual" as const;
  constructor(value: string, probe?: FocusaHealthProbe) { super([value], probe); }
}
