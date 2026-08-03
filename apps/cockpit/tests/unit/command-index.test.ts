import { describe, expect, it } from "vitest";
import { phase0Cards } from "../../src/lib/cards/phase0-card-manifest";
import { COCKPIT_WORKPOINT_RESUME_SCHEMA, type CockpitWorkpointResume } from "../../src/lib/contracts/workpoint-resume";
import { buildCommandIndex, filterCommandIndex } from "../../src/lib/navigation/command-index";
import { footerDestinations, workspaceManifest } from "../../src/lib/navigation/sidebar-manifest";

const resume: CockpitWorkpointResume = {
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
};

describe("global command index", () => {
  it("indexes every enabled workspace, registered object, and unique capability", () => {
    const commands = buildCommandIndex(workspaceManifest, phase0Cards, footerDestinations, null);
    const commandIds = new Set(commands.map((command) => command.id));

    for (const workspace of workspaceManifest.filter((item) => item.state !== "planned")) {
      expect(commandIds).toContain(`workspace:${workspace.id}`);
    }
    for (const workspace of workspaceManifest.filter((item) => item.state === "planned")) {
      expect(commandIds).not.toContain(`workspace:${workspace.id}`);
    }
    for (const card of phase0Cards) expect(commandIds).toContain(`object:${card.card_id}`);
    for (const capability of new Set(phase0Cards.flatMap((card) => card.capabilities))) {
      expect(commandIds).toContain(`capability:${capability}`);
    }
    expect(commands.map((command) => command.id)).toHaveLength(commandIds.size);
  });

  it("does not fabricate Resume Workpoint and adds exactly one canonical resume command", () => {
    const absent = buildCommandIndex(workspaceManifest, phase0Cards, footerDestinations, null);
    expect(absent.some((command) => command.kind === "resume")).toBe(false);

    const present = buildCommandIndex(workspaceManifest, phase0Cards, footerDestinations, resume);
    expect(present.filter((command) => command.kind === "resume")).toEqual([
      expect.objectContaining({ href: resume.target.href, label: `Resume ${resume.label}` }),
    ]);

    const mismatched = buildCommandIndex(workspaceManifest, phase0Cards, footerDestinations, {
      ...resume,
      target: { ...resume.target, workspace_id: "live" },
    });
    expect(mismatched.some((command) => command.kind === "resume")).toBe(false);
  });

  it("matches aliases and all query terms", () => {
    const commands = buildCommandIndex(workspaceManifest, phase0Cards, footerDestinations, resume);
    expect(filterCommandIndex(commands, "session diagnostics").map((command) => command.id))
      .toContain("capability:uiai.session.diagnostics.read");
    expect(filterCommandIndex(commands, "focusa workpoint resume read").map((command) => command.id))
      .toContain("capability:focusa.workpoint.resume.read");
    expect(filterCommandIndex(commands, "workpoint:001").map((command) => command.kind))
      .toContain("resume");
  });
});
