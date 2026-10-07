import {MenuButton} from '../../layout/MenuButton'
import {useT} from '../../i18n'
import {Navigate, useLocation} from 'react-router-dom'
import {useDirectoryActions} from '../../directory/actions'
import {DirectoryTree} from '../../directory/DirectoryTree'
import {useMe} from '../../session/context'
import {displayName} from '../../session/account'
import {useDirectoryData, useSettings} from '../../sync/hooks'

/**
 * 主页。设置了「进入主页时打开主看板」且主看板仍在目录中时，直接打开主看板；否则显示目录。
 * 从目录或其他页面主动回到主页（带 stay 状态）时总是显示目录。
 */
export function HomePage() {
  const t = useT()
  const me = useMe()
  const settings = useSettings()
  const {boards} = useDirectoryData()
  const actions = useDirectoryActions()
  const location = useLocation()
  const stay = (location.state as {stay?: boolean} | null)?.stay === true
  const main = boards.find((board) => board.id === settings.main_board_id && board.archived_at === null)

  if (settings.open_main_board_on_home && main && !stay) {
    return <Navigate to={`/board/${main.id}`} replace />
  }

  return (
    <div className="page">
      <div className="page-header">
        <MenuButton />
        <h1>{t('home.greeting', {name: displayName(me.account)})}</h1>
        <span className="grow" />
        <button type="button" className="btn" onClick={() => actions.createFolder(null)}>
          {t('directory.createFolder')}
        </button>
        <button type="button" className="btn primary" onClick={() => actions.createBoard(null)}>
          {t('directory.createBoard')}
        </button>
      </div>
      <div className="card-section">
        <DirectoryTree activeBoardId={null} />
      </div>
    </div>
  )
}
