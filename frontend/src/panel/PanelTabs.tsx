import {
  DndContext,
  pointerWithin,
  useDraggable,
  useDroppable,
  type DragEndEvent,
  type DragMoveEvent,
} from '@dnd-kit/core'
import {useDragSensors} from '../hooks/useDragSensors'
import {useCallback, useState, type MouseEvent} from 'react'
import {Link} from 'react-router-dom'
import type {Panel} from '../api/types'
import {Menu, MenuItem, MenuSeparator} from '../components/Menu'
import {tabDropIndex} from './model'
import {t, useT} from '../i18n'

/** 标签页菜单中的操作。 */
export interface PanelTabActions {
  rename: (panel: Panel) => void
  toggleMain: (panel: Panel) => void
  options: (panel: Panel) => void
  moveToBoard: (panel: Panel) => void
  archive: (panel: Panel) => void
  /** 打开该面板的列表归档或卡片归档。 */
  openListArchive: (panel: Panel) => void
  openCardArchive: (panel: Panel) => void
  reorder: (panel: Panel, index: number) => void
  create: () => void
}

interface PanelTabsProps {
  boardId: number
  tabs: Panel[]
  activePanelId: number | null
  mainPanelId: number | null
  actions: PanelTabActions
  /** 只读时不能拖动、新建或打开标签页菜单。 */
  readOnly?: boolean
}

/** 拖动中的落点：放在某个标签页的前面或后面。 */
interface TabHint {
  overId: number
  after: boolean
}

/** 开始拖动时指针的横坐标，鼠标和触摸都适用。 */
function startX(event: Event): number {
  if ('touches' in event) {
    const touch = (event as TouchEvent).touches[0] ?? (event as TouchEvent).changedTouches[0]
    return touch?.clientX ?? 0
  }
  return (event as PointerEvent).clientX
}

/**
 * 面板标签页。可以拖动调整顺序：鼠标按下后移动 5 像素开始拖动，触屏长按 250 毫秒开始拖动。
 * 每个标签页的菜单提供重命名、设为主面板、标签和优先级、列表归档、卡片归档、移到其他看板、归档。
 */
export function PanelTabs({boardId, tabs, activePanelId, mainPanelId, actions, readOnly = false}: PanelTabsProps) {
  const t = useT()
  const [hint, setHint] = useState<TabHint | null>(null)
  const sensors = useDragSensors()

  function onDragMove(event: DragMoveEvent) {
    if (!event.over) {
      setHint(null)
      return
    }
    const x = startX(event.activatorEvent) + event.delta.x
    const rect = event.over.rect
    setHint({overId: Number(event.over.id), after: x > rect.left + rect.width / 2})
  }

  function onDragEnd(event: DragEndEvent) {
    const current = hint
    setHint(null)
    if (!current || !event.over) return
    const draggedId = Number(event.active.id)
    const index = tabDropIndex(tabs, draggedId, current.overId, current.after)
    const panel = tabs.find((p) => p.id === draggedId)
    if (index !== null && panel) {
      actions.reorder(panel, index)
    }
  }

  return (
    <DndContext sensors={sensors} collisionDetection={pointerWithin} onDragMove={onDragMove} onDragOver={onDragMove} onDragEnd={onDragEnd} onDragCancel={() => setHint(null)}>
      <nav className="panel-tabs" aria-label={t('common.panel')}>
        {tabs.map((panel) => (
          <PanelTab
            key={panel.id}
            boardId={boardId}
            panel={panel}
            active={panel.id === activePanelId}
            main={panel.id === mainPanelId}
            hint={hint?.overId === panel.id ? hint : null}
            actions={actions}
            readOnly={readOnly}
          />
        ))}
        {!readOnly && (
          <button type="button" className="tab-add" onClick={actions.create}>
            {t('panel.createShort')}
          </button>
        )}
      </nav>
    </DndContext>
  )
}

interface PanelTabProps {
  boardId: number
  panel: Panel
  active: boolean
  main: boolean
  hint: TabHint | null
  actions: PanelTabActions
  readOnly: boolean
}

function PanelTab({boardId, panel, active, main, hint, actions, readOnly}: PanelTabProps) {
  const {attributes, listeners, setNodeRef: setDragRef, isDragging} = useDraggable({id: panel.id, disabled: readOnly})
  const {setNodeRef: setDropRef} = useDroppable({id: panel.id})
  const setRef = useCallback(
    (element: HTMLElement | null) => {
      setDragRef(element)
      setDropRef(element)
    },
    [setDragRef, setDropRef],
  )
  const [menuOpen, setMenuOpen] = useState(false)
  let className = active ? 'tab active' : 'tab'
  if (isDragging) className += ' dragging'
  if (hint) className += hint.after ? ' drop-after' : ' drop-before'

  return (
    <Link
      ref={setRef}
      {...(readOnly ? {} : {...attributes, ...listeners})}
      to={`/board/${boardId}/panel/${panel.id}`}
      className={className}
      aria-current={active ? 'page' : undefined}
      onContextMenu={(event: MouseEvent) => {
        if (readOnly) return
        event.preventDefault()
        // 触屏长按拖动时系统也会触发 contextmenu，拖动期间不打开菜单。
        if (!isDragging) setMenuOpen(true)
      }}
    >
      {main && (
        <span className="home" title={t('panel.main')} aria-label={t('panel.main')}>
          ★
        </span>
      )}
      {panel.name}
      {!readOnly && (
      <Menu
        open={menuOpen}
        onOpenChange={setMenuOpen}
        align="start"
        trigger={
          <button
            type="button"
            className="tab-more"
            aria-label={t('common.menuOf', {name: panel.name})}
            onClick={(e) => {
              e.preventDefault()
              e.stopPropagation()
            }}
          >
            ⋯
          </button>
        }
      >
        <MenuItem onSelect={() => actions.rename(panel)}>{t('common.rename')}</MenuItem>
        <MenuItem onSelect={() => actions.toggleMain(panel)}>{main ? t('panel.unsetMain') : t('panel.setMain')}</MenuItem>
        <MenuItem onSelect={() => actions.options(panel)}>{t('panel.options')}</MenuItem>
        <MenuSeparator />
        <MenuItem onSelect={() => actions.openListArchive(panel)}>{t('list.archive')}</MenuItem>
        <MenuItem onSelect={() => actions.openCardArchive(panel)}>{t('card.archive')}</MenuItem>
        <MenuSeparator />
        <MenuItem onSelect={() => actions.moveToBoard(panel)}>{t('panel.moveToBoard')}</MenuItem>
        <MenuItem danger onSelect={() => actions.archive(panel)}>
          {t('common.archive')}
        </MenuItem>
      </Menu>
      )}
    </Link>
  )
}
