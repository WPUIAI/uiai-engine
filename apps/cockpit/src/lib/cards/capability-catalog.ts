import type { CardManifest } from "../contracts/card-manifest";
import type { Phase0CardPlacement } from "./phase0-card-placement";

export interface CapabilityCatalogEntry {
  capability_id: string;
  labels: string[];
  card_ids: string[];
  workspaces: string[];
  source_planes: string[];
  side_effects: string[];
  required_scopes: string[];
  locality: "local" | "cloud" | "mixed" | "hosted";
  status: "registered" | "adapter_only";
  license: "not_declared";
  experimental: false;
  artifact_types: string[];
}

export interface CapabilityCatalogFilters {
  query?: string;
  workspace?: string;
  status?: string;
  source_plane?: string;
  side_effect?: string;
  required_scope?: string;
  locality?: string;
  license?: string;
  experimental?: string;
  artifact_type?: string;
}

function workspaceFromHref(href: string): string {
  const path = href.split(/[?#]/, 1)[0];
  if (path === "/") return "overview";
  return path.replace(/^\//, "").split("/", 1)[0] || "overview";
}

function locality(surfaces: Set<string>): CapabilityCatalogEntry["locality"] {
  const hasCloud = surfaces.has("focusa_cloud");
  const hasLocal = [...surfaces].some((surface) => surface === "uiai_engine" || surface === "focusa_local" || surface === "wirebot");
  if (hasCloud && hasLocal) return "mixed";
  if (hasCloud) return "cloud";
  if (surfaces.has("ai_api")) return "hosted";
  return "local";
}

export function buildCapabilityCatalog(
  cards: readonly CardManifest[],
  placements: Readonly<Record<string, readonly Phase0CardPlacement[]>>,
): CapabilityCatalogEntry[] {
  const catalog = new Map<string, {
    labels: Set<string>;
    cardIds: Set<string>;
    workspaces: Set<string>;
    sources: Set<string>;
    sideEffects: Set<string>;
    scopes: Set<string>;
    artifactTypes: Set<string>;
    statuses: Set<"registered" | "adapter_only">;
  }>();

  for (const card of cards) {
    for (const capability of card.capabilities) {
      const current = catalog.get(capability) || {
        labels: new Set<string>(), cardIds: new Set<string>(), workspaces: new Set<string>(),
        sources: new Set<string>(), sideEffects: new Set<string>(), scopes: new Set<string>(),
        artifactTypes: new Set<string>(), statuses: new Set<"registered" | "adapter_only">(),
      };
      current.labels.add(card.label);
      current.cardIds.add(card.card_id);
      current.sources.add(card.product_surface);
      current.sideEffects.add(card.side_effect_class);
      current.scopes.add(card.required_scope);
      current.statuses.add(card.contract_ref ? "registered" : "adapter_only");
      if (card.receipt_behavior !== "none") current.artifactTypes.add(card.receipt_behavior);
      for (const placement of placements[card.card_id] || []) current.workspaces.add(workspaceFromHref(placement.href));
      catalog.set(capability, current);
    }
  }

  return [...catalog.entries()].map<CapabilityCatalogEntry>(([capabilityId, value]) => ({
    capability_id: capabilityId,
    labels: [...value.labels].sort(),
    card_ids: [...value.cardIds].sort(),
    workspaces: [...value.workspaces].sort(),
    source_planes: [...value.sources].sort(),
    side_effects: [...value.sideEffects].sort(),
    required_scopes: [...value.scopes].sort(),
    locality: locality(value.sources),
    status: value.statuses.has("registered") ? "registered" : "adapter_only",
    license: "not_declared",
    experimental: false,
    artifact_types: [...value.artifactTypes].sort(),
  })).sort((a, b) => a.capability_id.localeCompare(b.capability_id));
}

export function filterCapabilityCatalog(
  entries: readonly CapabilityCatalogEntry[],
  filters: CapabilityCatalogFilters,
): CapabilityCatalogEntry[] {
  const query = filters.query?.trim().toLowerCase() || "";
  return entries.filter((entry) => {
    if (query && ![entry.capability_id, ...entry.labels, ...entry.card_ids].join(" ").toLowerCase().includes(query)) return false;
    if (filters.workspace && !entry.workspaces.includes(filters.workspace)) return false;
    if (filters.status && entry.status !== filters.status) return false;
    if (filters.source_plane && !entry.source_planes.includes(filters.source_plane)) return false;
    if (filters.side_effect && !entry.side_effects.includes(filters.side_effect)) return false;
    if (filters.required_scope && !entry.required_scopes.includes(filters.required_scope)) return false;
    if (filters.locality && entry.locality !== filters.locality) return false;
    if (filters.license && entry.license !== filters.license) return false;
    if (filters.experimental && String(entry.experimental) !== filters.experimental) return false;
    if (filters.artifact_type && !entry.artifact_types.includes(filters.artifact_type)) return false;
    return true;
  });
}
