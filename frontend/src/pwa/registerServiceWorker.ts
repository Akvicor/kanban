/**
 * 注册 Service Worker（构建时生成的 /sw.js，见 pwa/serviceWorkerPlugin.ts），用于安装到桌面和缓存页面外壳。
 * 只在生产构建中注册：开发服务器没有 sw.js，并且缓存会干扰热更新。浏览器只在安全上下文（HTTPS 或 localhost）
 * 中提供 Service Worker。注册失败不影响页面使用。
 */
export function registerServiceWorker(): void {
  if (!import.meta.env.PROD || !window.isSecureContext || !('serviceWorker' in navigator)) return
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js').catch(() => {
      // 注册失败时页面照常使用，只是不能离线打开页面外壳。
    })
  })
}
