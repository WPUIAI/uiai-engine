import { describe, expect, it, vi } from "vitest";
import { runFocusaDiscoverySources } from "../../src/lib/discovery/focusa-discovery-source";
import { SiblingAndSavedHintDiscoverySource } from "../../src/lib/discovery/focusa-hint-source";

const hint = (url: string, daemon = "daemon_hint") => ({
  schema: "focusa.platform_hint.v1",
  daemon_url: url,
  daemon_id: daemon,
  device_id: "menubar_device_01",
  paired: true,
  client_types_seen: ["menubar"],
  last_verified_at: "2026-08-03T00:00:00Z",
});
const context = { localOnly: false, observedAt: "2026-08-03T00:01:00Z" };

describe("sibling and saved Focusa hint discovery", () => {
  it("deduplicates validated non-secret hints and prefers live daemon identity", async () => {
    const probe = vi.fn(async () => ({ healthy: true, latencyMs: 7, daemonId: "daemon_live", version: "1.0.0" }));
    const source = new SiblingAndSavedHintDiscoverySource(
      () => [hint("https://focusa.example.com")],
      () => [hint("https://focusa.example.com/", "daemon_stale")],
      probe,
    );
    const result = await runFocusaDiscoverySources([source], context);
    expect(probe).toHaveBeenCalledTimes(1);
    expect(result.candidates[0]).toMatchObject({
      source: "saved_hint",
      daemon_id: "daemon_live",
      base_url: "https://focusa.example.com",
      health_status: "healthy",
    });
  });

  it("ignores malformed and secret-bearing hints without creating candidates", async () => {
    const probe = vi.fn(async () => ({ healthy: true, latencyMs: 1 }));
    const source = new SiblingAndSavedHintDiscoverySource(
      () => [{ ...hint("https://focusa.example.com"), token: "must-not-cross" }],
      () => [{ schema: "focusa.platform_hint.v1", daemon_url: "not a url" }],
      probe,
    );
    const result = await runFocusaDiscoverySources([source], context);
    expect(result.candidates).toEqual([]);
    expect(probe).not.toHaveBeenCalled();
  });

  it("filters remote hints before probing in local-only mode", async () => {
    const probe = vi.fn(async () => ({ healthy: true, latencyMs: 1 }));
    const source = new SiblingAndSavedHintDiscoverySource(
      () => [hint("https://focusa.example.com")],
      () => [],
      probe,
    );
    const result = await runFocusaDiscoverySources([source], { ...context, localOnly: true });
    expect(result.candidates).toEqual([]);
    expect(probe).not.toHaveBeenCalled();
  });
});
