import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { parseAuthenticatedDaemonProfile, parseCockpitPairingRoom, parseDaemonProjectCandidate, parseFocusaDaemonCandidate, parseFocusaPlatformHint, parseProjectScopeBinding, parseScopeReconciliationDecision } from "../../src/lib/contracts/focusa-pairing";

const bundle = JSON.parse(readFileSync(new URL("../fixtures/focusa-pairing-valid.json", import.meta.url), "utf8"));

describe("UIAI-COCKPIT-005 pairing contracts", () => {
  it("validates the complete daemon-qualified contract bundle", () => {
    expect(parseFocusaDaemonCandidate(bundle.candidate).daemon_id).toBe("daemon_local");
    expect(parseFocusaPlatformHint(bundle.hint).paired).toBe(true);
    expect(parseCockpitPairingRoom(bundle.room).client_type).toBe("cockpit");
    expect(parseAuthenticatedDaemonProfile(bundle.profile).token_handle).toBe("keychain_profile_vps");
    expect(parseDaemonProjectCandidate(bundle.project).daemon_id).toBe("daemon_vps");
    expect(parseProjectScopeBinding(bundle.binding).continuity_id).toBe("cad135-mission-canvas");
    expect(parseScopeReconciliationDecision(bundle.reconciliation).relation).toBe("same_project_separate_authority");
  });

  it("rejects secret-bearing hints and unknown fields", () => {
    expect(() => parseFocusaPlatformHint({ ...bundle.hint, token: "secret" })).toThrow(/forbidden secret|unknown field/);
    expect(() => parseCockpitPairingRoom({ ...bundle.room, unexpected: true })).toThrow("unknown field");
    expect(() => parseFocusaDaemonCandidate({ ...bundle.candidate, source: "authority" })).toThrow("source unsupported");
  });

  it("rejects cross-daemon reconciliation without explicit confirmation", () => {
    expect(() => parseScopeReconciliationDecision({ ...bundle.reconciliation, operator_confirmed: false })).toThrow("operator confirmation");
    expect(() => parseScopeReconciliationDecision({ ...bundle.reconciliation, right_binding_id: bundle.reconciliation.left_binding_id })).toThrow("must differ");
  });
});
