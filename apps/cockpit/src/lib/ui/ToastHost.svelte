<script lang="ts">
  import { onMount } from "svelte";
  import { COCKPIT_TOAST_EVENT, type CockpitToast } from "./toast";

  type VisibleToast = Required<Pick<CockpitToast, "id" | "title" | "level">> & CockpitToast;
  let toasts: VisibleToast[] = [];
  const timers = new Map<string, number>();

  function dismiss(id: string) {
    const timer = timers.get(id);
    if (timer) window.clearTimeout(timer);
    timers.delete(id);
    toasts = toasts.filter((toast) => toast.id !== id);
  }

  function show(input: CockpitToast) {
    const id = input.id || crypto.randomUUID();
    const toast: VisibleToast = { ...input, id, level: input.level || "info", title: input.title };
    const existing = toasts.findIndex((item) => item.id === id);
    toasts = existing >= 0
      ? toasts.map((item) => item.id === id ? toast : item)
      : [...toasts.slice(-3), toast];
    const prior = timers.get(id);
    if (prior) window.clearTimeout(prior);
    const duration = input.durationMs ?? (toast.level === "error" ? 9000 : 5000);
    if (duration > 0) timers.set(id, window.setTimeout(() => dismiss(id), duration));
  }

  onMount(() => {
    const listener = (event: Event) => show((event as CustomEvent<CockpitToast>).detail);
    window.addEventListener(COCKPIT_TOAST_EVENT, listener);
    return () => {
      window.removeEventListener(COCKPIT_TOAST_EVENT, listener);
      for (const timer of timers.values()) window.clearTimeout(timer);
    };
  });
</script>

<div class="toast-region" aria-live="polite" aria-label="Notifications">
  {#each toasts as toast (toast.id)}
    <article class="toast" data-level={toast.level} role={toast.level === "error" ? "alert" : "status"}>
      <span class="toast-mark" aria-hidden="true"></span>
      <div class="toast-copy">
        <strong>{toast.title}</strong>
        {#if toast.message}<span>{toast.message}</span>{/if}
        {#if toast.progress !== undefined}
          <progress max="1" value={Math.max(0, Math.min(1, toast.progress))} aria-label={toast.title}></progress>
        {/if}
      </div>
      <button type="button" aria-label={`Dismiss ${toast.title}`} onclick={() => dismiss(toast.id)}>×</button>
    </article>
  {/each}
</div>

<style>
  .toast-region { position: fixed; z-index: 1000; inset: 68px 18px auto auto; display: grid; gap: 10px; width: min(380px, calc(100vw - 36px)); pointer-events: none; }
  .toast { pointer-events: auto; display: grid; grid-template-columns: 9px 1fr auto; gap: 11px; align-items: start; padding: 13px 14px; border: 1px solid var(--color-border); border-radius: 12px; background: color-mix(in srgb, var(--color-surface-elevated) 94%, transparent); color: var(--color-text); box-shadow: 0 12px 32px rgb(0 0 0 / 24%); backdrop-filter: blur(18px); }
  .toast-mark { width: 8px; height: 8px; margin-top: 5px; border-radius: 50%; background: var(--color-accent); }
  .toast[data-level="success"] .toast-mark { background: var(--color-success); }
  .toast[data-level="warning"] .toast-mark { background: var(--color-warn); }
  .toast[data-level="error"] .toast-mark { background: var(--color-error); }
  .toast-copy { min-width: 0; display: grid; gap: 4px; }
  .toast-copy strong { font-size: .82rem; }
  .toast-copy span { color: var(--color-text-muted); font-size: .76rem; line-height: 1.35; overflow-wrap: anywhere; }
  progress { width: 100%; height: 5px; accent-color: var(--color-accent); }
  button { border: 0; background: transparent; color: var(--color-text-muted); font: inherit; font-size: 1.1rem; cursor: pointer; }
  button:focus-visible { outline: 3px solid var(--color-focus-ring); outline-offset: 2px; }
  @media (max-width: 700px) { .toast-region { inset: 60px 10px auto 10px; width: auto; } }
  @media (prefers-reduced-motion: no-preference) { .toast { animation: toast-in 160ms ease-out; } @keyframes toast-in { from { opacity: 0; transform: translateY(-7px); } } }
</style>
