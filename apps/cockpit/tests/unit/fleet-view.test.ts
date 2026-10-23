import { describe, it, expect, beforeEach } from "vitest";
import { saveProfileMetadata, __resetMemoryForTests } from "../../src/lib/profiles/metadata";
import { __resetResolverForTests } from "../../src/lib/profiles/resolver";
import { __resetLifecycleForTests, transitionProfileHealth } from "../../src/lib/profiles/lifecycle";
import { buildFleetView } from "../../src/lib/profiles/fleet";
import { __resetSelectionForTests } from "../../src/lib/profiles/selection";
import { clearActiveProfile } from "../../src/lib/profiles/selection";

describe("T005-08.01 fleet view model", () => {
  beforeEach(async () => {
    __resetMemoryForTests();
    __resetResolverForTests();
    __resetLifecycleForTests();
    __resetSelectionForTests();
    await clearActiveProfile();
  });

  it("retains health/location/version/scopes/source/expiry/lastUse provenance", async () => {
    const base = {
      profileId: "profile:fleet-1",
      daemonId: "daemon:fleet-a",
      endpoint: "https://127.0.0.1:8787",
      pairedAt: "2026-08-04T00:00:00Z",
      lastSeenAt: "2026-08-04T01:00:00Z",
      scopes: ["read", "write"],
      source: "pairing" as const,
      displayName: "OVH Fleet",
    };
    await saveProfileMetadata(base as unknown as never);
    // inject version via extension field (non-secret)
    const withVer = { ...base, _version: "2.0.0-dev" } as unknown as never;
    await saveProfileMetadata(withVer);

    let view = await buildFleetView();
    expect(view).toHaveLength(1);
    expect(view[0].health).toBe("active");
    expect(view[0].provenance).toBe("pairing:2026-08-04T00:00:00Z");
    expect(view[0].scopes).toEqual(["read", "write"]);
    expect(view[0].version).toBe("2.0.0-dev");
    expect(view[0].lastSeenAt).toBeDefined();

    await transitionProfileHealth("profile:fleet-1", "expired", "ev:exp-001", "ttl");
    view = await buildFleetView();
    expect(view[0].health).toBe("expired");
    expect(view[0].provenance).toBe("pairing:2026-08-04T00:00:00Z"); // provenance retained through expiry
  });

  it("multiple profiles remain isolated in view", async () => {
    await saveProfileMetadata({ profileId: "profile:fleet-2", daemonId: "daemon:b", endpoint: "https://127.0.0.1:8788", pairedAt: "2026-08-04T00:00:00Z", source: "menubar" as const, scopes: ["read"] });
    await saveProfileMetadata({ profileId: "profile:fleet-3", daemonId: "daemon:c", endpoint: "https://127.0.0.1:8789", pairedAt: "2026-08-04T00:00:00Z", source: "manual" as const });
    const view = await buildFleetView();
    expect(view.length).toBeGreaterThanOrEqual(2);
    const ids = view.map((v) => v.profileId);
    expect(ids).toContain("profile:fleet-2");
    expect(ids).toContain("profile:fleet-3");
    // no secrets in view
    expect(JSON.stringify(view)).not.toMatch(/token|secret/i);
  });
});
