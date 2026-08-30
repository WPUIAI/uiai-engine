export type EvidenceSharePacket = {
  packet_id: string;
  descriptor: string;
  artifact_ref: string;
  artifact_path: string;
  artifact_url: string;
  thumbnail_url: string;
  captured_at: string;
  source_url?: string;
  availability: string;
  workpoint_ref?: string;
  continuity_ref?: string;
};
export type EvidenceShareList = { packets: EvidenceSharePacket[]; count: number };
export type EvidenceShareManifest = {
  schema: string; artifact_ref: string; artifact_sha256: string; screenshot_ref: string;
  screenshot_sha256: string; format: string; mime: string; bytes: number; width: number; height: number;
  source_url?: string; captured_at: string; duration_ms: number; availability: string; access: string;
  interaction: string; scope?: { workpoint_ref?: string; continuity_ref?: string }; truth_notice: string;
};
export type EvidenceShareVerification = { packet_id: string; descriptor: string; valid: boolean; issues: string[] };

export function sourceHost(value?: string): string {
  if (!value) return "Source not disclosed";
  try { return new URL(value).hostname || "Source not disclosed"; } catch { return "Source unavailable"; }
}
export function packetMatchesWorkpoint(packet: EvidenceShareManifest | undefined, workpoint?: string): boolean {
  return !workpoint || packet?.scope?.workpoint_ref === workpoint;
}
export function humanBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return "Unknown size";
  return new Intl.NumberFormat(undefined, { style: "unit", unit: "byte", notation: bytes >= 1_000_000 ? "compact" : "standard" }).format(bytes);
}
