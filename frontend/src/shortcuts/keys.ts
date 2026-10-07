import {t} from '../i18n'
/**
 * 快捷键按键字符串，与后端 common/shortcut 的格式一致：
 * 修饰键按 Mod、Alt、Shift 的顺序在前，用 + 连接，最后是主键，例如 E、Enter、Mod+C、Mod+Shift+K。
 * Mod 在 macOS 上对应 Cmd，在其他系统上对应 Ctrl。
 */

/** 生成按键字符串需要的键盘事件字段。 */
export interface KeyInput {
  code: string
  ctrlKey: boolean
  metaKey: boolean
  altKey: boolean
  shiftKey: boolean
}

const NAMED_KEYS: Record<string, string> = {
  Enter: 'Enter',
  NumpadEnter: 'Enter',
  Space: 'Space',
  Backspace: 'Backspace',
  Delete: 'Delete',
  ArrowUp: 'ArrowUp',
  ArrowDown: 'ArrowDown',
  ArrowLeft: 'ArrowLeft',
  ArrowRight: 'ArrowRight',
}

/**
 * 主键按物理按键（event.code）识别，这样 Shift+1 仍记为 Shift+1，而不是符号 !。
 * 只按下修饰键，或按下不能绑定的键（Escape、Tab 等）时返回 null。
 */
function mainKey(code: string): string | null {
  let match = /^Key([A-Z])$/.exec(code)
  if (match) {
    return match[1]
  }
  match = /^(?:Digit|Numpad)([0-9])$/.exec(code)
  if (match) {
    return match[1]
  }
  if (/^F([1-9]|1[0-2])$/.test(code)) {
    return code
  }
  return NAMED_KEYS[code] ?? null
}

/** 把键盘事件转换为按键字符串，不能绑定时返回 null。 */
export function keyFromEvent(event: KeyInput): string | null {
  const main = mainKey(event.code)
  if (!main) {
    return null
  }
  const parts: string[] = []
  if (event.ctrlKey || event.metaKey) parts.push('Mod')
  if (event.altKey) parts.push('Alt')
  if (event.shiftKey) parts.push('Shift')
  parts.push(main)
  return parts.join('+')
}

/** 判断当前是否为 Apple 平台，决定 Mod 显示为 ⌘ 还是 Ctrl。 */
export function isApplePlatform(): boolean {
  return /Mac|iPhone|iPad|iPod/.test(navigator.platform || navigator.userAgent)
}

/** 按键名到显示文字：符号键用符号，Space 用语言包文案（按当前语言取用，不做模块级缓存）。 */
function displayName(part: string): string {
  if (part === 'Space') return t('shortcut.keySpace')
  const symbols: Record<string, string> = {ArrowUp: '↑', ArrowDown: '↓', ArrowLeft: '←', ArrowRight: '→'}
  return symbols[part] ?? part
}

/** 按键字符串的显示文字，例如 Mod+C 在 macOS 上显示为 ⌘ + C。 */
export function formatKey(key: string, apple: boolean): string {
  return key
    .split('+')
    .map((part) => {
      if (part === 'Mod') return apple ? '⌘' : 'Ctrl'
      if (part === 'Alt') return apple ? '⌥' : 'Alt'
      if (part === 'Shift') return apple ? '⇧' : 'Shift'
      return displayName(part)
    })
    .join(' + ')
}

/** 找出已经绑定某个按键的操作，没有时返回 null。 */
export function findOwner(bindings: Record<string, string[]>, key: string): string | null {
  for (const [action, keys] of Object.entries(bindings)) {
    if (keys.includes(key)) {
      return action
    }
  }
  return null
}
