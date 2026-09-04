"use strict";

const byId = (id) => document.getElementById(id);
const locale = window.EvidenceLocale;
const tr = (key, values) => locale?.t(key, values) ?? key;
const text = (node, value) => { node.textContent = value ?? "—"; };
const availabilityStates = new Set(["loading", "ready", "unavailable", "blocked", "corrupt", "stale", "redacted", "degraded"]);
const expectedSections = ["overview", "evidence", "timeline", "inspect", "developer"];
const deploymentBase = new URL("../", document.baseURI);
const registryAPI = new URL("api/evidence/registry/public", deploymentBase).href.replace(/\/$/, "");
const lowMemory = typeof navigator.deviceMemory === "number" && navigator.deviceMemory <= 2;
const registryPageSize = lowMemory ? 25 : 100;
const registryState = { project: "", query: "", status: "", type: "", artifactCursor: "", workItemCursor: "", artifacts: [], workItems: [], eventSource: null, reloadTimer: 0 };

const safeRef = (value) => typeof value === "string" && /^\.\/[A-Za-z0-9._/-]+$/.test(value) && !value.includes("../") ? value : null;
const validSHA256 = (value) => typeof value === "string" && /^[a-f0-9]{64}$/.test(value);
const formatBytes = (value) => new Intl.NumberFormat(locale?.locale, { style: "unit", unit: "byte", notation: value > 999999 ? "compact" : "standard" }).format(value);
const formatTime = (value) => {
  const date = new Date(value);
  return Number.isNaN(date.valueOf()) ? tr("invalid_time") : (locale?.time(date) ?? date.toLocaleString());
};

function fact(label, value) {
  const wrap = document.createElement("div");
  const dt = document.createElement("dt");
  const dd = document.createElement("dd"); dd.dir = "auto";
  wrap.className = "fact";
  text(dt, label);
  text(dd, value);
  wrap.append(dt, dd);
  return wrap;
}

function datum(label, value) {
  const wrap = document.createElement("div");
  const small = document.createElement("small");
  const code = document.createElement("code"); code.dir = "ltr";
  wrap.className = "datum";
  text(small, label);
  text(code, value);
  wrap.append(small, code);
  return wrap;
}

function lineageItem(label, value) {
  const item = document.createElement("li");
  const strong = document.createElement("strong");
  const code = document.createElement("code"); code.dir = "ltr";
  text(strong, label + " ");
  text(code, value || tr("unbound"));
  item.append(strong, code);
  return item;
}

function setStatus(status, state, label) {
  delete status.dataset.readyLabel;
  status.className = `status ${state}`;
  status.dataset.epwaStatus = state;
  document.querySelector("main > :not([hidden])")?.setAttribute("aria-busy", state === "loading" ? "true" : "false");
  text(status.querySelector("span"), label);
}

function setReadyStatus(status, label) {
  const offlineSnapshot = navigator.onLine === false || location.protocol === "file:";
  if (offlineSnapshot) setStatus(status, "stale", `${label} · ${tr("offline_snapshot")}`);
  else setStatus(status, "ready", label);
  status.dataset.readyLabel = label;
}

function refreshConnectionStatus() {
  const status = byId("status");
  if (status.dataset.readyLabel) setReadyStatus(status, status.dataset.readyLabel);
  if (navigator.onLine === false) document.body.dataset.pwaStatus = "offline";
}

addEventListener("online", refreshConnectionStatus);
addEventListener("offline", refreshConnectionStatus);

function setValidity(layer, value, state) {
  const node = byId(`validity-${layer}`);
  text(node, value);
  node.dataset.state = state;
}

function setUnavailableValidity(state) {
  setValidity("integrity", state === "corrupt" ? tr("record_corrupt") : tr("unavailable_value"), state);
  setValidity("provenance", tr("unavailable_value"), state);
  setValidity("observation", tr("unavailable_value"), state);
  setValidity("sufficiency", tr("not_assessable"), "blocked");
  setValidity("verification", tr("not_determined"), "not_determined");
  setValidity("completion", tr("not_determined"), "not_determined");
  setValidity("settlement", tr("not_determined"), "not_determined");
  setValidity("legal", tr("not_determined"), "not_determined");
}

