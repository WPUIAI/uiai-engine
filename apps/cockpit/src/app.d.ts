// See https://kit.svelte.dev/docs/types#app
declare global {
  namespace App {
    // interface Error {}
    // interface Locals {}
    // interface PageData {}
    // interface Platform {}
  }

  interface Window {
    __UIAI_COCKPIT_CONTRACTS__?: {
      workpoint_resume?: unknown;
      entitlement?: unknown;
    };
  }
}
export {};