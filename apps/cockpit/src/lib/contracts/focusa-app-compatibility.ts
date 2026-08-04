import { parseFocusaAppManifest, type FocusaAppManifestV2 } from "./desktop-presentation";

export const FOCUSA_PAIRING_PROTOCOLS = {
  pairing: "1",
  bridge: "1",
  scope_context: "1",
} as const;

export const FOCUSA_PAIRING_CAPABILITIES = [
  "pair.start",
  "pair.status",
  "pair.inherited",
  "scope.enumerate",
] as const;

export type FocusaPairingProtocol = keyof typeof FOCUSA_PAIRING_PROTOCOLS;
export type FocusaPairingCapability = (typeof FOCUSA_PAIRING_CAPABILITIES)[number];
export type PairingCompatibilityReason =
  | "compatible"
  | "wrong_sibling_app"
  | "missing_protocol"
  | "unsupported_protocol_version"
  | "missing_capability";

export interface PairingCompatibilityResultV1 {
  schema: "focusa.pairing_compatibility_result.v1";
  compatible: boolean;
  reason: PairingCompatibilityReason;
  protocol?: FocusaPairingProtocol;
  expected_version?: string;
  observed_version?: string;
  capability?: FocusaPairingCapability;
}

const result = (
  compatible: boolean,
  reason: PairingCompatibilityReason,
  detail: Partial<PairingCompatibilityResultV1> = {},
): PairingCompatibilityResultV1 => ({
  schema: "focusa.pairing_compatibility_result.v1",
  compatible,
  reason,
  ...detail,
});

/**
 * Parses the shared read-only manifest and applies the UIAI-COCKPIT-005
 * pairing compatibility policy. This function grants no authentication or
 * scope authority; it only decides whether Path B may be offered.
 */
export function negotiateFocusaPairingCompatibility(value: unknown): {
  manifest: FocusaAppManifestV2;
  compatibility: PairingCompatibilityResultV1;
} {
  const manifest = parseFocusaAppManifest(value);
  if (manifest.app !== "focusa-menubar") {
    return { manifest, compatibility: result(false, "wrong_sibling_app") };
  }

  for (const [protocol, expected] of Object.entries(FOCUSA_PAIRING_PROTOCOLS) as [FocusaPairingProtocol, string][]) {
    const observed = manifest.protocols[protocol];
    if (observed === undefined) {
      return { manifest, compatibility: result(false, "missing_protocol", { protocol, expected_version: expected }) };
    }
    if (observed !== expected) {
      return {
        manifest,
        compatibility: result(false, "unsupported_protocol_version", {
          protocol,
          expected_version: expected,
          observed_version: observed,
        }),
      };
    }
  }

  for (const capability of FOCUSA_PAIRING_CAPABILITIES) {
    if (!manifest.capabilities.includes(capability)) {
      return { manifest, compatibility: result(false, "missing_capability", { capability }) };
    }
  }

  return { manifest, compatibility: result(true, "compatible") };
}
