import type { DiscoverySource, FocusaDaemonCandidateV1 } from "$lib/contracts/focusa-pairing";

export interface MergedFocusaCandidateV1 extends FocusaDaemonCandidateV1 {
  provenance_sources: DiscoverySource[];
  alternative_urls: string[];
}

const SOURCE_PRIORITY: Record<DiscoverySource, number> = {
  manual: 60,
  environment: 50,
  saved_hint: 40,
  tailscale: 30,
  bonjour: 20,
  loopback: 10,
};

function score(candidate: FocusaDaemonCandidateV1): number {
  return (candidate.health_status === "healthy" ? 10_000 : 0)
    + (candidate.daemon_id ? 1_000 : 0)
    + SOURCE_PRIORITY[candidate.source]
    - Math.min(candidate.latency_ms, 999) / 1_000;
}

/** Deduplicates discovery observations without converting them into authority. */
export function mergeFocusaDaemonCandidates(candidates: readonly FocusaDaemonCandidateV1[]): MergedFocusaCandidateV1[] {
  const daemonByUrl = new Map<string, string>();
  for (const candidate of candidates) if (candidate.daemon_id) daemonByUrl.set(candidate.base_url, candidate.daemon_id);
  const groups = new Map<string, FocusaDaemonCandidateV1[]>();
  for (const candidate of candidates) {
    const daemonId = candidate.daemon_id ?? daemonByUrl.get(candidate.base_url);
    const key = daemonId ? `daemon:${daemonId}` : `url:${candidate.base_url}`;
    groups.set(key, [...(groups.get(key) ?? []), candidate]);
  }

  return [...groups.values()].map((group) => {
    const ranked = [...group].sort((left, right) => score(right) - score(left));
    const preferred = ranked[0];
    const liveIdentity = ranked.find((candidate) => candidate.daemon_id);
    return {
      ...preferred,
      daemon_id: liveIdentity?.daemon_id,
      machine_id: liveIdentity?.machine_id ?? preferred.machine_id,
      version: liveIdentity?.version ?? preferred.version,
      capabilities: [...new Set(group.flatMap((candidate) => candidate.capabilities ?? []))].sort(),
      provenance_sources: [...new Set(group.map((candidate) => candidate.source))]
        .sort((left, right) => SOURCE_PRIORITY[right] - SOURCE_PRIORITY[left]),
      alternative_urls: [...new Set(group.map((candidate) => candidate.base_url).filter((url) => url !== preferred.base_url))].sort(),
    };
  }).sort((left, right) => score(right) - score(left) || left.candidate_id.localeCompare(right.candidate_id));
}
