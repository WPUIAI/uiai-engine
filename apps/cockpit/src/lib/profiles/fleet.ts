/**
 * T005-08.01 — Fleet view model
 * Health, location, version, scopes, source, expiry and lastUse retain profile provenance.
 * No secrets — view derives only from metadata + health.
 */
import { listProfileMetadata, type DaemonProfileMetadata } from "./metadata";
import { getProfileHealth, type ProfileHealthStatus } from "./lifecycle";

export interface FleetEntry {
  profileId: string;
  daemonId: string;
  endpoint: string;
  displayName?: string;
  health: ProfileHealthStatus;
  source: DaemonProfileMetadata["source"];
  scopes: string[];
  pairedAt: string;
  lastSeenAt?: string;
  version?: string; // optional, from metadata extension if present
  provenance: string; // e.g. "pairing:2026-08-04T00:00:00Z"
}

function provenanceOf(m: DaemonProfileMetadata): string {
  return `${m.source}:${m.pairedAt}`;
}

export async function buildFleetView(): Promise<FleetEntry[]> {
  const all = await listProfileMetadata();
  const entries: FleetEntry[] = [];
  for (const m of all) {
    const health = await getProfileHealth(m.profileId).catch(() => "unavailable" as ProfileHealthStatus);
    const ext = m as unknown as Record<string, unknown>;
    entries.push({
      profileId: m.profileId,
      daemonId: m.daemonId,
      endpoint: m.endpoint,
      displayName: m.displayName,
      health,
      source: m.source,
      scopes: [...(m.scopes ?? [])],
      pairedAt: m.pairedAt,
      lastSeenAt: m.lastSeenAt,
      version: typeof ext._version === "string" ? (ext._version as string) : undefined,
      provenance: provenanceOf(m),
    });
  }
  // deterministic sort: health priority then profileId
  const prio: Record<ProfileHealthStatus, number> = { active: 0, unavailable: 1, expired: 2, conflict: 3, revoked: 4 };
  entries.sort((a, b) => (prio[a.health] - prio[b.health]) || a.profileId.localeCompare(b.profileId));
  return entries;
}
