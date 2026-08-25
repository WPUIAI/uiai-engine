import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { cockpitMessages, directionForLocale, message } from "../../src/lib/ui/messages";

describe("Cockpit accessibility and i18n contract", () => {
  it("uses stable message keys with interpolation and RTL direction", () => {
    expect(message("sidebar.moved", { workspace: "Live", group: "work" })).toBe("Live moved within work.");
    expect(Object.keys(cockpitMessages)).toContain("shell.find_do");
    expect(directionForLocale("ar-SA")).toBe("rtl");
    expect(directionForLocale("he")).toBe("rtl");
    expect(directionForLocale("en-US")).toBe("ltr");
  });

  it("implements the required keyboard, live-region, contrast, and motion surfaces", () => {
    const shell = readFileSync(new URL("../../src/routes/+layout.svelte", import.meta.url), "utf8");
    for (const marker of [
      'event.key.toLowerCase() === "k"', 'event.key === ","', 'event.key === "?"',
      'event.key === "ArrowDown"', 'event.key === "ArrowUp"', 'event.key === "ArrowLeft"',
      'event.key === "ArrowRight"', 'event.key === "F10"', 'event.key === "Escape"',
      'aria-live="polite"', 'aria-current=', 'aria-expanded=', 'aria-keyshortcuts=',
      'prefers-reduced-motion: reduce', 'forced-colors: active', 'data-workspace-row',
    ]) expect(shell).toContain(marker);
  });

  it("uses desktop-native link styling, stronger contrast tokens, and visible version posture", () => {
    const tokens = readFileSync(new URL("../../src/lib/ui/design-tokens.css", import.meta.url), "utf8");
    const shell = readFileSync(new URL("../../src/routes/+layout.svelte", import.meta.url), "utf8");
    expect(tokens).toContain("a { color: inherit; text-decoration: none; }");
    expect(tokens).toContain("--color-border-strong");
    expect(tokens).toContain("--color-text-muted: #b6b6bf");
    expect(shell).toContain("v{APP_VERSION}");
    expect(shell).toContain("focusaDaemonSummary(focusaConnections)");
  });

  it("keeps keyboard reorder as an alternative to drag and drop", () => {
    const shell = readFileSync(new URL("../../src/routes/+layout.svelte", import.meta.url), "utf8");
    expect(shell).toContain("keyboardMoveWorkspace(item.id, -1)");
    expect(shell).toContain("keyboardMoveWorkspace(item.id, 1)");
    expect(shell).toContain("sidebarAnnouncement");
  });
});
