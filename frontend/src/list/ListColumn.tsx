import {useDraggable, useDroppable} from '@dnd-kit/core'
import {useCallback, useState, type CSSProperties, type MouseEvent} from 'react'
import type {Card, List, ListEnd} from '../api/types'
import {getClipboard, type CardClipboard} from '../card/clipboard'
import {EMPTY_SUMMARY, type AttachmentSummary} from '../attachment/summary'
import {CardComposer} from '../card/CardComposer'
import {CardItem, type CardFaceContext} from '../card/CardItem'
import {Menu, MenuItem, MenuSeparator} from '../components/Menu'
import {sortLabel} from './model'
import {useT} from '../i18n'

/** 拖动卡片时的落点：某张卡片的上方或下方，cardId 为 null 时放在列表空白处。 */
export interface CardHint {
  listId: number
  cardId: number | null
  after: boolean
}

export interface ListColumnActions {
  rename: () => void
  settings: () => void
  rules: () => void
  archive: () => void
  archiveCards: () => void
  /** 把剪贴板中的卡片粘贴到这列列首。 */
  paste: () => void
  copyCard: (card: Card) => void
  cutCard: (card: Card) => void
  /** 打开卡片详情，不随双击切换关闭。 */
  editCard: (card: Card) => void
  openComposer: (end: ListEnd) => void
  closeComposer: () => void
  createCard: (title: string, end: ListEnd) => Promise<void>
  openCard: (card: Card) => void
  hoverCard: (card: Card | null) => void
  hoverList: (listId: number | null) => void
}

interface ListColumnProps {
  list: List
  /** 搜索和筛选后显示的卡片，按列表的排序方式排列。 */
  cards: Card[]
  /** 列中未归档的卡片数，不受搜索、筛选和「只看今日」影响。 */
  count: number
  /** 卡片正面需要的公共数据，showAge、progress 和 attachments 按列和卡片补全。 */
  face: Omit<CardFaceContext, 'showAge' | 'progress' | 'attachments'>
  progress: Map<number, {done: number; total: number}>
  attachments: Map<number, AttachmentSummary>
  /** 开启「只看今日」时，是否为今日卡片的判断；未开启时为 null。 */
  isToday: ((card: Card) => boolean) | null
  /** 已松手、移动尚未生效的列表：保持拖动时的半透明，直到列表出现在新位置或请求结束。 */
  moving: boolean
  /** 卡片是否已松手、移动尚未生效。 */
  isCardMoving: (card: Card) => boolean
  selectedCardId: number | null
  /** 拖动列表时的落点：放在这个列表的左边或右边。 */
  listHint: {after: boolean} | null
  cardHint: CardHint | null
  composer: ListEnd | null
  actions: ListColumnActions
  /** 看板或面板在归档中时只能查看：不能拖动、新建或打开会修改的菜单。 */
  readOnly?: boolean
}

/**
 * 一个列表：列头显示颜色、名称、卡片数、排序方式和菜单，拖动列头可以调整列表顺序。
 * 列首和列尾各有一个添加卡片的按钮，卡片的插入位置由列表设置决定。
 */
export function ListColumn({list, cards, count, face, progress, attachments, isToday, moving, isCardMoving, selectedCardId, listHint, cardHint, composer, actions, readOnly = false}: ListColumnProps) {
  const t = useT()
  const dragId = `list:${list.id}`
  const {attributes, listeners, setNodeRef: setDragRef, isDragging} = useDraggable({id: dragId, disabled: readOnly})
  const [clip, setClip] = useState<CardClipboard | null>(null)
  const {setNodeRef: setDropRef} = useDroppable({id: dragId})
  const setRef = useCallback(
    (element: HTMLElement | null) => {
      setDragRef(element)
      setDropRef(element)
    },
    [setDragRef, setDropRef],
  )
  const [menuOpen, setMenuOpen] = useState(false)
  let className = 'list'
  if (isDragging || moving) className += ' dragging'
  if (listHint) className += listHint.after ? ' drop-after' : ' drop-before'
  if (cardHint?.cardId === null) className += ' card-over'
  const noProgress = {done: 0, total: 0}

  const addButton = (end: ListEnd) =>
    composer === end ? (
      <CardComposer onCreate={(title) => actions.createCard(title, end)} onClose={actions.closeComposer} />
    ) : (
      <button type="button" className="add-card" onClick={() => actions.openComposer(end)}>
        {t('list.addCard')}
      </button>
    )

  return (
    <section
      ref={setRef}
      data-flip-key={`l:${list.id}`}
      className={className}
      style={{'--list-color': list.color || 'transparent'} as CSSProperties}
      aria-label={list.name}
      onMouseEnter={() => actions.hoverList(list.id)}
      onMouseLeave={() => actions.hoverList(null)}
    >
      <header
        className="list-head"
        {...(readOnly ? {} : {...attributes, ...listeners})}
        onContextMenu={(event: MouseEvent) => {
          if (readOnly) return
          event.preventDefault()
          // 触屏长按拖动时系统也会触发 contextmenu，拖动期间不打开菜单。
          if (!isDragging) setMenuOpen(true)
        }}
      >
        {list.color && <span className="list-dot" aria-hidden />}
        <h3>{list.name}</h3>
        <span className="count" title={t('list.cardCount')}>
          {count}
        </span>
        <span className="sort">{sortLabel(list.sort_mode, list.sort_dir)}</span>
        {!readOnly && (
        <Menu
          open={menuOpen}
          onOpenChange={(open) => {
            setMenuOpen(open)
            if (open) setClip(getClipboard())
          }}
          trigger={
            <button type="button" className="icon-btn sm" aria-label={t('common.menuOf', {name: list.name})}>
              ⋯
            </button>
          }
        >
          <MenuItem disabled={clip === null} onSelect={actions.paste}>
            {t('common.paste')}
          </MenuItem>
          <MenuSeparator />
          <MenuItem onSelect={actions.rename}>{t('common.rename')}</MenuItem>
          <MenuItem onSelect={actions.settings}>{t('list.settings')}</MenuItem>
          <MenuItem onSelect={actions.rules}>{t('list.rules')}</MenuItem>
          <MenuSeparator />
          <MenuItem disabled={count === 0} onSelect={actions.archiveCards}>
            {t('list.archiveAllCards')}
          </MenuItem>
          <MenuItem danger onSelect={actions.archive}>
            {t('list.archiveList')}
          </MenuItem>
        </Menu>
        )}
      </header>
      {!readOnly && addButton('head')}
      <div className="cards">
        {cards.map((card) => (
          <CardItem
            key={card.id}
            card={card}
            context={{
              ...face,
              showAge: list.show_age,
              progress: progress.get(card.id) ?? noProgress,
              attachments: attachments.get(card.id) ?? EMPTY_SUMMARY,
            }}
            dimmed={isToday !== null && !isToday(card)}
            selected={card.id === selectedCardId}
            moving={isCardMoving(card)}
            dropHint={cardHint?.cardId === card.id ? (cardHint.after ? 'after' : 'before') : null}
            onOpen={() => actions.openCard(card)}
            onHover={(hovered) => actions.hoverCard(hovered ? card : null)}
            onCopy={readOnly ? undefined : () => actions.copyCard(card)}
            onCut={readOnly ? undefined : () => actions.cutCard(card)}
            onEdit={readOnly ? undefined : () => actions.editCard(card)}
          />
        ))}
      </div>
      {!readOnly && addButton('tail')}
    </section>
  )
}
