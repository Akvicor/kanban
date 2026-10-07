import {describe, expect, it} from 'vitest'
import type {Board, Panel} from '../api/types'
import {boardTabs, entryPanelId, tabDropIndex} from './model'

function panel(id: number, boardId: number, position: number, archived = false): Panel {
  return {id, board_id: boardId, name: `P${id}`, position, archived_at: archived ? '2026-01-01T00:00:00Z' : null}
}

const board = (mainPanelId: number | null): Board => ({
  id: 1,
  folder_id: null,
  name: '看板',
  position: 1,
  main_panel_id: mainPanelId,
  archived_at: null,
  created_at: '2026-01-01T00:00:00Z',
})

describe('boardTabs', () => {
  it('只包含该看板未归档的面板，按顺序排列', () => {
    const panels = [panel(1, 1, 300), panel(2, 1, 100), panel(3, 1, 200, true), panel(4, 2, 50)]
    expect(boardTabs(panels, 1).map((p) => p.id)).toEqual([2, 1])
  })
})

describe('entryPanelId', () => {
  const tabs = [panel(2, 1, 100), panel(1, 1, 300)]
  it('有主面板时进入主面板，否则进入第一个面板', () => {
    expect(entryPanelId(board(1), tabs)).toBe(1)
    expect(entryPanelId(board(null), tabs)).toBe(2)
  })
  it('主面板不在标签页中时进入第一个面板；没有面板时为 null', () => {
    expect(entryPanelId(board(9), tabs)).toBe(2)
    expect(entryPanelId(board(null), [])).toBeNull()
  })
})

describe('tabDropIndex', () => {
  const tabs = [panel(1, 1, 100), panel(2, 1, 200), panel(3, 1, 300)]
  it('换算成在其他标签页中的位置', () => {
    expect(tabDropIndex(tabs, 3, 1, false)).toBe(0)
    expect(tabDropIndex(tabs, 1, 3, true)).toBe(2)
    expect(tabDropIndex(tabs, 1, 2, true)).toBe(1)
  })
  it('位置没有变化时返回 null', () => {
    expect(tabDropIndex(tabs, 2, 2, true)).toBeNull()
    expect(tabDropIndex(tabs, 2, 1, true)).toBeNull()
    expect(tabDropIndex(tabs, 2, 3, false)).toBeNull()
  })
})
