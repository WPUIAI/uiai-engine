export type EvidenceSharePacket = {
  packet_id: string;
  descriptor: string;
  artifact_ref: string;
  artifact_url: string;
  portable_url?: string;
  captured_at: string;
  source_url?: string;
  availability: string;
  workpoint_ref?: string;
  continuity_ref?: string;
};
export type EvidenceShareList = { packets: EvidenceSharePacket[]; count: number };
export type EvidenceShareManifest = {
  schema: string; artifact_ref: string; artifact_sha256: string; digest_label?: string;
  format: string; bytes?: number; width?: number; height?: number;
  captured_at?: string; availability?: string; access?: string; kind?: string;
  scope?: { workpoint_ref?: string; continuity_ref?: string }; truth_notice: string;
};
export type EvidenceShareVerification = { packet_id: string; descriptor: string; valid: boolean; issues: string[] };
export type EvidenceShareSettings = { schema: string; scope: { project_ref?: string; workstream_ref?: string }; revision: number; sources: string[]; values: Record<string, any>; warnings?: string[] };

const object = (value: unknown): Record<string, unknown> => value !== null && typeof value === "object" && !Array.isArray(value) ? value as Record<string, unknown> : {};
const string = (value: unknown): string => typeof value === "string" ? value : "";
const number = (value: unknown): number | undefined => typeof value === "number" && Number.isFinite(value) && value >= 0 ? value : undefined;

// The package endpoint serves three existing contracts, not only screenshots.
export function normalizeEvidenceShareManifest(value: unknown): EvidenceShareManifest {
  const manifest = object(value);
  const schema = string(manifest.schema);
  if (!["uiai.screenshot_evidence_share.v1", "uiai.evidence_artifact_manifest.v1", "uiai.epwa_generic_artifact.v1"].includes(schema)) {
    throw new Error("Unsupported evidence package contract");
  }
  const scope = object(manifest.scope);
  const workpoint = object(scope.workpoint);
  const integrity = object(manifest.integrity);
  const policy = object(manifest.policy);
  return {
    schema,
    artifact_ref: string(manifest.artifact_ref) || string(manifest.artifact_id),
    artifact_sha256: string(manifest.artifact_sha256) || string(manifest.asset_sha256) || string(integrity.manifest_sha256),
    digest_label: string(manifest.artifact_sha256) ? "Artifact SHA-256" : string(manifest.asset_sha256) ? "Payload SHA-256" : "Manifest SHA-256",
    format: string(manifest.format) || string(manifest.media_type) || "Evidence package",
    bytes: number(manifest.bytes), width: number(manifest.width), height: number(manifest.height),
    captured_at: string(manifest.captured_at), availability: string(manifest.availability), kind: string(manifest.kind),
    access: string(manifest.access) || string(policy.access_class),
    scope: { workpoint_ref: string(scope.workpoint_ref) || string(workpoint.workpoint_ref), continuity_ref: string(scope.continuity_ref) },
    truth_notice: string(manifest.truth_notice) || "Delivery does not establish review, task completion, settlement, or legal admissibility.",
  };
}

export function secureEvidenceURL(value?: string): string | undefined {
  try {
    const url = new URL(value || "");
    return url.protocol === "https:" && !url.username && !url.password ? url.href : undefined;
  } catch { return undefined; }
}
export function sourceHost(value?: string): string {
  if (!value) return "Source not disclosed";
  try { return new URL(value).hostname || "Source not disclosed"; } catch { return "Source unavailable"; }
}
export function packetMatchesWorkpoint(packet: EvidenceShareManifest | undefined, workpoint?: string): boolean {
  return !workpoint || packet?.scope?.workpoint_ref === workpoint;
}
export function humanBytes(bytes?: number): string {
  if (typeof bytes !== "number" || !Number.isFinite(bytes) || bytes < 0) return "Unknown size";
  return new Intl.NumberFormat(undefined, { style: "unit", unit: "byte", notation: bytes >= 1_000_000 ? "compact" : "standard" }).format(bytes);
}
