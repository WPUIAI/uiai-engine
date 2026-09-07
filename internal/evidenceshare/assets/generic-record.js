"use strict";

window.renderGenericEvidenceRecord = async (manifest) => {
  if (manifest?.schema === "uiai.epwa_generic_artifact.v1") return renderPortableArtifactRecord(manifest);
  if (manifest?.schema !== "uiai.evidence_artifact_manifest.v1") throw new Error("Artifact manifest contract is unsupported");
  const scope = manifest.scope || {};
  const flatScope = {
    project_ref: scope.project?.project_ref,
    workstream_ref: scope.workstream?.workstream_ref,
    workset_ref: scope.workset?.workset_ref,
    callgraph_ref: scope.callgraph?.frame_ref || scope.callgraph?.run_ref,
    workpoint_ref: scope.workpoint?.workpoint_ref,
    work_item_ref: scope.work_items?.[0]?.work_item_ref,
    work_items: (scope.work_items || []).map((item) => ({
      ...item,
      description_state: item.description ? "visible" : "unavailable",
      revision_state: "current",
      authority: {
        acceptance_atom_refs: item.acceptance_atom_refs || [],
        evidence_requirement_refs: item.evidence_requirement_refs || [],
        review_requirement_refs: item.review_requirement_refs || [],
      },
    })),
  };
  text(byId("title"), manifest.title || "Evidence record");
  text(byId("truth"), manifest.summary || "Bound immutable evidence artifact.");
  text(byId("record-id"), manifest.artifact_id);
  text(byId("record-revision"), manifest.revision);
  renderLineage(flatScope);

  const assets = Array.isArray(manifest.assets) ? manifest.assets : [];
  const primary = assets.find((asset) => String(asset.media_type || "").startsWith("image/"));
  const frame = byId("primary-evidence-frame");
  const source = primary ? safeRef(`./${primary.path}`) : null;
  if (primary && source && validSHA256(primary.sha256)) {
    frame.hidden = false;
    byId("screenshot").src = source;
    byId("screenshot").alt = primary.alt_text || `Evidence asset ${primary.asset_id}`;
    byId("image-link").href = source;
    text(byId("source-label"), primary.source_ref || "Source not disclosed");
    text(byId("capture-label"), `${primary.width || "—"} × ${primary.height || "—"} · ${primary.media_type}`);
    text(byId("caption"), `Captured ${formatTime(primary.captured_at || manifest.captured_at)} · ${formatBytes(primary.byte_size)} · SHA-256 ${primary.sha256}`);
  } else {
    frame.hidden = true;
  }

  const claims = Array.isArray(manifest.claims) ? manifest.claims : [];
  setValidity("integrity", manifest.integrity?.manifest_sha256 ? "Digest-bound" : "Digest recorded by commit", "recorded");
  setValidity("provenance", `${(manifest.provenance?.custody || []).length} custody event(s)`, "recorded");
  setValidity("observation", claims[0]?.status || "Recorded", "recorded");
  setValidity("sufficiency", `Limited · ${claims.length} claim / ${assets.length} asset`, "limited");
  setValidity("verification", manifest.verification?.status || "Not determined", manifest.verification?.status || "not_determined");
  setValidity("completion", "Not asserted by this artifact", "not_determined");
  setValidity("settlement", "Not asserted by this artifact", "not_determined");
  setValidity("legal", "Not determined", "not_determined");
  byId("facts").replaceChildren(
    fact("Captured", formatTime(manifest.captured_at)), fact("Created", formatTime(manifest.created_at)),
    fact("Evidence assets", assets.length), fact("Claims", claims.length), fact("Access", manifest.policy?.access_class),
    fact("Redaction", manifest.policy?.redaction_state), fact("Authority posture", manifest.authority?.posture),
    fact("Retention", manifest.policy?.retention_class),
  );
  renderTimeline((manifest.provenance?.custody || []).map((event) => ({ event_type: event.action, occurred_at: event.occurred_at })));
  byId("inspect-grid").replaceChildren(
    datum("Artifact", manifest.artifact_id), datum("Revision", manifest.revision),
    datum("Manifest SHA-256", manifest.integrity?.manifest_sha256), datum("Bundle SHA-256", manifest.integrity?.bundle_sha256),
    datum("Project", flatScope.project_ref), datum("Workstream", flatScope.workstream_ref), datum("Workset", flatScope.workset_ref),
    datum("CallGraph", flatScope.callgraph_ref), datum("Workpoint", flatScope.workpoint_ref),
    ...workItemInspectData(flatScope), datum("Evidence authority", manifest.authority?.evidence_authority_ref),
    datum("Completion authority", manifest.authority?.completion_authority_ref), datum("Verification", manifest.verification?.status),
  );
  byId("detail-json-link").href = "./artifact.json";
  byId("manifest-json-link").href = "./artifact.json";
  byId("inspection-json-link").href = "./projection.json";
  text(byId("limitations-copy"), "This immutable artifact is a bounded evidence input. Its existence and delivery do not establish review, verification, task completion, provider closure, settlement, or legal admissibility.");
  setReadyStatus(byId("status"), "Read-only artifact loaded");
  byId("title").focus({ preventScroll: true });
};

