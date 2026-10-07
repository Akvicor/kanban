/**
 * 看板目录的结构计算：由同步数据中的文件夹和看板得出目录树、看板归档树、路径和可选的目标位置。
 * 排序规则与后端 service/directory.go 一致：同一父级下按位置排列，位置相同时文件夹在前，再按 ID。
 */
import type {Board, Folder} from '../api/types'
import {t} from '../i18n'

/** 文件夹的最大深度，与后端一致。根下第一层为 1，看板不增加深度。 */
export const MAX_FOLDER_DEPTH = 16

export type DirKind = 'folder' | 'board'

/** 目录树的节点。nav 为 true 表示这是看板归档中通向归档内容、但本身仍在目录中的文件夹。 */
export type DirNode =
  | {kind: 'folder'; folder: Folder; children: DirNode[]; nav: boolean}
  | {kind: 'board'; board: Board}

/** 目录中的一项，用于计算同级顺序。 */
export type DirItem = {kind: 'folder'; item: Folder} | {kind: 'board'; item: Board}

function parentOf(item: DirItem): number | null {
  return item.kind === 'folder' ? item.item.parent_id : item.item.folder_id
}

export function compareItems(a: DirItem, b: DirItem): number {
  if (a.item.position !== b.item.position) {
    return a.item.position - b.item.position
  }
  if (a.kind !== b.kind) {
    return a.kind === 'folder' ? -1 : 1
  }
  return a.item.id - b.item.id
}

function allItems(folders: Folder[], boards: Board[]): DirItem[] {
  return [
    ...folders.map((item) => ({kind: 'folder', item}) as DirItem),
    ...boards.map((item) => ({kind: 'board', item}) as DirItem),
  ]
}

/** 父级下未归档的文件夹和看板，按目录顺序，可以排除正在移动的那一项。 */
export function activeSiblings(
  folders: Folder[],
  boards: Board[],
  parentId: number | null,
  exclude?: {kind: DirKind; id: number},
): DirItem[] {
  return allItems(folders, boards)
    .filter((entry) => entry.item.archived_at === null && parentOf(entry) === parentId)
    .filter((entry) => !(exclude && entry.kind === exclude.kind && entry.item.id === exclude.id))
    .sort(compareItems)
}

/** 由一组项按父子关系建树，include 决定哪些项出现在树中。 */
function buildTree(items: DirItem[], include: (entry: DirItem) => boolean, nav: (folder: Folder) => boolean): DirNode[] {
  const byParent = new Map<number | null, DirItem[]>()
  for (const entry of items.filter(include)) {
    const parent = parentOf(entry)
    byParent.set(parent, [...(byParent.get(parent) ?? []), entry])
  }
  const build = (parent: number | null): DirNode[] =>
    (byParent.get(parent) ?? []).sort(compareItems).map((entry) =>
      entry.kind === 'folder'
        ? {kind: 'folder', folder: entry.item, children: build(entry.item.id), nav: nav(entry.item)}
        : {kind: 'board', board: entry.item},
    )
  return build(null)
}

/** 目录：未归档的文件夹和看板。 */
export function buildDirectory(folders: Folder[], boards: Board[]): DirNode[] {
  return buildTree(allItems(folders, boards), (entry) => entry.item.archived_at === null, () => false)
}

/**
 * 看板归档：已归档的文件夹和看板按归档前的层级展示；
 * 通向它们、但本身仍在目录中的文件夹作为导航节点出现。
 */
export function buildArchive(folders: Folder[], boards: Board[]): DirNode[] {
  const byId = new Map(folders.map((folder) => [folder.id, folder]))
  const navFolders = new Set<number>()
  const markAncestors = (parentId: number | null) => {
    let current = parentId
    while (current !== null && !navFolders.has(current)) {
      const folder = byId.get(current)
      if (!folder) break
      if (folder.archived_at === null) navFolders.add(current)
      current = folder.parent_id
    }
  }
  for (const entry of allItems(folders, boards)) {
    if (entry.item.archived_at !== null) markAncestors(parentOf(entry))
  }
  return buildTree(
    allItems(folders, boards),
    (entry) => entry.item.archived_at !== null || (entry.kind === 'folder' && navFolders.has(entry.item.id)),
    (folder) => folder.archived_at === null,
  )
}

/** 从根到 folderId 的文件夹路径，folderId 为 null 时为空。 */
export function folderPath(folders: Folder[], folderId: number | null): Folder[] {
  const byId = new Map(folders.map((folder) => [folder.id, folder]))
  const path: Folder[] = []
  let current = folderId
  while (current !== null && path.length <= MAX_FOLDER_DEPTH) {
    const folder = byId.get(current)
    if (!folder) break
    path.unshift(folder)
    current = folder.parent_id
  }
  return path
}

/** folderId 自己及其全部下级文件夹的 ID。 */
export function descendantFolderIds(folders: Folder[], folderId: number): Set<number> {
  const result = new Set([folderId])
  let added = true
  while (added) {
    added = false
    for (const folder of folders) {
      if (folder.parent_id !== null && result.has(folder.parent_id) && !result.has(folder.id)) {
        result.add(folder.id)
        added = true
      }
    }
  }
  return result
}

