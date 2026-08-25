/** Workstream-rooted canonical runtime — Focusa docs/164, WorkstreamKey = ProjectRootKey + continuity_id.
 *  One partition per workstream: state/evidence/compaction. Mirrors
 *  focusa-core workstream_root.rs resolution_key = workstream_id|continuity|canonical_root.
 */
export const WORKSTREAM_SCHEMA = "focusa.workstream_root.v1" as const;

export function workstreamKey(projectRootKey: string, continuityId: string): string {
  if (!projectRootKey || !continuityId) throw new Error("workstreamKey requires project_root_key and continuity_id");
  const root = projectRootKey.replace(/\/+$/, "");
  // Dual-form: filesystem path (must be absolute, may contain /) OR opaque project id (e.g. proj_a).
  const isPath = root.includes("/");
  if (isPath) {
    if (!root.startsWith("/")) throw new Error("project_root_key path must be absolute");
    if (!/^[A-Za-z0-9._~:\/-]{1,512}$/.test(root)) throw new Error("project_root_key path is not allowed");
  } else {
    if (!/^[A-Za-z0-9][A-Za-z0-9._:-]{0,159}$/.test(root)) throw new Error("project_root_key id must be opaque");
  }
  if (!/^[A-Za-z0-9._~:-]{1,256}$/.test(continuityId)) throw new Error("continuity_id is not opaque");
  return `${root}::${continuityId}`;
}

export function parseWorkstreamKey(value: string): { projectRootKey: string; continuityId: string } {
  const idx = value.indexOf("::");
  if (idx < 1) throw new Error("workstream_key must be project_root_key::continuity_id");
  return { projectRootKey: value.slice(0, idx), continuityId: value.slice(idx + 2) };
}

export function deriveWorkstreamKeyFromScope(scope: { project_root_key?: string; workstream_key?: string; continuity_id?: string }): string | undefined {
  if (scope.workstream_key) return scope.workstream_key;
  if (scope.project_root_key && scope.continuity_id) return workstreamKey(scope.project_root_key, scope.continuity_id);
  return undefined;
}

export function assertWorkstreamKeyMatchesScope(scope: { project_root_key?: string; workstream_key?: string; continuity_id?: string }): void {
  if (scope.project_root_key && scope.continuity_id && scope.workstream_key) {
    const derived = workstreamKey(scope.project_root_key, scope.continuity_id);
    if (scope.workstream_key !== derived) throw new Error(`workstream_key mismatch: expected ${derived}`);
  }
  if (scope.workstream_key) parseWorkstreamKey(scope.workstream_key);
}
