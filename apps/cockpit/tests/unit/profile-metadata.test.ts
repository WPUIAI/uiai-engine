import { describe, it, expect, beforeEach } from "vitest";
import {
  saveProfileMetadata,
  loadProfileMetadata,
  listProfileMetadata,
  removeProfileMetadata,
  validateProfileMetadata,
  __resetMemoryForTests,
  __getMemorySnapshot,
} from "../../src/lib/profiles/metadata";

const valid = {
  profileId: "profile:test-001",
  daemonId: "daemon:abc123",
  endpoint: "https://127.0.0.1:8787",
  pairedAt: "2026-08-04T00:00:00Z",
  source: "pairing" as const,
};

describe("T005-07.01 profile metadata repository", () => {
  beforeEach(() => __resetMemoryForTests());

  it("persists atomically without token material", async () => {
    await saveProfileMetadata(valid);
    expect(await loadProfileMetadata(valid.profileId)).toEqual(expect.objectContaining({ profileId: valid.profileId }));
    expect(Object.keys(__getMemorySnapshot())).toContain(valid.profileId);
    const list = await listProfileMetadata();
    expect(list).toHaveLength(1);
  });

  it("fail-closes on token material", async () => {
    expect(() => validateProfileMetadata({ ...valid, token: "secret123456789012345" } as unknown as typeof valid)).toThrow(/token material/i);
    expect(() => validateProfileMetadata({ ...valid, secret: "abc" } as unknown as typeof valid)).toThrow(/token material/i);
    await expect(saveProfileMetadata({ ...valid, profileId: "profile:bad", token: "x" } as unknown as typeof valid)).rejects.toThrow(
      /token material/i,
    );
    expect(Object.keys(__getMemorySnapshot())).not.toContain("profile:bad");
  });

  it("removes atomically", async () => {
    await saveProfileMetadata(valid);
    expect(await removeProfileMetadata(valid.profileId)).toBe(true);
    expect(await loadProfileMetadata(valid.profileId)).toBeNull();
    expect(__getMemorySnapshot()[valid.profileId]).toBeUndefined();
  });

  it("rejects invalid endpoint/query/token-shaped values", async () => {
    expect(() => validateProfileMetadata({ ...valid, endpoint: "https://example.com?token=abc" })).toThrow(/endpoint/i);
    expect(() => validateProfileMetadata({ ...valid, profileId: "https://bad" })).toThrow(/opaque/i);
  });
});
