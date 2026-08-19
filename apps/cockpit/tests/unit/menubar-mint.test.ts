import { describe, expect, it } from "vitest";
import { createMenubarCrossReferenceRequest } from "../../src/lib/pairing/menubar-cross-reference";
import { mintDistinctCockpitDevice } from "../../src/lib/pairing/menubar-mint";

const cross = createMenubarCrossReferenceRequest({ device_id: "menubar_dev_1234567890abcd", nonce: "nonce_1234567890abcdef", daemon_url: "http://127.0.0.1:8787" });
const proof = { schema: "focusa.daemon_menubar_proof.v1", verified_device_id: "menubar_dev_1234567890abcd", daemon_url: "http://127.0.0.1:8787", verified_at: new Date().toISOString(), scopes: ["read"] };

describe("T005-06.04 distinct Cockpit token minting", () => {
  it("mints distinct device + handle from valid proof", () => {
    const out = mintDistinctCockpitDevice({ crossRef: { schema: "focusa.menubar_cross_reference.v1", device_id: cross.device_id, nonce: cross.nonce, daemon_url: cross.daemon_url, created_at: cross.created_at }, daemonProof: proof, cockpit_device_id: "cockpit_dev_0987654321abcd", token_handle: "profile:2" });
    expect(out.distinct).toBe(true);
    expect(out.cockpit_device_id).not.toBe(cross.device_id);
    expect(out.menubar_device_id).toBe(cross.device_id);
  });
  it("rejects copy of menubar device id", () => {
    expect(() => mintDistinctCockpitDevice({ crossRef: { schema: "focusa.menubar_cross_reference.v1", device_id: cross.device_id, nonce: cross.nonce, daemon_url: cross.daemon_url, created_at: cross.created_at }, daemonProof: proof, cockpit_device_id: cross.device_id, token_handle: "profile:2" })).toThrow(/distinct/);
  });
  it("rejects secret-bearing proof or origin mismatch", () => {
    expect(() => mintDistinctCockpitDevice({ crossRef: { schema: "focusa.menubar_cross_reference.v1", device_id: cross.device_id, nonce: cross.nonce, daemon_url: cross.daemon_url, created_at: cross.created_at }, daemonProof: { ...proof, verified_device_id: "other_dev_12345678abcd" }, cockpit_device_id: "cockpit_dev_0987654321abcd", token_handle: "profile:2" })).toThrow(/mismatch/);
    expect(() => mintDistinctCockpitDevice({ crossRef: { schema: "focusa.menubar_cross_reference.v1", device_id: cross.device_id, nonce: cross.nonce, daemon_url: cross.daemon_url, created_at: cross.created_at }, daemonProof: { ...proof, token: "secret" } as unknown as Record<string,unknown>, cockpit_device_id: "cockpit_dev_0987654321abcd", token_handle: "profile:2" })).toThrow(/forbidden/);
  });
});
