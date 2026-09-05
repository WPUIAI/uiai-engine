import { describe, expect, it } from "vitest";
import {
  EngineDeliveryError,
  evidenceScopeHeaders,
  validateArtifactDelivery,
  type ArtifactDeliveryResult,
} from "../../src/lib/engine-client";

function deliveryResult(overrides: Partial<ArtifactDeliveryResult> = {}): ArtifactDeliveryResult {
  const artifactUrl = "https://evidence.example/api/screenshot/share/package/";
  const portableUrl = `${artifactUrl}portable.zip`;
  const base: ArtifactDeliveryResult = {
    schema: "uiai.artifact_result.v2",
    artifact_ref: "artifact:report",
    delivery_state: "ready",
    artifact_url: artifactUrl,
    portable_url: portableUrl,
    epwa_delivery: {
      schema: "uiai.epwa_delivery.v1",
      delivery_id: "uiai-epwa-delivery:sha256:" + "d".repeat(64),
      revision: 1,
      artifact: { artifact_ref: "artifact:report", revision: 1, manifest_sha256: "a".repeat(64), output_sha256: "b".repeat(64) },
      epwa: { package_id: "c".repeat(64), package_ref: "uiai-epwa-package:sha256:" + "c".repeat(64), package_sha256: "c".repeat(64), record_url: artifactUrl, portable_url: portableUrl, access: "public_safe" },
      scope: { posture: "complete" },
      state: "ready",
    },
  };
  return { ...base, ...overrides };
}

describe("Cockpit EPWA delivery consumer", () => {
  it("forwards the full evidence scope without inventing absent values", () => {
    const workItems = [{ work_item_ref: "work-item:196", revision: "1" }];
    const headers = evidenceScopeHeaders({
      project_ref: "project:uiai-engine", workstream_ref: "workstream:epwa", workset_ref: "workset:completion",
      callgraph_ref: "callgraph:106q", workpoint_ref: "workpoint:196", work_item_ref: "work-item:196",
      continuity_ref: "epwa-callgraph-full-closure", work_items: workItems,
    });
    expect(headers.get("X-UIAI-Project-Ref")).toBe("project:uiai-engine");
    expect(headers.get("X-UIAI-Workstream-Ref")).toBe("workstream:epwa");
    expect(headers.get("X-UIAI-Workset-Ref")).toBe("workset:completion");
    expect(headers.get("X-UIAI-CallGraph-Ref")).toBe("callgraph:106q");
    expect(headers.get("X-UIAI-Workpoint-Ref")).toBe("workpoint:196");
    expect(headers.get("X-UIAI-Work-Item-Ref")).toBe("work-item:196");
    expect(headers.get("X-UIAI-Continuity-Ref")).toBe("epwa-callgraph-full-closure");
    expect(headers.get("X-UIAI-Work-Items")).toBe(JSON.stringify(workItems));
  });

  it("accepts only one identity-bound ready HTTPS viewer and portable package", () => {
    const result = deliveryResult();
    expect(validateArtifactDelivery(result)).toBe(result);
  });

  it("fails closed for pending delivery, raw fields, identity drift, and non-HTTPS links", () => {
    const pending = deliveryResult({ delivery_state: "pending_reconcile" });
    pending.epwa_delivery = { ...pending.epwa_delivery, state: "pending_reconcile", recovery_ref: "reconcile:epwa" };
    expect(() => validateArtifactDelivery(pending)).toThrow(EngineDeliveryError);

    expect(() => validateArtifactDelivery({ ...deliveryResult(), screenshot: "base64" })).toThrow(/forbidden raw artifact/i);

    const drifted = deliveryResult();
    drifted.epwa_delivery = { ...drifted.epwa_delivery, artifact: { ...drifted.epwa_delivery.artifact, artifact_ref: "artifact:other" } };
    expect(() => validateArtifactDelivery(drifted)).toThrow(/invalid EPWA delivery binding/i);

    expect(() => validateArtifactDelivery({ ...deliveryResult(), artifact_url: "http://evidence.example/raw" })).toThrow(/HTTPS EPWA URLs/i);
  });
});
