import { beforeEach, describe, expect, it } from "vitest";
import {
  ENTITLEMENT_PROJECTION_SCHEMA,
  EntitlementDeniedError,
  installEntitlementProjection,
  parseEntitlementProjection,
  requireCapabilityEntitlement,
} from "../../src/lib/contracts/entitlement";
import { engineClient } from "../../src/lib/engine-client";
import { workspaceManifest } from "../../src/lib/navigation/sidebar-manifest";

const projection = {
  schema: ENTITLEMENT_PROJECTION_SCHEMA,
  source: "uiai_authority",
  verified: true,
  product: "uiai-engine",
  state: "active_evaluation",
  capabilities: [
    { capability_id: "uiai.browser.session.create", status: "allowed", remaining: 2, limit_bucket: "browser_sessions" },
    { capability_id: "uiai.search.execute", status: "limit_reached", remaining: 0, limit_bucket: "searches" },
  ],
  recovery_actions: [{ kind: "manage_license", label: "Manage entitlement", href: "/nodes-services?view=uiai-engine" }],
  protected_worker: { worker_status: "ready", capsule_status: "mounted", version: "1.0.0", compatibility: "compatible" },
  local_artifacts: { access: "preserved", evidence: "preserved" },
  observed_at: "2026-08-03T10:00:00Z",
} as const;

beforeEach(() => installEntitlementProjection(null));

describe("canonical entitlement and allocation guard", () => {
  it("parses authority, capability limits, worker/capsule, and artifact preservation", () => {
    const parsed = parseEntitlementProjection(projection);
    expect(parsed.state).toBe("active_evaluation");
    expect(parsed.capabilities[0].remaining).toBe(2);
    expect(parsed.protected_worker).toEqual(expect.objectContaining({ worker_status: "ready", capsule_status: "mounted" }));
    expect(parsed.local_artifacts.evidence).toBe("preserved");
  });

  it.each([
    { ...projection, verified: false },
    { ...projection, source: "fixture" },
    { ...projection, token: "secret" },
    { ...projection, recovery_actions: [{ ...projection.recovery_actions[0], href: "https://example.com/license" }] },
    { ...projection, local_artifacts: { access: "blocked", evidence: "blocked" } },
  ])("rejects noncanonical or unsafe entitlement input", (candidate) => {
    expect(() => parseEntitlementProjection(candidate)).toThrow();
  });

  it("blocks allocation before entitlement success with the stable denial envelope", () => {
    expect(() => engineClient.openSession("https://example.com")).toThrow(EntitlementDeniedError);
    try {
      requireCapabilityEntitlement("uiai.browser.session.create");
    } catch (error) {
      expect((error as EntitlementDeniedError).denial).toEqual(expect.objectContaining({
        schema: "uiai.entitlement_denial.v1",
        code: "license_required",
        local_artifacts: "preserved",
      }));
    }
  });

  it("allows only explicitly signed capabilities and reports signed limits", () => {
    installEntitlementProjection(projection);
    expect(() => requireCapabilityEntitlement("uiai.browser.session.create")).not.toThrow();
    expect(() => requireCapabilityEntitlement("uiai.search.execute")).toThrowError(expect.objectContaining({
      denial: expect.objectContaining({ code: "evaluation_limit_reached" }),
    }));
    expect(() => requireCapabilityEntitlement("uiai.browser.session.control")).toThrowError(expect.objectContaining({
      denial: expect.objectContaining({ code: "license_required" }),
    }));
  });

  it("does not create a duplicate License workspace", () => {
    expect(workspaceManifest.some((workspace) => workspace.id.toLowerCase().includes("license"))).toBe(false);
  });
});
