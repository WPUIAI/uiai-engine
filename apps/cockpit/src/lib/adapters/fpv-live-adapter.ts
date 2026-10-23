import { requireCapabilityEntitlement } from "$lib/contracts/entitlement";
import { artifactRequest, engineRequest, engineUrl, type ArtifactDeliveryResult, type ReadyArtifactDeliveryResult } from "$lib/engine-client";

export interface FpvLiveAttachment {
  token: string;
  session_id: string;
  controls: boolean;
  expires_at: string;
  viewer_url: string;
  stream_url: string;
  fallback_url: string;
  evidence_url: string;
  portable_url: string;
  delivery_id: string;
}

interface FpvShareResponse extends ArtifactDeliveryResult {
  token: string;
  session_id: string;
  expires_at: string;
  operational_mirror: {
    token: string;
    session_id: string;
    controls: boolean;
    mirror_url_expires_at: string;
  };
}

export function fpvAttachmentUrls(share: FpvShareResponse & ReadyArtifactDeliveryResult): FpvLiveAttachment {
  const base = engineUrl().replace(/\/$/, "");
  const mirror = share.operational_mirror;
  const token = encodeURIComponent(mirror.token);
  return {
    token: mirror.token,
    session_id: mirror.session_id,
    controls: mirror.controls,
    expires_at: mirror.mirror_url_expires_at || share.expires_at,
    viewer_url: `${base}/m/${token}`,
    stream_url: `${base}/m/${token}/stream.cdp.mjpg`,
    fallback_url: `${base}/m/${token}/screenshot.jpg`,
    evidence_url: share.artifact_url,
    portable_url: share.portable_url,
    delivery_id: share.epwa_delivery.delivery_id,
  };
}

export async function attachFpvSession(sessionId: string, controls = false, expiresMinutes = 60): Promise<FpvLiveAttachment> {
  requireCapabilityEntitlement(controls ? "uiai.browser.session.control" : "uiai.browser.screenshot.execute");
  const share = await artifactRequest<FpvShareResponse>("/api/fpv/share", {
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