const validProjection = (value) => value?.schema === "uiai.evidence_pwa_projection.v1" &&
  availabilityStates.has(value.availability) && value.interaction === "read_only" &&
  expectedSections.every((id, index) => value.sections?.[index]?.id === id);

async function fetchJSON(ref) {
  const response = await fetch(ref, { cache: "no-store", credentials: "omit" });
  if (!response.ok) throw new Error(`${tr("unavailable_value")} (${response.status})`);
  return response.json();
}

function renderTimeline(entries) {
  const events = byId("events");
  const rows = (entries || []).map((entry) => {
    const item = document.createElement("li");
    const time = document.createElement("time");
    const copy = document.createElement("div"); copy.dir = "auto";
    time.dateTime = entry.occurred_at;
    text(time, formatTime(entry.occurred_at));
    text(copy, entry.event_type);
    item.append(time, copy);
    return item;
  });
  if (rows.length === 0) {
    const item = document.createElement("li");
    text(item, tr("no_timeline"));
    rows.push(item);
  }
  events.replaceChildren(...rows);
}

const queryString = (values) => {
  const params = new URLSearchParams();
  Object.entries(values).forEach(([key, value]) => { if (value) params.set(key, String(value)); });
  return params.toString();
};

const publicPath = (value) => typeof value === "string" && /^\/[A-Za-z0-9._~!$&'()*+,;=:@%/?-]+$/.test(value) && !value.includes("..") ? new URL(value.slice(1), deploymentBase).href : null;

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
  const cellLabels = ["kind", "record", "status", "binding", "revision"].map((key) => tr(key));
  [kindCell, recordCell, statusCell, bindingCell, revisionCell].forEach((cell, index) => { cell.dataset.label = cellLabels[index]; cell.dir = index === 1 ? "ltr" : "auto"; });
  text(kindCell, kind === "artifact" ? tr("kind_artifact") : record.item_type || tr("kind_work_item"));
  const strong = document.createElement("strong");
  const code = document.createElement("code"); strong.dir = "auto"; code.dir = "ltr";
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
  const copy = document.createElement("p"); copy.dir = "auto";
  const facts = document.createElement("dl");
  title.textContent = record.title || record.item_id || record.artifact_ref;
  copy.textContent = kind === "artifact" ? tr("public_artifact_revision", { revision: record.revision }) : (record.description || tr("no_provider_description"));
  facts.className = "registry-detail-facts";
  const entries = kind === "artifact" ? [
    [tr("artifact"), record.artifact_ref], [tr("kind_work_item"), record.first_work_item_ref], [tr("verification"), record.verification], [tr("closure"), record.closure], [tr("captured"), formatTime(record.captured_at)],
  ] : [
    [tr("kind_work_item"), record.work_item_ref], [tr("provider"), record.provider_surface], [tr("type"), record.item_type], [tr("status"), record.status], [tr("binding"), record.binding_state], [tr("revision"), record.revision],
  ];
  facts.replaceChildren(...entries.map(([label, value]) => fact(label, value || tr("unavailable_value"))));
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
      relationships.textContent = tr("relationships", { outgoing: locale.number(forward.edges?.length || 0), incoming: locale.number(reverse.edges?.length || 0) });
      children.push(relationships);
    } catch { /* Detail remains useful when relationship projection is degraded. */ }
  }
  panel.replaceChildren(...children);
  panel.hidden = false;
  panel.focus();
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
  byId("registry").setAttribute("aria-busy", "true");
  setStatus(status, "loading", tr("syncing_registry"));
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
  text(byId("registry-count"), tr("records_count", { count: locale.number(rows.length) }));
  text(byId("registry-revision"), Math.max(artifacts.index_revision || 0, workItems.index_revision || 0));
  text(byId("registry-freshness"), syncStatus.freshness || tr("unavailable_value"));
  byId("registry-facets").replaceChildren(
    fact(tr("kind_artifact"), locale.number(registryState.artifacts.length)), fact(tr("kind_work_item"), locale.number(registryState.workItems.length)),
    fact(tr("epic"), locale.number(registryState.workItems.filter((item) => item.item_type === "epic").length)),
    fact(tr("state_blocked"), locale.number(registryState.workItems.filter((item) => item.status === "blocked").length)),
  );
  byId("registry").dataset.registryState = syncStatus.freshness || "degraded";
  text(byId("registry-truth"), rows.length ? tr("registry_has_records") : tr("registry_no_records"));
  byId("registry").setAttribute("aria-busy", "false");
  if (syncStatus.freshness === "live") setReadyStatus(status, tr("registry_live"));
  else setStatus(status, "degraded", tr("registry_degraded", { state: syncStatus.freshness || "degraded" }));
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
    } catch { showRegistryUnavailable(new Error(tr("registry_unavailable"))); }
  });
  source.onerror = () => {
    text(byId("registry-freshness"), tr("reconnecting"));
    setStatus(byId("status"), "degraded", tr("registry_reconnecting"));
  };
}