/** 以 folderId 为根的子树层数，只有它自己时为 1。已归档的下级也计入，与后端一致。 */
export function subtreeHeight(folders: Folder[], folderId: number): number {
  const children = folders.filter((folder) => folder.parent_id === folderId)
  return 1 + Math.max(0, ...children.map((child) => subtreeHeight(folders, child.id)))
}

/** 拖动到某一项上的落点：放在它前面、后面，或放进它里面（只对文件夹有效）。 */
export type DropIntent = 'before' | 'after' | 'inside'

/** 落点换算成的目标位置。index 为 null 表示排在末尾。 */
export interface DropTarget {
  parentId: number | null
  index: number | null
}

/**
 * 把「拖动 dragged 到 over 的 intent 处」换算成目标父级和同级位置。
 * 不能放的位置（放到自己身上、文件夹放进自己或下级、超过 16 层）以及没有变化的位置返回 null。
 * over 为 null 表示拖到目录空白处，放到根的末尾。
 */
export function resolveDrop(
  folders: Folder[],
  boards: Board[],
  dragged: {kind: DirKind; id: number},
  over: {kind: DirKind; id: number} | null,
  intent: DropIntent,
): DropTarget | null {
  if (over && over.kind === dragged.kind && over.id === dragged.id) {
    return null
  }
  let target: DropTarget
  if (!over) {
    target = {parentId: null, index: null}
  } else if (intent === 'inside' && over.kind === 'folder') {
    target = {parentId: over.id, index: null}
  } else {
    const overItem = over.kind === 'folder' ? folders.find((f) => f.id === over.id) : boards.find((b) => b.id === over.id)
    if (!overItem) return null
    const parentId = over.kind === 'folder' ? (overItem as Folder).parent_id : (overItem as Board).folder_id
    const siblings = activeSiblings(folders, boards, parentId, dragged)
    const at = siblings.findIndex((entry) => entry.kind === over.kind && entry.item.id === over.id)
    target = {parentId, index: intent === 'before' ? at : at + 1}
  }

  if (dragged.kind === 'folder') {
    if (target.parentId !== null && descendantFolderIds(folders, dragged.id).has(target.parentId)) {
      return null
    }
    const depth = target.parentId === null ? 0 : folderPath(folders, target.parentId).length
    if (depth + subtreeHeight(folders, dragged.id) > MAX_FOLDER_DEPTH) {
      return null
    }
  }

  // 位置没有变化时不发请求。
  const current =
    dragged.kind === 'folder' ? folders.find((f) => f.id === dragged.id) : boards.find((b) => b.id === dragged.id)
  if (!current) return null
  const currentParent = dragged.kind === 'folder' ? (current as Folder).parent_id : (current as Board).folder_id
  if (currentParent === target.parentId) {
    const siblings = activeSiblings(folders, boards, currentParent, dragged)
    const currentIndex = activeSiblings(folders, boards, currentParent).findIndex(
      (entry) => entry.kind === dragged.kind && entry.item.id === dragged.id,
    )
    const targetIndex = target.index ?? siblings.length
    if (targetIndex === currentIndex) return null
  }
  return target
}

/** 可以选择的看板：目录中的全部看板，按目录顺序，带上所在文件夹的路径。 */
export interface BoardOption {
  board: Board
  /** 例如「工作 / 项目 A / 迭代」。 */
  label: string
}

export function boardOptions(folders: Folder[], boards: Board[]): BoardOption[] {
  const options: BoardOption[] = []
  const visit = (nodes: DirNode[], path: string[]) => {
    for (const node of nodes) {
      if (node.kind === 'folder') {
        visit(node.children, [...path, node.folder.name])
      } else {
        options.push({board: node.board, label: [...path, node.board.name].join(' / ')})
      }
    }
  }
  visit(buildDirectory(folders, boards), [])
  return options
}

/** 可以作为目标父级的位置：根，以及按目录顺序展开的全部未归档文件夹。 */
export interface ParentOption {
  id: number | null
  name: string
  /** 根为 0，根下第一层为 1。 */
  depth: number
  /** 不能放到这里的原因，可以放时为 null。 */
  disabledReason: string | null
}

/**
 * 列出移动或恢复时可选的目标父级。moving 是要移动的文件夹或看板：
 * 文件夹不能移到自己或下级中，移动后子树深度不能超过 16 层；看板没有这些限制。
 */
export function parentOptions(folders: Folder[], boards: Board[], moving?: {kind: DirKind; id: number}): ParentOption[] {
  const blocked = moving?.kind === 'folder' ? descendantFolderIds(folders, moving.id) : new Set<number>()
  const height = moving?.kind === 'folder' ? subtreeHeight(folders, moving.id) : 0
  const options: ParentOption[] = [{id: null, name: t('directory.root'), depth: 0, disabledReason: null}]
  const visit = (nodes: DirNode[], depth: number) => {
    for (const node of nodes) {
      if (node.kind !== 'folder') continue
      let disabledReason: string | null = null
      if (blocked.has(node.folder.id)) {
        disabledReason = t('directory.moveIntoSelf')
      } else if (depth + height > MAX_FOLDER_DEPTH) {
        disabledReason = t('directory.tooDeep')
      }
      options.push({id: node.folder.id, name: node.folder.name, depth, disabledReason})
      visit(node.children, depth + 1)
    }
  }
  visit(buildDirectory(folders, boards), 1)
  return options
}
