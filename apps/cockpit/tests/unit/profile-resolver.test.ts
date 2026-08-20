import { describe, it, expect, beforeEach } from "vitest";
import { saveProfileMetadata, __resetMemoryForTests } from "../../src/lib/profiles/metadata";
import {
  resolveProfileCredential,
  __resetResolverForTests,
  __setProfileLocked,
  __setProfileExpired,
  __setProfileRevoked,
  __tokenHandleForTests,
} from "../../src/lib/profiles/resolver";
import { saveSecret, clearSecret } from "../../src/lib/secure-store";

const valid = {
  profileId: "profile:resolver-001",
  daemonId: "daemon:abc123",
  endpoint: "https://127.0.0.1:8787",
  pairedAt: "2026-08-04T00:00:00Z",
  source: "pairing" as const,
};

describe("T005-07.02 profile credential resolver", () => {
  beforeEach(async () => {
    __resetMemoryForTests();
    __resetResolverForTests();
    await clearSecret(__tokenHandleForTests(valid.profileId)).catch(() => {});
  });

  it("missing when no secret", async () => {
    await saveProfileMetadata(valid);
    const res = await resolveProfileCredential(valid.profileId);
    expect(res.kind).toBe("missing");
    expect(res.profile?.profileId).toBe(valid.profileId);
  });

  it("available when secret present", async () => {
    await saveProfileMetadata(valid);
    await saveSecret(__tokenHandleForTests(valid.profileId), "secret-value-1234567890");
    const res = await resolveProfileCredential(valid.profileId);
    expect(res.kind).toBe("available");
    if (res.kind === "available") {
      expect(res.tokenHandle).toBe(__tokenHandleForTests(valid.profileId));
      // @ts-expect-error — should not expose secret
      expect((res as unknown as { secret?: string }).secret).toBeUndefined();
    }
  });

  it("locked produces typed locked", async () => {
    await saveProfileMetadata(valid);
    await saveSecret(__tokenHandleForTests(valid.profileId), "secret-value-1234567890");
    __setProfileLocked(valid.profileId, true);
    const res = await resolveProfileCredential(valid.profileId);
    expect(res.kind).toBe("locked");
    expect(res.reason).toMatch(/locked/i);
  });

  it("expired produces typed expired", async () => {
    await saveProfileMetadata(valid);
    await saveSecret(__tokenHandleForTests(valid.profileId), "secret-value-1234567890");
    __setProfileExpired(valid.profileId, true);
    const res = await resolveProfileCredential(valid.profileId);
    expect(res.kind).toBe("expired");
  });

  it("revoked produces typed revoked", async () => {
    await saveProfileMetadata(valid);
    await saveSecret(__tokenHandleForTests(valid.profileId), "secret-value-1234567890");
    __setProfileRevoked(valid.profileId, true);
    const res = await resolveProfileCredential(valid.profileId);
    expect(res.kind).toBe("revoked");
  });

  it("missing profile produces missing with null profile", async () => {
    const res = await resolveProfileCredential("profile:does-not-exist");
    expect(res.kind).toBe("missing");
    expect(res.profile).toBeNull();
  });
});
