// hashcards service worker
// Caches the app shell on install; serves from cache when offline.
const CACHE = "hashcards-v1";
const PRECACHE = ["/manifest.json", "/icon-192.png", "/icon-512.png"];

self.addEventListener("install", e => {
  e.waitUntil(
    caches.open(CACHE).then(c => c.addAll(PRECACHE))
  );
  self.skipWaiting();
});

self.addEventListener("activate", e => {
  e.waitUntil(
    caches.keys().then(keys =>
      Promise.all(keys.filter(k => k !== CACHE).map(k => caches.delete(k)))
    )
  );
  self.clients.claim();
});

self.addEventListener("fetch", e => {
  // Network-first for navigation and API; cache-first for static assets.
  const url = new URL(e.request.url);
  const isStatic = /\.(png|jpg|jpeg|gif|webp|svg|woff2?|ttf|css|js)$/.test(url.pathname);

  if (isStatic) {
    e.respondWith(
      caches.match(e.request).then(cached => cached || fetch(e.request).then(resp => {
        const clone = resp.clone();
        caches.open(CACHE).then(c => c.put(e.request, clone));
        return resp;
      }))
    );
  }
  // For everything else, go to network (dynamic card content stays fresh).
});
