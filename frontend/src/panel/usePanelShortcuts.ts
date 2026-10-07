import {useCallback, useEffect, useMemo, useRef} from 'react'
import * as cardApi from '../api/card'
import {errorMessage} from '../api/client'
import type {Card, Label, Panel} from '../api/types'
import {classifyPaste, getClipboard, setClipboard} from '../card/clipboard'
import type {CardFocus} from '../card/detail/CardDetail'
import {findOwner, keyFromEvent} from '../shortcuts/keys'
import {useSettings, useSyncStore} from '../sync/hooks'
import {EntityType} from '../sync/store'
import {t} from '../i18n'

/** 鼠标当前悬停的卡片和列。 */
interface HoverTarget {
  cardId: number | null
  listId: number | null
}

interface PanelShortcutOptions {
  panel: Panel
  /** 面板的标签，按面板中的顺序。 */
  labels: Label[]
  cards: Card[]
  openCard: (cardId: number, focus: CardFocus) => void
  notify: (text: string) => void
  /** 为 false 时不响应快捷键，例如看板或面板在归档中只能查看。 */
  enabled?: boolean
}

/**
 * 焦点在输入控件中，或有弹窗、下拉菜单打开时，按键交给它们处理，不触发快捷键。
 * 按键已被处理（defaultPrevented，例如 Radix 关闭弹层）时同样跳过。
 */
function shortcutBlocked(event: KeyboardEvent): boolean {
  if (event.defaultPrevented || event.repeat || event.isComposing) return true
  const target = event.target as HTMLElement | null
  if (target?.closest('input, textarea, select, [contenteditable="true"]')) return true
  return document.querySelector('.dialog, [data-radix-popper-content-wrapper], .pswp') !== null
}

/** 页面上有选中的文字时，复制、剪切键留给浏览器复制文字。 */
function hasTextSelection(): boolean {
  return (window.getSelection()?.toString() ?? '') !== ''
}

/**
 * 面板中的快捷键，作用于鼠标悬停的卡片或列，按键绑定来自个人设置。
 * 返回记录悬停目标的函数，由卡片和列在鼠标进入、离开时调用。
 */
export function usePanelShortcuts({panel, labels, cards, openCard, notify, enabled = true}: PanelShortcutOptions) {
  const store = useSyncStore()
  const {shortcuts} = useSettings()
  const hover = useRef<HoverTarget>({cardId: null, listId: null})
  const report = useCallback((promise: Promise<unknown>) => promise.catch((err: unknown) => notify(errorMessage(err))), [notify])
  const applyCard = useCallback(
    (result: {data: Card; revision: number}) => store.applyLocal(EntityType.Card, result.data.id, result.data, result.revision),
    [store],
  )

  const remember = useCallback(
    (card: Card, mode: 'copy' | 'cut') => {
      setClipboard({mode, cardId: card.id, panelId: panel.id})
      notify(mode === 'copy' ? t('panel.copiedNotice') : t('panel.cutNotice'))
    },
    [panel.id, notify],
  )

  /** 粘贴到指定列的列首。没有内容时提示；不能跨面板。剪切在请求发出前清掉剪贴板。 */
  const pasteToList = useCallback(
    (listId: number) => {
      const kind = classifyPaste(getClipboard(), panel.id)
      if (kind === 'empty') {
        notify(t('panel.nothingToPaste'))
        return
      }
      if (kind === 'cross-panel') {
        notify(t('panel.crossPanelPaste'))
        return
      }
      const clip = getClipboard()
      if (!clip) return
      if (kind === 'copy') {
        void report(cardApi.copyCard(clip.cardId, listId).then(applyCard))
      } else {
        setClipboard(null)
        void report(cardApi.moveCard(clip.cardId, listId, 0).then(applyCard))
      }
    },
    [panel.id, notify, report, applyCard],
  )

  const run = useCallback(
    (action: string): boolean => {
      const {cardId, listId} = hover.current
      const card = cardId === null ? undefined : cards.find((c) => c.id === cardId && c.archived_at === null)

      if (action === 'paste_card') {
        if (listId === null) return false
        pasteToList(listId)
        return true
      }

      if (!card) return false
      const toggle = /^toggle_label_(\d+)$/.exec(action)
      if (toggle) {
        const label = labels[Number(toggle[1]) - 1]
        if (!label) return false
        void report(cardApi.setCardLabel(card.id, label.id, !card.label_ids.includes(label.id)).then(applyCard))
        return true
      }
      switch (action) {
        case 'open_card':
          openCard(card.id, null)
          return true
        case 'edit_title':
          openCard(card.id, 'title')
          return true
        case 'edit_labels':
          openCard(card.id, 'labels')
          return true
        case 'archive_card':
          void report(cardApi.archiveCard(card.id))
          return true
        case 'copy_card':
        case 'cut_card':
          if (hasTextSelection()) return false
          remember(card, action === 'copy_card' ? 'copy' : 'cut')
          return true
        default:
          return false
      }
    },
    [labels, cards, openCard, pasteToList, remember, report, applyCard],
  )

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (shortcutBlocked(event)) return
      const key = keyFromEvent(event)
      const action = key === null ? null : findOwner(shortcuts, key)
      if (action && run(action)) event.preventDefault()
    }
    if (!enabled) return
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [shortcuts, run, enabled])

  return useMemo(
    () => ({
      /** 鼠标进入卡片时传入卡片和它所在的列，离开时传入 null。 */
      hoverCard: (card: Card | null) => {
        hover.current = card ? {cardId: card.id, listId: card.list_id} : {...hover.current, cardId: null}
      },
      /** 鼠标进入列时传入列 ID，离开时传入 null。 */
      hoverList: (listId: number | null) => {
        hover.current = {cardId: listId === null ? null : hover.current.cardId, listId}
      },
      copyCard: (card: Card) => remember(card, 'copy'),
      cutCard: (card: Card) => remember(card, 'cut'),
      pasteToList,
    }),
    [remember, pasteToList],
  )
}
