import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { projectBindingRequiresReconciliation, type FocusaProjectBinding } from "../../src/lib/focusa-projects";

const binding = (daemonKey: string, projectRoot = "/projects/uiai"): FocusaProjectBinding => ({
  bindingId: `${daemonKey}::${projectRoot}`,
  daemonKey,
  daemonBaseUrl: daemonKey === "local" ? "http://127.0.0.1:8787" : "https://focusa-vps:8787",
  daemonLocation: daemonKey === "local" ? "local" : "remote",
  projectRoot,
  canonicalName: "UIAI Engine",
  status: "verified",
});

describe("Focusa project bindings", () => {
  it("keeps the same project on different daemons separate until explicit reconciliation", () => {
    const local = binding("local");
    const remote = binding("vps");
    expect(projectBindingRequiresReconciliation(local, [local, remote])).toBe(true);
    expect(projectBindingRequiresReconciliation(local, [local, binding("vps", "/projects/other")])).toBe(false);
  });

  it("requires daemon verification before writing the selected ScopeRef", () => {
    const source = readFileSync(new URL("../../src/lib/focusa-projects.ts", import.meta.url), "utf8");
    expect(source).toContain("/v1/project/identity");
    expect(source).toContain('binding.status !== "verified"');
    expect(source).toContain('uiai.scope.focusa_daemon_key');
    expect(source).toContain("projectBindingRequiresReconciliation");
  });
});
