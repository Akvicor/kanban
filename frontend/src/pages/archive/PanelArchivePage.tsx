import {Link} from 'react-router-dom'
import {MenuButton} from '../../layout/MenuButton'
import {useState} from 'react'
import * as panelApi from '../../api/panel'
import type {Panel} from '../../api/types'
import {BoardSelectDialog} from '../../panel/BoardSelectDialog'
import {useDirectoryData, usePanelData, useSettings, useSyncStore} from '../../sync/hooks'
import {EntityType} from '../../sync/store'
import {formatDateTime} from '../../utils/datetime'
import '../../directory/DirectoryTree.css'
import './BoardArchivePage.css'
import {useT} from '../../i18n'

/**
 * 面板归档：单独归档的面板，显示面板名、归档时间和原来所在的看板，最近归档的在前。
 * 点面板名进入只读查看。恢复时选择一个目录中的看板，面板排在它的最后一个标签页。归档不能清空。
 */
export function PanelArchivePage() {
  const t = useT()
  const store = useSyncStore()
  const settings = useSettings()
  const {boards} = useDirectoryData()
  const {panels} = usePanelData()
  const [restoring, setRestoring] = useState<Panel | null>(null)
  const archived = panels
    .filter((panel) => panel.archived_at !== null)
    .sort((a, b) => b.archived_at!.localeCompare(a.archived_at!) || b.id - a.id)
  const boardName = (id: number) => {
    const board = boards.find((b) => b.id === id)
    if (!board) return t('card.linkBoardGone')
    return board.archived_at ? t('archive.boardArchivedSuffix', {name: board.name}) : board.name
  }

  return (
    <div className="page">
      <div className="page-header">
        <MenuButton />
        <h1>{t('nav.panelArchive')}</h1>
      </div>
      <p className="hint">{t('archive.panelArchiveHint')}</p>
      <div className="card-section archive-tree">
        {archived.length === 0 && <p className="hint">{t('archive.panelArchiveEmpty')}</p>}
        {archived.map((panel) => (
          <div key={panel.id} className="dir-row archived">
            <Link to={`/board/${panel.board_id}/panel/${panel.id}`} className="dir-name">
              {panel.name}
            </Link>
            <span className="dir-meta">
              {t('archive.originalBoard', {name: boardName(panel.board_id)})} · {t('archive.archivedAt', {time: formatDateTime(panel.archived_at!, settings.timezone)})}
            </span>
            <button type="button" className="btn sm" onClick={() => setRestoring(panel)}>
              {t('common.restore')}
            </button>
          </div>
        ))}
      </div>
      {restoring && (
        <BoardSelectDialog
          title={t('archive.restorePanelTitle', {name: restoring.name})}
          description={t('archive.restorePanelDesc')}
          confirmText={t('common.restore')}
          initialBoardId={restoring.board_id}
          onClose={() => setRestoring(null)}
          onSubmit={async (boardId) => {
            const result = await panelApi.restorePanel(restoring.id, boardId)
            store.applyLocal(EntityType.Panel, result.data.id, result.data, result.revision)
          }}
        />
      )}
    </div>
  )
}
