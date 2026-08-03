import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

const updater = readFileSync(new URL("../../src/lib/updater.ts", import.meta.url), "utf8");
const host = readFileSync(new URL("../../src/lib/ui/ToastHost.svelte", import.meta.url), "utf8");
const config = JSON.parse(readFileSync(new URL("../../src-tauri/tauri.conf.json", import.meta.url), "utf8"));
const releaseScript = readFileSync(new URL("../../scripts/cockpit-release.sh", import.meta.url), "utf8");

describe("Cockpit dev OTA notifications", () => {
  it("polls the signed dev channel without overlapping update operations", () => {
    expect(updater).toContain("window.setInterval(run, 15_000)");
    expect(updater).toContain("if (inFlight) return");
    expect(config.plugins.updater.endpoints[0]).toContain("cockpit-latest/latest.json");
    expect(config.plugins.updater.pubkey).toMatch(/^dW50cnVzdGVk/);
    expect(releaseScript).toContain('UPDATER_NAME="uaiengine-cockpit_${VERSION}_${UPDATER_ARCH}.app.tar.gz"');
    expect(releaseScript).toContain('URL="${UPDATER_ASSET_BASE_URL%/}/$UPDATER_NAME"');
  });

  it("updates one persistent toast through background download and install", () => {
    expect(updater.match(/id: 'cockpit-ota'/g)?.length).toBeGreaterThanOrEqual(4);
    expect(updater).toContain("progress: 1");
    expect(host).toContain("aria-live=\"polite\"");
    expect(host).toContain("existing >= 0");
    expect(host).toContain("<progress");
  });

  it("keeps a reusable global toast event for non-OTA updates", () => {
    const api = readFileSync(new URL("../../src/lib/ui/toast.ts", import.meta.url), "utf8");
    expect(api).toContain('COCKPIT_TOAST_EVENT = "uiai-cockpit-toast"');
    expect(api).toContain("export function pushToast");
  });
});
