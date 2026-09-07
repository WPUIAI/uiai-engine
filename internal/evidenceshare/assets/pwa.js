"use strict";

async function registerEvidenceWorker() {
  if (location.protocol === "file:") {
    document.body.dataset.pwaStatus = "offline_file";
    return;
  }
  if (navigator.onLine === false) {
    document.body.dataset.pwaStatus = "offline";
    return;
  }
  if (!isSecureContext || !("serviceWorker" in navigator)) {
    document.body.dataset.pwaStatus = "unsupported";
    return;
  }
  try {
    const version = document.body.dataset.assetVersion;
    const worker = `./sw.js?v=${encodeURIComponent(version)}`;
    const registration = await navigator.serviceWorker.register(worker, { scope: "./", updateViaCache: "none" });
    document.body.dataset.pwaStatus = "ready";
    registration.update().catch(() => { document.body.dataset.pwaStatus = "degraded"; });
    navigator.serviceWorker.addEventListener("controllerchange", () => { document.body.dataset.pwaStatus = "updated"; }, { once: true });
  } catch {
    document.body.dataset.pwaStatus = "degraded";
  }
}

registerEvidenceWorker();
addEventListener("online", registerEvidenceWorker);
