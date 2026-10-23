import { describe, it, expect, beforeEach } from "vitest";
import { saveProfileMetadata, __resetMemoryForTests } from "../../src/lib/profiles/metadata";
import { __resetResolverForTests } from "../../src/lib/profiles/resolver";
import { transitionProfileHealth, getProfileHealth, listLifecycleTransitions, __resetLifecycleForTests } from "../../src/lib/profiles/lifecycle";
import { resolveProfileCredential } from "../../src/lib/profiles/resolver";

const valid = {
  profileId: "profile:lc-001",
  daemonId: "daemon:lc-abc",
  endpoint: "https://127.0.0.1:8787",
  pairedAt: "2026-08-04T00:00:00Z",
  source: "pairing" as const,
};

describe("T005-07.04 profile health lifecycle", () => {
  beforeEach(async () => {
    __resetMemoryForTests();
    __resetResolverForTests();
    __resetLifecycleForTests();
  });

  it("active -> unavailable -> active with evidence", async () => {
    await saveProfileMetadata(valid);
    expect(await getProfileHealth(valid.profileId)).toBe("active");
    const t1 = await transitionProfileHealth(valid.profileId, "unavailable", "ev:health-check-001", "health probe failed");
    expect(t1.from).toBe("active");
    expect(t1.to).toBe("unavailable");
    expect(t1.evidenceRef).toBe("ev:health-check-001");
    expect(await getProfileHealth(valid.profileId)).toBe("unavailable");

    const t2 = await transitionProfileHealth(valid.profileId, "active", "ev:health-check-002", "recovered");
    expect(t2.from).toBe("unavailable");
    expect(listLifecycleTransitions(valid.profileId)).toHaveLength(2);
  });

  it("expired and revoked are evidence-backed and map to resolver states", async () => {
    await saveProfileMetadata(valid);
    await transitionProfileHealth(valid.profileId, "expired", "ev:expiry-001", "token TTL exceeded");
    expect(await getProfileHealth(valid.profileId)).toBe("expired");
    const resExp = await resolveProfileCredential(valid.profileId);
    expect(resExp.kind).toBe("expired");

    // expired -> revoked allowed, active not allowed directly
    await expect(transitionProfileHealth(valid.profileId, "active", "ev:bad", "should fail")).rejects.toThrow(/not allowed/i);
    const t2 = await transitionProfileHealth(valid.profileId, "revoked", "ev:revoke-001", "operator revoke");
    expect(t2.to).toBe("revoked");
    const resRev = await resolveProfileCredential(valid.profileId);
    expect(resRev.kind).toBe("revoked");
  });

  it("conflict transitions are evidence-backed", async () => {
    await saveProfileMetadata(valid);
    const t = await transitionProfileHealth(valid.profileId, "conflict", "ev:conflict-001", "daemon identity mismatch");
    expect(t.to).toBe("conflict");
    expect(await getProfileHealth(valid.profileId)).toBe("conflict");
  });

  it("requires evidenceRef and reason", async () => {
    await saveProfileMetadata(valid);
    await expect(transitionProfileHealth(valid.profileId, "unavailable", "", "no ref")).rejects.toThrow(/evidenceRef/i);
    await expect(transitionProfileHealth(valid.profileId, "unavailable", "ev:001", "")).rejects.toThrow(/reason required/i);
  });
});
