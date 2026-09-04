"use strict";

function normalizeWorkItem(item) {
  const authority = item.authority || {
    acceptance_atom_refs: item.acceptance_atom_refs,
    evidence_requirement_refs: item.evidence_requirement_refs,
    review_requirement_refs: item.review_requirement_refs,
  };
  const descriptionState = item.description_state || (item.description ? "visible" : item.description_ref ? "redacted" : "unavailable");
  return { ...item, authority, description_state: descriptionState, revision_state: item.revision_state || "unknown" };
}

function scopeWorkItems(scope) {
  if (Array.isArray(scope?.work_items) && scope.work_items.length > 0) return scope.work_items.map(normalizeWorkItem);
  return scope?.work_item_ref ? [normalizeWorkItem({ work_item_ref: scope.work_item_ref, item_type: "work_item", revision_state: "unknown" })] : [];
}

function workItemSummary(item) {
  const descriptor = item.title || item.work_item_ref || tr("work_item_unavailable");
  const identity = item.item_id || item.work_item_ref || "unknown";
  const revision = item.revision || tr("revision_unavailable");
  const state = item.revision_state || "unknown";
  const status = item.status_at_capture || item.status || tr("status_unavailable");
  return `${descriptor} · ${identity} · ${revision} · ${state} · ${status}`;
}

function workItemRefs(values) {
  return Array.isArray(values) && values.length > 0 ? values.join(", ") : tr("none");
}

function workItemInspectData(scope) {
  return scopeWorkItems(scope).flatMap((item, index) => {
    const authority = item.authority || {};
    const position = locale.number(index + 1);
    const label = `${tr("kind_work_item")} ${position}`;
    return [
      datum(`${label} · ${item.item_type || tr("item")}`, workItemSummary(item)),
      datum(`${label} · ${tr("description")}`, `${item.description_state || "unavailable"}${item.description ? ` · ${item.description}` : ""}`),
      datum(`${label} · ${tr("relationships_label")}`, `${tr("parents")}: ${workItemRefs(item.parent_refs)} · ${tr("dependencies")}: ${workItemRefs(item.dependency_refs)} · ${tr("blockers")}: ${workItemRefs(item.blocker_refs)}`),
      datum(`${label} · ${tr("requirements")}`, `${tr("acceptance")}: ${workItemRefs(authority.acceptance_atom_refs)} · ${tr("evidence")}: ${workItemRefs(authority.evidence_requirement_refs)} · ${tr("review")}: ${workItemRefs(authority.review_requirement_refs)}`),
      datum(`${label} · ${tr("closure")}`, `${tr("record_posture")}: ${item.closure_posture || "unknown"} · ${tr("case_label")}: ${authority.completion_case_ref || tr("none")} · ${tr("decision")}: ${authority.completion_decision_ref || tr("none")} · ${tr("provider_close")}: ${authority.provider_close_receipt_ref || tr("none")} · ${tr("reopen")}: ${authority.reopen_ref || tr("none")} · ${tr("settlement")}: ${authority.settlement_posture || "unknown"}`),
    ];
  });
}

function renderLineage(scope) {
  const lineage = [
    lineageItem(tr("project_label"), scope.project_ref),
    lineageItem(tr("workstream"), scope.workstream_ref),
    lineageItem(tr("workset"), scope.workset_ref),
    lineageItem(tr("callgraph"), scope.callgraph_ref),
    lineageItem(tr("workpoint"), scope.workpoint_ref),
  ];
  scopeWorkItems(scope).forEach((item) => lineage.push(lineageItem(item.item_type || tr("kind_work_item"), workItemSummary(item))));
  byId("lineage-list").replaceChildren(...lineage);
}
