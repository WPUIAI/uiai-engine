import { afterEach, describe, expect, it, vi } from "vitest";
import { fpvAttachmentUrls } from "../../src/lib/adapters/fpv-live-adapter";

afterEach(() => vi.unstubAllGlobals());

describe("Cockpit FPV live adapter", () => {
  it("binds the existing opaque session share to local Engine viewer and fallback URLs", () => {
    vi.stubGlobal("window", { localStorage: { getItem: () => "http://127.0.0.1:7456" } });
    const attachment = fpvAttachmentUrls({
      token: "opaque-token",
      session_id: "session_01",
      controls: false,
      mirror_url_expires_at: "2026-08-03T01:00:00Z",
    });
    expect(attachment.viewer_url).toBe("http://127.0.0.1:7456/m/opaque-token");
    expect(attachment.stream_url).toBe("http://127.0.0.1:7456/m/opaque-token/stream.cdp.mjpg");
    expect(attachment.fallback_url).toBe("http://127.0.0.1:7456/m/opaque-token/screenshot.jpg");
    expect(attachment.session_id).toBe("session_01");
    expect(attachment.controls).toBe(false);
  });

  it("encodes share tokens instead of treating them as paths", () => {
    vi.stubGlobal("window", { localStorage: { getItem: () => "http://127.0.0.1:7456/" } });
    const attachment = fpvAttachmentUrls({ token: "opaque/token", session_id: "session_01", controls: true, mirror_url_expires_at: "later" });
    expect(attachment.viewer_url.endsWith("/m/opaque%2Ftoken")).toBe(true);
  });
});
