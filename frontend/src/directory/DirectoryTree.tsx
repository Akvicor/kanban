import {
  DndContext,
  pointerWithin,
  useDraggable,
  useDroppable,
  type DragEndEvent,
  type DragMoveEvent,
} from '@dnd-kit/core'
import {useDragSensors} from '../hooks/useDragSensors'
import {createContext, useCallback, useContext, useMemo, useState, type MouseEvent} from 'react'
import {Link} from 'react-router-dom'
import type {Board, Folder} from '../api/types'
import {Menu, MenuItem, MenuSeparator} from '../components/Menu'
import {useDirectoryData, useSettings} from '../sync/hooks'
import {useDirectoryActions, type DirTarget} from './actions'
import {useExpandedFolders} from './expanded'
import {buildDirectory, folderPath, resolveDrop, type DirKind, type DirNode, type DropIntent} from './tree'
import './DirectoryTree.css'
import {useT} from '../i18n'

/** 拖动中的落点，用于在目标行上显示提示。 */
interface DropHint {
  overId: string
  intent: DropIntent
}

const DropHintContext = createContext<DropHint | null>(null)

/** 根目录空白处的放置区，拖到这里时放到根的末尾。 */
const ROOT_DROP_ID = 'root'

function dragId(kind: DirKind, id: number): string {
  return `${kind}:${id}`
}

function parseDragId(value: string): {kind: DirKind; id: number} | null {
  const [kind, id] = value.split(':')
  return kind === 'folder' || kind === 'board' ? {kind, id: Number(id)} : null
}

/** 拖动开始时指针的纵坐标，鼠标和触摸都适用。 */
function startY(event: Event): number {
  if ('touches' in event) {
    const touch = (event as TouchEvent).touches[0] ?? (event as TouchEvent).changedTouches[0]
    return touch?.clientY ?? 0
  }
  return (event as PointerEvent).clientY
}

/** 按指针在目标行中的高度决定落点：文件夹上下各四分之一是前后，中间是放进去；看板只分前后。 */
function intentAt(y: number, rect: {top: number; height: number}, kind: DirKind): DropIntent {
  const ratio = (y - rect.top) / rect.height
  if (kind === 'folder') {
    if (ratio < 0.25) return 'before'
    if (ratio > 0.75) return 'after'
    return 'inside'
  }
  return ratio < 0.5 ? 'before' : 'after'
}

function findTarget(folders: Folder[], boards: Board[], ref: {kind: DirKind; id: number}): DirTarget | null {
  if (ref.kind === 'folder') {
    const folder = folders.find((f) => f.id === ref.id)
    return folder ? {kind: 'folder', folder} : null
  }
  const board = boards.find((b) => b.id === ref.id)
  return board ? {kind: 'board', board} : null
}

interface DirectoryTreeProps {
  /** 当前打开的看板，在目录中高亮并展开它所在的文件夹。 */
  activeBoardId: number | null
}

/**
 * 看板目录：文件夹和看板组成的树。可以拖动调整顺序和所属文件夹；每一项的菜单提供新建、重命名、移动、归档等操作。
 * 鼠标拖动 5 像素后开始拖动；触屏长按 250 毫秒开始拖动，避免与滚动冲突。
 */
