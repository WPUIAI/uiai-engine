const SCOPE_HEADER_ALIASES = {
  "X-UIAI-Project-Ref": ["project_ref", "project", "project_root"],
  "X-UIAI-Workstream-Ref": ["workstream_ref", "workstream"],
  "X-UIAI-Workset-Ref": ["workset_ref", "workset"],
  "X-UIAI-CallGraph-Ref": ["callgraph_ref", "callgraph"],
  "X-UIAI-Workpoint-Ref": ["workpoint_ref", "workpoint", "workpoint_id"],
  "X-UIAI-Work-Item-Ref": ["work_item_ref", "work_item"],
  "X-UIAI-Continuity-Ref": ["continuity_ref", "continuity", "continuity_id"],
};

const RAW_ARTIFACT_FIELDS = new Set([
  "screenshot",
  "imageBase64",
  "image_base64",
  "artifact_path",
  "result_path",
  "result_url",
  "screenshot_path",
  "inline_bytes",
]);

export function evidenceScopeHeaders(scope) {
  if (!scope || typeof scope !== "object") return {};
  const headers = {};
  for (const [header, keys] of Object.entries(SCOPE_HEADER_ALIASES)) {
    const value = keys
      .map((key) => scope[key])
      .find((candidate) => typeof candidate === "string" && candidate.trim());
    if (value) headers[header] = value.trim();
  }
  if (Array.isArray(scope.work_items)) {
    headers["X-UIAI-Work-Items"] = JSON.stringify(scope.work_items);
  }
  return headers;
}

export function findRawArtifactField(value, path = "$") {
  if (!value || typeof value !== "object") return "";
  if (Array.isArray(value)) {
    for (let index = 0; index < value.length; index += 1) {
      const found = findRawArtifactField(value[index], `${path}[${index}]`);
      if (found) return found;
    }
    return "";
  }
  for (const [key, child] of Object.entries(value)) {
    // A capability's JSON Schema property name is not an artifact payload.
    // Accept only the narrow scalar descriptors used by discovery; continue
    // recursively checking their contents and reject actual values/extra keys.
    const schemaProperty = path.endsWith(".properties") &&
      (path.includes(".parameters.") || path.includes(".inputSchema."));
    const descriptor = schemaProperty && child && typeof child === "object" && !Array.isArray(child) &&
      ["string", "number", "integer", "boolean", "object", "array", "null"].includes(child.type) &&
      Object.keys(child).every(field => field === "type" || field === "description") &&
      (child.description === undefined || typeof child.description === "string");
    if (RAW_ARTIFACT_FIELDS.has(key) && child !== undefined && child !== null && child !== "" && !descriptor) {
      return `${path}.${key}`;
    }
    const found = findRawArtifactField(child, `${path}.${key}`);
    if (found) return found;
  }
  return "";
}

export function unsealedAdhocLink(value, path = "$") {
  if (!value || typeof value !== "object") return "";
  if (Array.isArray(value)) {
    for (const item of value) {
      const found = unsealedAdhocLink(item, `${path}[?]`);
      if (found) return found;
    }
    return "";
  }
  // #299: an unscoped capture that succeeded is productive work, not a pending
  // evidence failure. Accept it only in the narrow engine shape: ok:true,
  // unsealed work state, a private loopback image route, an epwa sub-envelope
  // still honestly blocked, and NO scope references attempted anywhere — a
  // scoped request can never downgrade into the ad hoc path.
  if (value.delivery_state === "unsealed" && value.ok === true &&
      typeof value.private_image_url === "string" && value.private_image_url.startsWith("/api/screenshot/adhoc/") &&
      value.epwa_delivery?.state === "blocked" &&
      !value.epwa_delivery?.scope?.project_ref &&
      !value.epwa_delivery?.scope?.workpoint_ref &&
      !value.epwa_delivery?.scope?.continuity_ref) {
    return value.private_image_url;
  }
  for (const child of Object.values(value)) {
    const found = unsealedAdhocLink(child, `${path}.?`);
    if (found) return found;
  }
  return "";
}

export function findNonReadyArtifactDelivery(value, path = "$") {
  if (!value || typeof value !== "object") return "";
  if (Array.isArray(value)) {
    for (let index = 0; index < value.length; index += 1) {
      const found = findNonReadyArtifactDelivery(value[index], `${path}[${index}]`);
      if (found) return found;
    }
    return "";
  }
  if (Object.hasOwn(value, "epwa_delivery_error") || value.schema === "uiai.epwa_delivery_error.v1") return `${path}.epwa_delivery_error`;
  if (value.schema === "uiai.epwa_delivery.v1" && value.state !== "ready") return `${path}.state`;
  if (Object.hasOwn(value, "delivery_state") || Object.hasOwn(value, "epwa_delivery")) {
    const deliveryState = value.delivery_state || value.epwa_delivery?.state || "missing";
    if (deliveryState !== "ready" || value.epwa_delivery?.state !== "ready") return `${path}.delivery_state`;
  }
  for (const [key, child] of Object.entries(value)) {
    const found = findNonReadyArtifactDelivery(child, `${path}.${key}`);
    if (found) return found;
  }
  return "";
}

