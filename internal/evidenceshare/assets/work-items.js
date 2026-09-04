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
  const descriptor = item.title || item.work_item_ref || "Work item unavailable";
  const identity = item.item_id || item.work_item_ref || "unknown";
  const revision = item.revision || "revision unavailable";
  const state = item.revision_state || "unknown";
  const status = item.status_at_capture || item.status || "status unavailable";
  return `${descriptor} · ${identity} · ${revision} · ${state} · ${status}`;
}

function workItemRefs(values) {
  return Array.isArray(values) && values.length > 0 ? values.join(", ") : "None";
}

function workItemInspectData(scope) {
  return scopeWorkItems(scope).flatMap((item, index) => {
    const authority = item.authority || {};
    const position = index + 1;
    return [
      datum(`Work item ${position} · ${item.item_type || "item"}`, workItemSummary(item)),
      datum(`Work item ${position} · description`, `${item.description_state || "unavailable"}${item.description ? ` · ${item.description}` : ""}`),
      datum(`Work item ${position} · relationships`, `parents: ${workItemRefs(item.parent_refs)} · dependencies: ${workItemRefs(item.dependency_refs)} · blockers: ${workItemRefs(item.blocker_refs)}`),
      datum(`Work item ${position} · requirements`, `acceptance: ${workItemRefs(authority.acceptance_atom_refs)} · evidence: ${workItemRefs(authority.evidence_requirement_refs)} · review: ${workItemRefs(authority.review_requirement_refs)}`),
      datum(`Work item ${position} · closure`, `posture: ${item.closure_posture || "unknown"} · case: ${authority.completion_case_ref || "none"} · decision: ${authority.completion_decision_ref || "none"} · provider close: ${authority.provider_close_receipt_ref || "none"} · reopen: ${authority.reopen_ref || "none"} · settlement: ${authority.settlement_posture || "unknown"}`),
    ];
  });
}

function renderLineage(scope) {
  const lineage = [
    lineageItem("Project", scope.project_ref),
    lineageItem("Workstream", scope.workstream_ref),
    lineageItem("Workset", scope.workset_ref),
    lineageItem("CallGraph", scope.callgraph_ref),
    lineageItem("Workpoint", scope.workpoint_ref),
  ];
  scopeWorkItems(scope).forEach((item) => lineage.push(lineageItem(item.item_type || "Work item", workItemSummary(item))));
  byId("lineage-list").replaceChildren(...lineage);
}
