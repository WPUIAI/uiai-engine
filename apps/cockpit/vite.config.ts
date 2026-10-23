import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";

/** Vite config — Tauri dev server.
 *
 *  Port 1420 matches `tauri.conf.json` `devUrl`.
 */
export default defineConfig({
  plugins: [sveltekit()],
  // Keep mutable caches in this worktree, not a shared dependency directory.
  cacheDir: ".svelte-kit/vite-cache",
  server: { port: 1420, strictPort: true },
  build: { target: "es2022", sourcemap: true },
  clearScreen: false,
});
