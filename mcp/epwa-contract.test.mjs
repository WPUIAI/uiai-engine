import assert from "node:assert/strict";
import test from "node:test";

import { assertEvidenceDelivery, evidenceScopeHeaders, findNonReadyArtifactDelivery, findRawArtifactField } from "./epwa-contract.mjs";

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
  assert.equal(findRawArtifactField({ results: [{ delivery: { result_path: "/tmp/report.json" } }] }), "$.results[0].delivery.result_path");
  assert.equal(findRawArtifactField({ result: { result_url: "http://localhost/raw" } }), "$.result.result_url");
  assert.equal(findRawArtifactField({ imageBase64: "bytes" }), "$.imageBase64");
});

test("findNonReadyArtifactDelivery rejects pending and malformed envelopes recursively", () => {
  assert.equal(findNonReadyArtifactDelivery({ delivery_state: "ready", epwa_delivery: { state: "ready" } }), "");
  assert.equal(findNonReadyArtifactDelivery({ delivery_state: "pending_reconcile", epwa_delivery: { state: "pending_reconcile" } }), "$.delivery_state");
  assert.equal(findNonReadyArtifactDelivery({ results: [{ delivery_state: "ready" }] }), "$.results[0].delivery_state");
  assert.equal(findNonReadyArtifactDelivery({ nested: { epwa_delivery: { state: "blocked" } } }), "$.nested.delivery_state");
  assert.equal(findNonReadyArtifactDelivery({ epwa_delivery_error: { state: "pending_reconcile" } }), "$.epwa_delivery_error");
  assert.equal(findNonReadyArtifactDelivery({ schema: "uiai.epwa_delivery.v1", state: "blocked" }), "$.state");
});

const ready = (origin = "https://evidence.example/nested/") => ({
  artifact_url: origin + "record/", portable_url: origin + "record/portable.zip",
  epwa_delivery: { schema: "uiai.epwa_delivery.v1", state: "ready",
    artifact: { artifact_ref: "artifact:fixture" },
    epwa: { record_url: origin + "record/", portable_url: origin + "record/portable.zip" } },
});
test("shared adapter yields ready evidence at any configured HTTPS origin/subpath", () => {
  for (const base of ["https://one.example/", "https://two.example/client/deep/"]) {
    assert.deepEqual(assertEvidenceDelivery(ready(base), true), {
      recordURL: base + "record/", portableURL: base + "record/portable.zip",
    });
  }
});
test("required delivery cannot silently become metadata or an ephemeral preview", () => {
  for (const body of [{ width: 1280 }, { fpv_share: { public_url: "https://preview.example/m/temp" } },
    { screenshot: "private-bytes" }, { epwa_delivery: { state: "ready" } }]) {
    assert.throws(() => assertEvidenceDelivery(body, true), /EPWA delivery unavailable/);
  }
  assert.equal(assertEvidenceDelivery({ status: "healthy" }), null);
});
test("shared boundary rejects unsafe or conflicting delivery URLs", () => {
  for (const url of ["http://evidence.example/record/", "https://user:secret@evidence.example/record/",
    "https://evidence.example/record/#fragment"]) {
    const data = ready(); data.epwa_delivery.epwa.record_url = url;
    assert.throws(() => assertEvidenceDelivery(data, true));
  }
  const mismatch = ready(); mismatch.artifact_url = "https://other.example/record/";
  assert.throws(() => assertEvidenceDelivery(mismatch, true), /disagrees/);
  const invalid = ready(); invalid.epwa_delivery.epwa.portable_url = "https://other.example/not-a-zip";
  assert.throws(() => assertEvidenceDelivery(invalid, true), /not a ZIP/);
});
test("pending publication retains reconciliation without returning raw evidence", () => {
  assert.throws(() => assertEvidenceDelivery({ schema: "uiai.epwa_delivery_error.v1",
    state: "pending_reconcile", recovery_ref: "reconcile:fixture" }, true), /reconcile:fixture/);
  assert.throws(() => assertEvidenceDelivery({ screenshot: "private-bytes" }, true),
    error => error.message.includes("$.screenshot") && !error.message.includes("private-bytes"));
});

test("failed session publication retains a cleanup handle without raw pixels", () => {
  assert.throws(() => assertEvidenceDelivery({ session: { id: "created-session" }, screenshot: "private-bytes" }),
    error => error.message.includes('session_id="created-session"') && !error.message.includes("private-bytes"));
});

test("nested terminal evidence is validated and surfaced, never treated as dispatch completion", () => {
  assert.equal(assertEvidenceDelivery({ job_id: "job:test", status: "pending" }), null);
  assert.equal(assertEvidenceDelivery({ job: { result: ready() } }).recordURL, ready().artifact_url);
  const bad = ready(); bad.epwa_delivery.epwa.record_url = "http://unsafe.example/record/";
  assert.throws(() => assertEvidenceDelivery({ jobs: [ready(), bad] }));
});

test("consumer preserves the producer's portable-origin and query contract", () => {
  const data = ready();
  data.artifact_url = data.epwa_delivery.epwa.record_url = "https://one.example/record";
  data.portable_url = data.epwa_delivery.epwa.portable_url = "https://other.example/download.zip?revision=1";
  assert.deepEqual(assertEvidenceDelivery(data, true), { recordURL: data.artifact_url, portableURL: data.portable_url });
});
