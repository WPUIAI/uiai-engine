import { requireCapabilityEntitlement } from "$lib/contracts/entitlement";
import { engineRequest, engineUrl } from "$lib/engine-client";

export interface FpvLiveAttachment {
  token: string;
  session_id: string;
  controls: boolean;
  expires_at: string;
  viewer_url: string;
  stream_url: string;
  fallback_url: string;
}

interface FpvShareResponse {
  token: string;
  session_id: string;
  controls: boolean;
  mirror_url_expires_at: string;
}

export function fpvAttachmentUrls(share: FpvShareResponse): FpvLiveAttachment {
  const base = engineUrl().replace(/\/$/, "");
  const token = encodeURIComponent(share.token);
  return {
    token: share.token,
    session_id: share.session_id,
    controls: share.controls,
    expires_at: share.mirror_url_expires_at,
    viewer_url: `${base}/m/${token}`,
    stream_url: `${base}/m/${token}/stream.cdp.mjpg`,
    fallback_url: `${base}/m/${token}/screenshot.jpg`,
  };
}

export async function attachFpvSession(sessionId: string, controls = false, expiresMinutes = 60): Promise<FpvLiveAttachment> {
  requireCapabilityEntitlement(controls ? "uiai.browser.session.control" : "uiai.browser.screenshot.execute");
  const share = await engineRequest<FpvShareResponse>("/api/fpv/share", {
    method: "POST",
    body: JSON.stringify({ session_id: sessionId, controls, expires_minutes: expiresMinutes, one_time: false }),
  });
  if (share.session_id !== sessionId || !share.token) throw new Error("Engine returned a mismatched FPV attachment.");
  return fpvAttachmentUrls(share);
}

export async function revokeFpvSession(attachment: FpvLiveAttachment | null): Promise<void> {
  if (!attachment) return;
  await engineRequest<{ ok: boolean }>("/api/fpv/revoke", { method: "POST", body: JSON.stringify({ token: attachment.token }) });
}
