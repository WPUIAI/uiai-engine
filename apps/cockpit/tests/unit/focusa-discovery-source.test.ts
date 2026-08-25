import { describe, expect, it } from "vitest";
import {
  MAX_CANDIDATES_PER_SOURCE,
  runFocusaDiscoverySources,
  type FocusaDiscoverySourceAdapter,
} from "../../src/lib/discovery/focusa-discovery-source";

const candidate = (source: string, id: string, location: "local" | "remote" = "local") => ({
  schema: "focusa.daemon_candidate.v1",
  candidate_id: id,
  base_url: location === "local" ? `http://127.0.0.1:${8787 + Number(id.replace(/\D/g, "") || 0)}` : "https://focusa.example.com",
  source,
  location,
  observed_at: "2026-08-03T00:00:00Z",
  health_status: "healthy",
  latency_ms: 5,
});
const adapter = (source: FocusaDiscoverySourceAdapter["source"], values: unknown[]): FocusaDiscoverySourceAdapter => ({
  source,
  async discover() { return values; },
});
const context = { localOnly: false, observedAt: "2026-08-03T00:00:00Z" };

describe("Focusa discovery source boundary", () => {
  it("returns only canonical unprivileged candidates", async () => {
    const result = await runFocusaDiscoverySources([
      adapter("loopback", [candidate("loopback", "candidate_1")]),
      adapter("manual", [{ ...candidate("manual", "candidate_2", "remote"), token: "forbidden" }]),
    ], context);
    expect(result.candidates.map((item) => item.candidate_id)).toEqual(["candidate_1"]);
    expect(result.failures).toContainEqual({ source: "manual", reason: "invalid_candidate" });
  });

  it("isolates source failures and rejects source impersonation", async () => {
    const failing: FocusaDiscoverySourceAdapter = { source: "bonjour", async discover() { throw new Error("offline"); } };
    const result = await runFocusaDiscoverySources([
      failing,
      adapter("environment", [candidate("manual", "candidate_3")]),
    ], context);
    expect(result.candidates).toEqual([]);
    expect(result.failures).toEqual(expect.arrayContaining([
      { source: "bonjour", reason: "source_failed" },
      { source: "environment", reason: "source_mismatch" },
    ]));
  });

  it("enforces local-only filtering and per-source bounds", async () => {
    const values = Array.from({ length: MAX_CANDIDATES_PER_SOURCE + 3 }, (_, index) => candidate("saved_hint", `candidate_${index}`, index === 0 ? "remote" : "local"));
    const result = await runFocusaDiscoverySources([adapter("saved_hint", values)], { ...context, localOnly: true });
    expect(result.candidates).toHaveLength(MAX_CANDIDATES_PER_SOURCE - 1);
    expect(result.failures).toContainEqual({ source: "saved_hint", reason: "candidate_limit_exceeded" });
  });
});
