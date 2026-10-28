import { readFileSync } from "node:fs";
import { afterEach, describe, expect, it, vi } from "vitest";
import { testConnectionDraft } from "../../src/lib/controllers/connection-test";
import { probeEngineConnection, type EngineConnection } from "../../src/lib/engine-discovery";
import type { FocusaDaemonConnection } from "../../src/lib/focusa-daemon-discovery";

afterEach(() => { vi.useRealTimers(); vi.unstubAllGlobals(); });

function dependencies() {
  const probe = vi.fn(async (_candidate: string): Promise<EngineConnection> => ({ baseUrl: "http://candidate:7456", source: "saved", status: "connected", service: "uiai-engine" }));
  const discover = vi.fn(async (): Promise<FocusaDaemonConnection[]> => []);
  return { probe, discover };
}

function installBrowser() {
  const storage = new Map([["uiai.engine.url", "http://known-working:7456"], ["uiai.scope.project_root", "/unchanged"]]);
  const setItem = vi.fn((key: string, value: string) => { storage.set(key, value); });
  vi.stubGlobal("window", { setTimeout, clearTimeout, localStorage: { getItem: (key: string) => storage.get(key) ?? null, setItem } });
  vi.stubGlobal("localStorage", window.localStorage);
  return { storage, setItem };
}

describe("Read-only Cockpit draft connection test", () => {
  it("passes the draft, not the saved Engine URL, to the canonical probe", async () => {
    const d = dependencies();
    const result = await testConnectionDraft("http://candidate:7456", d.probe, d.discover);
    expect(d.probe).toHaveBeenCalledWith("http://candidate:7456");
    expect(result.success).toBe(true);
    expect(result.message).toContain("Saved settings unchanged");
  });

  it("does not report a healthy Engine as failed when Focusa discovery fails", async () => {
    const d = dependencies();
    d.discover.mockRejectedValue(new Error("independent discovery failure"));
    const result = await testConnectionDraft("http://candidate:7456", d.probe, d.discover);
    expect(result.success).toBe(true);
    expect(result.message).toContain("healthy");
    expect(result.message).toContain("Focusa discovery unavailable");
  });

  it("reports an unavailable candidate without implying it was saved", async () => {
    const d = dependencies();
    d.probe.mockResolvedValue({ baseUrl: "http://candidate:7456", source: "saved", status: "unavailable" });
    const result = await testConnectionDraft("http://candidate:7456", d.probe, d.discover);
    expect(result.success).toBe(false);
    expect(result.message).toContain("Engine unavailable");
    expect(result.message).toContain("Saved settings unchanged");
  });

  it("retains an Engine probe failure even when independent discovery succeeds", async () => {
    const d = dependencies();
    d.probe.mockRejectedValue(new Error("invalid candidate"));
    const result = await testConnectionDraft("invalid", d.probe, d.discover);
    expect(result.success).toBe(false);
    expect(result.message).toContain("invalid candidate");
    expect(d.discover).toHaveBeenCalledOnce();
  });

  it("isolates synchronous discovery failures too", async () => {
    const d = dependencies();
    d.discover.mockImplementation(() => { throw new Error("discovery adapter failure"); });
    expect((await testConnectionDraft("http://candidate:7456", d.probe, d.discover)).success).toBe(true);
  });

  it("preserves the saved connection and scope on a real failed candidate probe", async () => {
    const browser = installBrowser();
    vi.stubGlobal("fetch", vi.fn(async () => { throw new Error("offline"); }));
    const d = dependencies();
    const result = await testConnectionDraft("http://offline:7456", probeEngineConnection, d.discover);
    expect(result.success).toBe(false);
    expect(browser.storage.get("uiai.engine.url")).toBe("http://known-working:7456");
    expect(browser.storage.get("uiai.scope.project_root")).toBe("/unchanged");
    expect(browser.setItem).not.toHaveBeenCalled();
  });

  it("preserves saved settings on a real successful candidate probe", async () => {
    const browser = installBrowser();
    const fetch = vi.fn(async () => ({ ok: true, json: async () => ({ status: "healthy", service: "uiai-engine" }) }));
    vi.stubGlobal("fetch", fetch);
    const d = dependencies();
    const result = await testConnectionDraft("http://candidate:7456", probeEngineConnection, d.discover);
    expect(result.success).toBe(true);
    expect(fetch).toHaveBeenCalledWith("http://candidate:7456/health", expect.objectContaining({ cache: "no-store", signal: expect.any(AbortSignal) }));
    expect(browser.setItem).not.toHaveBeenCalled();
    expect(browser.storage.get("uiai.engine.url")).toBe("http://known-working:7456");
  });

  it("keeps an explicit reverse-proxy API base path when probing", async () => {
    installBrowser();
    const fetch = vi.fn(async () => ({ ok: true, json: async () => ({ status: "healthy", service: "uiai-engine" }) }));
    vi.stubGlobal("fetch", fetch);
    const result = await probeEngineConnection(" https://candidate.example/engine/ ");
    expect(result.baseUrl).toBe("https://candidate.example/engine");
    expect(fetch).toHaveBeenCalledWith("https://candidate.example/engine/health", expect.any(Object));
  });

  it("rejects invalid schemes without fetching or persisting", async () => {
    const browser = installBrowser();
    const fetch = vi.fn();
    vi.stubGlobal("fetch", fetch);
    await expect(probeEngineConnection("javascript:alert(1)")).rejects.toThrow("HTTP or HTTPS");
    expect(fetch).not.toHaveBeenCalled();
    expect(browser.setItem).not.toHaveBeenCalled();
  });

  it("does not accept another service as UIAI merely because health says healthy", async () => {
    installBrowser();
    vi.stubGlobal("fetch", vi.fn(async () => ({ ok: true, json: async () => ({ status: "healthy", service: "other" }) })));
    expect((await probeEngineConnection("http://other:7456")).status).toBe("unavailable");
  });

  it("bounds the probe and treats timeout as unavailable without persistence", async () => {
    vi.useFakeTimers();
    const browser = installBrowser();
    vi.stubGlobal("fetch", vi.fn((_url: string, { signal }: RequestInit) => new Promise((_resolve, reject) => {
      signal?.addEventListener("abort", () => reject(new DOMException("aborted", "AbortError")), { once: true });
    })));
    const result = probeEngineConnection("http://slow:7456");
    await vi.advanceTimersByTimeAsync(1800);
    expect((await result).status).toBe("unavailable");
    expect(browser.setItem).not.toHaveBeenCalled();
  });

  it("keeps Save distinct in the actual Settings component and errors non-successful", () => {
    const page = readFileSync(new URL("../../src/routes/settings/+page.svelte", import.meta.url), "utf8");
    const testHandler = page.slice(page.indexOf("async function testConnection()"), page.indexOf("onMount(() =>"));
    expect(testHandler).toContain("testConnectionDraft(candidate)");
    expect(testHandler).not.toContain("saveSettings()");
    expect(testHandler).not.toContain("setItem(");
    expect(page).toContain('onclick={saveSettings} disabled={testing}');
    expect(page).toContain('class:error={!connectionSuccess}');
    expect(page).toContain('window.localStorage.setItem("uiai.engine.url"');
  });
});
