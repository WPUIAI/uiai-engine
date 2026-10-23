import { describe, expect, it } from "vitest";
import { createMenubarCrossReferenceRequest, parseMenubarCrossReferenceRequest } from "../../src/lib/pairing/menubar-cross-reference";

describe("T005-06.03 menubar cross-reference request", () => {
  it("creates and parses a valid request with device_id + nonce and no secret", () => {
    const req = createMenubarCrossReferenceRequest({
      device_id: "menubar_device_abc123",
      nonce: "room_nonce_xyz-789",
      daemon_url: "http://127.0.0.1:8787",
      created_at: "2026-08-19T12:00:00.000Z",
    });
    expect(req.schema).toBe("focusa.menubar_cross_reference.v1");
    const parsed = parseMenubarCrossReferenceRequest(req);
    expect(parsed.device_id).toBe("menubar_device_abc123");
    expect(parsed.nonce).toBe("room_nonce_xyz-789");
    expect((parsed as unknown as Record<string, unknown>).token).toBeUndefined();
  });

  it("rejects any secret-bearing field", () => {
    expect(() => parseMenubarCrossReferenceRequest({ schema: "focusa.menubar_cross_reference.v1", device_id: "a", nonce: "b", daemon_url: "http://127.0.0.1:8787", created_at: new Date().toISOString(), token: "secret" })).toThrow(/forbidden secret/);
    expect(() => parseMenubarCrossReferenceRequest({ schema: "focusa.menubar_cross_reference.v1", device_id: "a", nonce: "b", daemon_url: "http://127.0.0.1:8787", created_at: new Date().toISOString(), secret: "x" })).toThrow(/forbidden secret/);
  });

  it("rejects unknown fields and invalid opaque/url", () => {
    expect(() => parseMenubarCrossReferenceRequest({ schema: "focusa.menubar_cross_reference.v1", device_id: "a", nonce: "b", daemon_url: "http://127.0.0.1:8787", created_at: new Date().toISOString(), extra: "nope" })).toThrow("unknown field");
    expect(() => createMenubarCrossReferenceRequest({ device_id: "", nonce: "b", daemon_url: "http://127.0.0.1:8787" })).toThrow();
    expect(() => createMenubarCrossReferenceRequest({ device_id: "a", nonce: "b", daemon_url: "not-a-url" })).toThrow();
  });
});
