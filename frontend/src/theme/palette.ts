/**
 * 配色选择与应用。
 *
 * 用户在个人设置中选择配色偏好：跟随系统，或固定使用三种配色之一。
 * 跟随系统时，系统亮色用清爽，系统暗色用暗夜。
 */

/** 实际显示的配色：clean 清爽（A）、dark 暗夜（B）、paper 纸感（C）。 */
export type Palette = 'clean' | 'dark' | 'paper'

/** 用户的配色偏好，与个人设置中的选项一一对应。 */
export type PalettePreference = 'system' | Palette

/** 新用户和未登录时使用的配色偏好。 */
export const DEFAULT_PREFERENCE: PalettePreference = 'clean'

/** 跟随系统时，系统亮色和暗色分别使用的配色。 */
const SYSTEM_LIGHT: Palette = 'clean'
const SYSTEM_DARK: Palette = 'dark'

/**
 * 本设备缓存最近一次的配色偏好，用于页面脚本加载前先套用配色，避免闪烁。
 * 偏好本身以服务端的个人设置为准，缓存只在拿到服务端设置前使用。
 * public/palette-init.js（由 index.html 在 head 中加载）读取同一个键，修改时需同步。
 */
export const PREFERENCE_STORAGE_KEY = 'kanban.palette'

/** 系统暗色外观的媒体查询。 */
export const SYSTEM_DARK_QUERY = '(prefers-color-scheme: dark)'

const PREFERENCES: readonly PalettePreference[] = ['system', 'clean', 'dark', 'paper']

/** 判断任意值是否为合法的配色偏好。 */
export function isPalettePreference(value: unknown): value is PalettePreference {
  return typeof value === 'string' && (PREFERENCES as readonly string[]).includes(value)
}

/** 按偏好和系统外观得出实际配色。 */
export function resolvePalette(preference: PalettePreference, systemDark: boolean): Palette {
  if (preference === 'system') {
    return systemDark ? SYSTEM_DARK : SYSTEM_LIGHT
  }
  return preference
}

/** 读取本设备缓存的配色偏好，缓存缺失或无效时返回默认偏好。 */
export function readCachedPreference(): PalettePreference {
  try {
    const stored = localStorage.getItem(PREFERENCE_STORAGE_KEY)
    return isPalettePreference(stored) ? stored : DEFAULT_PREFERENCE
  } catch {
    return DEFAULT_PREFERENCE
  }
}

/** 把配色偏好缓存到本设备。存储不可用时忽略，只影响下次打开时的首屏配色。 */
export function cachePreference(preference: PalettePreference): void {
  try {
    localStorage.setItem(PREFERENCE_STORAGE_KEY, preference)
  } catch {
    // 隐私模式等情况下 localStorage 不可写，下次打开时回到默认配色。
  }
}

/** 把配色应用到页面，并让浏览器地址栏颜色与顶栏一致。 */
export function applyPalette(palette: Palette): void {
  const root = document.documentElement
  root.dataset.palette = palette
  const themeColor = getComputedStyle(root).getPropertyValue('--topbar-bg').trim()
  const meta = document.querySelector<HTMLMetaElement>('meta[name="theme-color"]')
  if (meta && themeColor) {
    meta.content = themeColor
  }
}
