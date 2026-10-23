import { existsSync, readFileSync, readdirSync, statSync } from "node:fs";
import { describe, expect, it } from "vitest";

const sourceRoot = new URL("../../src/", import.meta.url);
function sourceFiles(directory: URL): URL[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const child = new URL(entry.name + (entry.isDirectory() ? "/" : ""), directory);
    return entry.isDirectory() ? sourceFiles(child) : /\.(svelte|ts|js)$/.test(entry.name) ? [child] : [];
  });
}

describe("Cockpit-003 rollout cleanup", () => {
  it("has one production shell and no temporary migration flag", () => {
    expect(existsSync(new URL("../../src/routes/+layout.svelte", import.meta.url))).toBe(true);
    const sources = sourceFiles(sourceRoot).map((file) => readFileSync(file, "utf8")).join("\n");
    for (const removed of ["primitive navigator", "scope-chip-strip", "process-ribbon", "COCKPIT_003_MIGRATION_FLAG"]) {
      expect(sources.toLowerCase()).not.toContain(removed.toLowerCase());
    }
    const overview = readFileSync(new URL("../../src/routes/+page.svelte", import.meta.url), "utf8");
    expect(overview).not.toContain("phase0Cards");
  });

  it("ships Help and rollout support diagnostics for the current shell", () => {
    const help = readFileSync(new URL("../../src/routes/help/+page.svelte", import.meta.url), "utf8");
    for (const marker of ["Shift F10", "Alt ↑ / ↓", "Recovery", "Local Evidence", "Capability catalog"]) expect(help).toContain(marker);
    const rollout = new URL("../../../../docs/cockpit/003/UIAI_COCKPIT_003_ROLLOUT_AND_SUPPORT.md", import.meta.url);
    expect(statSync(rollout).size).toBeGreaterThan(500);
  });
});
