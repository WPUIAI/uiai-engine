import { describe, it, expect, beforeEach } from "vitest";
import { saveProfileMetadata, loadProfileMetadata, __resetMemoryForTests } from "../../src/lib/profiles/metadata";
import {
  __resetResolverForTests,
  __tokenHandleForTests,
  resolveProfileCredential,
  __setProfileExpired,
} from "../../src/lib/profiles/resolver";
import { createProfileScopedClient } from "../../src/lib/profiles/client";
import { saveSecret, clearSecret } from "../../src/lib/secure-store";
import { __resetLifecycleForTests, transitionProfileHealth } from "../../src/lib/profiles/lifecycle";
import { __resetSelectionForTests, setActiveProfileId, getActiveProfileId, clearActiveProfile } from "../../src/lib/profiles/selection";

const mk = (id: string, daemon: string) => ({
  profileId: id,
  daemonId: daemon,
  endpoint: "https://127.0.0.1:8787",
  pairedAt: "2026-08-04T00:00:00Z",
  source: "pairing" as const,
  scopes: ["read"],
});

describe("T005-07.06 profile isolation proof", () => {
  beforeEach(async () => {
    __resetMemoryForTests();
    __resetResolverForTests();
    __resetLifecycleForTests();
    __resetSelectionForTests();
    await clearActiveProfile();
  });

  it("concurrent profiles — no cross-bind", async () => {
    const a = mk("profile:iso-a", "daemon:a");
    const b = mk("profile:iso-b", "daemon:b");
    await saveProfileMetadata(a);
    await saveProfileMetadata(b);
    await saveSecret(__tokenHandleForTests(a.profileId), "tok-a-1234567890-aaaa");
    await saveSecret(__tokenHandleForTests(b.profileId), "tok-b-1234567890-bbbb");

    const ca = await createProfileScopedClient(a.profileId);
    const cb = await createProfileScopedClient(b.profileId);

    const ra = (await ca.invoke("ping", { x: 1 })) as unknown as Record<string, unknown>;
    const rb = (await cb.invoke("ping", { x: 2 })) as unknown as Record<string, unknown>;

    expect(ra.profileId).toBe(a.profileId);
    expect(rb.profileId).toBe(b.profileId);
    expect(ra.daemonId).not.toBe(rb.daemonId);
    expect(JSON.stringify(ra)).not.toContain("tok-b");
    expect(JSON.stringify(rb)).not.toContain("tok-a");
    await expect(ca.invoke("ping", { profile_id: b.profileId } as unknown as Record<string, unknown>)).rejects.toThrow(/ambiguous/i);
  });

  it("scope denial — wrong scope cannot be elevated via client", async () => {
    await saveProfileMetadata({ ...mk("profile:iso-scope", "daemon:scope"), scopes: ["read"] });
    await saveSecret(__tokenHandleForTests("profile:iso-scope"), "tok-scope-1234567890");
    const c = await createProfileScopedClient("profile:iso-scope");
    expect(c.scopes).toEqual(["read"]);
    // client does not magically add scopes; selection preserves stored scopes
    const res = (await c.invoke("checked", {})) as unknown as Record<string, unknown>;
    expect((res as { scopes: string[] }).scopes).toEqual(["read"]);
  });

  it("restart / expiry — expired profile cannot be selected or used", async () => {
    await saveProfileMetadata(mk("profile:iso-exp", "daemon:exp"));
    await saveSecret(__tokenHandleForTests("profile:iso-exp"), "tok-exp-1234567890");
    await transitionProfileHealth("profile:iso-exp", "expired", "ev:exp-001", "ttl");
    expect((await resolveProfileCredential("profile:iso-exp")).kind).toBe("expired");
    await expect(setActiveProfileId("profile:iso-exp")).rejects.toThrow(/expired/i);
    await expect(createProfileScopedClient("profile:iso-exp")).rejects.toThrow(/expired/i);

    // Even if secret is renewed, expired health still blocks until lifecycle transition
    await saveSecret(__tokenHandleForTests("profile:iso-exp"), "tok-exp-1234567890-renewed");
    await expect(createProfileScopedClient("profile:iso-exp")).rejects.toThrow(/expired/i);
    expect((await resolveProfileCredential("profile:iso-exp")).kind).toBe("expired");
  });

  it("identity-change — daemon identity mismatch moves to conflict and blocks use", async () => {
    await saveProfileMetadata(mk("profile:iso-id", "daemon:orig"));
    await saveSecret(__tokenHandleForTests("profile:iso-id"), "tok-id-1234567890");
    // identity change detected
    await transitionProfileHealth("profile:iso-id", "conflict", "ev:identity-001", "daemon_id changed");

    // conflict blocks selection
    await expect(setActiveProfileId("profile:iso-id")).rejects.toThrow(/conflict/i);

    // but metadata still present, no secret leak
    const meta = await loadProfileMetadata("profile:iso-id");
    expect(meta?.daemonId).toBe("daemon:orig");
    expect(JSON.stringify(meta)).not.toContain("tok-id");
  });

  it("durable selection reversible without credential leak across restarts", async () => {
    await saveProfileMetadata(mk("profile:iso-dur", "daemon:dur"));
    await saveSecret(__tokenHandleForTests("profile:iso-dur"), "tok-dur-1234567890");
    await setActiveProfileId("profile:iso-dur");
    expect(await getActiveProfileId()).toBe("profile:iso-dur");
    await clearActiveProfile();
    expect(await getActiveProfileId()).toBeNull();
    await setActiveProfileId("profile:iso-dur");
    expect(await getActiveProfileId()).toBe("profile:iso-dur");
  });
});
