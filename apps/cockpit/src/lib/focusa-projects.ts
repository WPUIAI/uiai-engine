import type { FocusaDaemonConnection } from "$lib/focusa-daemon-discovery";

const PROJECT_BINDINGS_KEY = "uiai.cockpit.focusa_project_bindings.v1";

export interface FocusaProjectBinding {
  bindingId: string;
  daemonKey: string;
  daemonBaseUrl: string;
  daemonLocation: "local" | "remote";
  projectRoot: string;
  projectId?: string;
  canonicalName: string;
  continuityId?: string;
  status: "verified" | "unverified" | "conflict";
  verifiedAt?: string;
}

function daemonKey(daemon: FocusaDaemonConnection): string {
  return daemon.nodeId || daemon.machineId || daemon.baseUrl;
}

function normalizeRoot(value: string): string {
  const root = value.trim().replace(/\/+$/, "");
  return root.startsWith("/") ? root : "";
}

export function readFocusaProjectBindings(): FocusaProjectBinding[] {
  if (typeof window === "undefined") return [];
  try {
    const parsed = JSON.parse(window.localStorage.getItem(PROJECT_BINDINGS_KEY) || "[]");
    return Array.isArray(parsed) ? parsed.filter((item) => item && typeof item === "object" && typeof item.projectRoot === "string" && typeof item.daemonBaseUrl === "string").slice(0, 30) : [];
  } catch {
    return [];
  }
}

function saveFocusaProjectBindings(bindings: FocusaProjectBinding[]): FocusaProjectBinding[] {
  const bounded = bindings.slice(0, 30);
  if (typeof window !== "undefined") window.localStorage.setItem(PROJECT_BINDINGS_KEY, JSON.stringify(bounded));
  return bounded;
}

export async function verifyFocusaProject(daemon: FocusaDaemonConnection, projectRootValue: string, continuityIdValue = ""): Promise<FocusaProjectBinding> {
  const projectRoot = normalizeRoot(projectRootValue);
  if (!projectRoot) throw new Error("Enter an absolute project folder.");
  if (daemon.status !== "connected") throw new Error("The selected Focusa node is not connected.");
  const url = new URL(`${daemon.baseUrl}/v1/project/identity`);
  url.searchParams.set("cwd", projectRoot);
  url.searchParams.set("project_root", projectRoot);
  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), 10_000);
  try {
    const response = await fetch(url, { signal: controller.signal, cache: "no-store" });
    const body = await response.json().catch(() => ({})) as Record<string, any>;
    if (!response.ok) throw new Error(typeof body.error === "string" ? body.error : `Focusa project verification failed (${response.status}).`);
    const identity = body.project_identity && typeof body.project_identity === "object" ? body.project_identity : body;
    const verifiedRoot = normalizeRoot(String(identity.project_root || identity.root_path || projectRoot));
    const verified = identity.verified === true || identity.status === "verified" || body.status === "verified";
    const conflict = identity.status === "conflict" || body.status === "conflict" || (verifiedRoot && verifiedRoot !== projectRoot);
    const canonicalName = String(identity.canonical_name || identity.project_name || projectRoot.split("/").pop() || "Project");
    return {
      bindingId: `${daemonKey(daemon)}::${verifiedRoot || projectRoot}`,
      daemonKey: daemonKey(daemon),
      daemonBaseUrl: daemon.baseUrl,
      daemonLocation: daemon.location,
      projectRoot: verifiedRoot || projectRoot,
      projectId: typeof identity.project_id === "string" ? identity.project_id : undefined,
      canonicalName,
      continuityId: continuityIdValue.trim() || undefined,
      status: conflict ? "conflict" : verified ? "verified" : "unverified",
      verifiedAt: new Date().toISOString(),
    };
  } catch (error) {
    if (error instanceof DOMException && error.name === "AbortError") throw new Error("Focusa project verification timed out.");
    throw error;
  } finally {
    window.clearTimeout(timeout);
  }
}

export function rememberFocusaProjectBinding(binding: FocusaProjectBinding): FocusaProjectBinding[] {
  const rest = readFocusaProjectBindings().filter((item) => item.bindingId !== binding.bindingId);
  return saveFocusaProjectBindings([binding, ...rest]);
}

export function selectFocusaProjectBinding(binding: FocusaProjectBinding): void {
  if (binding.status !== "verified") throw new Error("Focusa project binding must be verified before scope selection.");
  if (typeof window === "undefined") return;
  window.localStorage.setItem("uiai.scope.project_root", binding.projectRoot);
  if (binding.continuityId) window.localStorage.setItem("uiai.scope.continuity_id", binding.continuityId);
  else window.localStorage.removeItem("uiai.scope.continuity_id");
  // Workstream-rooted: persist derived WorkstreamKey = ProjectRoot::ContinuityId (one partition per workstream).
  if (binding.projectRoot && binding.continuityId) window.localStorage.setItem("uiai.scope.workstream_key", `${binding.projectRoot.replace(/\/+$/, "")}::${binding.continuityId}`);
  else window.localStorage.removeItem("uiai.scope.workstream_key");
  window.localStorage.removeItem("uiai.scope.workpoint_id");
  window.localStorage.setItem("uiai.scope.focusa_daemon_key", binding.daemonKey);
}

export function projectBindingRequiresReconciliation(binding: FocusaProjectBinding, bindings: FocusaProjectBinding[]): boolean {
  return bindings.some((item) => item.bindingId !== binding.bindingId && item.projectRoot === binding.projectRoot && item.daemonKey !== binding.daemonKey);
}
