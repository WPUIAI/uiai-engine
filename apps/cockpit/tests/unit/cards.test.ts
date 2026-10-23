import { readFileSync } from "node:fs";
import { describe, it, expect } from "vitest";
import {
  phase0Cards,
  validateCardManifest,
  phase0ContractCoverage,
} from "../../src/lib/cards/phase0-card-manifest";
import { overviewCardIds, phase0CardPlacements } from "../../src/lib/cards/phase0-card-placement";

describe("Phase 0 card manifest", () => {
  it("has 14 cards", () => {
    expect(phase0Cards).toHaveLength(14);
  });

  it("validates without errors", () => {
    const v = validateCardManifest(phase0Cards);
    expect(v.ok).toBe(true);
  });

  it("every focusa_local card has a contract_ref or notes=adapter_only", () => {
    for (const card of phase0Cards) {
      if (card.product_surface === "focusa_local") {
        const has = card.contract_ref !== null && card.contract_ref !== undefined;
        const notes = card.notes ?? "";
        const isAdapterOnly = notes.includes("adapter_only");
        expect(has || isAdapterOnly).toBe(true);
      }
    }
  });

  it("every card has at least one capability", () => {
    for (const card of phase0Cards) {
      expect(card.capabilities.length).toBeGreaterThan(0);
    }
  });

  it("every card has a unique card_id", () => {
    const ids = phase0Cards.map((c) => c.card_id);
    const set = new Set(ids);
    expect(set.size).toBe(ids.length);
  });

  it("Spec 90 contract coverage matrix matches the manifest", () => {
    expect(phase0ContractCoverage).toHaveLength(phase0Cards.length);
  });

  it("places every Phase 0 card in at least one safe Cockpit destination", () => {
    expect(Object.keys(phase0CardPlacements).sort()).toEqual(phase0Cards.map((card) => card.card_id).sort());
    for (const placements of Object.values(phase0CardPlacements)) {
      expect(placements.length).toBeGreaterThan(0);
      for (const placement of placements) {
        expect(placement.href).toMatch(/^\/(?!\/)/);
      }
    }
  });

  it("maps only the specified summary contracts into Overview", () => {
    expect(overviewCardIds().sort()).toEqual([
      "focusa.project_card",
      "focusa.project_identity",
      "focusa.trajectory_view",
      "focusa.work_loop_status",
      "focusa.workpoint_resume",
      "uiai.health",
    ]);
  });

  it("uses the final Overview composition instead of the Phase 0 card grid", () => {
    const overview = readFileSync(new URL("../../src/routes/+page.svelte", import.meta.url), "utf8");
    expect(overview).not.toContain("phase0Cards");
    for (const section of ["Continue", "Active now", "Recent work", "System posture", "Suggested next actions"]) {
      expect(overview).toContain(section);
    }
  });
});
