"use strict";

const byId = (id) => document.getElementById(id);
const text = (node, value) => { node.textContent = value ?? "—"; };
const availabilityStates = new Set(["loading", "ready", "unavailable", "blocked", "corrupt", "stale", "redacted", "degraded"]);
const expectedSections = ["overview", "evidence", "timeline", "inspect", "developer"];

const safeRef = (value) => typeof value === "string" && /^\.\/[A-Za-z0-9._/-]+$/.test(value) && !value.includes("../") ? value : null;
const validSHA256 = (value) => typeof value === "string" && /^[a-f0-9]{64}$/.test(value);
const formatBytes = (value) => new Intl.NumberFormat(undefined, { style: "unit", unit: "byte", notation: value > 999999 ? "compact" : "standard" }).format(value);
const formatTime = (value) => {
  const date = new Date(value);
  return Number.isNaN(date.valueOf()) ? "Invalid recorded time" : date.toLocaleString();
};

function fact(label, value) {
  const wrap = document.createElement("div");
  const dt = document.createElement("dt");
  const dd = document.createElement("dd");
  wrap.className = "fact";
  text(dt, label);
  text(dd, value);
  wrap.append(dt, dd);
  return wrap;
}

function datum(label, value) {
  const wrap = document.createElement("div");
  const small = document.createElement("small");
  const code = document.createElement("code");
  wrap.className = "datum";
  text(small, label);
  text(code, value);
  wrap.append(small, code);
  return wrap;
}

function lineageItem(label, value) {
  const item = document.createElement("li");
  const strong = document.createElement("strong");
  const code = document.createElement("code");
  text(strong, label + " ");
  text(code, value || "Unbound");
  item.append(strong, code);
  return item;
}

function setStatus(status, state, label) {
  status.className = `status ${state === "ready" ? "ready" : "error"}`;
  status.dataset.epwaStatus = state;
  text(status.querySelector("span"), label);
}

function setValidity(layer, value, state) {
  const node = byId(`validity-${layer}`);
  text(node, value);
  node.dataset.state = state;
}

function setUnavailableValidity(state) {
  setValidity("integrity", state === "corrupt" ? "Failed" : "Unavailable", state);
  setValidity("provenance", "Unavailable", state);
  setValidity("observation", "Unavailable", state);
  setValidity("sufficiency", "Not assessable", "blocked");
  setValidity("verification", "Not determined", "not_determined");
  setValidity("completion", "Not determined", "not_determined");
  setValidity("settlement", "Not determined", "not_determined");
  setValidity("legal", "Not determined", "not_determined");
}

const validProjection = (value) => value?.schema === "uiai.evidence_pwa_projection.v1" &&
  availabilityStates.has(value.availability) && value.interaction === "read_only" &&
  expectedSections.every((id, index) => value.sections?.[index]?.id === id);

async function fetchJSON(ref) {
  const response = await fetch(ref, { cache: "no-store", credentials: "omit" });
  if (!response.ok) throw new Error(`Record unavailable (${response.status})`);
  return response.json();
}

function renderTimeline(entries) {
  const events = byId("events");
  const rows = (entries || []).map((entry) => {
    const item = document.createElement("li");
    const time = document.createElement("time");
    const copy = document.createElement("div");
    time.dateTime = entry.occurred_at;
    text(time, formatTime(entry.occurred_at));
    text(copy, entry.event_type);
    item.append(time, copy);
    return item;
  });
  if (rows.length === 0) {
    const item = document.createElement("li");
    text(item, "No bounded timeline entries are available in this projection.");
    rows.push(item);
  }
  events.replaceChildren(...rows);
}

function renderLineage(scope) {
  byId("lineage-list").replaceChildren(
    lineageItem("Project", scope.project_ref),
    lineageItem("Workstream", scope.workstream_ref),
    lineageItem("Workset", scope.workset_ref),
    lineageItem("CallGraph", scope.callgraph_ref),
    lineageItem("Workpoint", scope.workpoint_ref),
    lineageItem("Work Item", scope.work_item_ref),
  );
}

