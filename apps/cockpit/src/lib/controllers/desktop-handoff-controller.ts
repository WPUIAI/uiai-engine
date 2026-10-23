import { parseAppHandoffReceipt, type AppHandoffIntentV1, type AppHandoffReceiptV1 } from "$lib/contracts/desktop-presentation";
import { APP_CHANNEL, APP_VERSION } from "$lib/version";

export type FocusaHandoffRoute = "mission" | "card" | "workpoint" | "connect";

function isTauriHost(): boolean {
  return typeof window !== "undefined" && ("__TAURI_INTERNALS__" in window || "__TAURI__" in window);
}

export function createFocusaHandoff(route: FocusaHandoffRoute, targetRef: string, now = new Date()): AppHandoffIntentV1 {
  const expires = new Date(now.getTime() + 2 * 60_000);
  return {
    schema: "uiai.app_handoff_intent.v1",
    intent_id: `handoff_${crypto.randomUUID().replaceAll("-", "")}`,
    scheme: "focusa",
    route,
    target_ref: targetRef,
    requested_by: { client_type: "cockpit", client_id: `uaiengine-cockpit:${APP_VERSION}:${APP_CHANNEL}` },
    protocol_version: "1",
    created_at: now.toISOString(),
    expires_at: expires.toISOString(),
  };
}

export async function openInFocusa(route: FocusaHandoffRoute, targetRef: string): Promise<AppHandoffReceiptV1> {
  const intent = createFocusaHandoff(route, targetRef);
  if (!isTauriHost()) {
    return {
      schema: "uiai.app_handoff_receipt.v1",
      intent_id: intent.intent_id,
      status: "unavailable",
      target_app: "focusa-menubar",
      reason_code: "native_shell_required",
      observed_at: new Date().toISOString(),
    };
  }
  const { invoke } = await import("@tauri-apps/api/core");
  return parseAppHandoffReceipt(await invoke("cockpit_open_focusa_handoff", { intent }));
}
