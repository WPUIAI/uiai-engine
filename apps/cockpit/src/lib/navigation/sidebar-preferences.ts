import { workspaceManifest, type SidebarGroup } from "./sidebar-manifest";

export type SidebarMode = "expanded" | "compact" | "hidden";
export type SidebarLayoutMode = "recommended" | "custom";
export type SidebarDensity = "comfortable" | "compact";

export interface SidebarPreferencesV1 {
  schema: "uaiengine.cockpit.sidebar_preferences.v1";
  mode: SidebarMode;
  layout_mode: SidebarLayoutMode;
  density: SidebarDensity;
  width_px: number;
  collapsed_groups: SidebarGroup[];
  workspace_placements: Array<{ workspace_id: string; display_group: SidebarGroup; order: number }>;
  hidden_workspace_ids: string[];
  pinned_refs: Array<{ ref: string; kind: "workspace" | "saved_view" | "work_object"; workspace_id: string; order: number }>;
  last_updated_at: string;
  migration_diagnostics?: string[];
}

const STORAGE_KEY = "uiai.cockpit.sidebar_preferences.v1";
const DEFAULT_WIDTH = 240;
const validGroups: SidebarGroup[] = ["work", "create", "prove", "system"];

export function defaultSidebarPreferences(): SidebarPreferencesV1 {
  return { schema: "uaiengine.cockpit.sidebar_preferences.v1", mode: "expanded", layout_mode: "recommended", density: "comfortable", width_px: DEFAULT_WIDTH, collapsed_groups: [], workspace_placements: [], hidden_workspace_ids: [], pinned_refs: [], last_updated_at: new Date().toISOString(), migration_diagnostics: [] };
}

export function readSidebarPreferences(): SidebarPreferencesV1 {
  if (typeof window === "undefined") return defaultSidebarPreferences();
  try {
    const currentValue = window.localStorage.getItem(STORAGE_KEY);
    const legacyValue = window.localStorage.getItem("uiai.cockpit.sidebar_preferences");
    const raw = JSON.parse(currentValue || legacyValue || "null") as Record<string, unknown> | null;
    const schema = typeof raw?.schema === "string" ? raw.schema : "legacy";
    if (!raw || !["uaiengine.cockpit.sidebar_preferences.v1", "uaiengine.cockpit.sidebar_preferences.v0", "legacy"].includes(schema)) return defaultSidebarPreferences();

    const knownWorkspaceIds = new Set(workspaceManifest.map((workspace) => workspace.id));
    const diagnostics: string[] = [];
    const hiddenRaw = Array.isArray(raw.hidden_workspace_ids) ? raw.hidden_workspace_ids.filter((id): id is string => typeof id === "string") : [];
    const hidden = hiddenRaw.filter((id) => knownWorkspaceIds.has(id));
    hiddenRaw.filter((id) => !knownWorkspaceIds.has(id)).forEach((id) => diagnostics.push(`unknown_workspace:${id}`));
    const placementsRaw = Array.isArray(raw.workspace_placements) ? raw.workspace_placements : [];
    const placements = placementsRaw.flatMap((placement) => {
      if (!placement || typeof placement !== "object") return [];
      const item = placement as Record<string, unknown>;
      const id = typeof item.workspace_id === "string" ? item.workspace_id : "";
      const group = validGroups.includes(item.display_group as SidebarGroup) ? item.display_group as SidebarGroup : "work";
      if (!knownWorkspaceIds.has(id)) { if (id) diagnostics.push(`unknown_workspace:${id}`); return []; }
      return [{ workspace_id: id, display_group: group, order: Number(item.order) || 0 }];
    });
    const normalized: SidebarPreferencesV1 = {
      ...defaultSidebarPreferences(),
      mode: raw.mode === "compact" || raw.mode === "hidden" ? raw.mode : "expanded",
      layout_mode: raw.layout_mode === "custom" ? "custom" : "recommended",
      density: raw.density === "compact" ? "compact" : "comfortable",
      width_px: Math.min(320, Math.max(208, Number(raw.width_px) || DEFAULT_WIDTH)),
      collapsed_groups: (Array.isArray(raw.collapsed_groups) ? raw.collapsed_groups : []).filter((group): group is SidebarGroup => validGroups.includes(group as SidebarGroup)),
      workspace_placements: placements,
      hidden_workspace_ids: hidden,
      pinned_refs: Array.isArray(raw.pinned_refs) ? raw.pinned_refs.filter((ref) => ref && typeof ref === "object") as SidebarPreferencesV1["pinned_refs"] : [],
      last_updated_at: typeof raw.last_updated_at === "string" ? raw.last_updated_at : new Date().toISOString(),
      migration_diagnostics: [...new Set(diagnostics)],
    };
    if (schema !== "uaiengine.cockpit.sidebar_preferences.v1" || !currentValue) saveSidebarPreferences(normalized);
    return normalized;
  } catch {
    return defaultSidebarPreferences();
  }
}

export function saveSidebarPreferences(preferences: SidebarPreferencesV1): boolean {
  if (typeof window === "undefined") return false;
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify({ ...preferences, last_updated_at: new Date().toISOString() }));
    return true;
  } catch {
    return false;
  }
}

export function resetSidebarPreferences(): SidebarPreferencesV1 {
  const preferences = defaultSidebarPreferences();
  saveSidebarPreferences(preferences);
  return preferences;
}