async function render() {
  const status = byId("status");
  try {
    const manifest = await fetchJSON("./artifact.json");
    if (manifest.schema !== "uiai.screenshot_evidence_share.v1") throw new Error("Artifact manifest contract is unsupported");
    if (manifest.availability !== "ready" || !safeRef(manifest.projection_ref)) {
      const state = availabilityStates.has(manifest.availability) ? manifest.availability : "corrupt";
      setStatus(status, state, `Record ${state}`);
      text(byId("truth"), "Canonical projection lineage is incomplete; evidentiary rendering is prohibited.");
      setUnavailableValidity(state);
      byId("evidence").hidden = true;
      return;
    }

    const projection = await fetchJSON(manifest.projection_ref);
    if (!validProjection(projection)) throw new Error("Canonical Evidence PWA projection is invalid");
    if (projection.availability !== "ready") {
      setStatus(status, projection.availability, `Record ${projection.availability}`);
      text(byId("truth"), projection.summary);
      setUnavailableValidity(projection.availability);
      byId("evidence").hidden = true;
      return;
    }

    const asset = projection.assets?.find((item) => item.modality === "screenshot");
    const source = safeRef(asset?.ref);
    if (!source || source !== safeRef(manifest.screenshot_ref) || asset.sha256 !== manifest.screenshot_sha256 || !validSHA256(asset.sha256)) {
      throw new Error("Screenshot binding is corrupt");
    }
    if (!validSHA256(projection.artifact.manifest_sha256) || !validSHA256(projection.artifact.bundle_sha256)) {
      throw new Error("Artifact integrity binding is corrupt");
    }

    text(byId("title"), projection.title || "Evidence record");
    text(byId("truth"), projection.summary);
    text(byId("record-id"), projection.artifact.artifact_ref);
    text(byId("record-revision"), projection.artifact.revision);
    renderLineage(projection.artifact.scope);

    const image = byId("screenshot");
    image.src = source;
    byId("image-link").href = source;
    text(byId("source-label"), manifest.source_url || "Source not disclosed");
    text(byId("capture-label"), `${manifest.width} × ${manifest.height} · ${manifest.mime}`);
    text(byId("caption"), `Captured ${formatTime(manifest.captured_at)} · ${formatBytes(asset.bytes)} · SHA-256 ${asset.sha256}`);

    const claimCount = projection.claims?.length || 0;
    const assetCount = projection.assets?.length || 0;
    setValidity("integrity", "Digest-bound", "recorded");
    setValidity("provenance", `Engine-recorded · ${projection.federation_posture}`, "recorded");
    setValidity("observation", projection.claims?.[0]?.posture || "Captured visual only", "recorded");
    setValidity("sufficiency", `Limited · ${claimCount} claim / ${assetCount} asset`, "limited");
    setValidity("verification", "Not asserted by this artifact", "not_determined");
    setValidity("completion", "Not asserted by this artifact", "not_determined");
    setValidity("settlement", "Not asserted by this artifact", "not_determined");
    setValidity("legal", "Not determined", "not_determined");

    byId("facts").replaceChildren(
      fact("Captured", formatTime(manifest.captured_at)),
      fact("Freshness observed", formatTime(projection.freshness_observed_at)),
      fact("Target", manifest.source_url || "Not disclosed"),
      fact("Viewport", `${manifest.width} × ${manifest.height}`),
      fact("Access", projection.access),
      fact("Redaction", projection.redaction?.state || "unknown"),
      fact("Interaction", projection.interaction),
      fact("Inspection receipt", manifest.inspection_ref || "Unavailable"),
    );

    text(byId("limitations-copy"), "This is one bounded visual observation with digest and inspection bindings. It does not independently establish completeness, absence of contrary evidence, review acceptance, task completion, provider closure, settlement, or legal admissibility. Legal use depends on jurisdiction, authentication, relevance, custody, applicable evidentiary rules, and independent challenge.");
    renderTimeline(projection.timeline);

    const scope = projection.artifact.scope;
    byId("inspect-grid").replaceChildren(
      datum("Artifact", projection.artifact.artifact_ref),
      datum("Revision", projection.artifact.revision),
      datum("Manifest SHA-256", projection.artifact.manifest_sha256),
      datum("Bundle SHA-256", projection.artifact.bundle_sha256),
      datum("Screenshot SHA-256", asset.sha256),
      datum("Inspection", manifest.inspection_ref),
      datum("Project", scope.project_ref),
      datum("Workstream", scope.workstream_ref),
      datum("Workset", scope.workset_ref),
      datum("CallGraph", scope.callgraph_ref),
      datum("Workpoint", scope.workpoint_ref),
      datum("Work Item", scope.work_item_ref),
      datum("Provenance posture", projection.federation_posture),
      datum("Redaction", projection.redaction.state),
    );
    setStatus(status, "ready", "Read-only record loaded");
  } catch (error) {
    setStatus(status, "corrupt", "Record corrupt or unavailable");
    text(byId("truth"), error instanceof Error ? error.message : "Evidence record unavailable");
    setUnavailableValidity("corrupt");
    byId("evidence").hidden = true;
  }
}

render();
