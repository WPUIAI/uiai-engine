"use strict";

const byId = (id) => document.getElementById(id);
const text = (node, value) => { node.textContent = value ?? "—"; };
const availabilityStates = new Set(["loading", "ready", "unavailable", "blocked", "corrupt", "stale", "redacted", "degraded"]);
const expectedSections = ["overview", "evidence", "timeline", "inspect", "developer"];
const registryAPI = "/api/evidence/registry/public";
const lowMemory = typeof navigator.deviceMemory === "number" && navigator.deviceMemory <= 2;
const registryPageSize = lowMemory ? 25 : 100;
const registryState = { project: "", query: "", status: "", type: "", artifactCursor: "", workItemCursor: "", artifacts: [], workItems: [], eventSource: null, reloadTimer: 0 };

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

const queryString = (values) => {
  const params = new URLSearchParams();
  Object.entries(values).forEach(([key, value]) => { if (value) params.set(key, String(value)); });
  return params.toString();
};

const publicPath = (value) => typeof value === "string" && /^\/[A-Za-z0-9._~!$&'()*+,;=:@%/?-]+$/.test(value) && !value.includes("..") ? value : null;

function registryRow(kind, record) {
  const tr = document.createElement("tr");
  const kindCell = document.createElement("td");
  const recordCell = document.createElement("td");
  const statusCell = document.createElement("td");
  const bindingCell = document.createElement("td");
  const revisionCell = document.createElement("td");
  const button = document.createElement("button");
  button.type = "button";
  button.className = "registry-record";
  button.dataset.kind = kind;
  button.dataset.ref = kind === "artifact" ? record.artifact_ref : record.work_item_ref;
  text(kindCell, kind === "artifact" ? "Artifact" : record.item_type || "Work item");
  const strong = document.createElement("strong");
  const code = document.createElement("code");
  text(strong, kind === "artifact" ? record.title : record.title);
  text(code, kind === "artifact" ? record.artifact_ref : record.item_id);
  button.append(strong, code);
  recordCell.append(button);
  text(statusCell, kind === "artifact" ? record.verification : record.status);
  text(bindingCell, kind === "artifact" ? record.closure : record.binding_state);
  text(revisionCell, kind === "artifact" ? record.revision : record.revision);
  tr.append(kindCell, recordCell, statusCell, bindingCell, revisionCell);
  button.addEventListener("click", () => {
    if (kind === "artifact") location.assign(artifactViewURL(record));
    else showRegistryDetail(kind, record);
  });
  return tr;
}

async function showRegistryDetail(kind, record) {
  const panel = byId("registry-detail");
  const title = document.createElement("h2");
  const copy = document.createElement("p");
  const facts = document.createElement("dl");
  title.textContent = record.title || record.item_id || record.artifact_ref;
  copy.textContent = kind === "artifact" ? `Public-safe immutable artifact revision ${record.revision}.` : (record.description || "No provider description was projected.");
  facts.className = "registry-detail-facts";
  const entries = kind === "artifact" ? [
    ["Artifact", record.artifact_ref], ["Work item", record.first_work_item_ref], ["Verification", record.verification], ["Closure", record.closure], ["Captured", formatTime(record.captured_at)],
  ] : [
    ["Work item", record.work_item_ref], ["Provider", record.provider_surface], ["Type", record.item_type], ["Status", record.status], ["Binding", record.binding_state], ["Revision", record.revision],
  ];
  facts.replaceChildren(...entries.map(([label, value]) => fact(label, value || "Unavailable")));
  const children = [title, copy, facts];
  if (kind === "artifact" && publicPath(record.pwa_path)) {
    const link = document.createElement("a");
    link.href = record.pwa_path;
    link.textContent = "Open forensic evidence record";
    children.push(link);
  }
  if (kind === "work_item") {
    try {
      const base = { project_ref: registryState.project, object_ref: record.work_item_ref, limit: 100 };
      const [forward, reverse] = await Promise.all([
        fetchJSON(`${registryAPI}/edges?${queryString({ ...base, direction: "forward" })}`),
        fetchJSON(`${registryAPI}/edges?${queryString({ ...base, direction: "reverse" })}`),
      ]);
      const relationships = document.createElement("p");
      relationships.textContent = `${forward.edges?.length || 0} outgoing · ${reverse.edges?.length || 0} incoming relationships`;
      children.push(relationships);
    } catch { /* Detail remains useful when relationship projection is degraded. */ }
  }
  panel.replaceChildren(...children);
  panel.hidden = false;
  const url = new URL(location.href);
  url.searchParams.set("selected", kind === "artifact" ? record.artifact_ref : record.work_item_ref);
  history.replaceState(null, "", url);
}

function artifactViewURL(record) {
  const url = new URL(location.href);
  url.searchParams.set("project", registryState.project || record.project_ref);
  url.searchParams.set("artifact", record.artifact_ref);
  url.searchParams.set("revision", String(record.revision));
  url.searchParams.delete("selected");
  return url.href;
}

function updateRegistryURL() {
  const url = new URL(location.href);
  [["project", registryState.project], ["q", registryState.query], ["status", registryState.status], ["type", registryState.type]].forEach(([key, value]) => value ? url.searchParams.set(key, value) : url.searchParams.delete(key));
  history.replaceState(null, "", url);
}

async function loadRegistry({ append = false } = {}) {
  const status = byId("status");
  if (!registryState.project) return;
  if (!append) {
    registryState.artifactCursor = "";
    registryState.workItemCursor = "";
    registryState.artifacts = [];
    registryState.workItems = [];
  }
  setStatus(status, "loading", "Synchronizing registry");
  const shared = { project_ref: registryState.project, q: registryState.query };
  const [artifacts, workItems, syncStatus] = await Promise.all([
    append && !registryState.artifactCursor ? Promise.resolve({ rows: [] }) : fetchJSON(`${registryAPI}/artifacts?${queryString({ ...shared, page_size: registryPageSize, cursor: registryState.artifactCursor })}`),
    append && !registryState.workItemCursor ? Promise.resolve({ work_items: [] }) : fetchJSON(`${registryAPI}/work-items?${queryString({ ...shared, status: registryState.status, item_type: registryState.type, limit: registryPageSize, cursor: registryState.workItemCursor })}`),
    fetchJSON(`${registryAPI}/sync-status`),
  ]);
  registryState.artifacts.push(...(artifacts.rows || []));
  registryState.workItems.push(...(workItems.work_items || []));
  registryState.artifactCursor = artifacts.next_cursor || "";
  registryState.workItemCursor = workItems.next_cursor || "";
  const rows = [...registryState.artifacts.map((row) => registryRow("artifact", row)), ...registryState.workItems.map((row) => registryRow("work_item", row))];
  byId("registry-rows").replaceChildren(...rows);
  byId("registry-empty").hidden = rows.length !== 0;
  byId("registry-more").hidden = !registryState.artifactCursor && !registryState.workItemCursor;
  text(byId("registry-count"), `${rows.length} records`);
  text(byId("registry-revision"), Math.max(artifacts.index_revision || 0, workItems.index_revision || 0));
  text(byId("registry-freshness"), syncStatus.freshness || "unavailable");
  byId("registry-facets").replaceChildren(
    fact("Artifacts", registryState.artifacts.length), fact("Work items", registryState.workItems.length),
    fact("Epics", registryState.workItems.filter((item) => item.item_type === "epic").length),
    fact("Blocked", registryState.workItems.filter((item) => item.status === "blocked").length),
  );
  byId("registry").dataset.registryState = syncStatus.freshness || "degraded";
  text(byId("registry-truth"), rows.length ? "Public-safe immutable artifacts and provider-observed Work Items. Focusa bindings remain independently revisioned." : "The project is available, but no public-safe records are currently indexed.");
  setStatus(status, syncStatus.freshness === "live" ? "ready" : "degraded", `Registry ${syncStatus.freshness || "degraded"}`);
  updateRegistryURL();
  const selected = new URL(location.href).searchParams.get("selected");
  const selectedArtifact = registryState.artifacts.find((record) => record.artifact_ref === selected);
  const selectedWorkItem = registryState.workItems.find((record) => record.work_item_ref === selected);
  if (selectedArtifact) showRegistryDetail("artifact", selectedArtifact);
  else if (selectedWorkItem) showRegistryDetail("work_item", selectedWorkItem);
}

function connectRegistryEvents() {
  registryState.eventSource?.close();
  const source = new EventSource(`${registryAPI}/events`);
  registryState.eventSource = source;
  source.addEventListener("registry_revision", (message) => {
    try {
      const event = JSON.parse(message.data);
      const relevant = !event.results?.length || event.results.some((result) => result.project_ref === registryState.project);
      if (relevant) {
        clearTimeout(registryState.reloadTimer);
        registryState.reloadTimer = setTimeout(() => loadRegistry().catch(showRegistryUnavailable), 150);
      }
    } catch { showRegistryUnavailable(new Error("Registry event contract is invalid")); }
  });
  source.onerror = () => {
    text(byId("registry-freshness"), "degraded · reconnecting");
    setStatus(byId("status"), "degraded", "Registry reconnecting");
  };
}

function showRegistryUnavailable(error) {
  byId("registry").dataset.registryState = "unavailable";
  text(byId("registry-truth"), error instanceof Error ? error.message : "Registry unavailable");
  text(byId("registry-freshness"), "unavailable");
  byId("registry-empty").hidden = false;
  byId("registry-rows").replaceChildren();
  setStatus(byId("status"), "unavailable", "Registry unavailable");
}

async function renderRegistry() {
  const url = new URL(location.href);
  registryState.query = url.searchParams.get("q") || "";
  registryState.status = url.searchParams.get("status") || "";
  registryState.type = url.searchParams.get("type") || "";
  byId("registry-query").value = registryState.query;
  byId("registry-status").value = registryState.status;
  byId("registry-type").value = registryState.type;
  const response = await fetchJSON(`${registryAPI}/projects`);
  const projects = response.projects || [];
  const select = byId("registry-project");
  select.replaceChildren(...projects.map((project) => {
    const option = document.createElement("option");
    option.value = project.project_ref;
    option.textContent = project.display_name;
    return option;
  }));
  registryState.project = projects.some((project) => project.project_ref === url.searchParams.get("project")) ? url.searchParams.get("project") : (projects[0]?.project_ref || "");
  select.value = registryState.project;
  if (!registryState.project) throw new Error("No public-safe project registry is configured.");
  await loadRegistry();
  connectRegistryEvents();
  requestAnimationFrame(() => scrollTo(0, Number(sessionStorage.getItem("epwa-registry-scroll") || 0)));
}

function wireRegistryControls() {
  let debounce = 0;
  byId("registry-project").addEventListener("change", (event) => { registryState.project = event.target.value; loadRegistry().catch(showRegistryUnavailable); });
  byId("registry-query").addEventListener("input", (event) => { registryState.query = event.target.value.trim(); clearTimeout(debounce); debounce = setTimeout(() => loadRegistry().catch(showRegistryUnavailable), 250); });
  byId("registry-status").addEventListener("change", (event) => { registryState.status = event.target.value; loadRegistry().catch(showRegistryUnavailable); });
  byId("registry-type").addEventListener("change", (event) => { registryState.type = event.target.value; loadRegistry().catch(showRegistryUnavailable); });
  byId("registry-more").addEventListener("click", () => loadRegistry({ append: true }).catch(showRegistryUnavailable));
  byId("registry-rows").addEventListener("keydown", (event) => {
    if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return;
    const controls = [...byId("registry-rows").querySelectorAll("button")];
    const current = controls.indexOf(document.activeElement);
    controls[Math.max(0, Math.min(controls.length - 1, current + (event.key === "ArrowDown" ? 1 : -1)))]?.focus();
    event.preventDefault();
  });
  addEventListener("beforeunload", () => {
    if (!byId("registry").hidden) sessionStorage.setItem("epwa-registry-scroll", String(scrollY));
  });
}

async function renderPublicRecord() {
  const status = byId("status");
  const url = new URL(location.href);
  const artifactRef = url.searchParams.get("artifact") || "";
  const revision = Number(url.searchParams.get("revision"));
  const back = new URL(location.href);
  back.searchParams.delete("artifact"); back.searchParams.delete("revision");
  byId("registry-back").href = back.href;
  try {
    if (!artifactRef || !Number.isSafeInteger(revision) || revision < 1) throw new Error("Exact artifact identity is invalid");
    const base = `${registryAPI}/artifacts/${encodeURIComponent(artifactRef)}/revisions/${revision}`;
    const detail = await fetchJSON(base);
    const manifest = detail.manifest;
    if (detail.schema !== "uiai.public_evidence_artifact_detail.v1" || detail.artifact_id !== artifactRef || detail.revision !== revision || manifest?.artifact_id !== artifactRef || manifest?.revision !== revision) throw new Error("Artifact detail contract is invalid");
    if (!validSHA256(detail.manifest_sha256) || manifest.integrity?.manifest_sha256 !== detail.manifest_sha256) throw new Error("Artifact integrity binding is corrupt");
    const scope = manifest.scope || {};
    const flatScope = { project_ref: scope.project?.project_ref, workstream_ref: scope.workstream?.workstream_ref, workset_ref: scope.workset?.workset_ref, callgraph_ref: scope.callgraph?.frame_ref || scope.callgraph?.run_ref, workpoint_ref: scope.workpoint?.workpoint_ref, work_item_ref: scope.work_items?.[0]?.work_item_ref };
    text(byId("title"), manifest.title || "Evidence record"); text(byId("truth"), manifest.summary || "Bound immutable evidence artifact.");
    text(byId("record-id"), artifactRef); text(byId("record-revision"), revision); renderLineage(flatScope);
    const assets = Array.isArray(detail.assets) ? detail.assets : [];
    const primary = assets.find((asset) => asset.media_type?.startsWith("image/"));
    const frame = byId("primary-evidence-frame");
    if (primary && publicPath(primary.href) && validSHA256(primary.sha256)) {
      frame.hidden = false; byId("screenshot").src = primary.href; byId("screenshot").alt = primary.alt_text || `Evidence asset ${primary.asset_id}`; byId("image-link").href = primary.href;
      text(byId("source-label"), primary.source_ref || "Source not disclosed"); text(byId("capture-label"), `${primary.width || "—"} × ${primary.height || "—"} · ${primary.media_type}`);
      text(byId("caption"), `Captured ${formatTime(primary.captured_at || manifest.captured_at)} · ${formatBytes(primary.byte_size)} · SHA-256 ${primary.sha256}`);
    } else frame.hidden = true;
    const claimCount = manifest.claims?.length || 0;
    setValidity("integrity", "Digest-bound", "recorded"); setValidity("provenance", `${manifest.provenance?.custody?.length || 0} custody event(s)`, "recorded");
    setValidity("observation", manifest.claims?.[0]?.status || "Recorded", "recorded"); setValidity("sufficiency", `Limited · ${claimCount} claim / ${assets.length} asset`, "limited");
    setValidity("verification", manifest.verification?.status || "indeterminate", manifest.verification?.status || "not_determined"); setValidity("completion", "Not asserted by this artifact", "not_determined");
    setValidity("settlement", "Not asserted by this artifact", "not_determined"); setValidity("legal", "Not determined", "not_determined");
    byId("facts").replaceChildren(fact("Captured", formatTime(manifest.captured_at)), fact("Created", formatTime(manifest.created_at)), fact("Evidence assets", assets.length), fact("Claims", claimCount), fact("Access", manifest.policy?.access_class), fact("Redaction", manifest.policy?.redaction_state), fact("Authority posture", manifest.authority?.posture), fact("Retention", manifest.policy?.retention_class));
    renderTimeline((manifest.provenance?.custody || []).map((event) => ({ event_type: event.action, occurred_at: event.occurred_at, refs: [...(event.input_refs || []), ...(event.output_refs || [])] })));
    byId("inspect-grid").replaceChildren(datum("Artifact", artifactRef), datum("Revision", revision), datum("Manifest SHA-256", detail.manifest_sha256), datum("Bundle SHA-256", manifest.integrity?.bundle_sha256), datum("Project", flatScope.project_ref), datum("Workstream", flatScope.workstream_ref), datum("Workset", flatScope.workset_ref), datum("CallGraph", flatScope.callgraph_ref), datum("Workpoint", flatScope.workpoint_ref), datum("Work Item", flatScope.work_item_ref), datum("Evidence authority", manifest.authority?.evidence_authority_ref), datum("Completion authority", manifest.authority?.completion_authority_ref), datum("Verification", manifest.verification?.status), datum("Redaction", manifest.policy?.redaction_state));
    byId("detail-json-link").href = base; byId("manifest-json-link").href = detail.manifest_path || `${base}/manifest`; byId("inspection-json-link").hidden = true;
    text(byId("limitations-copy"), "This immutable artifact is a bounded evidence input. Its existence does not independently establish completeness, review acceptance, task completion, provider closure, settlement, or legal admissibility.");
    setStatus(status, "ready", "Read-only artifact loaded");
  } catch (error) {
    setStatus(status, "corrupt", "Artifact unavailable"); text(byId("truth"), error instanceof Error ? error.message : "Evidence artifact unavailable"); setUnavailableValidity("corrupt"); byId("evidence").hidden = true;
  }
}

async function renderRecord() {
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

wireRegistryControls();
const route = new URL(location.href);
if (route.searchParams.get("artifact")) {
  byId("registry").hidden = true;
  byId("record-detail").hidden = false;
  renderPublicRecord();
} else {
  renderRegistry().catch(showRegistryUnavailable);
  if (route.searchParams.get("view") === "record") {
    byId("record-detail").hidden = false;
    renderRecord();
  }
}
