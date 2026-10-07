import {describe, expect, it} from 'vitest'
import type {List} from '../api/types'
import {activeLists, archivedLists, listDropIndex, sortLabel} from './model'

function list(id: number, panelId: number, position: number, archivedAt: string | null = null): List {
  const op = {add_labels: [], remove_labels: [], start: '' as const, complete: '' as const}
  return {
    id,
    panel_id: panelId,
    name: `L${id}`,
    color: '',
    position,
    show_age: true,
    sort_mode: 'manual',
    sort_dir: 'asc',
    head_add: 'head',
    tail_add: 'tail',
    remind_off: false,
    due_off: false,
    rules: {create: op, enter: op, exit: op},
    archived_at: archivedAt,
  }
}

describe('activeLists / archivedLists', () => {
  const lists = [
    list(1, 10, 300),
    list(2, 10, 100),
    list(3, 10, 200, '2026-01-02T00:00:00Z'),
    list(4, 10, 50, '2026-01-03T00:00:00Z'),
    list(5, 20, 10),
  ]
  it('按面板分开，未归档的从左到右，归档的最近在前', () => {
    expect(activeLists(lists, 10).map((l) => l.id)).toEqual([2, 1])
    expect(archivedLists(lists, 10).map((l) => l.id)).toEqual([4, 3])
  })
})

describe('sortLabel', () => {
  it('默认排序不显示方向，其他方式显示方式、方向和排序', () => {
    expect(sortLabel('manual', 'desc')).toBe('默认排序')
    expect(sortLabel('due_at', 'asc')).toBe('截止时间·正序排序')
    expect(sortLabel('priority', 'desc')).toBe('优先级·倒序排序')
  })
})

describe('listDropIndex', () => {
  const lists = [list(1, 10, 100), list(2, 10, 200), list(3, 10, 300)]
  it('换算成在其他列表中的位置，位置不变时返回 null', () => {
    expect(listDropIndex(lists, 3, 1, false)).toBe(0)
    expect(listDropIndex(lists, 1, 3, true)).toBe(2)
    expect(listDropIndex(lists, 2, 1, true)).toBeNull()
    expect(listDropIndex(lists, 2, 2, false)).toBeNull()
  })
})
