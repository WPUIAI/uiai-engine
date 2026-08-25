import { describe, expect, it, vi } from "vitest";
import { runFocusaDiscoverySources } from "../../src/lib/discovery/focusa-discovery-source";
import { TailscaleDiscoverySource } from "../../src/lib/discovery/focusa-tailscale-source";

const context = { localOnly: false, observedAt: "2026-08-03T00:00:00Z" };

describe("Focusa Tailscale discovery", () => {
  it("probes only explicit deduplicated MagicDNS names over HTTPS", async () => {
    const probe = vi.fn(async () => ({ healthy: true, latencyMs: 11, daemonId: "daemon_tailnet" }));
    const source = new TailscaleDiscoverySource([
      "focusa.tail123.ts.net",
      "FOCUSA.tail123.ts.net",
      "invalid host",
      "focusa.local",
    ], 8787, probe);
    const result = await runFocusaDiscoverySources([source], context);
    expect(probe).toHaveBeenCalledOnce();
    expect(probe).toHaveBeenCalledWith("https://focusa.tail123.ts.net:8787", expect.any(AbortSignal));
    expect(result.candidates[0]).toMatchObject({ source: "tailscale", location: "remote", daemon_id: "daemon_tailnet" });
  });

  it("blocks all tailnet access before probing in local-only mode", async () => {
    const probe = vi.fn(async () => ({ healthy: true, latencyMs: 1 }));
    const result = await runFocusaDiscoverySources([
      new TailscaleDiscoverySource(["focusa.tail123.ts.net"], 8787, probe),
    ], { ...context, localOnly: true });
    expect(result.candidates).toEqual([]);
    expect(probe).not.toHaveBeenCalled();
  });

  it("rejects invalid ports and bounds approved names", async () => {
    const probe = vi.fn(async () => ({ healthy: false, latencyMs: 1 }));
    const invalid = await runFocusaDiscoverySources([new TailscaleDiscoverySource(["focusa.ts.net"], 0, probe)], context);
    expect(invalid.candidates).toEqual([]);
    expect(probe).not.toHaveBeenCalled();

    const names = Array.from({ length: 20 }, (_, index) => `focusa-${index}.tail.ts.net`);
    await runFocusaDiscoverySources([new TailscaleDiscoverySource(names, 8787, probe)], context);
    expect(probe).toHaveBeenCalledTimes(16);
  });
});
