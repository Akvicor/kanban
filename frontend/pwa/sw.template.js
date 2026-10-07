// Service Worker 模板。构建时由 pwa/serviceWorkerPlugin.ts 填入版本号和预缓存列表，生成站点根目录下的 sw.js。
//
// 只缓存页面本身，用来加快打开速度；数据读写都走网络，不做离线编辑：
// - 页面（导航请求）网络优先：每次先向服务端取最新的 index.html 并更新缓存，取不到时使用缓存的页面外壳。
// - /static/ 下的资源文件名带内容哈希，按版本缓存：安装时预缓存入口资源，其余在访问时缓存。
// - 新版本激活时删除旧版本的缓存。/api 和其他请求不经过 Service Worker。

const VERSION = '__SW_VERSION__'
const PRECACHE = JSON.parse('__SW_PRECACHE__')
const CACHE_PREFIX = 'kanban-'
const CACHE = CACHE_PREFIX + VERSION
// 页面外壳的缓存键。所有页面地址都由同一份 index.html 渲染。
const SHELL = '/'

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches
      .open(CACHE)
      .then((cache) => cache.addAll([SHELL, ...PRECACHE]))
      .then(() => self.skipWaiting()),
  )
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((key) => key.startsWith(CACHE_PREFIX) && key !== CACHE).map((key) => caches.delete(key))))
      .then(() => self.clients.claim()),
  )
})

self.addEventListener('fetch', (event) => {
  const request = event.request
  if (request.method !== 'GET') return
  const url = new URL(request.url)
  if (url.origin !== self.location.origin || url.pathname.startsWith('/api/')) return
  if (request.mode === 'navigate') {
    event.respondWith(networkFirstPage(request))
  } else if (url.pathname.startsWith('/static/')) {
    event.respondWith(cachedAsset(request))
  }
})

/** 页面网络优先。只缓存 HTML 响应，避免把错误页或其他内容当作页面外壳。 */
async function networkFirstPage(request) {
  const cache = await caches.open(CACHE)
  try {
    const response = await fetch(request)
    if (response.ok && (response.headers.get('Content-Type') ?? '').includes('text/html')) {
      await cache.put(SHELL, response.clone())
    }
    return response
  } catch (err) {
    const shell = await cache.match(SHELL)
    if (shell) return shell
    throw err
  }
}

/** 带哈希的资源内容不变，命中缓存直接返回；否则从网络取，成功后放入当前版本的缓存。 */
async function cachedAsset(request) {
  const cache = await caches.open(CACHE)
  const cached = await cache.match(request)
  if (cached) return cached
  const response = await fetch(request)
  if (response.ok) await cache.put(request, response.clone())
  return response
}
