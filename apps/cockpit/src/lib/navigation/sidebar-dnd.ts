import type { SidebarGroup, WorkspaceManifestEntry } from "./sidebar-manifest";
import type { SidebarPreferencesV1 } from "./sidebar-preferences";

export interface SidebarDropTarget {
  workspace_id: string;
  display_group: SidebarGroup;
  order: number;
}

export interface SidebarDropDecision {
  accepted: boolean;
  reason?: string;
}

export interface SidebarDndAdapter {
  beginDrag(itemRef: string): void;
  previewDrop(target: SidebarDropTarget): SidebarDropDecision;
  commitDrop(target: SidebarDropTarget): Promise<void>;
  cancelDrag(): void;
}

export function createSidebarDndAdapter(commit: (target: SidebarDropTarget, sourceRef: string) => Promise<void>): SidebarDndAdapter {
  let sourceRef = "";
  return {
    beginDrag(itemRef) { sourceRef = itemRef; },
    previewDrop(target) {
      if (!sourceRef) return { accepted: false, reason: "No workspace is being dragged." };
      if (sourceRef === target.workspace_id) return { accepted: false, reason: "A workspace cannot be dropped onto itself." };
      return { accepted: true };
    },
    async commitDrop(target) {
      const decision = this.previewDrop(target);
      if (!decision.accepted) throw new Error(decision.reason || "Drop target is not valid.");
      const source = sourceRef;
      sourceRef = "";
      await commit(target, source);
    },
    cancelDrag() { sourceRef = ""; },
  };
}

export function applyWorkspaceDrop(preferences: SidebarPreferencesV1, manifest: WorkspaceManifestEntry[], sourceId: string, target: SidebarDropTarget): SidebarPreferencesV1 {
  const source = manifest.find((item) => item.id === sourceId);
  if (!source || source.id === target.workspace_id) return preferences;
  const placements = manifest.map((item) => {
    const existing = preferences.workspace_placements.find((placement) => placement.workspace_id === item.id);
    return existing || { workspace_id: item.id, display_group: item.group, order: item.order };
  });
  const sourceIndex = placements.findIndex((item) => item.workspace_id === sourceId);
  const targetIndex = placements.findIndex((item) => item.workspace_id === target.workspace_id);
  if (sourceIndex < 0 || targetIndex < 0) return preferences;
  const [moved] = placements.splice(sourceIndex, 1);
  moved.display_group = target.display_group;
  placements.splice(Math.min(targetIndex, placements.length), 0, moved);
  const reordered = placements.map((placement, index) => ({ ...placement, order: (index + 1) * 10 }));
  return { ...preferences, layout_mode: "custom", workspace_placements: reordered };
}
