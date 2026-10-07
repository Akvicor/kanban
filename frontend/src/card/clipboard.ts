/**
 * 卡片的复制、剪切内容。只保存在当前设备的内存中，不同步；刷新页面后清空。
 * 粘贴只在同一面板内有效。
 */

export interface CardClipboard {
  mode: 'copy' | 'cut'
  cardId: number
  panelId: number
}

let current: CardClipboard | null = null

export function getClipboard(): CardClipboard | null {
  return current
}

export function setClipboard(value: CardClipboard | null): void {
  current = value
}

/** 粘贴到指定面板时的结果：没有内容、跨面板，或按复制、剪切执行。 */
export function classifyPaste(clip: CardClipboard | null, panelId: number): 'empty' | 'cross-panel' | 'copy' | 'cut' {
  if (!clip) return 'empty'
  if (clip.panelId !== panelId) return 'cross-panel'
  return clip.mode
}
