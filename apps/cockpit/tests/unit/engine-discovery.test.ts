import { readFileSync } from "node:fs";
import { afterEach, describe, expect, it, vi } from "vitest";
import { discoverEngines, readEngineHints, saveEngineHints, selectBestEngine, type EngineConnection } from "../../src/lib/engine-discovery";

afterEach(() => vi.unstubAllGlobals());

describe("UIAI Engine discovery", () => {
  it("keeps a verified configured Engine instead of replacing it on latency alone", () => {
    const choices: EngineConnection[] = [
      { baseUrl: "http://127.0.0.1:7456", status: "connected", source: "loopback", latencyMs: 1 },
      { baseUrl: "https://remote.example/engine", status: "connected", source: "saved", latencyMs: 80 },
    ];
    expect(selectBestEngine(choices, "https://remote.example/engine/")?.baseUrl).toBe("https://remote.example/engine");
  });

  it("retains verified fallback when the configured Engine is unavailable", () => {
    const choices: EngineConnection[] = [
      { baseUrl: "https://remote.example", status: "unavailable", source: "saved", latencyMs: 0 },
      { baseUrl: "http://127.0.0.1:7456", status: "connected", source: "loopback", latencyMs: 5 },
    ];
    expect(selectBestEngine(choices, "https://remote.example")?.baseUrl).toBe("http://127.0.0.1:7456");
  });

  it("treats zero latency as valid when selecting a verified fallback", () => {
    const choices: EngineConnection[] = [
      { baseUrl: "https://zero.example", status: "connected", source: "saved", latencyMs: 0 },
      { baseUrl: "https://one.example", status: "connected", source: "saved", latencyMs: 1 },
    ];
    expect(selectBestEngine(choices)?.baseUrl).toBe("https://zero.example");
  });

  it("probes the configured endpoint even when it is not a discovery hint", async () => {
    const storage = new Map([["uiai.engine.url", "https://configured.example/engine/"]]);
    vi.stubGlobal("window", { setTimeout, clearTimeout, localStorage: { getItem: (key: string) => storage.get(key) || null } });
    vi.stubGlobal("localStorage", window.localStorage);
    const fetch = vi.fn(async () => ({ ok: true, json: async () => ({ status: "healthy", service: "uiai-engine" }) }));
    vi.stubGlobal("fetch", fetch);
    const results = await discoverEngines();
    expect(fetch).toHaveBeenCalledWith("https://configured.example/engine/health", expect.any(Object));
    expect(results.some((item) => item.baseUrl === "https://configured.example/engine" && item.status === "connected")).toBe(true);
  });

  it("fences shell discovery against a selection changed while probing", () => {
    const layout = readFileSync(new URL("../../src/routes/+layout.svelte", import.meta.url), "utf8");
    expect(layout).toContain("const preferred = engineUrl()");
    expect(layout).toContain("if (engineUrl() !== preferred) return");
    expect(layout).toContain("selectBestEngine(connections, preferred)");
  });
  it("still normalizes every saved discovery hint to its origin", () => {
    const storage = new Map<string, string>();
    vi.stubGlobal("window", { localStorage: { getItem: (key: string) => storage.get(key) || null, setItem: (key: string, value: string) => storage.set(key, value) } });
    vi.stubGlobal("localStorage", window.localStorage);
    expect(saveEngineHints(["https://one.example/path", "https://two.example/another"])).toEqual(["https://one.example", "https://two.example"]);
    expect(readEngineHints()).toEqual(["https://one.example", "https://two.example"]);
  });
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
