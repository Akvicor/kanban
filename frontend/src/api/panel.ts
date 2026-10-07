import {http} from './client'
import type {Board, Label, Panel, PriorityLevel} from './types'

/** 面板、标签和优先级挡位的写入接口。index 是在同级中的位置（从 0 开始），为 null 时放到最后。 */

export function createPanel(boardId: number, name: string) {
  return http.write<Panel>('/panel/create', {board_id: boardId, name})
}

export function renamePanel(id: number, name: string) {
  return http.write<Panel>('/panel/rename', {id, name})
}

export function reorderPanel(id: number, index: number | null) {
  return http.write<Panel>('/panel/reorder', {id, index})
}

/** 把面板移到另一个看板，排在最后一个标签页。 */
export function movePanel(id: number, boardId: number) {
  return http.write<Panel>('/panel/move', {id, board_id: boardId})
}

export function archivePanel(id: number) {
  return http.write('/panel/archive', {id})
}

/** 把面板归档中的面板，或看板归档中某个看板里的面板，恢复到指定看板。 */
export function restorePanel(id: number, boardId: number) {
  return http.write<Panel>('/panel/restore', {id, board_id: boardId})
}

/** 设置看板的主面板，panelId 为 null 时取消。返回修改后的看板。 */
export function setMainPanel(boardId: number, panelId: number | null) {
  return http.write<Board>('/board/main_panel', {board_id: boardId, panel_id: panelId})
}

export function createLabel(panelId: number, name: string, color: string) {
  return http.write<Label>('/label/create', {panel_id: panelId, name, color})
}

export function updateLabel(id: number, name: string, color: string) {
  return http.write<Label>('/label/update', {id, name, color})
}

export function reorderLabel(id: number, index: number | null) {
  return http.write<Label>('/label/reorder', {id, index})
}

export function deleteLabel(id: number) {
  return http.write('/label/delete', {id})
}

export function createPriority(panelId: number, name: string, color: string) {
  return http.write<PriorityLevel>('/priority/create', {panel_id: panelId, name, color})
}

export function updatePriority(id: number, name: string, color: string) {
  return http.write<PriorityLevel>('/priority/update', {id, name, color})
}

export function reorderPriority(id: number, index: number | null) {
  return http.write<PriorityLevel>('/priority/reorder', {id, index})
}

export function deletePriority(id: number) {
  return http.write('/priority/delete', {id})
}
