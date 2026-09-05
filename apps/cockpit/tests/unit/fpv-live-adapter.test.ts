import { afterEach, describe, expect, it, vi } from "vitest";
import { fpvAttachmentUrls } from "../../src/lib/adapters/fpv-live-adapter";

afterEach(() => vi.unstubAllGlobals());

function fpvResponse(token: string, controls: boolean) {
  const artifactUrl = "https://evidence.example/share/package/";
  const portableUrl = `${artifactUrl}portable.zip`;
  return {
    schema: "uiai.fpv_share_result.v2",
    artifact_ref: "artifact:fpv",
    delivery_state: "ready",
    artifact_url: artifactUrl,
    portable_url: portableUrl,
    token,
    session_id: "session_01",
    expires_at: "2026-08-03T01:00:00Z",
    operational_mirror: { token, session_id: "session_01", controls, mirror_url_expires_at: "2026-08-03T01:00:00Z" },
    epwa_delivery: {
      schema: "uiai.epwa_delivery.v1" as const,
      delivery_id: "uiai-epwa-delivery:sha256:" + "d".repeat(64),
      revision: 1,
      artifact: { artifact_ref: "artifact:fpv", revision: 1, manifest_sha256: "a".repeat(64), output_sha256: "b".repeat(64) },
      epwa: { package_id: "c".repeat(64), package_ref: "uiai-epwa-package:sha256:" + "c".repeat(64), package_sha256: "c".repeat(64), record_url: artifactUrl, portable_url: portableUrl, access: "public_safe" },
      scope: { posture: "complete" }, state: "ready",
    },
  };
}

describe("Cockpit FPV live adapter", () => {
  it("binds an operational mirror to its canonical EPWA evidence", () => {
    vi.stubGlobal("window", { localStorage: { getItem: () => "http://127.0.0.1:7456" } });
    const attachment = fpvAttachmentUrls(fpvResponse("opaque-token", false));
    expect(attachment.viewer_url).toBe("http://127.0.0.1:7456/m/opaque-token");
    expect(attachment.stream_url).toBe("http://127.0.0.1:7456/m/opaque-token/stream.cdp.mjpg");
    expect(attachment.fallback_url).toBe("http://127.0.0.1:7456/m/opaque-token/screenshot.jpg");
    expect(attachment.evidence_url).toBe("https://evidence.example/share/package/");
    expect(attachment.portable_url).toBe("https://evidence.example/share/package/portable.zip");
    expect(attachment.delivery_id).toContain("uiai-epwa-delivery:sha256:");
    expect(attachment.session_id).toBe("session_01");
    expect(attachment.controls).toBe(false);
  });

  it("encodes share tokens instead of treating them as paths", () => {
    vi.stubGlobal("window", { localStorage: { getItem: () => "http://127.0.0.1:7456/" } });
    const attachment = fpvAttachmentUrls(fpvResponse("opaque/token", true));
    expect(attachment.viewer_url.endsWith("/m/opaque%2Ftoken")).toBe(true);
  });
});