function showRegistryUnavailable(error) {
  byId("registry").dataset.registryState = "unavailable";
  byId("registry").setAttribute("aria-busy", "false");
  text(byId("registry-truth"), error instanceof Error ? error.message : tr("registry_unavailable"));
  text(byId("registry-freshness"), tr("unavailable_value"));
  byId("registry-empty").hidden = false;
  byId("registry-rows").replaceChildren();
  setStatus(byId("status"), "unavailable", tr("registry_unavailable"));
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
  if (!registryState.project) throw new Error(tr("no_public_project"));
  await loadRegistry();
  connectRegistryEvents();
  requestAnimationFrame(() => scrollTo(0, Number(sessionStorage.getItem("epwa-registry-scroll") || 0)));
}

function wireRegistryControls() {
  let debounce = 0;
  byId("registry-controls").addEventListener("submit", (event) => event.preventDefault());
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
    if (!artifactRef || !Number.isSafeInteger(revision) || revision < 1) throw new Error(tr("exact_identity_invalid"));
    const base = `${registryAPI}/artifacts/${encodeURIComponent(artifactRef)}/revisions/${revision}`;
    const detail = await fetchJSON(base);
    const manifest = detail.manifest;
    if (detail.schema !== "uiai.public_evidence_artifact_detail.v1" || detail.artifact_id !== artifactRef || detail.revision !== revision || manifest?.artifact_id !== artifactRef || manifest?.revision !== revision) throw new Error(tr("artifact_contract_invalid"));
    if (!validSHA256(detail.manifest_sha256) || manifest.integrity?.manifest_sha256 !== detail.manifest_sha256) throw new Error(tr("integrity_corrupt"));
    const scope = manifest.scope || {};
    const flatScope = { project_ref: scope.project?.project_ref, workstream_ref: scope.workstream?.workstream_ref, workset_ref: scope.workset?.workset_ref, callgraph_ref: scope.callgraph?.frame_ref || scope.callgraph?.run_ref, workpoint_ref: scope.workpoint?.workpoint_ref, work_item_ref: scope.work_items?.[0]?.work_item_ref, work_items: scope.work_items };
    text(byId("title"), manifest.title || tr("evidence_record")); text(byId("truth"), manifest.summary || tr("bound_immutable_summary"));
    text(byId("record-id"), artifactRef); text(byId("record-revision"), revision); renderLineage(flatScope);
    const assets = Array.isArray(detail.assets) ? detail.assets : [];
    const primary = assets.find((asset) => asset.media_type?.startsWith("image/"));
    const frame = byId("primary-evidence-frame");
    if (primary && publicPath(primary.href) && validSHA256(primary.sha256)) {
      frame.hidden = false; byId("screenshot").src = primary.href; byId("screenshot").alt = primary.alt_text || `Evidence asset ${primary.asset_id}`; byId("image-link").href = primary.href;
      text(byId("source-label"), primary.source_ref || tr("source_not_disclosed")); text(byId("capture-label"), `${primary.width || "—"} × ${primary.height || "—"} · ${primary.media_type}`);
      text(byId("caption"), `${tr("captured")} ${formatTime(primary.captured_at || manifest.captured_at)} · ${formatBytes(primary.byte_size)} · SHA-256 ${primary.sha256}`);
    } else frame.hidden = true;
    const claimCount = manifest.claims?.length || 0;
    setValidity("integrity", tr("digest_bound"), "recorded"); setValidity("provenance", tr("custody_events", { count: locale.number(manifest.provenance?.custody?.length || 0) }), "recorded");
    setValidity("observation", manifest.claims?.[0]?.status || tr("recorded"), "recorded"); setValidity("sufficiency", tr("limited_counts", { claims: locale.number(claimCount), assets: locale.number(assets.length) }), "limited");
    setValidity("verification", manifest.verification?.status || "indeterminate", manifest.verification?.status || "not_determined"); setValidity("completion", tr("not_asserted"), "not_determined");
    setValidity("settlement", tr("not_asserted"), "not_determined"); setValidity("legal", tr("not_determined"), "not_determined");
    byId("facts").replaceChildren(fact(tr("captured"), formatTime(manifest.captured_at)), fact(tr("created"), formatTime(manifest.created_at)), fact(tr("evidence_assets"), locale.number(assets.length)), fact(tr("claims"), locale.number(claimCount)), fact(tr("access"), manifest.policy?.access_class), fact(tr("redaction"), manifest.policy?.redaction_state), fact(tr("authority_posture"), manifest.authority?.posture), fact(tr("retention"), manifest.policy?.retention_class));
    renderTimeline((manifest.provenance?.custody || []).map((event) => ({ event_type: event.action, occurred_at: event.occurred_at, refs: [...(event.input_refs || []), ...(event.output_refs || [])] })));
    byId("inspect-grid").replaceChildren(datum(tr("artifact"), artifactRef), datum(tr("revision"), revision), datum("Manifest SHA-256", detail.manifest_sha256), datum("Bundle SHA-256", manifest.integrity?.bundle_sha256), datum(tr("project_label"), flatScope.project_ref), datum(tr("workstream"), flatScope.workstream_ref), datum(tr("workset"), flatScope.workset_ref), datum(tr("callgraph"), flatScope.callgraph_ref), datum(tr("workpoint"), flatScope.workpoint_ref), ...workItemInspectData(flatScope), datum(tr("evidence_authority"), manifest.authority?.evidence_authority_ref), datum(tr("execution_authority"), manifest.authority?.completion_authority_ref), datum(tr("verification"), manifest.verification?.status), datum(tr("redaction"), manifest.policy?.redaction_state));
    byId("detail-json-link").href = base; byId("manifest-json-link").href = publicPath(detail.manifest_path) || `${base}/manifest`; byId("inspection-json-link").hidden = true;
    text(byId("limitations-copy"), tr("artifact_limitations"));
    setReadyStatus(status, tr("artifact_loaded"));
    byId("title").focus({ preventScroll: true });
  } catch (error) {
    setStatus(status, "corrupt", tr("artifact_unavailable")); text(byId("truth"), error instanceof Error ? error.message : tr("artifact_unavailable")); setUnavailableValidity("corrupt"); byId("evidence").hidden = true;
  }
}

