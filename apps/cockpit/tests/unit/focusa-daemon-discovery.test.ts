import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { focusaDaemonSummary, type FocusaDaemonConnection } from "../../src/lib/focusa-daemon-discovery";

const connected = (location: "local" | "remote"): FocusaDaemonConnection => ({
  baseUrl: location === "local" ? "http://127.0.0.1:8787" : "http://focusa-vps:8787",
  source: location === "local" ? "loopback" : "saved",
  location,
  status: "connected",
});

describe("Focusa daemon discovery", () => {
  it("reports local, remote, and simultaneous truthful connections", () => {
    expect(focusaDaemonSummary([connected("local")])).toBe("Focusa · Local");
    expect(focusaDaemonSummary([connected("remote")])).toBe("Focusa · VPS");
    expect(focusaDaemonSummary([connected("local"), connected("remote")])).toBe("Focusa · Local + VPS");
  });

  it("keeps discovery distinct from authority and probes the canonical health route", () => {
    const source = readFileSync(new URL("../../src/lib/focusa-daemon-discovery.ts", import.meta.url), "utf8");
    expect(source).toContain("discovery never grants scope or mutation authority");
    expect(source).toContain("http://127.0.0.1:8787");
    expect(source).toContain("/v1/health");
    expect(source).toContain("discoverViaBonjour");
    expect(source).toContain("readSavedFocusaDaemonHints");
  });
});
