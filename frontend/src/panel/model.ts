/** 面板相关的计算：看板的标签页、打开看板时进入的面板、拖动标签页后的位置、面板中的标签和挡位。 */
import type {Board, Label, Panel, PriorityLevel} from '../api/types'

function byPosition<T extends {position: number; id: number}>(a: T, b: T): number {
  return a.position - b.position || a.id - b.id
}

/** 看板的标签页：属于该看板、未进入面板归档的面板，按顺序排列。 */
export function boardTabs(panels: Panel[], boardId: number): Panel[] {
  return panels.filter((panel) => panel.board_id === boardId && panel.archived_at === null).sort(byPosition)
}

/** 打开看板时进入的面板：主面板；没有主面板时进入第一个面板；看板没有面板时为 null。 */
export function entryPanelId(board: Board, tabs: Panel[]): number | null {
  if (board.main_panel_id !== null && tabs.some((panel) => panel.id === board.main_panel_id)) {
    return board.main_panel_id
  }
  return tabs[0]?.id ?? null
}

/**
 * 把标签页 draggedId 拖到 overId 的前面或后面时，在其他标签页中的位置（从 0 开始）。
 * 位置没有变化时返回 null。
 */
export function tabDropIndex(tabs: Panel[], draggedId: number, overId: number, after: boolean): number | null {
  if (draggedId === overId) return null
  const others = tabs.filter((panel) => panel.id !== draggedId)
  const at = others.findIndex((panel) => panel.id === overId)
  if (at < 0) return null
  const index = after ? at + 1 : at
  const current = tabs.findIndex((panel) => panel.id === draggedId)
  return index === current ? null : index
}

/** 面板中的标签，按顺序排列。 */
export function panelLabels(labels: Label[], panelId: number): Label[] {
  return labels.filter((label) => label.panel_id === panelId).sort(byPosition)
}

/** 面板中的优先级挡位，按优先级从高到低排列。 */
export function panelPriorityLevels(levels: PriorityLevel[], panelId: number): PriorityLevel[] {
  return levels.filter((level) => level.panel_id === panelId).sort(byPosition)
}
