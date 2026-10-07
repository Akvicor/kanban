import {useCallback, useEffect, useMemo} from 'react'
import * as cardApi from '../../api/card'
import {ApiError, errorMessage, ResultCode} from '../../api/client'
import type {List, Panel} from '../../api/types'
import {AttachmentSection} from '../../attachment/AttachmentSection'
import {useAttachmentInput} from '../../attachment/useAttachmentInput'
import {panelLabels, panelPriorityLevels} from '../../panel/model'
import {useCardData, useDirectoryData, useNotifyChannels, usePanelData, usePanelLists, useSettings, useSyncStore} from '../../sync/hooks'
import {EntityType} from '../../sync/store'
import {ActionLog} from './ActionLog'
import {CardProps} from './CardProps'
import {LinkSection} from './LinkSection'
import {TaskSection} from './TaskSection'
import {DescriptionField} from './DescriptionField'
import {TitleField} from './TitleField'
import {TimerBox} from './TimerBox'
import {CardWriteContext, type CardWrite} from './useCardWrite'
import './CardDetail.css'
import {t, useT} from '../../i18n'

/** 打开卡片时直接进入的编辑状态，来自快捷键「修改卡片标题」「编辑卡片标签」。 */
export type CardFocus = 'title' | 'labels' | null

interface CardDetailProps {
  panel: Panel
  cardId: number
  focus: CardFocus
  onClose: () => void
  /** 显示一条提示。 */
  onNotice: (text: string) => void
}

/** 卡片所在位置的说明：列表名，或所在的归档。 */
function whereOf(listId: number | null, lists: List[]): string {
  if (listId === null) return t('card.archive')
  const list = lists.find((l) => l.id === listId)
  if (!list) return ''
  return list.archived_at ? t('list.archiveTitle', {name: list.name}) : list.name
}

/**
 * Esc 是否已由其他元素处理：焦点在输入控件中，或弹窗、下拉菜单用它关闭了自己。
 * Radix 在文档捕获阶段处理 Esc，关闭弹层时调用 preventDefault，并且同步移除弹层，
 * 因此这里用 defaultPrevented 判断，弹层是否还在页面上不可靠。
 */
function escapeHandledElsewhere(event: KeyboardEvent): boolean {
  if (event.defaultPrevented) return true
  const target = event.target as HTMLElement | null
  return target?.closest('input, textarea, select, [contenteditable="true"]') != null
}

/**
 * 卡片详情抽屉。电脑和平板上从右侧滑出，手机上全屏显示。
 * 卡片在卡片归档或列表归档中时只能查看。所有修改通过 CardWriteContext 写入并立即更新本地数据。
 */
export function CardDetail({panel, cardId, focus, onClose, onNotice}: CardDetailProps) {
  const t = useT()
  const store = useSyncStore()
  const settings = useSettings()
  const {cards, tasks, cardActions, cardLinks, attachments, notifyDeliveries} = useCardData()
  const {labels, priorityLevels} = usePanelData()
  const {boards} = useDirectoryData()
  const channels = useNotifyChannels()
  const {lists} = usePanelLists(panel.id)
  const card = cards.find((c) => c.id === cardId && c.panel_id === panel.id)
  const list = card?.list_id == null ? undefined : lists.find((l) => l.id === card.list_id)
  const boardArchived = boards.find((b) => b.id === panel.board_id)?.archived_at != null
  const panelArchived = panel.archived_at !== null || boardArchived
  const readOnly = !card || card.archived_at !== null || list?.archived_at != null || panel.archived_at !== null || boardArchived
  const ownLabels = useMemo(() => panelLabels(labels, panel.id), [labels, panel.id])
  const ownLevels = useMemo(() => panelPriorityLevels(priorityLevels, panel.id), [priorityLevels, panel.id])

  const write = useCallback<CardWrite>(
    async (type, request, deletedId) => {
      try {
        const result = await request()
        if (deletedId !== undefined) {
          store.applyLocal(type, deletedId, null, result.revision)
        } else {
          const items = (Array.isArray(result.data) ? result.data : [result.data]) as {id: number}[]
          items.forEach((item) => store.applyLocal(type, item.id, item, result.revision))
        }
        return 'ok'
      } catch (err) {
        if (err instanceof ApiError && err.code === ResultCode.Conflict && err.data) {
          const current = err.data as {id: number}
          store.applyLocal(type, current.id, current, err.revision)
          return 'conflict'
        }
        onNotice(errorMessage(err))
        return 'error'
      }
    },
    [store, onNotice],
  )
  const attachmentInput = useAttachmentInput(card?.id ?? null, readOnly, write)

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape' && !escapeHandledElsewhere(event)) onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [onClose])

  async function copy() {
    if (!card) return
    if ((await write(EntityType.Card, () => cardApi.copyCard(card.id, null))) === 'ok') onNotice(t('card.copied'))
  }

  async function archive() {
    if (!card) return
    try {
      await cardApi.archiveCard(card.id)
      onClose()
    } catch (err) {
      onNotice(errorMessage(err))
    }
  }

  return (
    <aside className="drawer" aria-label={t('card.detail')} {...attachmentInput.dropProps}>
      {attachmentInput.dragging && <div className="drop-hint">{t('card.dropToUpload')}</div>}
      {attachmentInput.dialog}
      <div className="drawer-head">
        <span className="where">{card ? whereOf(card.list_id, lists) : ''}</span>
        <span className="grow" />
        {card && !readOnly && (
          <>
            <button type="button" className="icon-btn" title={t('shortcut.copyCard')} aria-label={t('shortcut.copyCard')} onClick={() => void copy()}>
              ⧉
            </button>
            <button type="button" className="icon-btn" title={t('shortcut.archiveCard')} aria-label={t('shortcut.archiveCard')} onClick={() => void archive()}>
              ⊟
            </button>
          </>
        )}
        <button type="button" className="icon-btn" title={t('common.close')} aria-label={t('common.close')} onClick={onClose}>
          ×
        </button>
      </div>
      <div className="drawer-body">
        {!card ? (
          <p className="hint">{t('card.notFound')}</p>
        ) : (
          <CardWriteContext.Provider value={write}>
            {readOnly && (
              <p className="read-only-note">
                {card.archived_at
                  ? t('card.archivedReadOnly')
                  : list?.archived_at
                    ? t('card.listArchivedReadOnly')
                    : panel.archived_at
                      ? t('card.panelArchivedReadOnly')
                      : t('card.boardArchivedReadOnly')}
              </p>
            )}
            <TitleField key={`title-${card.id}`} card={card} readOnly={readOnly} autoFocus={focus === 'title'} />
            <CardProps
              key={`props-${card.id}`}
              place={{card, list, panelArchived}}
              labels={ownLabels}
              levels={ownLevels}
              channels={channels}
              deliveries={notifyDeliveries}
              timeZone={settings.timezone}
              readOnly={readOnly}
              openLabels={focus === 'labels'}
            />
            <TimerBox card={card} readOnly={readOnly} />
            <DescriptionField key={`desc-${card.id}`} card={card} readOnly={readOnly} />
            <TaskSection cardId={card.id} tasks={tasks} readOnly={readOnly} />
            <AttachmentSection card={card} attachments={attachments} readOnly={readOnly} timeZone={settings.timezone} />
            <LinkSection cardId={card.id} links={cardLinks} readOnly={readOnly} onNotice={onNotice} />
            <ActionLog cardId={card.id} actions={cardActions} lists={lists} timeZone={settings.timezone} />
          </CardWriteContext.Provider>
        )}
      </div>
    </aside>
  )
}
