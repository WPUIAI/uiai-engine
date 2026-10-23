import { afterEach, describe, expect, it, vi } from "vitest";
import { discoverEngines, selectBestEngine } from "../../src/lib/engine-discovery";

afterEach(() => vi.unstubAllGlobals());

describe("UIAI Engine discovery", () => {
  it("auto-connects only health-verified UIAI Engine candidates", async () => {
    const storage = new Map<string, string>();
    vi.stubGlobal("window", { setTimeout, clearTimeout, localStorage: { getItem: (key: string) => storage.get(key) || null, setItem: (key: string, value: string) => storage.set(key, value) } });
    vi.stubGlobal("localStorage", window.localStorage);
    vi.stubGlobal("performance", { now: () => 10 });
    vi.stubGlobal("fetch", vi.fn(async (url: string) => ({ ok: true, json: async () => url.includes("127.0.0.1") ? { service: "uiai-engine", status: "healthy", browserless: true } : { service: "other", status: "healthy" } })));
    const connections = await discoverEngines();
    expect(selectBestEngine(connections)?.baseUrl).toBe("http://127.0.0.1:7456");
    expect(connections.filter((item) => item.status === "connected")).toHaveLength(1);
  });
});
