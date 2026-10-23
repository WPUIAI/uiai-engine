import { describe, it, expect, beforeEach } from "vitest";
import { saveProfileMetadata, __resetMemoryForTests } from "../../src/lib/profiles/metadata";
import { __resetResolverForTests, __setProfileLocked, __tokenHandleForTests } from "../../src/lib/profiles/resolver";
import { createProfileScopedClient } from "../../src/lib/profiles/client";
import { saveSecret, clearSecret } from "../../src/lib/secure-store";

const valid = {
  profileId: "profile:client-001",
  daemonId: "daemon:client-abc",
  endpoint: "https://127.0.0.1:8787",
  pairedAt: "2026-08-04T00:00:00Z",
  source: "pairing" as const,
  scopes: ["read", "write"],
};

describe("T005-07.03 profile-scoped authenticated client", () => {
  beforeEach(async () => {
    __resetMemoryForTests();
    __resetResolverForTests();
    await clearSecret(__tokenHandleForTests(valid.profileId)).catch(() => {});
  });

  it("binds exactly one profile daemon identity and scopes", async () => {
    await saveProfileMetadata(valid);
    await saveSecret(__tokenHandleForTests(valid.profileId), "tok-1234567890-abcdefghij");
    const client = await createProfileScopedClient(valid.profileId);
    expect(client.profile.profileId).toBe(valid.profileId);
    expect(client.daemonId).toBe(valid.daemonId);
    expect(client.endpoint).toBe(valid.endpoint);
    expect(client.scopes).toEqual(["read", "write"]);
    const res = (await client.invoke("test_cmd", { foo: "bar" })) as unknown as Record<string, unknown>;
    expect(res.profileId).toBe(valid.profileId);
    expect(res.daemonId).toBe(valid.daemonId);
    expect(res.tokenHandle).toBe(__tokenHandleForTests(valid.profileId));
    // should not leak secret value
    expect(JSON.stringify(res)).not.toContain("tok-1234567890");
  });

  it("fails closed on missing credential", async () => {
    await saveProfileMetadata(valid);
    await expect(createProfileScopedClient(valid.profileId)).rejects.toThrow(/missing/i);
  });

  it("fails closed on locked", async () => {
    await saveProfileMetadata(valid);
    await saveSecret(__tokenHandleForTests(valid.profileId), "tok-1234567890-abcdefghij");
    __setProfileLocked(valid.profileId, true);
    await expect(createProfileScopedClient(valid.profileId)).rejects.toThrow(/locked/i);
  });

  it("fails closed on ambiguous binding", async () => {
    await saveProfileMetadata(valid);
    await saveSecret(__tokenHandleForTests(valid.profileId), "tok-1234567890-abcdefghij");
    const client = await createProfileScopedClient(valid.profileId);
    await expect(client.invoke("test_cmd", { profile_id: "profile:other" } as unknown as Record<string, unknown>)).rejects.toThrow(
      /ambiguous/i,
    );
  });

  it("single binding enforced — scopes retained", async () => {
    await saveProfileMetadata({ ...valid, scopes: ["scope:a"] });
    await saveSecret(__tokenHandleForTests(valid.profileId), "tok-1234567890-abcdefghij");
    const client = await createProfileScopedClient(valid.profileId);
    expect(client.scopes).toEqual(["scope:a"]);
  });
});
