import {
  parseFocusaDaemonCandidate,
  type DiscoverySource,
  type FocusaDaemonCandidateV1,
} from "$lib/contracts/focusa-pairing";

export const MAX_CANDIDATES_PER_SOURCE = 16;
export const MAX_DISCOVERY_SOURCES = 8;

export interface FocusaDiscoveryContext {
  localOnly: boolean;
  signal?: AbortSignal;
  observedAt: string;
}

export interface FocusaDiscoverySourceAdapter {
  readonly source: DiscoverySource;
  discover(context: Readonly<FocusaDiscoveryContext>): Promise<unknown[]>;
}

export interface FocusaDiscoveryFailure {
  source: DiscoverySource;
  reason: "source_failed" | "invalid_candidate" | "source_mismatch" | "candidate_limit_exceeded";
}

export interface FocusaDiscoveryBatchV1 {
  schema: "focusa.discovery_batch.v1";
  candidates: FocusaDaemonCandidateV1[];
  failures: FocusaDiscoveryFailure[];
}

/**
 * Runs untrusted discovery adapters without granting authentication, profile,
 * project, or scope authority. Every adapter output must pass the canonical
 * candidate parser and must identify its actual source.
 */
export async function runFocusaDiscoverySources(
  adapters: readonly FocusaDiscoverySourceAdapter[],
  context: Readonly<FocusaDiscoveryContext>,
): Promise<FocusaDiscoveryBatchV1> {
  if (adapters.length > MAX_DISCOVERY_SOURCES) throw new Error("too many discovery sources");
  if (!Number.isFinite(Date.parse(context.observedAt))) throw new Error("discovery observedAt is invalid");

  const candidates: FocusaDaemonCandidateV1[] = [];
  const failures: FocusaDiscoveryFailure[] = [];
  await Promise.all(adapters.map(async (adapter) => {
    if (context.signal?.aborted) return;
    let raw: unknown[];
    try {
      raw = await adapter.discover(context);
    } catch {
      failures.push({ source: adapter.source, reason: "source_failed" });
      return;
    }
    if (!Array.isArray(raw)) {
      failures.push({ source: adapter.source, reason: "invalid_candidate" });
      return;
    }
    if (raw.length > MAX_CANDIDATES_PER_SOURCE) {
      failures.push({ source: adapter.source, reason: "candidate_limit_exceeded" });
      raw = raw.slice(0, MAX_CANDIDATES_PER_SOURCE);
    }
    for (const value of raw) {
      try {
        const candidate = parseFocusaDaemonCandidate(value);
        if (candidate.source !== adapter.source) {
          failures.push({ source: adapter.source, reason: "source_mismatch" });
          continue;
        }
        if (context.localOnly && candidate.location !== "local") continue;
        candidates.push(candidate);
      } catch {
        failures.push({ source: adapter.source, reason: "invalid_candidate" });
      }
    }
  }));

  return { schema: "focusa.discovery_batch.v1", candidates, failures };
}
