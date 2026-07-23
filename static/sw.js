const CACHE = 'lesprivate-v4';
self.addEventListener('install', e => {
  e.waitUntil(
    caches.open(CACHE).then(c => c.addAll([
      '/static/manifest.json?v=3',
      '/static/icon-192x192.png?v=3',
      '/static/icon-512x512.png?v=3',
      '/static/favicon.png?v=3'
    ]))
  );
  self.skipWaiting();
});
self.addEventListener('activate', e => {
  e.waitUntil(caches.keys().then(keys => Promise.all(
    keys.filter(k => k !== CACHE).map(k => caches.delete(k))
  )));
  self.clients.claim();
});
self.addEventListener('fetch', e => {
  if (e.request.method !== 'GET') return;
  if (e.request.url.includes('/static/')) {
    e.respondWith(
      caches.match(e.request).then(r =>
        r || fetch(e.request).then(fr => {
          const clone = fr.clone();
          caches.open(CACHE).then(c => c.put(e.request, clone));
          return fr;
        })
      )
    );
  }
});
