import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { buildCapabilityCatalog, filterCapabilityCatalog } from "../../src/lib/cards/capability-catalog";
import { phase0Cards } from "../../src/lib/cards/phase0-card-manifest";
import { phase0CardPlacements } from "../../src/lib/cards/phase0-card-placement";

const catalog = buildCapabilityCatalog(phase0Cards, phase0CardPlacements);

describe("Capabilities catalog projection", () => {
  it("projects every unique registered capability exactly once", () => {
    const expected = [...new Set(phase0Cards.flatMap((card) => card.capabilities))].sort();
    expect(catalog.map((entry) => entry.capability_id)).toEqual(expected);
    expect(new Set(catalog.map((entry) => entry.capability_id)).size).toBe(catalog.length);
  });

  it("keeps grant, license, and experimental state truthful", () => {
    for (const entry of catalog) {
      expect(["registered", "adapter_only"]).toContain(entry.status);
      expect(entry.license).toBe("not_declared");
      expect(entry.experimental).toBe(false);
      expect(entry.source_planes.length).toBeGreaterThan(0);
      expect(entry.workspaces.length).toBeGreaterThan(0);
    }
  });

  it("filters by required catalog dimensions", () => {
    expect(filterCapabilityCatalog(catalog, { query: "workpoint resume" }).map((entry) => entry.capability_id))
      .toContain("focusa.workpoint.resume.read");
    expect(filterCapabilityCatalog(catalog, { side_effect: "write" }).every((entry) => entry.side_effects.includes("write"))).toBe(true);
    expect(filterCapabilityCatalog(catalog, { locality: "cloud" }).every((entry) => entry.locality === "cloud")).toBe(true);
    expect(filterCapabilityCatalog(catalog, { license: "not_declared", experimental: "false" })).toHaveLength(catalog.length);
  });

  it("renders every required Evidence, Activity, Nodes, and capability filter surface", () => {
    const source = (route: string) => readFileSync(new URL(`../../src/routes/${route}/+page.svelte`, import.meta.url), "utf8");
    const evidence = source("evidence");
    for (const label of ["Current Workpoint", "Needs capture", "Needs review", "Verified", "Provisional / Surrogate", "Public-safe", "Receipts", "Reports"]) expect(evidence).toContain(label);
    const activity = source("activity");
    for (const label of ["Now", "Approvals", "History", "Jobs", "Notifications", "Audit"]) expect(activity).toContain(label);
    const nodes = source("nodes-services");
    for (const label of ["Nodes", "UIAI Engine", "Focusa Local", "Focusa Cloud", "AI API", "Pairing & Devices", "Capacity", "Sync", "Updates & Compatibility"]) expect(nodes).toContain(label);
    const capabilities = source("capabilities");
    for (const label of ["Task or capability", "Workspace", "Status", "Source plane", "Side effect", "Required scope", "Local / cloud", "License", "Experimental", "Artifact type"]) expect(capabilities).toContain(label);
  });
});
