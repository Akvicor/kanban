/** 列表相关的计算：面板中的列表、排序方式的显示文字、拖动列表后的位置。 */
import type {List, ListOp, SortDir, SortMode, TimeAction} from '../api/types'
import {t} from '../i18n'

function byPosition(a: List, b: List): number {
  return a.position - b.position || a.id - b.id
}

/** 面板中不在列表归档中的列表，从左到右。 */
export function activeLists(lists: List[], panelId: number): List[] {
  return lists.filter((list) => list.panel_id === panelId && list.archived_at === null).sort(byPosition)
}

/** 面板的列表归档，最近归档的在前。 */
export function archivedLists(lists: List[], panelId: number): List[] {
  return lists
    .filter((list) => list.panel_id === panelId && list.archived_at !== null)
    .sort((a, b) => b.archived_at!.localeCompare(a.archived_at!) || b.id - a.id)
}

/** 排序方式选项，按当前语言返回。 */
export function sortModes(): {value: SortMode; label: string}[] {
  return [
    {value: 'manual', label: t('common.default')},
    {value: 'title', label: t('common.title')},
    {value: 'priority', label: t('common.priority')},
    {value: 'remind_at', label: t('common.remindAt')},
    {value: 'due_at', label: t('common.dueAt')},
    {value: 'created_at', label: t('common.createdAt')},
    {value: 'started_at', label: t('common.startedAt')},
    {value: 'completed_at', label: t('common.completedAt')},
  ]
}

/** 列头显示的排序方式：默认排序不用方向；其他方式显示「方式·方向排序」，例如「截止时间·倒序排序」。 */
export function sortLabel(mode: SortMode, dir: SortDir): string {
  if (mode === 'manual') return t('list.sortManual')
  const label = sortModes().find((m) => m.value === mode)?.label ?? mode
  return t('list.sortLabel', {label, dir: dir === 'asc' ? t('list.sortAsc') : t('list.sortDesc')})
}

/** 操作配置的操作名称，按当前语言返回。 */
export function opLabels(): Record<ListOp, string> {
  return {
    create: t('common.create'),
    enter: t('list.opEnter'),
    exit: t('list.opExit'),
  }
}

/** 时间动作选项，按当前语言返回。 */
export function timeActions(): {value: TimeAction; label: string}[] {
  return [
    {value: '', label: t('list.timeNone')},
    {value: 'set', label: t('list.timeSet')},
    {value: 'update', label: t('list.timeUpdate')},
    {value: 'clear', label: t('common.clear')},
  ]
}

/**
 * 把列表 draggedId 拖到 overId 的左边或右边时，在其他列表中的位置（从 0 开始）。
 * 位置没有变化时返回 null。
 */
export function listDropIndex(lists: List[], draggedId: number, overId: number, after: boolean): number | null {
  if (draggedId === overId) return null
  const others = lists.filter((list) => list.id !== draggedId)
  const at = others.findIndex((list) => list.id === overId)
  if (at < 0) return null
  const index = after ? at + 1 : at
  return index === lists.findIndex((list) => list.id === draggedId) ? null : index
}
