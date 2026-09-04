"use strict";

const VERSION = "__UIAI_ASSET_VERSION__";
const SCOPE_URL = new URL(self.registration.scope);
const CACHE_PREFIX = `uiai-evidence-shell:${SCOPE_URL.pathname}:`;
const CACHE_NAME = `${CACHE_PREFIX}${VERSION}`;
const SHELL_ASSETS = [
  "./",
  "./index.html",
  `./styles.css?v=${VERSION}`,
  `./work-items.js?v=${VERSION}`,
  `./pwa.js?v=${VERSION}`,
  `./app.js?v=${VERSION}`,
  `./manifest.webmanifest?v=${VERSION}`,
  `./icon.svg?v=${VERSION}`,
];
const OPTIONAL_RECORD_ASSETS = ["./artifact.json", "./projection.json", "./inspection.json"];

function requestIsCacheable(request) {
  if (request.method !== "GET") return false;
  const url = new URL(request.url);
  return url.origin === SCOPE_URL.origin && url.pathname.startsWith(SCOPE_URL.pathname);
}

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then(async (cache) => {
        await cache.addAll(SHELL_ASSETS);
        await Promise.allSettled(OPTIONAL_RECORD_ASSETS.map((asset) => cache.add(asset)));
      })
      .then(() => self.skipWaiting()),
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches.keys()
      .then((names) => Promise.all(names.filter((name) => name.startsWith(CACHE_PREFIX) && name !== CACHE_NAME).map((name) => caches.delete(name))))
      .then(() => self.clients.claim()),
  );
});

self.addEventListener("fetch", (event) => {
  if (!requestIsCacheable(event.request)) return;
  event.respondWith((async () => {
    const cache = await caches.open(CACHE_NAME);
    const cached = await cache.match(event.request);
    if (cached) return cached;
    try {
      const response = await fetch(event.request);
      if (response.ok && response.type !== "opaque") await cache.put(event.request, response.clone());
      return response;
    } catch (error) {
      if (event.request.mode === "navigate") {
        const fallback = await cache.match("./index.html") || await cache.match("./");
        if (fallback) return fallback;
      }
      throw error;
    }
  })());
});

self.addEventListener("message", (event) => {
  if (event.data?.type !== "PURGE_SCOPE_CACHE") return;
  event.waitUntil(caches.delete(CACHE_NAME));
});
