// hashcards service worker — cache-first for static assets, network-first for pages.
const CACHE = "hashcards-v2";
const PRECACHE = [
  "/static/style.css",
  "/static/script.js",
  "/static/icon.svg",
];

self.addEventListener("install", e => {
  e.waitUntil(
    caches.open(CACHE).then(c => c.addAll(PRECACHE))
  );
  self.skipWaiting();
});

self.addEventListener("activate", e => {
  // Remove any caches from previous versions.
  e.waitUntil(
    caches.keys().then(keys =>
      Promise.all(keys.filter(k => k !== CACHE).map(k => caches.delete(k)))
    )
  );
  self.clients.claim();
});

self.addEventListener("fetch", e => {
  const url = new URL(e.request.url);

  // Cache-first only for static assets served under /static/.
  const isStaticAsset =
    url.pathname.startsWith("/static/") &&
    /\.(css|js|svg|woff2?|ttf|png|jpg|jpeg|gif|webp)$/.test(url.pathname);

  if (isStaticAsset) {
    e.respondWith(
      caches.match(e.request).then(cached => {
        if (cached) return cached;
        return fetch(e.request).then(resp => {
          const clone = resp.clone();
          caches.open(CACHE).then(c => c.put(e.request, clone));
          return resp;
        });
      })
    );
    return;
  }

  // For all other requests (pages, API, file serving) go to the network.
  // This ensures card content and session state are always fresh.
});
