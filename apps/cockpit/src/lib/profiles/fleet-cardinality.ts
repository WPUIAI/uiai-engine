/**
 * T005-08.02 — Zero / one / many fleet picker cardinality
 * Zero -> show pairing, One -> may default (healthy only), Many -> always requires explicit selection.
 */
import { buildFleetView, type FleetEntry } from "./fleet";
import { getActiveProfileId, setActiveProfileId } from "./selection";

export type Cardinality = "zero" | "one" | "many";
export type PickerAction = "needs_pairing" | "defaulted" | "needs_selection" | "selected";

export interface PickerState {
  cardinality: Cardinality;
  action: PickerAction;
  entries: FleetEntry[];
  activeProfileId: string | null;
}

export async function resolvePickerState(): Promise<PickerState> {
  const entries = await buildFleetView();
  const active = await getActiveProfileId();

  if (entries.length === 0) {
    return { cardinality: "zero", action: "needs_pairing", entries, activeProfileId: null };
  }
  if (entries.length === 1) {
    const sole = entries[0];
    // one may default if healthy and no active yet — but never force if unhealthy
    if (!active && sole.health === "active") {
      await setActiveProfileId(sole.profileId).catch(() => {});
      const now = await getActiveProfileId();
      if (now === sole.profileId) return { cardinality: "one", action: "defaulted", entries, activeProfileId: now };
    }
    if (active === sole.profileId) return { cardinality: "one", action: "selected", entries, activeProfileId: active };
    return { cardinality: "one", action: "needs_selection", entries, activeProfileId: active };
  }
  // many -> always requires explicit selection, never auto-default
  if (active && entries.some((e) => e.profileId === active)) {
    return { cardinality: "many", action: "selected", entries, activeProfileId: active };
  }
  return { cardinality: "many", action: "needs_selection", entries, activeProfileId: active };
}
