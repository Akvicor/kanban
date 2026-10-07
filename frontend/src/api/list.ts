import {http} from './client'
import type {List, ListEnd, ListRules, PanelContent, SortDir, SortMode} from './types'

/** 加载面板的全部列表（包括列表归档中的）及其操作配置。 */
export function fetchPanelContent(panelId: number) {
  return http.get<PanelContent>(`/panel/content?panel_id=${panelId}`)
}

/** 在面板最右边新建列表。 */
export function createList(panelId: number, name: string) {
  return http.write<List>('/list/create', {panel_id: panelId, name})
}

export function renameList(id: number, name: string) {
  return http.write<List>('/list/rename', {id, name})
}

/** 列表的展示和通知设置，各项都需要提供。color 为空字符串表示没有颜色。 */
export interface ListSettingsInput {
  color: string
  show_age: boolean
  sort_mode: SortMode
  sort_dir: SortDir
  head_add: ListEnd
  tail_add: ListEnd
  remind_off: boolean
  due_off: boolean
}

export function updateListSettings(id: number, settings: ListSettingsInput) {
  return http.write<List>('/list/settings', {id, ...settings})
}

/** 用新的操作配置替换列表原有的配置。 */
export function updateListRules(id: number, rules: ListRules) {
  return http.write<List>('/list/rules', {id, rules})
}

export function reorderList(id: number, index: number | null) {
  return http.write<List>('/list/reorder', {id, index})
}

export function archiveList(id: number) {
  return http.write('/list/archive', {id})
}

/** 从列表归档恢复列表，atStart 为 true 时放在面板最左边，否则放在最右边。 */
export function restoreList(id: number, atStart: boolean) {
  return http.write<List>('/list/restore', {id, at_start: atStart})
}
