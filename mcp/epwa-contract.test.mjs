import assert from "node:assert/strict";
import test from "node:test";

import { evidenceScopeHeaders, findRawArtifactField } from "./epwa-contract.mjs";

test("evidenceScopeHeaders forwards complete scope without inventing values", () => {
  const workItems = [{ work_item_ref: "work-item:196", revision: "4" }];
  assert.deepEqual(
    evidenceScopeHeaders({
      project_ref: "project:uiai-engine",
      workstream: "workstream:epwa",
      workset_ref: "workset:completion",
      callgraph: "callgraph:106q",
      workpoint_id: "workpoint:196",
      work_item_ref: "work-item:196",
      continuity_id: "epwa-callgraph-full-closure",
      work_items: workItems,
    }),
    {
      "X-UIAI-Project-Ref": "project:uiai-engine",
      "X-UIAI-Workstream-Ref": "workstream:epwa",
      "X-UIAI-Workset-Ref": "workset:completion",
      "X-UIAI-CallGraph-Ref": "callgraph:106q",
      "X-UIAI-Workpoint-Ref": "workpoint:196",
      "X-UIAI-Work-Item-Ref": "work-item:196",
      "X-UIAI-Continuity-Ref": "epwa-callgraph-full-closure",
      "X-UIAI-Work-Items": JSON.stringify(workItems),
    },
  );
  assert.deepEqual(evidenceScopeHeaders(undefined), {});
});

test("findRawArtifactField rejects nested legacy screenshot and path fields", () => {
  assert.equal(findRawArtifactField({ delivery_state: "ready", epwa_delivery: {} }), "");
  assert.equal(findRawArtifactField({ session: { screenshot: "base64-data" } }), "$.session.screenshot");
  assert.equal(findRawArtifactField({ results: [{ artifact_path: "/tmp/report.json" }] }), "$.results[0].artifact_path");
  assert.equal(findRawArtifactField({ imageBase64: "bytes" }), "$.imageBase64");
});
