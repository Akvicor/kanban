import {Link} from 'react-router-dom'
import {MenuButton} from '../../layout/MenuButton'
import {useMemo, useState} from 'react'
import * as panelApi from '../../api/panel'
import type {Board, Folder, Panel} from '../../api/types'
import {useDirectoryActions} from '../../directory/actions'
import {buildArchive, type DirNode} from '../../directory/tree'
import {BoardSelectDialog} from '../../panel/BoardSelectDialog'
import {boardTabs} from '../../panel/model'
import {useDirectoryData, usePanelData, useSettings, useSyncStore} from '../../sync/hooks'
import {EntityType} from '../../sync/store'
import {formatDateTime} from '../../utils/datetime'
import '../../directory/DirectoryTree.css'
import './BoardArchivePage.css'
import {useT} from '../../i18n'

/**
 * 看板归档：按归档前的层级展示已归档的文件夹和看板。路径上仍在目录中的文件夹作为导航节点显示。
 * 点看板名进入只读查看。看板可以整体恢复到目录，也可以只把其中一个面板恢复到某个看板；文件夹只作为层级结构，不能单独恢复。归档不能清空。
 */
export function BoardArchivePage() {
  const t = useT()
  const {folders, boards} = useDirectoryData()
  const tree = useMemo(() => buildArchive(folders, boards), [folders, boards])

  return (
    <div className="page">
      <div className="page-header">
        <MenuButton />
        <h1>{t('nav.boardArchive')}</h1>
      </div>
      <p className="hint">{t('archive.boardArchiveHint')}</p>
      <div className="card-section archive-tree">
        {tree.length === 0 ? <p className="hint">{t('board.archiveEmpty')}</p> : <ArchiveNodes nodes={tree} depth={0} />}
      </div>
    </div>
  )
}

function ArchiveNodes({nodes, depth}: {nodes: DirNode[]; depth: number}) {
  return (
    <>
      {nodes.map((node) =>
        node.kind === 'folder' ? (
          <ArchiveFolder key={`f${node.folder.id}`} folder={node.folder} nav={node.nav} nodes={node.children} depth={depth} />
        ) : (
          <ArchiveBoard key={`b${node.board.id}`} board={node.board} depth={depth} />
        ),
      )}
    </>
  )
}

function ArchiveFolder({folder, nav, nodes, depth}: {folder: Folder; nav: boolean; nodes: DirNode[]; depth: number}) {
  const t = useT()
  const settings = useSettings()
  return (
    <>
      <div className={nav ? 'dir-row path-node' : 'dir-row archived'} style={{paddingLeft: 8 + depth * 18}}>
        <span className="dir-icon folder" aria-hidden />
        <span className="dir-name">{folder.name}</span>
        <span className="dir-meta">
          {folder.archived_at ? t('archive.archivedAt', {time: formatDateTime(folder.archived_at, settings.timezone)}) : t('archive.folderInDirectory')}
        </span>
      </div>
      <ArchiveNodes nodes={nodes} depth={depth + 1} />
    </>
  )
}

/** 归档中的看板，以及仍在其中的面板。面板可以单独恢复到某个看板，恢复后从这个看板中移走。 */
function ArchiveBoard({board, depth}: {board: Board; depth: number}) {
  const t = useT()
  const settings = useSettings()
  const actions = useDirectoryActions()
  const store = useSyncStore()
  const {panels} = usePanelData()
  const [restoring, setRestoring] = useState<Panel | null>(null)
  const tabs = boardTabs(panels, board.id)

  return (
    <>
      <div className="dir-row archived" style={{paddingLeft: 8 + depth * 18}}>
        <span className="dir-icon board" aria-hidden />
        <Link to={`/board/${board.id}`} className="dir-name">
          {board.name}
        </Link>
        <span className="dir-meta">{t('archive.archivedAt', {time: formatDateTime(board.archived_at!, settings.timezone)})}</span>
        <button type="button" className="btn sm" onClick={() => actions.restore(board)}>
          {t('archive.restoreBoard')}
        </button>
      </div>
      {tabs.map((panel) => (
        <div key={panel.id} className="dir-row archived panel-row" style={{paddingLeft: 8 + (depth + 1) * 18 + 14}}>
          <span className="dir-name">{t('archive.panelOf', {name: panel.name})}</span>
          <button type="button" className="btn sm ghost" onClick={() => setRestoring(panel)}>
            {t('archive.restoreOnlyPanel')}
          </button>
        </div>
      ))}
      {restoring && (
        <BoardSelectDialog
          title={t('archive.restorePanelTitle', {name: restoring.name})}
          description={t('archive.restorePanelFromBoardDesc')}
          confirmText={t('common.restore')}
          initialBoardId={null}
          onClose={() => setRestoring(null)}
          onSubmit={async (boardId) => {
            const result = await panelApi.restorePanel(restoring.id, boardId)
            store.applyLocal(EntityType.Panel, result.data.id, result.data, result.revision)
          }}
        />
      )}
    </>
  )
}
