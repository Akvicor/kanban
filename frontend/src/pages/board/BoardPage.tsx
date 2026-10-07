import {useMemo, useState, type ReactNode} from 'react'
import {Navigate, useNavigate, useParams} from 'react-router-dom'
import * as panelApi from '../../api/panel'
import {errorMessage, type WriteResult} from '../../api/client'
import type {Panel} from '../../api/types'
import {ConfirmDialog} from '../../components/ConfirmDialog'
import {Menu, MenuItem, MenuSeparator} from '../../components/Menu'
import {useToast} from '../../components/Toast'
import {useDirectoryActions} from '../../directory/actions'
import {NameDialog} from '../../directory/NameDialog'
import {MenuButton} from '../../layout/MenuButton'
import {PanelArchives, type PanelArchiveKind} from '../../panel/PanelArchives'
import {folderPath} from '../../directory/tree'
import {PanelLists} from '../../list/PanelLists'
import {BoardSelectDialog} from '../../panel/BoardSelectDialog'
import {boardTabs, entryPanelId} from '../../panel/model'
import {PanelOptionsDialog} from '../../panel/PanelOptionsDialog'
import {PanelTabs, type PanelTabActions} from '../../panel/PanelTabs'
import {useDirectoryData, usePanelData, useSettings, useSyncStore} from '../../sync/hooks'
import {EntityType} from '../../sync/store'
import './BoardPage.css'
import {useT} from '../../i18n'

/** 当前打开的弹窗。 */
type Pending =
  | {type: 'create'}
  | {type: 'rename'; panel: Panel}
  | {type: 'options'; panel: Panel}
  | {type: 'move'; panel: Panel}
  | {type: 'archive'; panel: Panel}

/** 打开某个面板的列表归档或卡片归档。 */
interface ArchiveView {
  panel: Panel
  kind: PanelArchiveKind
}

/**
 * 看板页：顶部是路径、名称和看板菜单，下面是面板标签页和当前面板的内容。
 * 地址中没有面板，或面板不在这个看板的标签页中时，进入主面板，没有主面板时进入第一个面板。
 */
