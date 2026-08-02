import { describe, expect, it } from "vitest";
import { footerDestinations, sidebarGroups, workspaceForPath, workspaceManifest } from "../../src/lib/navigation/sidebar-manifest";
import { defaultSidebarPreferences, readSidebarPreferences, saveSidebarPreferences } from "../../src/lib/navigation/sidebar-preferences";
import { sidebarFixtures } from "../fixtures/sidebar-fixtures";
import { applyWorkspaceDrop, createSidebarDndAdapter } from "../../src/lib/navigation/sidebar-dnd";
import { inspectScope, requireScopedMutation } from "../../src/lib/navigation/scope-guard";

describe("Cockpit sidebar manifest", () => {
  it("contains the normative Work/Create/Prove/System taxonomy", () => {
    expect(sidebarGroups.map((group) => group.id)).toEqual(["work", "create", "prove", "system"]);
    expect(workspaceManifest.map((workspace) => workspace.id)).toEqual([
      "overview", "live", "test_lab", "documents", "research", "studio", "automations", "evidence", "activity", "nodes_services", "capabilities",
    ]);
    expect(new Set(workspaceManifest.map((workspace) => workspace.route)).size).toBe(workspaceManifest.length);
    expect(footerDestinations.map((item) => item.id)).toEqual(["settings", "help"]);
  });

  it("resolves the current browser route without inventing backend state", () => {
    expect(workspaceForPath("/")?.id).toBe("overview");
    expect(workspaceForPath("/runs")?.id).toBe("live");
    expect(workspaceForPath("/nodes-services")?.state).toBe("planned");
    expect(workspaceForPath("/missing")).toBeUndefined();
  });

  it("starts with local recommended preferences", () => {
    const preferences = defaultSidebarPreferences();
    expect(preferences.schema).toBe("uaiengine.cockpit.sidebar_preferences.v1");
    expect(preferences.layout_mode).toBe("recommended");
    expect(preferences.mode).toBe("expanded");
    expect(preferences.width_px).toBe(240);
    expect(preferences.workspace_placements).toEqual([]);
  });

  it("covers the required baseline shell fixtures without product data", () => {
    expect(sidebarFixtures.map((fixture) => fixture.id)).toEqual(["expanded", "compact", "hidden", "overlay", "empty-context", "missing-scope", "blocked-entitlement"]);
    expect(sidebarFixtures.every((fixture) => fixture.expected.length > 0)).toBe(true);
    expect(sidebarFixtures.find((fixture) => fixture.id === "blocked-entitlement")?.entitlement).toBe("blocked");
  });

  it("round-trips presentation preferences locally without a backend", () => {
    const values = new Map<string, string>();
    Object.defineProperty(globalThis, "window", { configurable: true, value: { localStorage: { getItem: (key: string) => values.get(key) || null, setItem: (key: string, value: string) => values.set(key, value) } } });
    const preferences = { ...defaultSidebarPreferences(), mode: "compact" as const, layout_mode: "custom" as const, width_px: 280 };
    expect(saveSidebarPreferences(preferences)).toBe(true);
    expect(readSidebarPreferences()).toMatchObject({ mode: "compact", layout_mode: "custom", width_px: 280 });
    delete (globalThis as { window?: unknown }).window;
  });

  it("keeps DnD presentation-only and supports a keyboard-equivalent reorder primitive", async () => {
    const preferences = defaultSidebarPreferences();
    const target = { workspace_id: "documents", display_group: "work" as const, order: 40 };
    const reordered = applyWorkspaceDrop(preferences, workspaceManifest, "studio", target);
    expect(reordered.layout_mode).toBe("custom");
    expect(reordered.workspace_placements.find((item) => item.workspace_id === "studio")?.display_group).toBe("work");
    expect(reordered.workspace_placements).toHaveLength(workspaceManifest.length);
    let commit: string | undefined;
    const adapter = createSidebarDndAdapter(async (_drop, source) => { commit = source; });
    adapter.beginDrag("studio");
    expect(adapter.previewDrop(target).accepted).toBe(true);
    await adapter.commitDrop(target);
    expect(commit).toBe("studio");
    expect(adapter.previewDrop(target).accepted).toBe(false);
  });

  it("blocks unscoped browser mutations with exact recovery guidance", () => {
    expect(inspectScope({}).status).toBe("missing");
    expect(inspectScope({ project_root: "/project" }).status).toBe("partial");
    expect(inspectScope({ project_root: "/project", continuity_id: "continuity" }).allowed).toBe(true);
    expect(() => requireScopedMutation({}, "Open browser session")).toThrow("Open browser session blocked");
  });

  it("migrates legacy preferences and records removed workspace diagnostics", () => {
    const values = new Map<string, string>([["uiai.cockpit.sidebar_preferences", JSON.stringify({ schema: "uaiengine.cockpit.sidebar_preferences.v0", mode: "compact", hidden_workspace_ids: ["removed_workspace"] })]]);
    Object.defineProperty(globalThis, "window", { configurable: true, value: { localStorage: { getItem: (key: string) => values.get(key) || null, setItem: (key: string, value: string) => values.set(key, value) } } });
    const migrated = readSidebarPreferences();
    expect(migrated.schema).toBe("uaiengine.cockpit.sidebar_preferences.v1");
    expect(migrated.mode).toBe("compact");
    expect(migrated.hidden_workspace_ids).toEqual([]);
    expect(migrated.migration_diagnostics).toContain("unknown_workspace:removed_workspace");
    expect(values.has("uiai.cockpit.sidebar_preferences.v1")).toBe(true);
    delete (globalThis as { window?: unknown }).window;
  });
});
