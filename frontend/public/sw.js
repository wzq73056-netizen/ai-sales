/* 极简 Service Worker：壳缓存 + GET 接口离线回退
   - 静态资源：stale-while-revalidate
   - 导航请求：网络优先，断网回退到缓存的 index.html
   - GET 接口：网络优先，断网回退到上次成功的响应
*/
const CACHE = 'gz-shell-v2';
const SHELL = ['/', '/index.html', '/manifest.webmanifest', '/pwa-192.png', '/pwa-512.png'];

self.addEventListener('install', event => {
  event.waitUntil(
    caches
      .open(CACHE)
      .then(cache => cache.addAll(SHELL))
      .then(() => self.skipWaiting())
  );
});

self.addEventListener('activate', event => {
  event.waitUntil(
    caches
      .keys()
      .then(keys => Promise.all(keys.filter(key => key !== CACHE).map(key => caches.delete(key))))
      .then(() => self.clients.claim())
  );
});

/** 是否是需要离线兜底的接口请求 */
function isApiRequest(url) {
  return url.pathname.startsWith('/api') || url.port === '8888';
}

self.addEventListener('fetch', event => {
  const { request } = event;

  if (request.method !== 'GET') return;

  const url = new URL(request.url);

  if (request.mode === 'navigate') {
    event.respondWith(
      fetch(request).catch(async () => {
        const cache = await caches.open(CACHE);

        return (await cache.match('/index.html')) || (await cache.match('/')) || Response.error();
      })
    );

    return;
  }

  if (isApiRequest(url)) {
    event.respondWith(
      fetch(request)
        .then(response => {
          if (response.ok) {
            const copy = response.clone();

            caches.open(CACHE).then(cache => cache.put(request, copy));
          }

          return response;
        })
        .catch(async () => {
          const cached = await caches.match(request);

          return cached || new Response(JSON.stringify({ code: 7, msg: '离线，暂无缓存数据', data: null }), { headers: { 'Content-Type': 'application/json' } });
        })
    );

    return;
  }

  if (url.origin !== self.location.origin) return;

  event.respondWith(
    caches.open(CACHE).then(async cache => {
      const cached = await cache.match(request);
      const network = fetch(request)
        .then(response => {
          if (response.ok) cache.put(request, response.clone());

          return response;
        })
        .catch(() => cached);

      return cached || network;
    })
  );
});
