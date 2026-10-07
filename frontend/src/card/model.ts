/**
 * 卡片相关的计算：列表中的排序、到期状态、搜索和筛选、卡龄、定时器读数、任务树。
 */
import type {Card, List, PriorityLevel, Task} from '../api/types'
import {t} from '../i18n'

const DAY_MS = 24 * 60 * 60 * 1000

/** 列表中的卡片（不含卡片归档），按手动顺序（隐藏序号）。 */
export function manualOrder(cards: Card[], listId: number): Card[] {
  return cards
    .filter((card) => card.list_id === listId && card.archived_at === null)
    .sort((a, b) => (a.position ?? 0) - (b.position ?? 0) || a.id - b.id)
}

/** 排序用的键：数字或字符串，null 表示该字段为空。 */
type SortKey = number | string | null

function sortKey(card: Card, list: List, priorityRank: Map<number, number>): SortKey {
  const time = (value: string | null) => (value === null ? null : Date.parse(value))
  switch (list.sort_mode) {
    case 'title':
      return card.title
    case 'priority':
      // 挡位顺序越靠前优先级越高；没有优先级相当于最低，与空值一样排在当前方向的最后。
      return card.priority_level_id === null ? null : (priorityRank.get(card.priority_level_id) ?? null)
    case 'remind_at':
      return time(card.remind_at)
    case 'due_at':
      return time(card.due_at)
    case 'created_at':
      return time(card.created_at)
    case 'started_at':
      return time(card.started_at)
    case 'completed_at':
      return time(card.completed_at)
    default:
      return null
  }
}

function compareKeys(a: SortKey, b: SortKey): number {
  if (a === null || b === null) {
    // 空值视为比任何值都大：正序在最下面，倒序在最上面。
    return a === b ? 0 : a === null ? 1 : -1
  }
  if (typeof a === 'string' && typeof b === 'string') {
    return a.localeCompare(b, 'zh-Hans-CN')
  }
  return (a as number) - (b as number)
}

/**
 * 列表中的卡片按列表的排序方式展示。默认排序按隐藏序号，不使用方向；
 * 其他方式按字段和方向排列，相同时按隐藏序号，不改写隐藏序号。
 */
export function sortedCards(cards: Card[], list: List, levels: PriorityLevel[]): Card[] {
  const manual = manualOrder(cards, list.id)
  if (list.sort_mode === 'manual') return manual
  const rank = new Map(levels.map((level, index) => [level.id, index]))
  const order = new Map(manual.map((card, index) => [card.id, index]))
  const direction = list.sort_dir === 'asc' ? 1 : -1
  return [...manual].sort(
    (a, b) => direction * compareKeys(sortKey(a, list, rank), sortKey(b, list, rank)) || order.get(a.id)! - order.get(b.id)!,
  )
}

/**
 * 把卡片 draggedId 拖到列表 target 时传给移动接口的位置：按隐藏序号的第 index 位（不计被拖动的卡片），
 * index 为 null 表示列尾。overCardId 是落点所在的卡片，after 表示放在它下方；为 null 表示放在列表空白处（列尾）。
 * 目标列表不是手动排序时，卡片在列中的显示位置由排序方式决定，与落点无关；隐藏序号放到列尾，
 * 只在列表改回手动排序时体现。在这种列表内部拖动不改变任何东西。
 * 位置没有变化时返回 null。
 */
export function resolveCardDrop(
  cards: Card[],
  target: List,
  draggedId: number,
  overCardId: number | null,
  after: boolean,
): {index: number | null} | null {
  if (overCardId === draggedId) return null
  const order = manualOrder(cards, target.id)
  const current = order.findIndex((card) => card.id === draggedId)
  if (target.sort_mode !== 'manual') {
    return current >= 0 ? null : {index: null}
  }
  const others = order.filter((card) => card.id !== draggedId)
  let index: number | null = null
  if (overCardId !== null) {
    const at = others.findIndex((card) => card.id === overCardId)
    if (at < 0) return null
    index = after ? at + 1 : at
  }
  if (current >= 0 && (index ?? others.length) === current) return null
  return {index}
}

