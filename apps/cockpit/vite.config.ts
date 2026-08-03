import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";

/** Vite config — Tauri dev server.
 *
 *  Port 1420 matches `tauri.conf.json` `devUrl`.
 */
const allowedHosts = (process.env.VITE_ALLOWED_HOSTS || "").split(",").map((host) => host.trim()).filter(Boolean);

export default defineConfig({
  plugins: [sveltekit()],
  server: { port: 1420, strictPort: true, host: true, allowedHosts },
  build: { target: "es2022", sourcemap: true },
  clearScreen: false,
});
