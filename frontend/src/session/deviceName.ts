import {t} from '../i18n'
/**
 * 按 User-Agent 生成设备名，例如「Chrome · macOS」，用于在设备列表中区分登录的设备。
 */

/**
 * 桌面客户端（kanban-app）在 User-Agent 末尾追加的标记 `KanbanApp/<版本>`。
 * 客户端基于 Chromium，标记优先于浏览器识别，设备名显示为「桌面客户端 · 系统」。
 */
const DESKTOP_APP = /KanbanApp\//

const BROWSERS: [RegExp, string][] = [
  [/Edg\//, 'Edge'],
  [/OPR\//, 'Opera'],
  [/Firefox\//, 'Firefox'],
  [/Chrome\//, 'Chrome'],
  [/Safari\//, 'Safari'],
]

const SYSTEMS: [RegExp, string][] = [
  [/iPhone|iPad|iPod/, 'iOS'],
  [/Android/, 'Android'],
  [/Windows/, 'Windows'],
  [/Mac OS X|Macintosh/, 'macOS'],
  [/Linux/, 'Linux'],
]

function match(userAgent: string, rules: [RegExp, string][]): string | null {
  return rules.find(([pattern]) => pattern.test(userAgent))?.[1] ?? null
}

export function deviceName(userAgent: string): string {
  const client = DESKTOP_APP.test(userAgent) ? t('session.desktopApp') : match(userAgent, BROWSERS)
  const parts = [client, match(userAgent, SYSTEMS)].filter(Boolean)
  return parts.length > 0 ? parts.join(' · ') : t('session.unknownDevice')
}