export function DirectoryTree({activeBoardId}: DirectoryTreeProps) {
  const t = useT()
  const {folders, boards} = useDirectoryData()
  const actions = useDirectoryActions()
  const tree = useMemo(() => buildDirectory(folders, boards), [folders, boards])
  const {expanded, setOpen} = useExpandedFolders()
  const [hint, setHint] = useState<DropHint | null>(null)
  const sensors = useDragSensors()

  // 已展开的文件夹，加上当前看板所在的各级文件夹。
  const isOpen = useMemo(() => {
    const board = boards.find((b) => b.id === activeBoardId)
    const ancestors = new Set(folderPath(folders, board?.folder_id ?? null).map((folder) => folder.id))
    return (id: number) => expanded.has(id) || ancestors.has(id)
  }, [folders, boards, activeBoardId, expanded])
  const view: TreeView = {activeBoardId, isOpen, setOpen}

  function onDragMove(event: DragMoveEvent) {
    const over = event.over
    if (!over) {
      setHint(null)
      return
    }
    if (over.id === ROOT_DROP_ID) {
      setHint({overId: ROOT_DROP_ID, intent: 'after'})
      return
    }
    const target = parseDragId(String(over.id))
    if (!target) return
    const y = startY(event.activatorEvent) + event.delta.y
    setHint({overId: String(over.id), intent: intentAt(y, over.rect, target.kind)})
  }

  function onDragEnd(event: DragEndEvent) {
    const currentHint = hint
    setHint(null)
    const dragged = parseDragId(String(event.active.id))
    if (!dragged || !event.over || !currentHint || currentHint.overId !== String(event.over.id)) {
      return
    }
    const over = currentHint.overId === ROOT_DROP_ID ? null : parseDragId(currentHint.overId)
    const target = resolveDrop(folders, boards, dragged, over, currentHint.intent)
    const item = findTarget(folders, boards, dragged)
    if (target && item) {
      actions.place(item, target.parentId, target.index)
    }
  }

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={pointerWithin}
      onDragMove={onDragMove}
      onDragOver={onDragMove}
      onDragEnd={onDragEnd}
      onDragCancel={() => setHint(null)}
    >
      <DropHintContext.Provider value={hint}>
        <div className="dir-tree" role="tree">
          {tree.length === 0 && <p className="dir-empty">{t('directory.noBoards')}</p>}
          <TreeNodes nodes={tree} depth={0} view={view} />
          <RootDropZone />
        </div>
      </DropHintContext.Provider>
    </DndContext>
  )
}

/** 各层共用的展示状态：当前看板和文件夹展开状态。 */
interface TreeView {
  activeBoardId: number | null
  isOpen: (folderId: number) => boolean
  setOpen: (folderId: number, open: boolean) => void
}

function TreeNodes({nodes, depth, view}: {nodes: DirNode[]; depth: number; view: TreeView}) {
  return (
    <>
      {nodes.map((node) =>
        node.kind === 'folder' ? (
          <FolderNode key={`f${node.folder.id}`} folder={node.folder} nodes={node.children} depth={depth} view={view} />
        ) : (
          <BoardRow key={`b${node.board.id}`} board={node.board} depth={depth} active={node.board.id === view.activeBoardId} />
        ),
      )}
    </>
  )
}

/** 一行目录项既可以拖动，也可以作为放置目标；返回合并后的 ref、拖动属性、包含落点提示的类名，以及是否正在拖动。 */
function useRow(kind: DirKind, id: number) {
  const draggableId = dragId(kind, id)
  const {attributes, listeners, setNodeRef: setDragRef, isDragging} = useDraggable({id: draggableId})
  const {setNodeRef: setDropRef} = useDroppable({id: draggableId})
  const hint = useContext(DropHintContext)
  const setRef = useCallback(
    (element: HTMLElement | null) => {
      setDragRef(element)
      setDropRef(element)
    },
    [setDragRef, setDropRef],
  )
  const dropClass = hint?.overId === draggableId ? ` drop-${hint.intent}` : ''
  return [setRef, {...attributes, ...listeners}, `dir-row${isDragging ? ' dragging' : ''}${dropClass}`, isDragging] as const
}

