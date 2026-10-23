import { describe, expect, it } from "vitest";
import {
  COCKPIT_WORKPOINT_RESUME_SCHEMA,
  parseCockpitWorkpointResume,
} from "../../src/lib/contracts/workpoint-resume";

const resumable = {
  schema: COCKPIT_WORKPOINT_RESUME_SCHEMA,
  source: "focusa_workpoint_resume",
  canonical: true,
  status: "resumable",
  workpoint_id: "workpoint:001",
  label: "Verify Cockpit navigation",
  target: {
    workspace_id: "overview",
    href: "/?workpoint=workpoint%3A001",
    object_ref: "workpoint:001",
  },
  observed_at: "2026-08-03T09:00:00Z",
} as const;

describe("Cockpit Workpoint Resume view model", () => {
  it("accepts an explicitly canonical resumable contract", () => {
    const parsed = parseCockpitWorkpointResume(resumable);
    expect(parsed.status).toBe("resumable");
    if (parsed.status === "resumable") expect(parsed.target.workspace_id).toBe("overview");
  });

  it("accepts blocked state only with an exact local recovery action", () => {
    const parsed = parseCockpitWorkpointResume({
      schema: COCKPIT_WORKPOINT_RESUME_SCHEMA,
      source: "focusa_workpoint_resume",
      canonical: true,
      status: "blocked",
      recovery: {
        message: "Verify project scope before resuming.",
        href: "/settings?section=scope",
        action_label: "Verify scope",
      },
      observed_at: "2026-08-03T09:00:00Z",
    });
    expect(parsed.status).toBe("blocked");
  });

  it.each([
    { ...resumable, canonical: false },
    { ...resumable, source: "fixture" },
    { ...resumable, workpoint_id: "/tmp/private" },
    { ...resumable, target: { ...resumable.target, href: "https://example.com/workpoint" } },
    { ...resumable, token: "secret" },
  ])("fails closed for non-canonical or unsafe state", (candidate) => {
    expect(() => parseCockpitWorkpointResume(candidate)).toThrow();
  });
});
