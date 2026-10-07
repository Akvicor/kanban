import {useEffect, useState} from 'react'
import {fetchListCards} from '../api/card'
import {restoreList} from '../api/list'
import {errorMessage} from '../api/client'
import type {Card, List} from '../api/types'
import {sortedCards} from '../card/model'
import {Dialog} from '../components/Dialog'
import {panelPriorityLevels} from '../panel/model'
import {useCardData, usePanelData, useSettings, useSyncStore} from '../sync/hooks'
import {EntityType} from '../sync/store'
import {formatDateTime} from '../utils/datetime'
import {t, useT} from '../i18n'

interface ListArchiveDialogProps {
  lists: List[]
  /** 打开列表归档中的卡片，卡片只能查看。 */
  onOpenCard: (card: Card) => void
  onClose: () => void
}

/**
 * 面板的列表归档。列表连同其中的卡片、配置和卡片顺序一起恢复到面板的最左边或最右边，
 * 不能单独恢复其中的某张卡片。可以打开某个列表查看其中的卡片。归档不能清空。挂载时即打开。
 */
export function ListArchiveDialog({lists, onOpenCard, onClose}: ListArchiveDialogProps) {
  const t = useT()
  const store = useSyncStore()
  const settings = useSettings()
  const [error, setError] = useState('')
  const [viewing, setViewing] = useState<List | null>(null)

  async function restore(list: List, atStart: boolean) {
    setError('')
    try {
      const result = await restoreList(list.id, atStart)
      store.applyLocal(EntityType.List, list.id, result.data, result.revision)
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  if (viewing) {
    return <ArchivedListCards list={viewing} onOpenCard={onOpenCard} onBack={() => setViewing(null)} onClose={onClose} />
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()} title={t('list.archive')} description={t('list.restoreDesc')}>
      <div className="archive-list">
        {lists.length === 0 && <p className="hint">{t('list.archiveEmpty')}</p>}
        {lists.map((list) => (
          <div key={list.id} className="archive-list-row">
            <div className="archive-list-info">
              <strong>{list.name}</strong>
              <span className="hint">{t('archive.archivedAt', {time: formatDateTime(list.archived_at!, settings.timezone)})}</span>
            </div>
            <button type="button" className="btn sm ghost" onClick={() => setViewing(list)}>
              {t('list.viewCards')}
            </button>
            <button type="button" className="btn sm" onClick={() => void restore(list, true)}>
              {t('list.restoreLeft')}
            </button>
            <button type="button" className="btn sm" onClick={() => void restore(list, false)}>
              {t('list.restoreRight')}
            </button>
          </div>
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

interface ArchivedListCardsProps {
  list: List
  onOpenCard: (card: Card) => void
  onBack: () => void
  onClose: () => void
}

/** 列表归档中某个列表的卡片，按列表的排序方式排列，只能查看。打开时加载。 */
function ArchivedListCards({list, onOpenCard, onBack, onClose}: ArchivedListCardsProps) {
  const store = useSyncStore()
  const {cards} = useCardData()
  const {priorityLevels} = usePanelData()
  const [loaded, setLoaded] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    fetchListCards(list.id)
      .then((bundle) => {
        if (cancelled) return
        store.applyBundle(bundle)
        setLoaded(true)
      })
      .catch((err: unknown) => !cancelled && setError(errorMessage(err)))
    return () => {
      cancelled = true
    }
  }, [store, list.id])

  const listCards = sortedCards(cards, list, panelPriorityLevels(priorityLevels, list.panel_id))

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()} title={t('list.archiveTitle', {name: list.name})} description={t('list.archivedReadOnly')}>
      <div className="archive-list">
        {!loaded && !error && <p className="hint">{t('common.loading')}</p>}
        {loaded && listCards.length === 0 && <p className="hint">{t('list.noCards')}</p>}
        {listCards.map((card) => (
          <div key={card.id} className="archive-list-row">
            <button type="button" className="archive-list-info archive-card-open" onClick={() => onOpenCard(card)}>
              <strong>{card.title}</strong>
            </button>
          </div>
        ))}
        {error && <p className="form-error">{error}</p>}
        <div className="form-actions">
          <button type="button" className="btn" onClick={onBack}>
            {t('common.back')}
          </button>
          <button type="button" className="btn" onClick={onClose}>
            {t('common.close')}
          </button>
        </div>
      </div>
    </Dialog>
  )
}
