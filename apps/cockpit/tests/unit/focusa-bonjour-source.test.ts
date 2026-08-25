import { describe, expect, it, vi } from "vitest";
import { BonjourDiscoverySource } from "../../src/lib/discovery/focusa-bonjour-source";
import { runFocusaDiscoverySources } from "../../src/lib/discovery/focusa-discovery-source";

const context = { localOnly: false, observedAt: "2026-08-03T00:00:00Z" };
const probe = vi.fn(async () => ({ healthy: true, latencyMs: 12, daemonId: "daemon_lan" }));

describe("Focusa Bonjour discovery", () => {
  it("accepts matching bounded .local service endpoints", async () => {
    const discover = vi.fn(async () => ({ url: "http://focusa-host.local:8787", host: "focusa-host.local.", port: 8787 }));
    const result = await runFocusaDiscoverySources([new BonjourDiscoverySource(discover, probe)], context);
    expect(discover).toHaveBeenCalledWith(2);
    expect(result.candidates[0]).toMatchObject({ source: "bonjour", location: "remote", base_url: "http://focusa-host.local:8787", daemon_id: "daemon_lan" });
  });

  it.each([
    { url: "http://evil.example:8787", host: "evil.example", port: 8787 },
    { url: "http://other.local:8787", host: "focusa.local", port: 8787 },
    { url: "http://user:secret@focusa.local:8787", host: "focusa.local", port: 8787 },
    { url: "http://focusa.local:8787/private", host: "focusa.local", port: 8787 },
  ])("rejects malformed or mismatched service %#", async (service) => {
    const localProbe = vi.fn(probe);
    const result = await runFocusaDiscoverySources([new BonjourDiscoverySource(async () => service, localProbe)], context);
    expect(result.candidates).toEqual([]);
    expect(localProbe).not.toHaveBeenCalled();
  });

  it("does not browse or probe LAN services in local-only mode", async () => {
    const discover = vi.fn(async () => ({ url: "http://focusa.local:8787", host: "focusa.local", port: 8787 }));
    const result = await runFocusaDiscoverySources([new BonjourDiscoverySource(discover, probe)], { ...context, localOnly: true });
    expect(result.candidates).toEqual([]);
    expect(discover).not.toHaveBeenCalled();
  });
});
