import { describe, it, expect, beforeEach } from "vitest";
import { saveProfileMetadata, __resetMemoryForTests } from "../../src/lib/profiles/metadata";
import { __resetResolverForTests } from "../../src/lib/profiles/resolver";
import { __resetLifecycleForTests } from "../../src/lib/profiles/lifecycle";
import { __resetSelectionForTests, clearActiveProfile, setActiveProfileId } from "../../src/lib/profiles/selection";
import { resolvePickerState } from "../../src/lib/profiles/fleet-cardinality";

describe("T005-08.02 zero/one/many", () => {
  beforeEach(async () => {
    __resetMemoryForTests();
    __resetResolverForTests();
    __resetLifecycleForTests();
    __resetSelectionForTests();
    await clearActiveProfile();
  });

  it("zero shows pairing", async () => {
    const s = await resolvePickerState();
    expect(s.cardinality).toBe("zero");
    expect(s.action).toBe("needs_pairing");
  });

  it("one may default to healthy sole profile", async () => {
    await saveProfileMetadata({ profileId: "profile:card-1", daemonId: "daemon:a", endpoint: "https://127.0.0.1:8787", pairedAt: "2026-08-04T00:00:00Z", source: "pairing" as const });
    const s = await resolvePickerState();
    expect(s.cardinality).toBe("one");
    expect(s.action).toBe("defaulted");
    expect(s.activeProfileId).toBe("profile:card-1");
  });

  it("many always requires selection — no auto-default", async () => {
    await saveProfileMetadata({ profileId: "profile:card-1", daemonId: "daemon:a", endpoint: "https://127.0.0.1:8787", pairedAt: "2026-08-04T00:00:00Z", source: "pairing" as const });
    await saveProfileMetadata({ profileId: "profile:card-2", daemonId: "daemon:b", endpoint: "https://127.0.0.1:8788", pairedAt: "2026-08-04T00:00:00Z", source: "manual" as const });
    const s = await resolvePickerState();
    expect(s.cardinality).toBe("many");
    expect(s.action).toBe("needs_selection");
    expect(s.activeProfileId).toBeNull();
  });

  it("many with explicit selection stays selected", async () => {
    await saveProfileMetadata({ profileId: "profile:card-1", daemonId: "daemon:a", endpoint: "https://127.0.0.1:8787", pairedAt: "2026-08-04T00:00:00Z", source: "pairing" as const });
    await saveProfileMetadata({ profileId: "profile:card-2", daemonId: "daemon:b", endpoint: "https://127.0.0.1:8788", pairedAt: "2026-08-04T00:00:00Z", source: "manual" as const });
    await setActiveProfileId("profile:card-2");
    const s = await resolvePickerState();
    expect(s.cardinality).toBe("many");
    expect(s.action).toBe("selected");
  });
});
