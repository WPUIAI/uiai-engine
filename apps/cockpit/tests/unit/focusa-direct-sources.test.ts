import { describe, expect, it, vi } from "vitest";
import { EnvironmentDiscoverySource, LoopbackDiscoverySource, ManualDiscoverySource, type FocusaHealthProbe } from "../../src/lib/discovery/focusa-direct-sources";
import { runFocusaDiscoverySources } from "../../src/lib/discovery/focusa-discovery-source";

const healthy: FocusaHealthProbe = vi.fn(async () => ({
  healthy: true,
  latencyMs: 9,
  daemonId: "daemon_01",
  version: "1.2.3",
  capabilities: ["pair.start"],
}));
const context = { localOnly: false, observedAt: "2026-08-03T00:00:00Z" };

describe("direct Focusa discovery sources", () => {
  it("probes the canonical loopback daemon and retains health provenance", async () => {
    const result = await runFocusaDiscoverySources([new LoopbackDiscoverySource(healthy)], context);
    expect(result.failures).toEqual([]);
    expect(result.candidates[0]).toMatchObject({
      base_url: "http://127.0.0.1:8787",
      source: "loopback",
      location: "local",
      health_status: "healthy",
      daemon_id: "daemon_01",
    });
  });

  it("accepts explicit HTTPS environment and manual endpoints", async () => {
    const result = await runFocusaDiscoverySources([
      new EnvironmentDiscoverySource(["https://focusa.example.com:443"], healthy),
      new ManualDiscoverySource("https://other.example.com:9443", healthy),
    ], context);
    expect(result.candidates.map((item) => [item.source, item.location])).toEqual([
      ["environment", "remote"],
      ["manual", "remote"],
    ]);
  });

  it("rejects credentials paths queries weak remote transport and implicit ports before probing", async () => {
    const probe = vi.fn(healthy);
    const invalid = [
      "http://user:secret@127.0.0.1:8787",
      "http://127.0.0.1:8787/private",
      "http://127.0.0.1:8787?token=no",
      "http://focusa.example.com:8787",
      "https://focusa.example.com",
    ];
    const result = await runFocusaDiscoverySources([new EnvironmentDiscoverySource(invalid, probe)], context);
    expect(result.candidates).toEqual([]);
    expect(probe).not.toHaveBeenCalled();
  });

  it("blocks remote candidates before network access in local-only mode", async () => {
    const probe = vi.fn(healthy);
    const result = await runFocusaDiscoverySources([
      new ManualDiscoverySource("https://focusa.example.com:443", probe),
    ], { ...context, localOnly: true });
    expect(result.candidates).toEqual([]);
    expect(probe).not.toHaveBeenCalled();
  });

  it("returns unavailable candidates when bounded health probes fail", async () => {
    const unavailable: FocusaHealthProbe = async () => ({ healthy: false, latencyMs: 1_500 });
    const result = await runFocusaDiscoverySources([new LoopbackDiscoverySource(unavailable)], context);
    expect(result.candidates[0]).toMatchObject({ health_status: "unavailable", latency_ms: 1_500 });
  });
});
