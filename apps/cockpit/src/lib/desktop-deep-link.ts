import { pushToast } from "$lib/ui/toast";

export type CockpitDeepLinkRoute = "live_session" | "focusa" | "evidence" | "settings_pairing";

export interface CockpitDeepLinkIntent {
  schema: "uiai.cockpit.deep_link.v1";
  route: CockpitDeepLinkRoute;
  target_ref: string | null;
  handoff_ref: string | null;
}

type Navigate = (path: string) => void;
type Unlisten = () => void;

const allowedKeys = new Set(["schema", "route", "target_ref", "handoff_ref"]);
const opaqueRef = /^[A-Za-z0-9._~:-]{1,256}$/;

export function parseCockpitDeepLinkIntent(value: unknown): CockpitDeepLinkIntent {
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error("Deep-link intent must be an object.");
  const item = value as Record<string, unknown>;
  if (Object.keys(item).some((key) => !allowedKeys.has(key))) throw new Error("Deep-link intent contains unknown fields.");
  if (item.schema !== "uiai.cockpit.deep_link.v1") throw new Error("Unsupported deep-link intent schema.");
  if (!(["live_session", "focusa", "evidence", "settings_pairing"] as unknown[]).includes(item.route)) throw new Error("Unsupported deep-link route.");
  const targetRef = item.target_ref === null ? null : typeof item.target_ref === "string" && opaqueRef.test(item.target_ref) ? item.target_ref : null;
  const handoffRef = item.handoff_ref === null ? null : typeof item.handoff_ref === "string" && opaqueRef.test(item.handoff_ref) ? item.handoff_ref : null;
  if (item.target_ref !== null && targetRef === null) throw new Error("Invalid deep-link target reference.");
  if (item.handoff_ref !== null && handoffRef === null) throw new Error("Invalid deep-link handoff reference.");
  const route = item.route as CockpitDeepLinkRoute;
  if (route === "settings_pairing" && (targetRef !== null || handoffRef !== null)) throw new Error("Pairing route does not accept references.");
  if (route !== "live_session" && handoffRef !== null) throw new Error("Only live-session routes accept handoff references.");
  if (route !== "settings_pairing" && targetRef === null) throw new Error("Deep-link target reference is required.");
  return { schema: "uiai.cockpit.deep_link.v1", route, target_ref: targetRef, handoff_ref: handoffRef };
}

export function applyCockpitDeepLink(value: unknown, navigate: Navigate): CockpitDeepLinkIntent {
  const intent = parseCockpitDeepLinkIntent(value);
  if (intent.route === "live_session") {
    window.localStorage.setItem("uiai.cockpit.requested_session_id", intent.target_ref!);
    if (intent.handoff_ref) window.localStorage.setItem("uiai.cockpit.handoff_ref", intent.handoff_ref);
    else window.localStorage.removeItem("uiai.cockpit.handoff_ref");
    navigate(`/live?session=${encodeURIComponent(intent.target_ref!)}`);
  } else if (intent.route === "evidence") {
    window.localStorage.setItem("uiai.cockpit.requested_evidence_ref", intent.target_ref!);
    navigate(`/evidence?ref=${encodeURIComponent(intent.target_ref!)}`);
  } else if (intent.route === "focusa") {
    window.localStorage.setItem("uiai.cockpit.requested_focusa_ref", intent.target_ref!);
    navigate(`/?focusa=${encodeURIComponent(intent.target_ref!)}`);
  } else {
    navigate("/settings?panel=pairing");
  }
  return intent;
}

function isTauriHost(): boolean {
  return typeof window !== "undefined" && ("__TAURI_INTERNALS__" in window || "__TAURI__" in window);
}

export async function installCockpitDeepLinkRouting(navigate: Navigate): Promise<Unlisten> {
  if (!isTauriHost()) return () => undefined;
  const [{ invoke }, { listen }] = await Promise.all([import("@tauri-apps/api/core"), import("@tauri-apps/api/event")]);
  const route = (value: unknown) => {
    try {
      applyCockpitDeepLink(value, navigate);
    } catch (error) {
      pushToast({ id: "deep-link-rejected", title: "Cockpit link rejected", message: error instanceof Error ? error.message : "The activation was not accepted.", level: "warning" });
    }
  };
  const unlistenOpen = await listen<unknown>("cockpit-deep-link", (event) => route(event.payload));
  const unlistenRejected = await listen<string>("cockpit-deep-link-rejected", (event) => pushToast({ id: "deep-link-rejected", title: "Cockpit link rejected", message: event.payload, level: "warning" }));
  const pending = await invoke<unknown>("cockpit_take_deep_link");
  if (pending !== null && pending !== undefined) route(pending);
  return () => { unlistenOpen(); unlistenRejected(); };
}
