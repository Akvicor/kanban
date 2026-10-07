import {describe, expect, it} from 'vitest'
import type {Card, List, PriorityLevel} from '../api/types'
import {
  dueStatus,
  formatAge,
  matchesFilter,
  parseDuration,
  resolveCardDrop,
  resolveTaskDrop,
  sortedCards,
  taskProgress,
  taskTree,
  timerSeconds,
  EMPTY_FILTER,
} from './model'

function card(id: number, fields: Partial<Card> = {}): Card {
  return {
    id,
    panel_id: 1,
    list_id: 10,
    position: id * 100,
    title: `卡片 ${id}`,
    description: '',
    label_ids: [],
    priority_level_id: null,
    remind_at: null,
    due_at: null,
    remind_notify: false,
    due_notify: false,
    created_at: '2026-01-01T00:00:00Z',
    started_at: null,
    completed_at: null,
    timer_seconds: 0,
    timer_started_at: null,
    archived_at: null,
    cover_attachment_id: null,
    notify_channel_ids: [],
    ...fields,
  }
}

function list(fields: Partial<List>): List {
  const op = {add_labels: [], remove_labels: [], start: '' as const, complete: '' as const}
  return {
    id: 10,
    panel_id: 1,
    name: '列表',
    color: '',
    position: 1,
    show_age: true,
    sort_mode: 'manual',
    sort_dir: 'asc',
    head_add: 'head',
    tail_add: 'tail',
    remind_off: false,
    due_off: false,
    rules: {create: op, enter: op, exit: op},
    archived_at: null,
    ...fields,
  }
}

const levels: PriorityLevel[] = [
  {id: 7, panel_id: 1, name: 'P0', color: '#000000', position: 1},
  {id: 8, panel_id: 1, name: 'P1', color: '#000000', position: 2},
]

const ids = (cards: Card[]) => cards.map((c) => c.id)

describe('sortedCards', () => {
  const cards = [
    card(1, {position: 300, due_at: '2026-03-01T00:00:00Z', priority_level_id: 8}),
    card(2, {position: 100, due_at: null, priority_level_id: null}),
    card(3, {position: 200, due_at: '2026-02-01T00:00:00Z', priority_level_id: 7}),
    card(4, {position: 50, list_id: 11}),
    card(5, {position: null, list_id: null, archived_at: '2026-01-02T00:00:00Z'}),
  ]

  it('默认排序按隐藏序号，只包含该列表中未归档的卡片', () => {
    expect(ids(sortedCards(cards, list({}), levels))).toEqual([2, 3, 1])
  })

  it('时间正序从早到晚，空值在最下面；倒序相反，空值在最上面', () => {
    expect(ids(sortedCards(cards, list({sort_mode: 'due_at', sort_dir: 'asc'}), levels))).toEqual([3, 1, 2])
    expect(ids(sortedCards(cards, list({sort_mode: 'due_at', sort_dir: 'desc'}), levels))).toEqual([2, 1, 3])
  })

  it('优先级正序从高到低，按挡位顺序；没有优先级视为最低', () => {
    expect(ids(sortedCards(cards, list({sort_mode: 'priority', sort_dir: 'asc'}), levels))).toEqual([3, 1, 2])
  })
})

describe('dueStatus', () => {
  const now = Date.parse('2026-03-10T10:00:00Z') // 上海时间 18:00
  const tz = 'Asia/Shanghai'
  it('逾期、今日、即将到期', () => {
    expect(dueStatus(card(1, {due_at: '2026-03-10T09:00:00Z'}), now, tz)).toBe('overdue')
    expect(dueStatus(card(1, {due_at: '2026-03-10T15:00:00Z'}), now, tz)).toBe('today') // 上海 23:00
    expect(dueStatus(card(1, {due_at: '2026-03-10T17:00:00Z'}), now, tz)).toBe('soon') // 上海次日 01:00
    expect(dueStatus(card(1, {due_at: '2026-03-12T10:00:00Z'}), now, tz)).toBeNull()
  })
  it('已完成或没有截止时间的卡片没有到期状态', () => {
    expect(dueStatus(card(1, {due_at: '2026-03-10T09:00:00Z', completed_at: '2026-03-09T00:00:00Z'}), now, tz)).toBeNull()
    expect(dueStatus(card(1), now, tz)).toBeNull()
  })
})

describe('matchesFilter', () => {
  const target = card(1, {title: '整理 Weekly 报告', description: '包含图表', label_ids: [1, 2], priority_level_id: null})
  it('关键词都要出现在标题或描述中，不区分大小写', () => {
    expect(matchesFilter(target, {...EMPTY_FILTER, query: 'weekly 图表'})).toBe(true)
    expect(matchesFilter(target, {...EMPTY_FILTER, query: 'weekly 月报'})).toBe(false)
  })
  it('标签包含与排除、优先级（含没有优先级）、所在列表同时满足', () => {
    expect(matchesFilter(target, {...EMPTY_FILTER, labels: {1: 'include'}})).toBe(true)
    expect(matchesFilter(target, {...EMPTY_FILTER, labels: {2: 'exclude'}})).toBe(false)
    expect(matchesFilter(target, {...EMPTY_FILTER, priorities: [null]})).toBe(true)
    expect(matchesFilter(target, {...EMPTY_FILTER, priorities: [7]})).toBe(false)
    expect(matchesFilter(target, {...EMPTY_FILTER, lists: [11]})).toBe(false)
  })
})

