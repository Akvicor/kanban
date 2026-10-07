import type {CardAction, List} from '../../api/types'
import {formatDateTime} from '../../utils/datetime'
import {t} from '../../i18n'

/** 操作记录的文字，列表名按当前名称显示。 */
function describe(action: CardAction, lists: Map<number, List>): string {
  const listName = (id: unknown) => (typeof id === 'number' ? t('actionlog.listName', {name: lists.get(id)?.name ?? t('actionlog.listMissing')}) : '')
  switch (action.type) {
    case 'create':
      return action.data.copied_from ? t('actionlog.copyTo', {list: listName(action.data.list_id)}) : t('actionlog.createIn', {list: listName(action.data.list_id)})
    case 'move':
      return t('actionlog.moveFromTo', {from: listName(action.data.from_list_id), to: listName(action.data.to_list_id)})
    case 'task_complete': {
      const children = typeof action.data.children === 'number' ? action.data.children : 0
      return t('actionlog.completeTask', {title: String(action.data.title ?? '')}) + (children > 0 ? t('actionlog.completeTaskChildren', {count: children}) : '')
    }
    case 'task_uncomplete':
      return t('actionlog.uncompleteTask', {title: String(action.data.title ?? '')})
    case 'archive':
      return t('actionlog.archiveFrom', {list: listName(action.data.list_id)})
    case 'restore':
      return t('actionlog.restoreFromArchive', {list: listName(action.data.list_id), end: action.data.end === 'head' ? t('card.head') : t('card.tail')})
    default:
      return action.type
  }
}

/** 卡片的操作记录，最近的在前。 */
export function ActionLog({cardId, actions, lists, timeZone}: {cardId: number; actions: CardAction[]; lists: List[]; timeZone: string}) {
  const own = actions
    .filter((action) => action.card_id === cardId)
    .sort((a, b) => b.created_at.localeCompare(a.created_at) || b.id - a.id)
  const byId = new Map(lists.map((list) => [list.id, list]))
  return (
    <section className="section">
      <h5>{t('card.actionLog')}</h5>
      <ul className="log">
        {own.map((action) => (
          <li key={action.id}>
            <time>{formatDateTime(action.created_at, timeZone)}</time>
            <span>{describe(action, byId)}</span>
          </li>
        ))}
      </ul>
    </section>
  )
}
