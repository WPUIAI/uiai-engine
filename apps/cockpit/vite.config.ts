import adapter from "@sveltejs/adapter-static";
import { sveltekit } from "@sveltejs/kit/vite";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";

/** Vite config — Tauri dev server.
 *
 *  Port 1420 matches `tauri.conf.json` `devUrl`.
 */
export default defineConfig({
  plugins: [
    sveltekit({
      preprocess: vitePreprocess(),
      alias: { $lib: "./src/lib" },
      adapter: adapter({
        pages: "build",
        assets: "build",
        fallback: "index.html",
        precompress: false,
        strict: true,
      }),
      prerender: { entries: [] },
    }),
  ],
  // Keep mutable caches in this worktree, not a shared dependency directory.
  cacheDir: ".svelte-kit/vite-cache",
  server: { port: 1420, strictPort: true },
  build: { target: "es2022", sourcemap: true },
  clearScreen: false,
});