/** 卡片的到期状态。 */
export type DueStatus = 'overdue' | 'today' | 'soon'

/** 时刻在时区中的日期，格式 YYYY-MM-DD。 */
function dayIn(instant: number, timeZone: string): string {
  return new Intl.DateTimeFormat('en-CA', {timeZone, year: 'numeric', month: '2-digit', day: '2-digit'}).format(instant)
}

/**
 * 到期状态，只针对完成时间为空、有截止时间的卡片：
 * 截止时间早于此刻为逾期；落在用户时区的今天且不早于此刻为今日；此刻之后 24 小时内为即将到期。
 */
export function dueStatus(card: Card, now: number, timeZone: string): DueStatus | null {
  if (card.completed_at !== null || card.due_at === null) return null
  const due = Date.parse(card.due_at)
  if (due < now) return 'overdue'
  if (dayIn(due, timeZone) === dayIn(now, timeZone)) return 'today'
  if (due - now <= DAY_MS) return 'soon'
  return null
}

/** 今日卡片：逾期和今日的卡片。 */
export function isTodayCard(card: Card, now: number, timeZone: string): boolean {
  const status = dueStatus(card, now, timeZone)
  return status === 'overdue' || status === 'today'
}

/** 面板的筛选条件。labels 中每个标签可以要求包含或排除；priorities 中 null 表示没有优先级。 */
export interface CardFilter {
  query: string
  labels: Record<number, 'include' | 'exclude'>
  priorities: (number | null)[]
  lists: number[]
}

export const EMPTY_FILTER: CardFilter = {query: '', labels: {}, priorities: [], lists: []}

/** 是否设置了任何筛选条件（不含搜索）。 */
export function filterActive(filter: CardFilter): boolean {
  return Object.keys(filter.labels).length > 0 || filter.priorities.length > 0 || filter.lists.length > 0
}

/**
 * 卡片是否满足搜索和筛选：关键词按空格分隔，都要出现在标题或描述中，不区分大小写；
 * 筛选条件同时满足才显示，同一类条件中选了多个时满足其一即可。
 */
export function matchesFilter(card: Card, filter: CardFilter): boolean {
  const keywords = filter.query.toLowerCase().split(/\s+/).filter(Boolean)
  if (keywords.length > 0) {
    const text = `${card.title}\n${card.description}`.toLowerCase()
    if (!keywords.every((keyword) => text.includes(keyword))) return false
  }
  const labelRules = Object.entries(filter.labels)
  const included = labelRules.filter(([, mode]) => mode === 'include').map(([id]) => Number(id))
  const excluded = labelRules.filter(([, mode]) => mode === 'exclude').map(([id]) => Number(id))
  if (included.length > 0 && !included.some((id) => card.label_ids.includes(id))) return false
  if (excluded.some((id) => card.label_ids.includes(id))) return false
  if (filter.priorities.length > 0 && !filter.priorities.includes(card.priority_level_id)) return false
  if (filter.lists.length > 0 && (card.list_id === null || !filter.lists.includes(card.list_id))) return false
  return true
}

/** 卡龄：从创建时间到此刻经过的时长，例如「3 天」「5 小时」「刚刚」。 */
export function formatAge(createdAt: string, now: number): string {
  const minutes = Math.floor((now - Date.parse(createdAt)) / 60000)
  if (minutes < 1) return t('card.justNow')
  if (minutes < 60) return t('card.ageMinutes', {count: minutes})
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return t('card.ageHours', {count: hours})
  const days = Math.floor(hours / 24)
  if (days < 30) return t('card.ageDays', {count: days})
  const months = Math.floor(days / 30)
  if (months < 12) return t('card.ageMonths', {count: months})
  return t('card.ageYears', {count: Math.floor(days / 365)})
}

/** 定时器当前读数（秒）：累计秒数加上本次开始到此刻的秒数。 */
export function timerSeconds(card: Card, now: number): number {
  if (card.timer_started_at === null) return card.timer_seconds
  return card.timer_seconds + Math.max(0, Math.floor((now - Date.parse(card.timer_started_at)) / 1000))
}

