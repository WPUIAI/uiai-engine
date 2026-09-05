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
    if (RAW_ARTIFACT_FIELDS.has(key) && child !== undefined && child !== null && child !== "") {
      return `${path}.${key}`;
    }
    const found = findRawArtifactField(child, `${path}.${key}`);
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
