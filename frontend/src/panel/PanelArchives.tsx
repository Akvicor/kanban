import {useNavigate} from 'react-router-dom'
import type {Card, Panel} from '../api/types'
import {CardArchiveDialog} from '../card/CardArchiveDialog'
import {ListArchiveDialog} from '../list/ListArchiveDialog'
import {activeLists, archivedLists} from '../list/model'
import {usePanelLists} from '../sync/hooks'

export type PanelArchiveKind = 'list' | 'card'

/**
 * 打开某个面板的列表归档或卡片归档。归档属于面板，由面板标签页菜单打开，
 * 因此这里按面板加载列表数据；打开归档中的卡片跳到该卡片，只读查看。挂载时即打开。
 */
export function PanelArchives({panel, kind, onClose}: {panel: Panel; kind: PanelArchiveKind; onClose: () => void}) {
  const {lists} = usePanelLists(panel.id)
  const navigate = useNavigate()
  const openCard = (card: Card) => {
    onClose()
    navigate(`/board/${panel.board_id}/panel/${panel.id}/card/${card.id}`)
  }

  if (kind === 'list') {
    return <ListArchiveDialog lists={archivedLists(lists, panel.id)} onOpenCard={openCard} onClose={onClose} />
  }
  return <CardArchiveDialog panel={panel} lists={activeLists(lists, panel.id)} onOpenCard={openCard} onClose={onClose} />
}
