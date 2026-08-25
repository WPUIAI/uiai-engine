import { describe, expect, it } from "vitest";
import { evaluateMenubarFallback, isFallbackDecision } from "../../src/lib/pairing/menubar-fallback";
import { createMenubarCrossReferenceRequest } from "../../src/lib/pairing/menubar-cross-reference";
import manifest from "../../../../tests/fixtures/desktop-presentation/focusa-app-manifest.valid.json";

const hint = { schema: "focusa.platform_hint.v1", daemon_url: "https://focusa.example", daemon_id: "daemon_1", device_id: "menubar_1", paired: true, client_types_seen: ["menubar"], last_verified_at: "2026-08-03T00:00:00Z" } as unknown;
const cross = createMenubarCrossReferenceRequest({ device_id: "menubar_1", nonce: "nonce_1234567890abcdef", daemon_url: "https://focusa.example" });
const proof = { schema: "focusa.daemon_menubar_proof.v1", verified_device_id: "menubar_1", daemon_url: "https://focusa.example", verified_at: new Date().toISOString(), scopes: ["read"] };

describe("T005-06.05 fallback — Path A remains available, no partial mutation", () => {
  it("falls back on denied policy/dismissed without mutation", () => {
    const d = evaluateMenubarFallback({ same_machine: true, same_user: true, policy_enabled: false, dismissed: false, selected_daemon_url: "https://focusa.example", hint, manifest, crossRef: cross, daemonProof: proof, cockpit_device_id: "cockpit_dev_0987654321abcd", token_handle: "profile:2" });
    expect(isFallbackDecision(d)).toBe(true);
    if (isFallbackDecision(d)) {
      expect(d.fallbackToPathA).toBe(true);
      expect(d.authorityMutated).toBe(false);
      expect(d.partialProfileCreated).toBe(false);
      expect(d.reason).toBe("denied_by_policy");
    }
  });
  it("falls back on mismatch without partial profile", () => {
    const badProof = { ...proof, verified_device_id: "other_dev_12345678abcd" };
    const d = evaluateMenubarFallback({ same_machine: true, same_user: true, policy_enabled: true, dismissed: false, selected_daemon_url: "https://focusa.example", hint, manifest, crossRef: cross, daemonProof: badProof, cockpit_device_id: "cockpit_dev_0987654321abcd", token_handle: "profile:2" });
    expect(isFallbackDecision(d)).toBe(true);
    if (isFallbackDecision(d)) {
      expect(d.reason).toBe("mismatch");
      expect(d.partialProfileCreated).toBe(false);
      expect(d.authorityMutated).toBe(false);
    }
  });
  it("falls back on absence and leaves Path A available", () => {
    const d = evaluateMenubarFallback({ same_machine: true, same_user: true, policy_enabled: true, dismissed: false, selected_daemon_url: "https://focusa.example", hint, manifest, crossRef: null, daemonProof: null });
    expect(isFallbackDecision(d)).toBe(true);
    if (isFallbackDecision(d)) expect(d.reason).toBe("absent");
  });
  it("passes through mint on compatible proof", () => {
    const d = evaluateMenubarFallback({ same_machine: true, same_user: true, policy_enabled: true, dismissed: false, selected_daemon_url: "https://focusa.example", hint, manifest, crossRef: { schema: "focusa.menubar_cross_reference.v1", device_id: cross.device_id, nonce: cross.nonce, daemon_url: cross.daemon_url, created_at: cross.created_at }, daemonProof: proof, cockpit_device_id: "cockpit_dev_0987654321abcd", token_handle: "profile:2" });
    expect((d as { schema: string }).schema).toBe("focusa.distinct_cockpit_mint.v1");
  });
});
