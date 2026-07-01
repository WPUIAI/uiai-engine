import adapter from "@sveltejs/adapter-static";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

/** SvelteKit config for Focusa Cockpit.
 *
 *  Static-adapter SPA: Tauri does not support server-based frontend
 *  solutions inside the app. Build output is `build/` which Tauri
 *  bundles as `frontendDist`.
 */
export default {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter({
      pages: "build",
      assets: "build",
      fallback: "index.html",
      precompress: false,
      strict: true,
    }),
    prerender: { entries: [] },
  },
};
