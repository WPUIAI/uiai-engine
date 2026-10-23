import { describe, expect, it } from "vitest";
import { mergeFocusaDaemonCandidates } from "../../src/lib/discovery/focusa-candidate-merge";
import type { DiscoverySource, FocusaDaemonCandidateV1 } from "../../src/lib/contracts/focusa-pairing";

const candidate = (id: string, source: DiscoverySource, url: string, daemon?: string, healthy = true): FocusaDaemonCandidateV1 => ({
  schema: "focusa.daemon_candidate.v1", candidate_id: id, base_url: url, source,
  location: url.includes("127.0.0.1") ? "local" : "remote", observed_at: "2026-08-03T00:00:00Z",
  health_status: healthy ? "healthy" : "unavailable", latency_ms: source === "manual" ? 20 : 10,
  daemon_id: daemon, capabilities: [`source.${source}`],
});

describe("Focusa candidate merge", () => {
  it("uses verified daemon identity across URLs and retains all provenance", () => {
    const merged = mergeFocusaDaemonCandidates([
      candidate("saved", "saved_hint", "https://focusa.example.com", "daemon_1"),
      candidate("tail", "tailscale", "https://focusa.tail.ts.net:8787", "daemon_1"),
    ]);
    expect(merged).toHaveLength(1);
    expect(merged[0]).toMatchObject({ daemon_id: "daemon_1", provenance_sources: ["saved_hint", "tailscale"] });
    expect([merged[0].base_url, ...merged[0].alternative_urls]).toEqual(expect.arrayContaining([
      "https://focusa.example.com", "https://focusa.tail.ts.net:8787",
    ]));
  });

  it("uses URL identity when no verified daemon ID exists and joins anonymous observations to verified URL", () => {
    const merged = mergeFocusaDaemonCandidates([
      candidate("anon", "bonjour", "http://focusa.local:8787"),
      candidate("verified", "environment", "http://focusa.local:8787", "daemon_lan"),
      candidate("other", "manual", "https://other.example:8787"),
    ]);
    expect(merged).toHaveLength(2);
    expect(merged.find((item) => item.daemon_id === "daemon_lan")?.provenance_sources).toEqual(["environment", "bonjour"]);
  });

  it("prefers healthy observations without discarding explicit-source provenance", () => {
    const merged = mergeFocusaDaemonCandidates([
      candidate("manual-down", "manual", "https://focusa.example:8787", "daemon_1", false),
      candidate("tail-live", "tailscale", "https://focusa.tail.ts.net:8787", "daemon_1", true),
    ]);
    expect(merged[0].candidate_id).toBe("tail-live");
    expect(merged[0].provenance_sources).toEqual(["manual", "tailscale"]);
  });
});
