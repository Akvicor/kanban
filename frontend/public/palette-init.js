// 在样式和脚本加载前按本设备缓存的配色偏好设置配色，避免首屏闪烁。
// 规则与 src/theme/palette.ts 一致：跟随系统时亮色用 clean、暗色用 dark。
// 以独立文件在 index.html 的 <head> 中同步加载，内容安全策略只允许本站脚本，不允许内联脚本。
(() => {
  let preference = 'clean';
  try {
    const stored = localStorage.getItem('kanban.palette');
    if (['system', 'clean', 'dark', 'paper'].includes(stored)) preference = stored;
  } catch {
    // localStorage 不可用时使用默认配色。
  }
  const palette = preference === 'system'
    ? (matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'clean')
    : preference;
  document.documentElement.dataset.palette = palette;
})();
