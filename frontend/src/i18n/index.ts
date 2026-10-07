/**
 * 界面文案取用与语言切换。
 *
 * 语言以模块级状态保存（见 locale.ts 的解析与缓存），组件用 useT 订阅变化；
 * 网络层等非组件代码直接调用 t/errorText。文案包是扁平键值表，键按「模块.名称」组织，
 * 简体中文包是缺键回退基准；英文包缺键会让 errors.test.ts 失败。
 */
import {useSyncExternalStore} from 'react'
import {errorMessages as errorsEn} from './errors.en'
import {errorMessages as errorsZhCN} from './errors.zh-CN'
import {cacheLocale, readCachedLocale, resolveLocale, type Locale} from './locale'
import {messages as messagesEn} from './messages.en'
import {messages as messagesZhCN} from './messages.zh-CN'

const PACKS: Record<Locale, Record<string, string>> = {'zh-CN': messagesZhCN, en: messagesEn}
const ERROR_PACKS: Record<Locale, Record<number, string>> = {'zh-CN': errorsZhCN, en: errorsEn}

let current: Locale = readCachedLocale()
const listeners = new Set<() => void>()

/** 把标签页标题和 PWA 名称改为当前语言的版本；首屏与切换语言时都要调用。 */
function applyDocumentTitle(): void {
  if (typeof document === 'undefined') return
  document.title = t('app.title')
  document.querySelector('meta[name="apple-mobile-web-app-title"]')?.setAttribute('content', t('app.title'))
}
applyDocumentTitle()

function subscribe(listener: () => void): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

/** 当前界面语言。 */
export function getLocale(): Locale {
  return current
}

/** 切换界面语言并缓存到本设备，通知所有订阅者。 */
export function setLocale(locale: Locale): void {
  if (current === locale) return
  current = locale
  cacheLocale(locale)
  applyDocumentTitle()
  for (const listener of listeners) listener()
}

/** 按个人设置中的语言偏好解析并应用；空值跟随浏览器语言。 */
export function applyLocalePreference(preference: string): void {
  setLocale(resolveLocale(preference, typeof navigator === 'undefined' ? [] : navigator.languages))
}

/** t 取当前语言的文案。缺键回退简体中文，仍缺则返回键名；{name} 占位符用 vars 替换。 */
export function t(key: string, vars?: Record<string, string | number>): string {
  const text = PACKS[current][key] ?? PACKS['zh-CN'][key] ?? key
  if (!vars) return text
  return text.replace(/\{(\w+)\}/g, (match, name: string) => (name in vars ? String(vars[name]) : match))
}

/** errorText 取错误码在当前语言下的文案，缺码回退简体中文。 */
export function errorText(code: number): string {
  return ERROR_PACKS[current][code] ?? ERROR_PACKS['zh-CN'][code] ?? t('error.unknown')
}

/** useLocale 订阅语言变化，供组件随语言重新渲染。 */
export function useLocale(): Locale {
  return useSyncExternalStore(subscribe, getLocale, getLocale)
}

/** useT 订阅语言变化并返回 t，组件在语言切换后按新语言取文案。 */
export function useT(): typeof t {
  useLocale()
  return t
}

export type {Locale}
