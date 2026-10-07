import {useEffect, useState} from 'react'
import * as cardApi from '../api/card'
import {errorMessage} from '../api/client'
import type {Card, List, ListEnd, Panel} from '../api/types'
import {Dialog} from '../components/Dialog'
import {Select} from '../components/Select'
import {useCardData, useSettings, useSyncStore} from '../sync/hooks'
import {EntityType} from '../sync/store'
import {formatDateTime} from '../utils/datetime'
import {t, useT} from '../i18n'

interface CardArchiveDialogProps {
  panel: Panel
  /** 面板中未归档的列表，作为恢复目标。 */
  lists: List[]
  onOpenCard: (card: Card) => void
  onClose: () => void
}

/**
 * 面板的卡片归档，最近归档的在前。打开时加载归档中的卡片。
 * 恢复时选择一个未归档的列表和列首或列尾，不执行目标列的移入配置。挂载时即打开。
 */
export function CardArchiveDialog({panel, lists, onOpenCard, onClose}: CardArchiveDialogProps) {
  const t = useT()
  const store = useSyncStore()
  const settings = useSettings()
  const {cards} = useCardData()
  const [loaded, setLoaded] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    cardApi
      .fetchArchivedCards(panel.id)
      .then((bundle) => {
        if (cancelled) return
        store.applyBundle(bundle)
        setLoaded(true)
      })
      .catch((err: unknown) => !cancelled && setError(errorMessage(err)))
    return () => {
      cancelled = true
    }
  }, [store, panel.id])

  const archived = cards
    .filter((card) => card.panel_id === panel.id && card.archived_at !== null)
    .sort((a, b) => b.archived_at!.localeCompare(a.archived_at!) || b.id - a.id)

  async function restore(card: Card, listId: number, end: ListEnd) {
    setError('')
    try {
      const result = await cardApi.restoreCard(card.id, listId, end)
      store.applyLocal(EntityType.Card, card.id, result.data, result.revision)
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()} title={t('card.archive')} description={t('card.restoreDesc')}>
      <div className="archive-list">
        {!loaded && !error && <p className="hint">{t('common.loading')}</p>}
        {loaded && archived.length === 0 && <p className="hint">{t('card.archiveEmpty')}</p>}
        {archived.map((card) => (
          <ArchivedCardRow
            key={card.id}
            card={card}
            lists={lists}
            archivedAt={formatDateTime(card.archived_at!, settings.timezone)}
            onOpen={() => onOpenCard(card)}
            onRestore={(listId, end) => void restore(card, listId, end)}
          />
        ))}
        {error && <p className="form-error">{error}</p>}
        <div className="form-actions">
          <button type="button" className="btn" onClick={onClose}>
            {t('common.close')}
          </button>
        </div>
      </div>
    </Dialog>
  )
}

interface ArchivedCardRowProps {
  card: Card
  lists: List[]
  archivedAt: string
  onOpen: () => void
  onRestore: (listId: number, end: ListEnd) => void
}

function ArchivedCardRow({card, lists, archivedAt, onOpen, onRestore}: ArchivedCardRowProps) {
  const [listId, setListId] = useState(lists[0]?.id ?? 0)
  const [end, setEnd] = useState<ListEnd>('head')
  const target = lists.some((list) => list.id === listId) ? listId : (lists[0]?.id ?? 0)

  return (
    <div className="archive-list-row archive-card-row">
      <button type="button" className="archive-list-info archive-card-open" onClick={onOpen}>
        <strong>{card.title}</strong>
        <span className="hint">{t('archive.archivedAt', {time: archivedAt})}</span>
      </button>
      <div className="archive-card-restore">
        <Select
          className="inline-select"
          label={t('card.restoreToList')}
          value={String(target)}
          options={lists.length === 0 ? [{value: '0', label: t('card.noLists'), disabled: true}] : lists.map((list) => ({value: String(list.id), label: list.name}))}
          onChange={(value) => setListId(Number(value))}
        />
        <Select
          className="inline-select"
          label={t('card.restoreTo')}
          value={end}
          options={[
            {value: 'head', label: t('card.head')},
            {value: 'tail', label: t('card.tail')},
          ]}
          onChange={(value) => setEnd(value as ListEnd)}
        />
        <button type="button" className="btn sm" disabled={target === 0} onClick={() => onRestore(target, end)}>
          {t('common.restore')}
        </button>
      </div>
    </div>
  )
}
