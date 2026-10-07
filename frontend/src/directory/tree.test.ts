import {describe, expect, it} from 'vitest'
import type {Board, Folder} from '../api/types'
import {buildArchive, buildDirectory, folderPath, parentOptions, resolveDrop, type DirNode} from './tree'

const ARCHIVED = '2026-01-01T00:00:00Z'

function folder(id: number, parent: number | null, position: number, archived = false): Folder {
  return {id, parent_id: parent, name: `F${id}`, position, archived_at: archived ? ARCHIVED : null}
}

function board(id: number, folderId: number | null, position: number, archived = false): Board {
  return {id, folder_id: folderId, name: `B${id}`, position, main_panel_id: null, archived_at: archived ? ARCHIVED : null, created_at: ARCHIVED}
}

/** 把树压平成「层级缩进 + 名称」，便于断言。 */
function outline(nodes: DirNode[], depth = 0): string[] {
  return nodes.flatMap((node) =>
    node.kind === 'folder'
      ? [`${'  '.repeat(depth)}${node.folder.name}${node.nav ? '(导航)' : ''}`, ...outline(node.children, depth + 1)]
      : [`${'  '.repeat(depth)}${node.board.name}`],
  )
}

describe('buildDirectory', () => {
  it('同级文件夹和看板共用排序，位置相同时文件夹在前，已归档的不出现', () => {
    const folders = [folder(1, null, 200), folder(2, 1, 100), folder(3, null, 100, true)]
    const boards = [board(10, null, 100), board(11, null, 200), board(12, 1, 50), board(13, null, 300, true)]
    expect(outline(buildDirectory(folders, boards))).toEqual(['B10', 'F1', '  B12', '  F2', 'B11'])
  })
})

describe('buildArchive', () => {
  it('按原层级展示归档内容，路径上仍在目录中的文件夹作为导航节点', () => {
    const folders = [folder(1, null, 100), folder(2, 1, 100, true), folder(3, null, 200)]
    const boards = [board(10, 2, 100, true), board(11, 1, 200, true), board(12, 3, 100), board(13, 1, 300)]
    expect(outline(buildArchive(folders, boards))).toEqual(['F1(导航)', '  F2', '    B10', '  B11'])
  })
})

describe('folderPath', () => {
  it('返回从根到文件夹的路径', () => {
    const folders = [folder(1, null, 1), folder(2, 1, 1), folder(3, 2, 1)]
    expect(folderPath(folders, 3).map((f) => f.id)).toEqual([1, 2, 3])
    expect(folderPath(folders, null)).toEqual([])
  })
})

describe('resolveDrop', () => {
  // 根：B10(100)、F1(200)、B11(300)；F1 中：B12(100)
  const folders = [folder(1, null, 200)]
  const boards = [board(10, null, 100), board(11, null, 300), board(12, 1, 100)]
  const b = (id: number) => ({kind: 'board' as const, id})
  const f = (id: number) => ({kind: 'folder' as const, id})

  it('放到某一项前后时换算成同级位置，位置不计被拖动的项', () => {
    expect(resolveDrop(folders, boards, b(11), b(10), 'before')).toEqual({parentId: null, index: 0})
    expect(resolveDrop(folders, boards, b(10), b(11), 'after')).toEqual({parentId: null, index: 2})
    expect(resolveDrop(folders, boards, b(12), b(10), 'after')).toEqual({parentId: null, index: 1})
  })

  it('放进文件夹时排在末尾，拖到空白处时放到根的末尾', () => {
    expect(resolveDrop(folders, boards, b(10), f(1), 'inside')).toEqual({parentId: 1, index: null})
    expect(resolveDrop(folders, boards, b(12), null, 'after')).toEqual({parentId: null, index: null})
  })

  it('没有变化或不能放的位置返回 null', () => {
    expect(resolveDrop(folders, boards, b(10), b(10), 'after')).toBeNull()
    expect(resolveDrop(folders, boards, b(10), f(1), 'before')).toBeNull()
    expect(resolveDrop(folders, boards, f(1), f(1), 'inside')).toBeNull()
    const nested = [...folders, folder(2, 1, 50)]
    expect(resolveDrop(nested, boards, f(1), f(2), 'inside')).toBeNull()
  })
})

describe('parentOptions', () => {
  it('移动文件夹时不能选自己和下级，也不能超过 16 层', () => {
    const chain = Array.from({length: 16}, (_, i) => folder(i + 1, i === 0 ? null : i, 1))
    const moving = folder(100, null, 2)
    const child = folder(101, 100, 1)
    const options = parentOptions([...chain, moving, child], [], {kind: 'folder', id: 100})

    const byId = new Map(options.map((option) => [option.id, option]))
    expect(byId.get(100)?.disabledReason).toBe('不能移到自己或下级中')
    expect(byId.get(101)?.disabledReason).toBe('不能移到自己或下级中')
    // 被移动的文件夹连同下级共 2 层：放到第 14 层下刚好 16 层，放到第 15 层下超过。
    expect(byId.get(14)?.disabledReason).toBeNull()
    expect(byId.get(15)?.disabledReason).toBe('超过 16 层')
    expect(byId.get(null)?.disabledReason).toBeNull()
  })
})
