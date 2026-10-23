import { afterEach, describe, expect, it, vi } from "vitest";
import { applyCockpitDeepLink, parseCockpitDeepLinkIntent } from "../../src/lib/desktop-deep-link";

const liveIntent = {
  schema: "uiai.cockpit.deep_link.v1",
  route: "live_session",
  target_ref: "session_01",
  handoff_ref: "handoff:abc-123",
} as const;

afterEach(() => vi.unstubAllGlobals());

describe("Cockpit desktop deep-link routing", () => {
  it("accepts the bounded native intent and rejects unknown fields", () => {
    expect(parseCockpitDeepLinkIntent(liveIntent)).toEqual(liveIntent);
    expect(() => parseCockpitDeepLinkIntent({ ...liveIntent, unexpected: true })).toThrow("unknown fields");
    expect(() => parseCockpitDeepLinkIntent({ ...liveIntent, target_ref: "../../escape" })).toThrow("target reference");
  });

  it("selects the opaque session through the canonical Live route", () => {
    const values = new Map<string, string>();
    vi.stubGlobal("window", {
      localStorage: {
        setItem: (key: string, value: string) => values.set(key, value),
        removeItem: (key: string) => values.delete(key),
      },
    });
    const navigate = vi.fn();
    applyCockpitDeepLink(liveIntent, navigate);
    expect(values.get("uiai.cockpit.requested_session_id")).toBe("session_01");
    expect(values.get("uiai.cockpit.handoff_ref")).toBe("handoff:abc-123");
    expect(navigate).toHaveBeenCalledWith("/live?session=session_01");
  });

  it("routes evidence and pairing without inventing object data", () => {
    const values = new Map<string, string>();
    vi.stubGlobal("window", { localStorage: { setItem: (key: string, value: string) => values.set(key, value), removeItem: () => undefined } });
    const navigate = vi.fn();
    applyCockpitDeepLink({ schema: "uiai.cockpit.deep_link.v1", route: "evidence", target_ref: "evidence_01", handoff_ref: null }, navigate);
    expect(values.get("uiai.cockpit.requested_evidence_ref")).toBe("evidence_01");
    expect(navigate).toHaveBeenLastCalledWith("/evidence?ref=evidence_01");
    applyCockpitDeepLink({ schema: "uiai.cockpit.deep_link.v1", route: "settings_pairing", target_ref: null, handoff_ref: null }, navigate);
    expect(navigate).toHaveBeenLastCalledWith("/settings?panel=pairing");
  });
});
