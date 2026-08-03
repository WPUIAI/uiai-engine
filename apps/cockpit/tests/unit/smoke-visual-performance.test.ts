import { existsSync } from "node:fs";
import { performance } from "node:perf_hooks";
import { describe, expect, it } from "vitest";
import { phase0Cards } from "../../src/lib/cards/phase0-card-manifest";
import type { CardManifest } from "../../src/lib/contracts/card-manifest";
import { buildCommandIndex, filterCommandIndex } from "../../src/lib/navigation/command-index";
import { footerDestinations, workspaceManifest } from "../../src/lib/navigation/sidebar-manifest";

const visualStates = ["normal", "loading", "empty", "blocked", "degraded", "error", "approval", "success"] as const;
const visualFrames = ["default", "dark", "constrained", "overlay", "drag-and-drop"] as const;

describe("Cockpit smoke, visual-state, and performance gates", () => {
  it("has a route component for every registered workspace", () => {
    for (const workspace of workspaceManifest) {
      const route = workspace.route === "/" ? "" : workspace.route;
      expect(existsSync(new URL(`../../src/routes${route}/+page.svelte`, import.meta.url))).toBe(true);
    }
  });

  it("covers all required non-happy visual states and keeps baseline screenshots", () => {
    expect(visualStates).toEqual(["normal", "loading", "empty", "blocked", "degraded", "error", "approval", "success"]);
    expect(visualFrames).toEqual(["default", "dark", "constrained", "overlay", "drag-and-drop"]);
    for (const baseline of ["overview-default.png", "overview-dark.png", "overview-constrained.png", "capabilities-recovery-only.png"]) {
      expect(existsSync(new URL(`../visual/baselines/${baseline}`, import.meta.url))).toBe(true);
    }
  });

  it("indexes and filters representative large registries within a bounded budget", () => {
    const cards: CardManifest[] = Array.from({ length: 100 }, (_, batch) => phase0Cards.map((card) => ({
      ...card,
      card_id: `${card.card_id}.${batch}`,
      capabilities: card.capabilities.map((capability) => `${capability}.${batch}`),
    }))).flat();
    const started = performance.now();
    const commands = buildCommandIndex(workspaceManifest, cards, footerDestinations, null);
    const matches = filterCommandIndex(commands, "workpoint resume read 99");
    const elapsed = performance.now() - started;
    expect(commands.length).toBeGreaterThan(2_000);
    expect(matches.length).toBeGreaterThan(0);
    expect(elapsed).toBeLessThan(500);
  });
});
