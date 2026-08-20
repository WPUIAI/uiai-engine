/**
 * T005-08.03 — Accessible fleet picker
 * Keyboard / screen-reader / filtering / details + explicit confirmation.
 */
import type { FleetEntry } from "./fleet";
import { setActiveProfileId } from "./selection";

export interface PickerFilter {
  query?: string;
  health?: FleetEntry["health"][];
}

export function filterEntries(entries: FleetEntry[], f: PickerFilter): FleetEntry[] {
  let out = entries;
  if (f.query?.trim()) {
    const q = f.query.trim().toLowerCase();
    out = out.filter((e) => [e.profileId, e.daemonId, e.displayName ?? "", e.endpoint].some((v) => v.toLowerCase().includes(q)));
  }
  if (f.health?.length) out = out.filter((e) => f.health!.includes(e.health));
  return out;
}

export function ariaLabel(e: FleetEntry): string {
  return `${e.displayName ?? e.daemonId} — ${e.health} — ${e.endpoint} — source ${e.source}`;
}

export function detailText(e: FleetEntry): string {
  return `Profile ${e.profileId}\nDaemon ${e.daemonId}\nEndpoint ${e.endpoint}\nHealth ${e.health}\nScopes ${e.scopes.join(", ") || "none"}\nProvenance ${e.provenance}`;
}

/** Explicit confirmation required — returns selected meta only after confirm(). */
export async function confirmSelection(profileId: string): Promise<string> {
  const m = await setActiveProfileId(profileId);
  return m.profileId;
}

/** Keyboard helper: index navigation wrap. */
export function nextIndex(current: number, delta: number, len: number): number {
  if (len === 0) return 0;
  return (current + delta + len) % len;
}