/** 把秒数显示为 H:MM:SS。 */
export function formatDuration(seconds: number): string {
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

/** 把 H:MM:SS、MM:SS 或秒数解析为秒数，格式不对时返回 null。 */
export function parseDuration(text: string): number | null {
  const parts = text.trim().split(':')
  if (parts.length > 3 || parts.some((part) => !/^\d+$/.test(part))) return null
  return parts.reduce((total, part) => total * 60 + Number(part), 0)
}

/** 任务树的节点。 */
export interface TaskNode {
  task: Task
  children: TaskNode[]
  /** 第一层为 1。 */
  depth: number
}

/** 卡片的任务树，同一父任务下按顺序排列。 */
export function taskTree(tasks: Task[], cardId: number): TaskNode[] {
  const own = tasks.filter((task) => task.card_id === cardId).sort((a, b) => a.position - b.position || a.id - b.id)
  const build = (parentId: number | null, depth: number): TaskNode[] =>
    own.filter((task) => task.parent_id === parentId).map((task) => ({task, depth, children: build(task.id, depth + 1)}))
  return build(null, 1)
}

/** 任务的最大层数，与后端一致。 */
export const MAX_TASK_DEPTH = 3

/** 拖动任务的落点：放在某个任务前面、后面，或成为它的子任务。 */
export type TaskDropIntent = 'before' | 'after' | 'inside'

/**
 * 把任务 draggedId 拖到 overId 的 intent 处时的目标父任务和同级位置（不计被拖动的任务）。
 * 不能放的位置（自己、自己的下级、移动后超过三层）以及没有变化的位置返回 null。
 */
export function resolveTaskDrop(
  tasks: Task[],
  draggedId: number,
  overId: number,
  intent: TaskDropIntent,
): {parentId: number | null; index: number | null} | null {
  const byId = new Map(tasks.map((task) => [task.id, task]))
  const dragged = byId.get(draggedId)
  const over = byId.get(overId)
  if (!dragged || !over || draggedId === overId) return null

  const subtree = new Set([draggedId])
  let grew = true
  while (grew) {
    grew = false
    for (const task of tasks) {
      if (task.parent_id !== null && subtree.has(task.parent_id) && !subtree.has(task.id)) {
        subtree.add(task.id)
        grew = true
      }
    }
  }
  if (subtree.has(overId)) return null

  const depthOf = (id: number | null): number => {
    let depth = 0
    for (let current = id; current !== null && depth <= MAX_TASK_DEPTH; current = byId.get(current)?.parent_id ?? null) depth++
    return depth
  }
  const heightOf = (id: number): number => 1 + Math.max(0, ...tasks.filter((task) => task.parent_id === id).map((task) => heightOf(task.id)))

  const parentId = intent === 'inside' ? overId : over.parent_id
  if (depthOf(parentId) + heightOf(draggedId) > MAX_TASK_DEPTH) return null

  const siblings = tasks
    .filter((task) => task.parent_id === parentId && task.id !== draggedId)
    .sort((a, b) => a.position - b.position || a.id - b.id)
  let index: number | null = null
  if (intent !== 'inside') {
    const at = siblings.findIndex((task) => task.id === overId)
    index = intent === 'before' ? at : at + 1
  }
  if (parentId === dragged.parent_id) {
    const current = tasks
      .filter((task) => task.parent_id === parentId)
      .sort((a, b) => a.position - b.position || a.id - b.id)
      .findIndex((task) => task.id === draggedId)
    if ((index ?? siblings.length) === current) return null
  }
  return {parentId, index}
}

/** 任务进度：所有层级的任务一起计数。 */
export function taskProgress(tasks: Task[], cardId: number): {done: number; total: number} {
  return taskProgressByCard(tasks).get(cardId) ?? {done: 0, total: 0}
}

/** 每张卡片的任务进度，用于一次显示整个面板的卡片；没有任务的卡片不在结果中。 */
export function taskProgressByCard(tasks: Task[]): Map<number, {done: number; total: number}> {
  const progress = new Map<number, {done: number; total: number}>()
  for (const task of tasks) {
    const entry = progress.get(task.card_id) ?? {done: 0, total: 0}
    entry.total++
    if (task.done) entry.done++
    progress.set(task.card_id, entry)
  }
  return progress
}