// Preserve handles for committed effects without exposing raw evidence payloads.
function evidenceRecoveryDetails(value) {
  return Object.fromEntries(Object.entries({
    artifact_ref: value?.artifact_ref || value?.artifact?.artifact_ref || value?.epwa_delivery?.artifact?.artifact_ref,
    job_id: value?.job_id,
    recovery_ref: value?.recovery_ref || value?.epwa_delivery?.recovery_ref,
    session_id: value?.session_id || value?.session?.id,
  }).filter(([, ref]) => typeof ref === "string" && ref.length > 0 && ref.length <= 512));
}

export function evidenceRecoveryHint(value) {
  return Object.entries(evidenceRecoveryDetails(value))
    .map(([key, ref]) => `${key}=${JSON.stringify(ref)}`).join("; ");
}

// One delivery boundary for MCP and native Pi. A ready label alone is not delivery.
// Raw artifact payload fields always fail closed. Non-ready delivery state fails
// closed for artifact consumers (required=true) and surfaces truthfully for
// read-only callers (required=false): pending reconciliation never becomes
// success, but it also no longer blocks record-page navigation or status reads.
export function assertEvidenceDelivery(value, required = false, options = {}) {
  const allowAdhoc = options.allowAdhoc === true;
  const rawProblem = findRawArtifactField(value);
  const problem = rawProblem || (required ? findNonReadyArtifactDelivery(value) : "");
  const recovery = evidenceRecoveryHint(value);
  const fail = (reason) => {
    const error = new Error(`EPWA delivery unavailable: ${reason}${recovery ? `; ${recovery}` : ""}`);
    error.code = "epwa_delivery_unavailable";
    // Explicit allowlist: never copy the rejected payload, title, scope or links.
    error.delivery_error = Object.freeze({
      schema: "uiai.epwa_delivery_error.v1",
      code: error.code,
      delivery_state: "blocked",
      message: "EPWA delivery unavailable: delivery was not verified; no raw artifact bytes were returned.",
      ...evidenceRecoveryDetails(value),
    });
    throw error;
  };
  if (allowAdhoc && !rawProblem && unsealedAdhocLink(value)) {
    // #299: the caller attempted no evidence scope, and the engine returned
    // the narrow unsealed ad hoc shape (private short-lived link, honestly
    // blocked formal envelope, no raw bytes). That is a delivered work result,
    // not a formal-evidence failure — read-only and evidence consumers may
    // proceed; formal evidence still requires a complete scope.
    return null;
  }
  if (problem) fail(problem);
  if (required && value?.schema !== "uiai.epwa_delivery.v1" && !value?.epwa_delivery) {
    fail("required evidence envelope missing; producer reconciliation required");
  }
  const visit = (entry, required) => {
    if (!entry || typeof entry !== "object") return null;
    const envelope = entry.schema === "uiai.epwa_delivery.v1" ? entry : entry.epwa_delivery;
    if (!envelope) {
      let first = null;
      for (const child of Object.values(entry)) {
        const links = visit(child, required);
      if (!first) first = links;
      }
      return first;
    }
    if (envelope.state !== "ready") {
      // Pending/blocked envelopes are truthful state, not delivery claims; only
      // artifact consumers (required=true) fail on them.
      if (required) fail("malformed ready evidence envelope");
      return null;
    }
    if (envelope.schema !== "uiai.epwa_delivery.v1" ||
        typeof envelope.artifact?.artifact_ref !== "string" || !envelope.artifact.artifact_ref.trim()) {
      fail("malformed ready evidence envelope");
    }
    const parse = (text) => {
      if (typeof text !== "string" || /[\x00-\x20\\]/.test(text)) fail("invalid evidence URL");
      let url;
      try { url = new URL(text); } catch { fail("invalid evidence URL"); }
      if (url.protocol !== "https:" || !url.hostname || url.username || url.password || url.hash) {
        fail("evidence URL must be credential-free HTTPS without a fragment");
      }
      return url;
    };
    const record = parse(envelope.epwa?.record_url);
    const portable = parse(envelope.epwa?.portable_url);
    // Match the producer contract: a portable ZIP may have a separately configured HTTPS origin.
    if (!portable.pathname.endsWith(".zip")) fail("portable evidence URL is not a ZIP");
    for (const [key, expected] of [["artifact_url", envelope.epwa.record_url], ["portable_url", envelope.epwa.portable_url]]) {
      if (entry[key] !== undefined && entry[key] !== expected) fail(`${key} disagrees with canonical delivery`);
    }
    return { recordURL: record.href, portableURL: portable.href };
  };
  return visit(value);
}