const renderPortableArtifactRecord = async (manifest) => {
  const scope = manifest.scope || {};
  text(byId("title"), manifest.title || "Portable artifact record");
  text(byId("truth"), manifest.truth_notice || "Bound portable artifact.");
  text(byId("record-id"), manifest.artifact_ref);
  text(byId("record-revision"), manifest.revision);
  renderLineage(scope);
  const frame = byId("primary-evidence-frame");
  const source = safeRef(manifest.asset_ref);
  const isImage = String(manifest.media_type || "").startsWith("image/");
  if (source && isImage && validSHA256(manifest.asset_sha256)) {
    frame.hidden = false;
    byId("screenshot").src = source;
    byId("screenshot").alt = manifest.title || "Artifact preview";
    byId("image-link").href = source;
    text(byId("source-label"), manifest.source_ref || "Source not disclosed");
    text(byId("capture-label"), manifest.media_type);
    text(byId("caption"), `Captured ${formatTime(manifest.captured_at)} · ${formatBytes(manifest.bytes)} · SHA-256 ${manifest.asset_sha256}`);
  } else {
    frame.hidden = true;
  }
  setValidity("integrity", validSHA256(manifest.asset_sha256) ? "Digest-bound" : "Invalid", validSHA256(manifest.asset_sha256) ? "recorded" : "invalid");
  setValidity("provenance", manifest.source_ref || "Source not disclosed", "recorded");
  setValidity("observation", manifest.kind || "Artifact", "recorded");
  setValidity("sufficiency", "Bound bytes only", "limited");
  setValidity("verification", "Not determined", "not_determined");
  setValidity("completion", "Not asserted by this artifact", "not_determined");
  setValidity("settlement", "Not asserted by this artifact", "not_determined");
  setValidity("legal", "Not determined", "not_determined");
  byId("facts").replaceChildren(
    fact("Captured", formatTime(manifest.captured_at)), fact("Kind", manifest.kind), fact("Media type", manifest.media_type),
    fact("Bytes", formatBytes(manifest.bytes)), fact("Availability", manifest.availability), fact("Access", manifest.access),
  );
  byId("inspect-grid").replaceChildren(
    datum("Artifact", manifest.artifact_ref), datum("Revision", manifest.revision), datum("Payload SHA-256", manifest.asset_sha256),
    datum("Project", scope.project_ref), datum("Workstream", scope.workstream_ref), datum("Workset", scope.workset_ref),
    datum("CallGraph", scope.callgraph_ref), datum("Workpoint", scope.workpoint_ref), ...workItemInspectData(scope),
    datum("Parent artifact", manifest.parent_artifact_ref || "None"), datum("Child artifacts", (manifest.child_artifact_refs || []).join(", ") || "None"),
  );
  byId("detail-json-link").href = source || "./artifact.json";
  byId("manifest-json-link").href = "./artifact.json";
  byId("inspection-json-link").href = "./projection.json";
  text(byId("limitations-copy"), manifest.truth_notice);
  setReadyStatus(byId("status"), "Read-only artifact loaded");
  byId("title").focus({ preventScroll: true });
};
