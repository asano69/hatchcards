// hashcards service worker — cache-first for static assets, network-first for pages.
const CACHE = "hashcards-v5;"
const PRECACHE = [
  "/static/css/tokens.css",
  "/static/css/base.css",
  "/static/css/components.css",
  "/static/css/index.css",
  "/static/css/drill.css",
  "/static/css/done.css",
  "/static/css/new.css",
  "/static/script.js",
  "/static/icons/icon-192.png",
  "/static/icons/icon-512.png",
  "/static/icons/icon-192-maskable.png",
  "/static/icons/icon-512-maskable.png",
];

self.addEventListener("install", e => {
  e.waitUntil(
    caches.open(CACHE).then(c => c.addAll(PRECACHE))
  );
  self.skipWaiting();
});

self.addEventListener("activate", e => {
  // Remove caches from previous versions.
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

  // All other requests (pages, API, file serving) go to the network.
});
