import {http} from './client'
import type {Card, CardBundle, CardLink, ListEnd, Task} from './types'

/** 卡片、任务和关联的接口。写入后返回受影响的实体和同步序号。 */

/** 面板卡片归档中的卡片。 */
export function fetchArchivedCards(panelId: number) {
  return http.get<CardBundle>(`/panel/archived_cards?panel_id=${panelId}`)
}

/** 列表归档中某个列表的卡片。 */
export function fetchListCards(listId: number) {
  return http.get<CardBundle>(`/list/cards?list_id=${listId}`)
}

/** 用列首（head）或列尾（tail）的创建按钮新建卡片。 */
export function createCard(listId: number, title: string, button: ListEnd) {
  return http.write<Card>('/card/create', {list_id: listId, title, button})
}

/** 修改标题。baseRevision 是开始编辑时本地卡片数据的同步序号。 */
export function updateCardTitle(id: number, text: string, baseRevision: number) {
  return http.write<Card>('/card/title', {id, text, base_revision: baseRevision})
}

/** 修改描述。baseRevision 是开始编辑时本地卡片数据的同步序号。 */
export function updateCardDescription(id: number, text: string, baseRevision: number) {
  return http.write<Card>('/card/description', {id, text, base_revision: baseRevision})
}

export function setCardPriority(id: number, priorityLevelId: number | null) {
  return http.write<Card>('/card/priority', {id, priority_level_id: priorityLevelId})
}

export interface CardDatesInput {
  remind_at: string | null
  due_at: string | null
  remind_notify: boolean
  due_notify: boolean
}

export function setCardDates(id: number, dates: CardDatesInput) {
  return http.write<Card>('/card/dates', {id, ...dates})
}

export function setCardLabel(id: number, labelId: number, on: boolean) {
  return http.write<Card>('/card/label', {id, label_id: labelId, on})
}

export function cardTimer(id: number, action: 'start' | 'stop' | 'set', seconds = 0) {
  return http.write<Card>('/card/timer', {id, action, seconds})
}

/** 把卡片移到列表中按隐藏序号的第 index 位，index 为 null 时放到列尾。 */
export function moveCard(id: number, listId: number, index: number | null) {
  return http.write<Card>('/card/move', {id, list_id: listId, index})
}

export function archiveCard(id: number) {
  return http.write('/card/archive', {id})
}

export function archiveAllCards(listId: number) {
  return http.write('/list/archive_cards', {id: listId})
}

export function restoreCard(id: number, listId: number, end: ListEnd) {
  return http.write<Card>('/card/restore', {id, list_id: listId, end})
}

/** 复制卡片。listId 为 null 时副本插在原卡片下方，否则插到该列表的列首。 */
export function copyCard(id: number, listId: number | null) {
  return http.write<Card>('/card/copy', {id, list_id: listId})
}

/** 新建任务，titles 有多项时每项一个任务。 */
export function createTasks(cardId: number, parentId: number | null, titles: string[]) {
  return http.write<Task[]>('/task/create', {card_id: cardId, parent_id: parentId, titles})
}

export function renameTask(id: number, title: string) {
  return http.write<Task>('/task/rename', {id, title})
}

/** 勾选或取消勾选任务。返回状态被改动的任务：勾选时含随之勾选的下级任务。 */
export function setTaskDone(id: number, done: boolean) {
  return http.write<Task[]>('/task/done', {id, done})
}

export function moveTask(id: number, parentId: number | null, index: number | null) {
  return http.write<Task>('/task/move', {id, parent_id: parentId, index})
}

export function deleteTask(id: number) {
  return http.write('/task/delete', {id})
}

export function addCardLink(cardId: number, target: {board_id: number} | {panel_id: number}) {
  return http.write<CardLink>('/card_link/add', {card_id: cardId, ...target})
}

export function deleteCardLink(id: number) {
  return http.write('/card_link/delete', {id})
}
