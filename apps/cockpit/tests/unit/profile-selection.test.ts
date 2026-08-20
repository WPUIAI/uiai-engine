import { describe, it, expect, beforeEach } from "vitest";
import { saveProfileMetadata, __resetMemoryForTests } from "../../src/lib/profiles/metadata";
import { __resetResolverForTests } from "../../src/lib/profiles/resolver";
import { __resetLifecycleForTests, transitionProfileHealth } from "../../src/lib/profiles/lifecycle";
import {
  getActiveProfileId,
  getActiveProfile,
  setActiveProfileId,
  clearActiveProfile,
  __resetSelectionForTests,
} from "../../src/lib/profiles/selection";

const base = (id: string) => ({
  profileId: id,
  daemonId: `daemon:${id}`,
  endpoint: "https://127.0.0.1:8787",
  pairedAt: "2026-08-04T00:00:00Z",
  source: "pairing" as const,
});

describe("T005-07.05 active profile selection", () => {
  beforeEach(async () => {
    __resetMemoryForTests();
    __resetResolverForTests();
    __resetLifecycleForTests();
    __resetSelectionForTests();
    await clearActiveProfile();
  });

  it("explicit durable selection is reversible", async () => {
    await saveProfileMetadata(base("profile:sel-a"));
    await saveProfileMetadata(base("profile:sel-b"));
    await setActiveProfileId("profile:sel-a");
    expect(await getActiveProfileId()).toBe("profile:sel-a");
    expect((await getActiveProfile())?.profileId).toBe("profile:sel-a");

    await setActiveProfileId("profile:sel-b");
    expect(await getActiveProfileId()).toBe("profile:sel-b");

    await clearActiveProfile();
    expect(await getActiveProfileId()).toBeNull();
  });

  it("cannot select revoked/expired/unavailable", async () => {
    await saveProfileMetadata(base("profile:sel-c"));
    await transitionProfileHealth("profile:sel-c", "revoked", "ev:revoke", "revoked");
    await expect(setActiveProfileId("profile:sel-c")).rejects.toThrow(/health.*revoked/i);

    await saveProfileMetadata(base("profile:sel-d"));
    await transitionProfileHealth("profile:sel-d", "expired", "ev:exp", "expired");
    await expect(setActiveProfileId("profile:sel-d")).rejects.toThrow(/health.*expired/i);
  });

  it("cannot cross-bind — selection does not leak credential", async () => {
    await saveProfileMetadata(base("profile:sel-e"));
    const meta = await setActiveProfileId("profile:sel-e");
    // selection returns metadata, never token material
    expect((meta as unknown as Record<string, unknown>).token).toBeUndefined();
    expect((meta as unknown as Record<string, unknown>).secret).toBeUndefined();
    expect(await getActiveProfileId()).toBe("profile:sel-e");
  });

  it("requires existing profile", async () => {
    await expect(setActiveProfileId("profile:missing")).rejects.toThrow(/not found/i);
  });
});
