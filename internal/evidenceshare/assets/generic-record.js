"use strict";

window.renderGenericEvidenceRecord = async (manifest) => {
  if (manifest?.schema === "uiai.epwa_generic_artifact.v1") return renderPortableArtifactRecord(manifest);
  if (manifest?.schema !== "uiai.evidence_artifact_manifest.v1") throw new Error(tr("artifact_contract_invalid"));
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
  text(byId("title"), manifest.title || tr("evidence_record"));
  text(byId("truth"), manifest.summary || tr("bound_immutable_summary"));
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
    byId("screenshot").alt = primary.alt_text || tr("evidence_asset", { id: primary.asset_id });
    byId("image-link").href = source;
    text(byId("source-label"), primary.source_ref || tr("source_not_disclosed"));
    text(byId("capture-label"), `${primary.width ? locale.number(primary.width) : "—"} × ${primary.height ? locale.number(primary.height) : "—"} · ${primary.media_type}`);
    text(byId("caption"), `${tr("captured")} ${formatTime(primary.captured_at || manifest.captured_at)} · ${formatBytes(primary.byte_size)} · SHA-256 ${primary.sha256}`);
  } else {
    frame.hidden = true;
  }

  const claims = Array.isArray(manifest.claims) ? manifest.claims : [];
  setValidity("integrity", manifest.integrity?.manifest_sha256 ? tr("digest_bound") : tr("digest_recorded_commit"), "recorded");
  setValidity("provenance", tr("custody_events", { count: locale.number((manifest.provenance?.custody || []).length) }), "recorded");
  setValidity("observation", claims[0]?.status || tr("recorded"), "recorded");
  setValidity("sufficiency", tr("limited_counts", { claims: locale.number(claims.length), assets: locale.number(assets.length) }), "limited");
  setValidity("verification", manifest.verification?.status || tr("not_determined"), manifest.verification?.status || "not_determined");
  setValidity("completion", tr("not_asserted"), "not_determined");
  setValidity("settlement", tr("not_asserted"), "not_determined");
  setValidity("legal", tr("not_determined"), "not_determined");
  byId("facts").replaceChildren(
    fact(tr("captured"), formatTime(manifest.captured_at)), fact(tr("created"), formatTime(manifest.created_at)),
    fact(tr("evidence_assets"), locale.number(assets.length)), fact(tr("claims"), locale.number(claims.length)), fact(tr("access"), manifest.policy?.access_class),
    fact(tr("redaction"), manifest.policy?.redaction_state), fact(tr("authority_posture"), manifest.authority?.posture),
    fact(tr("retention"), manifest.policy?.retention_class),
  );
  renderTimeline((manifest.provenance?.custody || []).map((event) => ({ event_type: event.action, occurred_at: event.occurred_at })));
  byId("inspect-grid").replaceChildren(
    datum(tr("artifact"), manifest.artifact_id), datum(tr("revision"), manifest.revision),
    datum(tr("manifest_digest"), manifest.integrity?.manifest_sha256), datum(tr("bundle_digest"), manifest.integrity?.bundle_sha256),
    datum(tr("project"), flatScope.project_ref), datum(tr("workstream"), flatScope.workstream_ref), datum(tr("workset"), flatScope.workset_ref),
    datum(tr("callgraph"), flatScope.callgraph_ref), datum(tr("workpoint"), flatScope.workpoint_ref),
    ...workItemInspectData(flatScope), datum(tr("evidence_authority"), manifest.authority?.evidence_authority_ref),
    datum(tr("completion_authority"), manifest.authority?.completion_authority_ref), datum(tr("verification"), manifest.verification?.status),
  );
  byId("detail-json-link").href = "./artifact.json";
  byId("manifest-json-link").href = "./artifact.json";
  byId("inspection-json-link").href = "./projection.json";
  text(byId("limitations-copy"), tr("artifact_limitations"));
  setReadyStatus(byId("status"), tr("artifact_loaded"));
  byId("title").focus({ preventScroll: true });
};

const renderPortableArtifactRecord = async (manifest) => {
  const scope = manifest.scope || {};
  text(byId("title"), manifest.title || tr("portable_artifact_record"));
  text(byId("truth"), manifest.truth_notice || tr("bound_portable_summary"));
  text(byId("record-id"), manifest.artifact_ref);
  text(byId("record-revision"), manifest.revision);
  renderLineage(scope);
  const frame = byId("primary-evidence-frame");
  const source = safeRef(manifest.asset_ref);
  const isImage = String(manifest.media_type || "").startsWith("image/");
  if (source && isImage && validSHA256(manifest.asset_sha256)) {
    frame.hidden = false;
    byId("screenshot").src = source;
    byId("screenshot").alt = manifest.title || tr("artifact_preview");
    byId("image-link").href = source;
    text(byId("source-label"), manifest.source_ref || tr("source_not_disclosed"));
    text(byId("capture-label"), manifest.media_type);
    text(byId("caption"), `${tr("captured")} ${formatTime(manifest.captured_at)} · ${formatBytes(manifest.bytes)} · SHA-256 ${manifest.asset_sha256}`);
  } else {
    frame.hidden = true;
  }
  setValidity("integrity", validSHA256(manifest.asset_sha256) ? tr("digest_bound") : tr("invalid"), validSHA256(manifest.asset_sha256) ? "recorded" : "invalid");
  setValidity("provenance", manifest.source_ref || tr("source_not_disclosed"), "recorded");
  setValidity("observation", manifest.kind || tr("artifact"), "recorded");
  setValidity("sufficiency", tr("bound_bytes_only"), "limited");
  setValidity("verification", tr("not_determined"), "not_determined");
  setValidity("completion", tr("not_asserted"), "not_determined");
  setValidity("settlement", tr("not_asserted"), "not_determined");
  setValidity("legal", tr("not_determined"), "not_determined");
  byId("facts").replaceChildren(
    fact(tr("captured"), formatTime(manifest.captured_at)), fact(tr("kind"), manifest.kind), fact(tr("media_type"), manifest.media_type),
    fact(tr("bytes"), formatBytes(manifest.bytes)), fact(tr("availability"), manifest.availability), fact(tr("access"), manifest.access),
  );
  byId("inspect-grid").replaceChildren(
    datum(tr("artifact"), manifest.artifact_ref), datum(tr("revision"), manifest.revision), datum(tr("payload_digest"), manifest.asset_sha256),
    datum(tr("project"), scope.project_ref), datum(tr("workstream"), scope.workstream_ref), datum(tr("workset"), scope.workset_ref),
    datum(tr("callgraph"), scope.callgraph_ref), datum(tr("workpoint"), scope.workpoint_ref), ...workItemInspectData(scope),
    datum(tr("parent_artifact"), manifest.parent_artifact_ref || tr("none")), datum(tr("child_artifacts"), (manifest.child_artifact_refs || []).join(", ") || tr("none")),
  );
  byId("detail-json-link").href = source || "./artifact.json";
  byId("manifest-json-link").href = "./artifact.json";
  byId("inspection-json-link").href = "./projection.json";
  text(byId("limitations-copy"), manifest.truth_notice || tr("artifact_limitations"));
  setReadyStatus(byId("status"), tr("artifact_loaded"));
  byId("title").focus({ preventScroll: true });
};