function FolderNode({folder, nodes, depth, view}: {folder: Folder; nodes: DirNode[]; depth: number; view: TreeView}) {
  const t = useT()
  const [setRowRef, dragProps, rowClass, isDragging] = useRow('folder', folder.id)
  const actions = useDirectoryActions()
  const [menuOpen, setMenuOpen] = useState(false)
  const target: DirTarget = {kind: 'folder', folder}
  const open = view.isOpen(folder.id)
  const onToggle = (next: boolean) => view.setOpen(folder.id, next)

  return (
    <div role="treeitem" aria-expanded={open}>
      <div
        ref={setRowRef}
        {...dragProps}
        className={rowClass}
        style={{paddingLeft: 8 + depth * 18}}
        onClick={() => onToggle(!open)}
        onContextMenu={(event: MouseEvent) => {
          event.preventDefault()
          // 触屏长按拖动时系统也会触发 contextmenu，拖动期间不打开菜单。
          if (!isDragging) setMenuOpen(true)
        }}
      >
        <span className={open ? 'dir-caret open' : 'dir-caret'} aria-hidden>
          ▸
        </span>
        <span className="dir-icon folder" aria-hidden />
        <span className="dir-name">{folder.name}</span>
        <Menu
          open={menuOpen}
          onOpenChange={setMenuOpen}
          trigger={
            <button type="button" className="dir-more" aria-label={t('common.menuOf', {name: folder.name})} onClick={(e) => e.stopPropagation()}>
              ⋯
            </button>
          }
        >
          <MenuItem onSelect={() => actions.createBoard(folder.id)}>{t('directory.createBoardHere')}</MenuItem>
          <MenuItem onSelect={() => actions.createFolder(folder.id)}>{t('directory.createFolderHere')}</MenuItem>
          <MenuSeparator />
          <MenuItem onSelect={() => actions.rename(target)}>{t('common.rename')}</MenuItem>
          <MenuItem onSelect={() => actions.move(target)}>{t('directory.moveTo')}</MenuItem>
          <MenuSeparator />
          <MenuItem danger onSelect={() => actions.archive(target)}>
            {t('common.archive')}
          </MenuItem>
        </Menu>
      </div>
      {open && (
        <div role="group">
          <TreeNodes nodes={nodes} depth={depth + 1} view={view} />
        </div>
      )}
    </div>
  )
}

function BoardRow({board, depth, active}: {board: Board; depth: number; active: boolean}) {
  const t = useT()
  const [setRowRef, dragProps, rowClass, isDragging] = useRow('board', board.id)
  const actions = useDirectoryActions()
  const settings = useSettings()
  const [menuOpen, setMenuOpen] = useState(false)
  const isMain = settings.main_board_id === board.id
  const target: DirTarget = {kind: 'board', board}

  return (
    <div role="treeitem" aria-selected={active}>
      <Link
        ref={setRowRef}
        {...dragProps}
        to={`/board/${board.id}`}
        className={`${rowClass}${active ? ' active' : ''}`}
        style={{paddingLeft: 8 + depth * 18 + 14}}
        onContextMenu={(event: MouseEvent) => {
          event.preventDefault()
          // 触屏长按拖动时系统也会触发 contextmenu，拖动期间不打开菜单。
          if (!isDragging) setMenuOpen(true)
        }}
      >
        <span className="dir-icon board" aria-hidden />
        <span className="dir-name">{board.name}</span>
        {isMain && (
          <span className="dir-star" title={t('directory.mainBoard')} aria-label={t('directory.mainBoard')}>
            ★
          </span>
        )}
        <Menu
          open={menuOpen}
          onOpenChange={setMenuOpen}
          trigger={
            <button
              type="button"
              className="dir-more"
              aria-label={t('common.menuOf', {name: board.name})}
              onClick={(e) => {
                e.preventDefault()
                e.stopPropagation()
              }}
            >
              ⋯
            </button>
          }
        >
          <MenuItem onSelect={() => actions.toggleMain(board)}>{isMain ? t('directory.unsetMainBoard') : t('directory.setMainBoard')}</MenuItem>
          <MenuSeparator />
          <MenuItem onSelect={() => actions.rename(target)}>{t('common.rename')}</MenuItem>
          <MenuItem onSelect={() => actions.move(target)}>{t('directory.moveTo')}</MenuItem>
          <MenuSeparator />
          <MenuItem danger onSelect={() => actions.archive(target)}>
            {t('common.archive')}
          </MenuItem>
        </Menu>
      </Link>
    </div>
  )
}

function RootDropZone() {
  const {setNodeRef} = useDroppable({id: ROOT_DROP_ID})
  const hint = useContext(DropHintContext)
  return <div ref={setNodeRef} className={hint?.overId === ROOT_DROP_ID ? 'dir-root-drop active' : 'dir-root-drop'} />
}
