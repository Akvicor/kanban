import {useRef, useState, type KeyboardEvent} from 'react'
import * as cardApi from '../../api/card'
import type {Card} from '../../api/types'
import {useSyncStore} from '../../sync/hooks'
import {EntityType} from '../../sync/store'
import {useCardWrite} from './useCardWrite'
import {useT} from '../../i18n'

/**
 * 卡片标题。点击后编辑，回车或失去焦点时保存，Esc 放弃。开始编辑时记下本地卡片数据的同步序号，保存时一起发送。
 * 标题在此期间被其他设备修改过时，服务端返回冲突：输入框保留草稿并显示其他设备保存的标题，
 * 用户选择用自己的标题覆盖（回车）或放弃（Esc）。冲突期间失去焦点不提交，避免在不知情时覆盖。
 */
export function TitleField({card, readOnly, autoFocus}: {card: Card; readOnly: boolean; autoFocus: boolean}) {
  const t = useT()
  const store = useSyncStore()
  const write = useCardWrite()
  const [draft, setDraft] = useState<string | null>(autoFocus && !readOnly ? card.title : null)
  const [conflict, setConflict] = useState(false)
  const base = useRef(store.versionOf(EntityType.Card, card.id))
  const saving = useRef(false)

  function begin() {
    if (readOnly) return
    base.current = store.versionOf(EntityType.Card, card.id)
    setDraft(card.title)
  }

  function discard() {
    setDraft(null)
    setConflict(false)
  }

  /** 保存草稿。overwrite 为 true 时以本地最新的同步序号提交，覆盖其他设备保存的标题。 */
  async function commit(overwrite: boolean) {
    if (draft === null || saving.current || (conflict && !overwrite)) return
    const title = draft.trim()
    if (!title || title === card.title) {
      discard()
      return
    }
    if (overwrite) base.current = store.versionOf(EntityType.Card, card.id)
    saving.current = true
    const outcome = await write(EntityType.Card, () => cardApi.updateCardTitle(card.id, title, base.current))
    saving.current = false
    if (outcome === 'ok') discard()
    else if (outcome === 'conflict') setConflict(true)
  }

  function onKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault()
      void commit(conflict)
    } else if (event.key === 'Escape') {
      event.stopPropagation()
      discard()
    }
  }

  if (draft === null) {
    return (
      <h2 className={readOnly ? 'card-detail-title' : 'card-detail-title editable'} onClick={begin}>
        {card.title}
      </h2>
    )
  }
  return (
    <div className="title-edit">
      <textarea
        className="textarea title-input"
        value={draft}
        aria-label={t('card.title')}
        autoFocus
        rows={2}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={() => void commit(false)}
        onKeyDown={onKeyDown}
      />
      {conflict && (
        <div className="conflict-note" role="alert">
          <span>{t('card.titleConflict', {title: card.title})}</span>
          <button type="button" className="btn sm primary" onMouseDown={(e) => e.preventDefault()} onClick={() => void commit(true)}>
            {t('card.overwriteTitle')}
          </button>
          <button type="button" className="btn sm" onMouseDown={(e) => e.preventDefault()} onClick={discard}>
            {t('card.discardMine')}
          </button>
        </div>
      )}
    </div>
  )
}
