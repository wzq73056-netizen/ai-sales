/**
 * 注册 Service Worker（只做壳缓存，接口不缓存）
 *
 * - 只在生产构建里注册，避免开发时的 HMR 被 SW 拦截
 * - 注册失败不影响应用启动
 */
export function setupServiceWorker() {
  if (!import.meta.env.PROD) return;

  if (!('serviceWorker' in navigator)) return;

  const register = () => {
    navigator.serviceWorker.register('/sw.js').catch(() => {
      /* 注册失败不阻断应用 */
    });
  };

  // 入口脚本可能在 load 之后才执行，此时直接注册，否则等 load
  if (document.readyState === 'complete') {
    register();

    return;
  }

  window.addEventListener('load', register);
}
