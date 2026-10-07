/** 重试等待时间从 1 秒开始逐次翻倍，最长 30 秒。同步连接重连和打开页面时确认登录状态共用。 */
const BASE_MS = 1000
const MAX_MS = 30000

/** 第 attempt 次重试（从 0 开始）前的等待时间。 */
export function backoffDelay(attempt: number): number {
  return Math.min(MAX_MS, BASE_MS * 2 ** attempt)
}
