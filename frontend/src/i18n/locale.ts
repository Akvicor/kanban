/**
 * 界面与通知语言。
 *
 * 用户在个人设置中选择语言：跟随系统，或固定简体中文/英文。跟随系统时按浏览器语言解析：
 * zh 开头用简体中文，其余用英文。本设备缓存最近一次解析出的语言，供登录页等设置未加载时使用；
 * 语言本身以服务端的个人设置为准。
 */

/** 受支持的界面语言，取值与后端 locale.Type 一致。 */
export type Locale = 'zh-CN' | 'en'

/** 新用户和未登录时的语言偏好（跟随系统）。 */
export const DEFAULT_PREFERENCE = ''

/** 本设备缓存键，保存最近一次解析出的语言。 */
export const LOCALE_STORAGE_KEY = 'kanban.locale'

const LOCALES: readonly Locale[] = ['zh-CN', 'en']

/** 判断任意值是否为受支持的语言。空值表示跟随系统。 */
export function isLocale(value: unknown): value is Locale {
  return typeof value === 'string' && (LOCALES as readonly string[]).includes(value)
}

/** 把浏览器语言标记解析为受支持的语言：zh 开头用简体中文，其余用英文。 */
export function resolveLocale(preference: string, browserLanguages: readonly string[]): Locale {
  if (isLocale(preference)) return preference
  for (const tag of browserLanguages) {
    if (tag.toLowerCase().startsWith('zh')) return 'zh-CN'
    if (tag.toLowerCase().startsWith('en')) return 'en'
  }
  return 'en'
}

/** 读取本设备缓存的语言；缓存缺失或无效时按浏览器语言解析。 */
export function readCachedLocale(): Locale {
  try {
    const stored = localStorage.getItem(LOCALE_STORAGE_KEY)
    if (isLocale(stored)) return stored
  } catch {
    // 存储不可用时退回浏览器语言。
  }
  return resolveLocale(DEFAULT_PREFERENCE, typeof navigator === 'undefined' ? [] : navigator.languages)
}

/** 把解析出的语言缓存到本设备。存储不可用时忽略，只影响下次打开时的首屏语言。 */
export function cacheLocale(locale: Locale): void {
  try {
    localStorage.setItem(LOCALE_STORAGE_KEY, locale)
  } catch {
    // 存储不可用时忽略。
  }
}