export function BoardPage() {
  const t = useT()
  const params = useParams()
  const boardId = Number(params.boardId)
  const panelId = params.panelId === undefined ? null : Number(params.panelId)
  const store = useSyncStore()
  const {folders, boards} = useDirectoryData()
  const {panels} = usePanelData()
  const settings = useSettings()
  const directoryActions = useDirectoryActions()
  const navigate = useNavigate()
  const {show: showToast, element: toastElement} = useToast()
  const [pending, setPending] = useState<Pending | null>(null)
  const [archiveView, setArchiveView] = useState<ArchiveView | null>(null)
  const board = boards.find((b) => b.id === boardId)
  const tabs = useMemo(() => boardTabs(panels, boardId), [panels, boardId])

  if (!board) {
    return (
      <div className="page">
        <div className="page-header">
          <MenuButton />
        </div>
        <p className="hint">{t('board.notFound')}</p>
      </div>
    )
  }
  const requested = panelId === null ? undefined : panels.find((panel) => panel.id === panelId && panel.board_id === board.id)
  const archivedPanel = requested?.archived_at ? requested : undefined
  const liveTabs = tabs
  const viewTabs = archivedPanel ? [archivedPanel] : liveTabs
  const readOnly = board.archived_at !== null || archivedPanel !== undefined
  if (!archivedPanel && !viewTabs.some((panel) => panel.id === panelId)) {
    const entry = entryPanelId(board, liveTabs)
    if (entry !== null) {
      return <Navigate to={`/board/${board.id}/panel/${entry}`} replace />
    }
  }
  const activePanel = archivedPanel ?? viewTabs.find((panel) => panel.id === panelId)

  const close = () => setPending(null)
  const apply = (type: string, result: WriteResult<{id: number}>) => store.applyLocal(type, result.data.id, result.data, result.revision)
  const run = (operation: () => Promise<void>) => {
    operation().catch((err: unknown) => showToast(errorMessage(err)))
  }

  const tabActions: PanelTabActions = {
    create: () => setPending({type: 'create'}),
    rename: (panel) => setPending({type: 'rename', panel}),
    options: (panel) => setPending({type: 'options', panel}),
    moveToBoard: (panel) => setPending({type: 'move', panel}),
    archive: (panel) => setPending({type: 'archive', panel}),
    openListArchive: (panel) => setArchiveView({panel, kind: 'list'}),
    openCardArchive: (panel) => setArchiveView({panel, kind: 'card'}),
    toggleMain: (panel) =>
      run(async () => apply(EntityType.Board, await panelApi.setMainPanel(board.id, board.main_panel_id === panel.id ? null : panel.id))),
    reorder: (panel, index) => run(async () => apply(EntityType.Panel, await panelApi.reorderPanel(panel.id, index))),
  }

  let dialog: ReactNode = null
  if (pending?.type === 'create') {
    dialog = (
      <NameDialog
        title={t('board.createPanel')}
        initialName=""
        confirmText={t('common.create')}
        onClose={close}
        onSubmit={async (name) => {
          const result = await panelApi.createPanel(board.id, name)
          apply(EntityType.Panel, result)
          navigate(`/board/${board.id}/panel/${result.data.id}`)
        }}
      />
    )
  } else if (pending?.type === 'rename') {
    const {panel} = pending
    dialog = (
      <NameDialog
        title={t('board.renamePanel')}
        initialName={panel.name}
        confirmText={t('common.save')}
        onClose={close}
        onSubmit={async (name) => apply(EntityType.Panel, await panelApi.renamePanel(panel.id, name))}
      />
    )
  } else if (pending?.type === 'options') {
    dialog = <PanelOptionsDialog panel={pending.panel} onClose={close} />
  } else if (pending?.type === 'move') {
    const {panel} = pending
    dialog = (
      <BoardSelectDialog
        title={t('board.movePanelTitle', {name: panel.name})}
        description={t('board.movePanelDesc')}
        confirmText={t('common.move')}
        initialBoardId={null}
        excludeBoardId={board.id}
        onClose={close}
        onSubmit={async (targetId) => apply(EntityType.Panel, await panelApi.movePanel(panel.id, targetId))}
      />
    )
  } else if (pending?.type === 'archive') {
    const {panel} = pending
    dialog = (
      <ConfirmDialog
        open
        title={t('board.archivePanelTitle', {name: panel.name})}
        description={t('board.archivePanelDesc')}
        confirmText={t('common.archive')}
        danger
        onCancel={close}
        onConfirm={() => {
          close()
          run(async () => {
            await panelApi.archivePanel(panel.id)
          })
        }}
      />
    )
  }

  const path = folderPath(folders, board.folder_id)
  const isMain = settings.main_board_id === board.id
  const boardTarget = {kind: 'board' as const, board}

  return (
    <div className="board-page">
      <header className="topbar">
        <MenuButton />
        <div className="crumbs">
          {path.map((folder) => (
            <span key={folder.id} className="dim">
              {folder.name}
              <span className="sep"> / </span>
            </span>
          ))}
          <strong>{board.name}</strong>
          {isMain && <span className="badge-main">{t('directory.mainBoard')}</span>}
        </div>
        <div className="topbar-actions">
          {!readOnly && (
          <button
            type="button"
            className={isMain ? 'icon-btn starred' : 'icon-btn'}
            aria-label={isMain ? t('directory.unsetMainBoard') : t('directory.setMainBoard')}
            title={isMain ? t('directory.unsetMainBoard') : t('directory.setMainBoard')}
            onClick={() => directoryActions.toggleMain(board)}
          >
            {isMain ? '★' : '☆'}
          </button>
          )}
          {!readOnly && (
          <Menu
            trigger={
              <button type="button" className="icon-btn" aria-label={t('board.menu')}>
                ⋯
              </button>
            }
          >
            <MenuItem onSelect={() => directoryActions.rename(boardTarget)}>{t('directory.renameBoard')}</MenuItem>
            <MenuItem onSelect={() => directoryActions.move(boardTarget)}>{t('directory.moveTo')}</MenuItem>
            <MenuSeparator />
            <MenuItem danger onSelect={() => directoryActions.archive(boardTarget)}>
              {t('board.archiveBoard')}
            </MenuItem>
          </Menu>
          )}
        </div>
      </header>
      {readOnly && (
        <p className="read-only-banner">{board.archived_at ? t('card.boardArchivedReadOnly') : t('card.panelArchivedReadOnly')}</p>
      )}
      {viewTabs.length > 0 && (
        <PanelTabs boardId={board.id} tabs={viewTabs} activePanelId={activePanel?.id ?? null} mainPanelId={board.main_panel_id} actions={tabActions} readOnly={readOnly} />
      )}
      {activePanel ? (
        <div className="board-content">
          <PanelLists key={activePanel.id} panel={activePanel} readOnly={readOnly} />
        </div>
      ) : (
        <div className="board-body board-empty">
          <p className="hint">{t('board.noPanels')}</p>
          {!readOnly && (
            <button type="button" className="btn primary" onClick={tabActions.create}>
              {t('board.createPanel')}
            </button>
          )}
        </div>
      )}
      {dialog}
      {archiveView && <PanelArchives panel={archiveView.panel} kind={archiveView.kind} onClose={() => setArchiveView(null)} />}
      {toastElement}
    </div>
  )
}
