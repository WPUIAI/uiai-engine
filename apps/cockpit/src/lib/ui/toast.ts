export type ToastLevel = "info" | "success" | "warning" | "error";

export interface CockpitToast {
  id?: string;
  title: string;
  message?: string;
  level?: ToastLevel;
  progress?: number;
  durationMs?: number;
}

export const COCKPIT_TOAST_EVENT = "uiai-cockpit-toast";

/** Publish a global, non-blocking Cockpit notification. Reusing an id updates the existing toast. */
export function pushToast(toast: CockpitToast): void {
  if (typeof window === "undefined") return;
  window.dispatchEvent(new CustomEvent<CockpitToast>(COCKPIT_TOAST_EVENT, { detail: toast }));
}
