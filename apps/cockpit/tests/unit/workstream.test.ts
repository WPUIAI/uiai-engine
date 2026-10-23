import { describe, expect, it } from "vitest";
import { workstreamKey, parseWorkstreamKey, assertWorkstreamKeyMatchesScope } from "../../src/lib/contracts/workstream";
import { parseDesktopPresentationRequest } from "../../src/lib/contracts/desktop-presentation";

describe("Workstream-rooted canonical runtime (docs/164)", () => {
  it("derives WorkstreamKey as ProjectRoot::Continuity", () => {
    expect(workstreamKey("/home/wirebot/focusa", "cad135")).toBe("/home/wirebot/focusa::cad135");
    expect(parseWorkstreamKey("/a::b")).toEqual({ projectRootKey: "/a", continuityId: "b" });
  });
  it("rejects mismatched workstream_key vs scope", () => {
    expect(() => assertWorkstreamKeyMatchesScope({ project_root_key: "/a", continuity_id: "c1", workstream_key: "/a::c2" })).toThrow(/mismatch/);
    expect(() => assertWorkstreamKeyMatchesScope({ project_root_key: "/a", continuity_id: "c1", workstream_key: "bad" })).toThrow();
  });
  it("desktop presentation parse rejects mismatch workstream_key (opaque style)", () => {
    const base: Record<string, unknown> = {
      schema: "uiai.desktop_presentation_request.v1",
      mode: "full",
      reason: "operator_request",
      scope_ref: { project_root_key: "proj_a", continuity_id: "c1", workstream_key: "proj_a::c2", authority_state: "verified" },
      requested_by: { client_type: "cockpit", client_id: "cockpit_1" },
      focus: true,
      expires_in_ms: 5000,
      idempotency_key: "idem_1",
    };
    expect(() => parseDesktopPresentationRequest(base)).toThrow(/workstream_key mismatch/);
  });
  it("desktop presentation accepts matching workstream_key", () => {
    const base: Record<string, unknown> = {
      schema: "uiai.desktop_presentation_request.v1",
      mode: "full",
      reason: "operator_request",
      scope_ref: { project_root_key: "proj_a", continuity_id: "c1", workstream_key: "proj_a::c1", authority_state: "verified" },
      requested_by: { client_type: "cockpit", client_id: "cockpit_1" },
      focus: true,
      expires_in_ms: 5000,
      idempotency_key: "idem_2",
    };
    expect(() => parseDesktopPresentationRequest(base)).not.toThrow();
  });
  it("auto-derives workstream_key in scope when missing", async () => {
    const { scopeWorkstreamKey } = await import("../../src/lib/contracts/scope-ref");
    expect(scopeWorkstreamKey({ project_root_key: "/a", continuity_id: "c1", authority_state: "verified" })).toBe("/a::c1");
    expect(scopeWorkstreamKey({ project_root_key: "proj_a", continuity_id: "c1", authority_state: "verified" })).toBe("proj_a::c1");
  });
});