async function renderRecord() {
  const status = byId("status");
  try {
    const manifest = await fetchJSON("./artifact.json");
    if (manifest.schema !== "uiai.screenshot_evidence_share.v1") throw new Error(tr("record_unsupported"));
    if (manifest.availability !== "ready" || !safeRef(manifest.projection_ref)) {
      const state = availabilityStates.has(manifest.availability) ? manifest.availability : "corrupt";
      setStatus(status, state, tr("record_state", { state }));
      text(byId("truth"), tr("incomplete_lineage"));
      setUnavailableValidity(state);
      byId("evidence").hidden = true;
      return;
    }

    const projection = await fetchJSON(manifest.projection_ref);
    if (!validProjection(projection)) throw new Error(tr("projection_invalid"));
    if (projection.availability !== "ready") {
      setStatus(status, projection.availability, tr("record_state", { state: projection.availability }));
      text(byId("truth"), projection.summary);
      setUnavailableValidity(projection.availability);
      byId("evidence").hidden = true;
      return;
    }

    const asset = projection.assets?.find((item) => item.modality === "screenshot");
    const source = safeRef(asset?.ref);
    if (!source || source !== safeRef(manifest.screenshot_ref) || asset.sha256 !== manifest.screenshot_sha256 || !validSHA256(asset.sha256)) {
      throw new Error(tr("screenshot_corrupt"));
    }
    if (!validSHA256(projection.artifact.manifest_sha256) || !validSHA256(projection.artifact.bundle_sha256)) {
      throw new Error(tr("integrity_corrupt"));
    }

    text(byId("title"), projection.title || tr("evidence_record"));
    text(byId("truth"), projection.summary);
    text(byId("record-id"), projection.artifact.artifact_ref);
    text(byId("record-revision"), projection.artifact.revision);
    renderLineage({ ...projection.artifact.scope, work_items: projection.work_items });

    const image = byId("screenshot");
    image.src = source;
    byId("image-link").href = source;
    text(byId("source-label"), manifest.source_url || tr("source_not_disclosed"));
    text(byId("capture-label"), `${manifest.width} × ${manifest.height} · ${manifest.mime}`);
    text(byId("caption"), `${tr("captured")} ${formatTime(manifest.captured_at)} · ${formatBytes(asset.bytes)} · SHA-256 ${asset.sha256}`);

    const claimCount = projection.claims?.length || 0;
    const assetCount = projection.assets?.length || 0;
    setValidity("integrity", tr("digest_bound"), "recorded");
    setValidity("provenance", `${tr("recorded")} · ${projection.federation_posture}`, "recorded");
    setValidity("observation", projection.claims?.[0]?.posture || tr("recorded"), "recorded");
    setValidity("sufficiency", tr("limited_counts", { claims: locale.number(claimCount), assets: locale.number(assetCount) }), "limited");
    setValidity("verification", tr("not_asserted"), "not_determined");
    setValidity("completion", tr("not_asserted"), "not_determined");
    setValidity("settlement", tr("not_asserted"), "not_determined");
    setValidity("legal", tr("not_determined"), "not_determined");

    byId("facts").replaceChildren(
      fact(tr("captured"), formatTime(manifest.captured_at)),
      fact(tr("freshness_observed"), formatTime(projection.freshness_observed_at)),
      fact(tr("target"), manifest.source_url || tr("source_not_disclosed")),
      fact(tr("viewport"), `${manifest.width} × ${manifest.height}`),
      fact(tr("access"), projection.access),
      fact(tr("redaction"), projection.redaction?.state || "unknown"),
      fact(tr("interaction"), projection.interaction),
      fact(tr("inspection_receipt"), manifest.inspection_ref || tr("unavailable_value")),
    );

    text(byId("limitations-copy"), tr("record_limitations"));
    renderTimeline(projection.timeline);

    const scope = { ...projection.artifact.scope, work_items: projection.work_items };
    byId("inspect-grid").replaceChildren(
      datum(tr("artifact"), projection.artifact.artifact_ref),
      datum(tr("revision"), projection.artifact.revision),
      datum("Manifest SHA-256", projection.artifact.manifest_sha256),
      datum("Bundle SHA-256", projection.artifact.bundle_sha256),
      datum("Screenshot SHA-256", asset.sha256),
      datum("Inspection", manifest.inspection_ref),
      datum(tr("project_label"), scope.project_ref),
      datum(tr("workstream"), scope.workstream_ref),
      datum(tr("workset"), scope.workset_ref),
      datum(tr("callgraph"), scope.callgraph_ref),
      datum(tr("workpoint"), scope.workpoint_ref),
      ...workItemInspectData(scope),
      datum(tr("provenance"), projection.federation_posture),
      datum(tr("redaction"), projection.redaction.state),
    );
    setReadyStatus(status, tr("record_loaded"));
    byId("title").focus({ preventScroll: true });
  } catch (error) {
    setStatus(status, "corrupt", tr("record_corrupt"));
    text(byId("truth"), error instanceof Error ? error.message : tr("record_corrupt"));
    setUnavailableValidity("corrupt");
    byId("evidence").hidden = true;
  }
}

wireRegistryControls();
const route = new URL(location.href);
const defaultView = document.body.dataset.defaultView || "registry";
if (route.searchParams.get("artifact")) {
  byId("registry").hidden = true;
  byId("record-detail").hidden = false;
  renderPublicRecord();
} else if (route.searchParams.get("view") === "record" || defaultView === "record") {
  byId("registry").hidden = true;
  byId("record-detail").hidden = false;
  if (defaultView === "record") byId("registry-back").hidden = true;
  renderRecord();
} else {
  renderRegistry().catch(showRegistryUnavailable);
}