describe('resolveCardDrop', () => {
  // 列表 10：1、2、3；列表 20：4
  const cards = [card(1), card(2), card(3), card(4, {list_id: 20})]
  const manual = list({id: 10, sort_mode: 'manual'})
  it('同列拖动换算成不计自己的位置，位置不变时返回 null', () => {
    expect(resolveCardDrop(cards, manual, 1, 3, true)).toEqual({index: 2})
    expect(resolveCardDrop(cards, manual, 3, 1, false)).toEqual({index: 0})
    expect(resolveCardDrop(cards, manual, 1, 2, false)).toBeNull()
    expect(resolveCardDrop(cards, manual, 3, null, false)).toBeNull()
    expect(resolveCardDrop(cards, manual, 2, 2, false)).toBeNull()
  })
  it('跨列拖到卡片上下或空白处', () => {
    expect(resolveCardDrop(cards, manual, 4, 2, true)).toEqual({index: 2})
    expect(resolveCardDrop(cards, manual, 4, null, false)).toEqual({index: null})
  })
  it('目标列表不是手动排序时放到列尾，列内拖动不变', () => {
    const sorted = list({id: 10, sort_mode: 'title'})
    expect(resolveCardDrop(cards, sorted, 4, 1, false)).toEqual({index: null})
    expect(resolveCardDrop(cards, sorted, 1, 3, true)).toBeNull()
  })
})

describe('resolveTaskDrop', () => {
  // 一(1) ─ 一.一(3) ─ 一.一.一(5)；二(2)；三(4)
  const tasks = [
    {id: 1, card_id: 1, parent_id: null, title: '一', done: false, position: 100},
    {id: 2, card_id: 1, parent_id: null, title: '二', done: false, position: 200},
    {id: 4, card_id: 1, parent_id: null, title: '三', done: false, position: 300},
    {id: 3, card_id: 1, parent_id: 1, title: '一.一', done: false, position: 100},
    {id: 5, card_id: 1, parent_id: 3, title: '一.一.一', done: false, position: 100},
  ]
  it('放到前后时换算成同级位置，放进去时成为最后一个子任务', () => {
    expect(resolveTaskDrop(tasks, 4, 1, 'before')).toEqual({parentId: null, index: 0})
    expect(resolveTaskDrop(tasks, 2, 3, 'inside')).toEqual({parentId: 3, index: null})
    expect(resolveTaskDrop(tasks, 5, 2, 'after')).toEqual({parentId: null, index: 2})
  })
  it('不能放到自己的下级下，移动后不能超过三层，位置不变时返回 null', () => {
    expect(resolveTaskDrop(tasks, 1, 5, 'inside')).toBeNull()
    expect(resolveTaskDrop(tasks, 3, 2, 'inside')).toEqual({parentId: 2, index: null})
    expect(resolveTaskDrop(tasks, 1, 2, 'inside')).toBeNull()
    expect(resolveTaskDrop(tasks, 2, 1, 'after')).toBeNull()
  })
})

describe('卡龄、定时器和任务', () => {
  it('卡龄按创建时间计算', () => {
    const created = '2026-01-01T00:00:00Z'
    expect(formatAge(created, Date.parse('2026-01-01T00:00:30Z'))).toBe('刚刚')
    expect(formatAge(created, Date.parse('2026-01-01T05:00:00Z'))).toBe('5 小时')
    expect(formatAge(created, Date.parse('2026-01-04T00:00:00Z'))).toBe('3 天')
  })

  it('定时器读数是累计秒数加上本次已经过的秒数', () => {
    const running = card(1, {timer_seconds: 100, timer_started_at: '2026-01-01T00:00:00Z'})
    expect(timerSeconds(running, Date.parse('2026-01-01T00:01:00Z'))).toBe(160)
    expect(parseDuration('1:02:03')).toBe(3723)
    expect(parseDuration('abc')).toBeNull()
  })

  it('任务树按层级和顺序，进度计算所有层级', () => {
    const tasks = [
      {id: 1, card_id: 1, parent_id: null, title: '一', done: true, position: 2},
      {id: 2, card_id: 1, parent_id: null, title: '二', done: false, position: 1},
      {id: 3, card_id: 1, parent_id: 1, title: '一.一', done: true, position: 1},
      {id: 4, card_id: 9, parent_id: null, title: '别的卡片', done: false, position: 1},
    ]
    const tree = taskTree(tasks, 1)
    expect(tree.map((node) => node.task.id)).toEqual([2, 1])
    expect(tree[1].children[0].depth).toBe(2)
    expect(taskProgress(tasks, 1)).toEqual({done: 2, total: 3})
  })
})
