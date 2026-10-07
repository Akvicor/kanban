import {
  DndContext,
  DragOverlay,
  pointerWithin,
  type CollisionDetection,
  type DragEndEvent,
  type DragMoveEvent,
  type DragStartEvent,
} from '@dnd-kit/core'
import {useDragSensors} from '../hooks/useDragSensors'
import {armFlip, disarmFlip, useFlip} from '../utils/flip'
import {usePanScroll} from '../utils/panScroll'
import {usePendingMoves} from './pendingMoves'
import {useCallback, useMemo, useRef, useState, type ReactNode} from 'react'
import {useLocation, useNavigate, useParams} from 'react-router-dom'
import * as cardApi from '../api/card'
import * as listApi from '../api/list'
import {errorMessage, type WriteResult} from '../api/client'
import type {Card, List, ListEnd, Panel} from '../api/types'
import {attachmentSummaries} from '../attachment/summary'
import type {CardFocus} from '../card/detail/CardDetail'
import {CardDrawer} from '../card/detail/CardDrawer'
import {isTodayCard, manualOrder, matchesFilter, resolveCardDrop, sortedCards, taskProgressByCard} from '../card/model'
import {ConfirmDialog} from '../components/ConfirmDialog'
import {useToast} from '../components/Toast'
import {NameDialog} from '../directory/NameDialog'
import {useNow} from '../hooks/useNow'
import {panelLabels, panelPriorityLevels} from '../panel/model'
import {PanelToolbar} from '../panel/PanelToolbar'
import {usePanelShortcuts} from '../panel/usePanelShortcuts'
import {usePanelView} from '../panel/viewState'
import {useCardData, usePanelData, usePanelLists, useSettings, useSyncStore} from '../sync/hooks'
import {EntityType} from '../sync/store'
import {ListColumn, type CardHint, type ListColumnActions} from './ListColumn'
import {ListRulesDialog} from './ListRulesDialog'
import {ListSettingsDialog} from './ListSettingsDialog'
import {activeLists, listDropIndex} from './model'
import './PanelLists.css'
import {useT} from '../i18n'

/** 当前打开的弹窗。 */
type Pending =
  | {type: 'create'}
  | {type: 'rename'; list: List}
  | {type: 'settings'; list: List}
  | {type: 'rules'; list: List}
  | {type: 'archive'; list: List}
  | {type: 'archive-cards'; list: List}

/** 拖动列表时的落点：放在某个列表的左边或右边。 */
interface ListHint {
  overId: number
  after: boolean
}

/** 拖动对象的 ID 带类型前缀，列表为 list:ID，卡片为 card:ID。 */
function parseDragId(id: string | number): {kind: 'list' | 'card'; id: number} {
  const [kind, value] = String(id).split(':')
  return {kind: kind === 'card' ? 'card' : 'list', id: Number(value)}
}

/** 开始拖动时指针的位置，鼠标和触摸都适用。 */
function startPoint(event: Event): {x: number; y: number} {
  if ('touches' in event) {
    const touch = (event as TouchEvent).touches[0] ?? (event as TouchEvent).changedTouches[0]
    return {x: touch?.clientX ?? 0, y: touch?.clientY ?? 0}
  }
  return {x: (event as PointerEvent).clientX, y: (event as PointerEvent).clientY}
}

/**
 * 拖动时的自动滚动：指针进入横向列表区左右各 15% 的范围时横向滚动到相邻的列，越靠边越快；
 * 列内上下滚动保持 dnd-kit 的默认范围（20%）。
 */
const LIST_AUTO_SCROLL = {threshold: {x: 0.15, y: 0.2}}

/**
 * 碰撞检测：拖动列表时只看列表；拖动卡片时优先取指针下的卡片，没有时取指针所在的列表。
 */
const collision: CollisionDetection = (args) => {
  const hits = pointerWithin(args)
  const ofKind = (kind: string) => hits.filter((hit) => parseDragId(hit.id).kind === kind)
  if (parseDragId(args.active.id).kind === 'list') return ofKind('list')
  const cards = ofKind('card')
  return cards.length > 0 ? cards : ofKind('list')
}

/**
 * 面板的内容：工具栏、横向排列的列表和卡片，以及卡片详情抽屉。
 * 列表可以拖动调整顺序（只在本面板内）；卡片可以在本面板的列表之间拖动。
 * 搜索、筛选只改变显示哪些卡片，「只看今日」让其余卡片变灰。
 */
export function PanelLists({panel, readOnly = false}: {panel: Panel; readOnly?: boolean}) {
  const t = useT()
  const store = useSyncStore()
  const settings = useSettings()
  const navigate = useNavigate()
  const location = useLocation()
  const params = useParams()
  const {lists, loaded, error} = usePanelLists(panel.id)
  const {cards, tasks, attachments} = useCardData()
  const {labels, priorityLevels} = usePanelData()
  const {view, update: updateView} = usePanelView(panel.id)
  const {show: showToast, element: toastElement} = useToast()
  const [pending, setPending] = useState<Pending | null>(null)
  const [listHint, setListHint] = useState<ListHint | null>(null)
  const [cardHint, setCardHint] = useState<CardHint | null>(null)
  const [draggingCard, setDraggingCard] = useState<Card | null>(null)
  // 拖动列表或卡片期间暂停手机宽度下的按列对齐，避免与自动滚动互相抵消。
  const [dragActive, setDragActive] = useState(false)
  const [composer, setComposer] = useState<{listId: number; end: ListEnd} | null>(null)
  const sensors = useDragSensors()
  const listsRef = useRef<HTMLDivElement>(null)
  const panelViewRef = useRef<HTMLDivElement>(null)
  useFlip(listsRef)
  usePanScroll(panelViewRef, listsRef, settings.pan_modifier)
  const moves = usePendingMoves()

  const active = activeLists(lists, panel.id)
  const ownLabels = useMemo(() => panelLabels(labels, panel.id), [labels, panel.id])
  const ownLevels = useMemo(() => panelPriorityLevels(priorityLevels, panel.id), [priorityLevels, panel.id])
  const panelCards = useMemo(() => cards.filter((card) => card.panel_id === panel.id && card.archived_at === null), [cards, panel.id])
  const progress = useMemo(() => taskProgressByCard(tasks), [tasks])
  const attachmentInfo = useMemo(() => attachmentSummaries(panelCards, attachments), [panelCards, attachments])
  const now = useNow(panelCards.some((card) => card.timer_started_at !== null) ? 1000 : 30000)
  const openCardId = params.cardId === undefined ? null : Number(params.cardId)
  const focus = (location.state as {focus?: CardFocus} | null)?.focus ?? null

  const panelPath = `/board/${panel.board_id}/panel/${panel.id}`
  const openCard = useCallback(
    (cardId: number, cardFocus: CardFocus) => navigate(`${panelPath}/card/${cardId}`, {state: {focus: cardFocus}}),
    [navigate, panelPath],
  )
  const closeCard = useCallback(() => navigate(panelPath), [navigate, panelPath])
  const shortcuts = usePanelShortcuts({panel, labels: ownLabels, cards: panelCards, openCard, notify: showToast, enabled: !readOnly})

  const close = () => setPending(null)
  const applyList = (result: WriteResult<{id: number}>) => store.applyLocal(EntityType.List, result.data.id, result.data, result.revision)
  const applyCard = (result: WriteResult<Card>) => store.applyLocal(EntityType.Card, result.data.id, result.data, result.revision)
  const report = (promise: Promise<unknown>) => promise.catch((err: unknown) => showToast(errorMessage(err)))
  // 移动等待生效期间按位置签名判断：卡片看所在列和序号，列表看序号。键与落点动画的 data-flip-key 一致。
  const cardSpot = (card: Card) => `${card.list_id}:${card.position}`
  const isCardMoving = (card: Card) => moves.isPending(`c:${card.id}`, cardSpot(card))
  const isListMoving = (list: List) => moves.isPending(`l:${list.id}`, String(list.position))

  function clearHints() {
    setListHint(null)
    setCardHint(null)
    setDraggingCard(null)
    setDragActive(false)
  }

  function onDragStart(event: DragStartEvent) {
    setDragActive(true)
    const dragged = parseDragId(event.active.id)
    // 记下当前位置，落点后给被挤开的卡片和列表播放动画。
    armFlip(listsRef.current, dragged.kind === 'card' ? `c:${dragged.id}` : `l:${dragged.id}`)
    if (dragged.kind === 'card') setDraggingCard(panelCards.find((card) => card.id === dragged.id) ?? null)
  }

  function onDragMove(event: DragMoveEvent) {
    if (!event.over) {
      setListHint(null)
      setCardHint(null)
      return
    }
    const start = startPoint(event.activatorEvent)
    const rect = event.over.rect
    const over = parseDragId(event.over.id)
    if (parseDragId(event.active.id).kind === 'list') {
      setListHint({overId: over.id, after: start.x + event.delta.x > rect.left + rect.width / 2})
      return
    }
    const listId = over.kind === 'card' ? panelCards.find((card) => card.id === over.id)?.list_id : over.id
    if (listId == null) return
    // 不是手动排序的列表按排序方式决定卡片的位置，只提示放进这个列表。
    const manual = active.find((list) => list.id === listId)?.sort_mode === 'manual'
    if (over.kind === 'card' && manual) {
      setCardHint({listId, cardId: over.id, after: start.y + event.delta.y > rect.top + rect.height / 2})
    } else {
      setCardHint({listId, cardId: null, after: true})
    }
  }

  function onDragEnd(event: DragEndEvent) {
    const currentList = listHint
    const currentCard = cardHint
    clearHints()
    const noChange = () => disarmFlip(listsRef.current)
    if (!event.over) return noChange()
    const dragged = parseDragId(event.active.id)
    if (dragged.kind === 'list') {
      const index = currentList ? listDropIndex(active, dragged.id, currentList.overId, currentList.after) : null
      if (index === null) return noChange()
      const request = listApi.reorderList(dragged.id, index).then(applyList)
      const list = active.find((item) => item.id === dragged.id)
      if (list) moves.track(`l:${list.id}`, String(list.position), request)
      void report(request)
      return
    }
    const target = currentCard ? active.find((list) => list.id === currentCard.listId) : undefined
    if (!target || !currentCard) return noChange()
    const drop = resolveCardDrop(panelCards, target, dragged.id, currentCard.cardId, currentCard.after)
    if (!drop) return noChange()
    const request = cardApi.moveCard(dragged.id, target.id, drop.index).then(applyCard)
    const card = panelCards.find((item) => item.id === dragged.id)
    if (card) moves.track(`c:${card.id}`, cardSpot(card), request)
    void report(request)
  }

  const columnActions = (list: List): ListColumnActions => ({
    rename: () => setPending({type: 'rename', list}),
    settings: () => setPending({type: 'settings', list}),
    rules: () => setPending({type: 'rules', list}),
    archive: () => setPending({type: 'archive', list}),
    archiveCards: () => setPending({type: 'archive-cards', list}),
    paste: () => shortcuts.pasteToList(list.id),
    copyCard: shortcuts.copyCard,
    cutCard: shortcuts.cutCard,
    editCard: (card) => openCard(card.id, null),
    openComposer: (end) => setComposer({listId: list.id, end}),
    closeComposer: () => setComposer(null),
    createCard: async (title, end) => {
      try {
        applyCard(await cardApi.createCard(list.id, title, end))
      } catch (err) {
        showToast(errorMessage(err))
        throw err
      }
    },
    openCard: (card) => (card.id === openCardId ? closeCard() : openCard(card.id, null)),
    hoverCard: shortcuts.hoverCard,
    hoverList: shortcuts.hoverList,
  })

  let dialog: ReactNode = null
  if (pending?.type === 'create') {
    dialog = (
      <NameDialog
        title={t('list.create')}
        initialName=""
        confirmText={t('common.create')}
        onClose={close}
        onSubmit={async (name) => applyList(await listApi.createList(panel.id, name))}
      />
    )
  } else if (pending?.type === 'rename') {
    const {list} = pending
    dialog = (
      <NameDialog
        title={t('list.rename')}
        initialName={list.name}
        confirmText={t('common.save')}
        onClose={close}
        onSubmit={async (name) => applyList(await listApi.renameList(list.id, name))}
      />
    )
  } else if (pending?.type === 'settings') {
    dialog = <ListSettingsDialog list={pending.list} onClose={close} />
  } else if (pending?.type === 'rules') {
    dialog = <ListRulesDialog list={pending.list} onClose={close} />
  } else if (pending?.type === 'archive') {
    const {list} = pending
    dialog = (
      <ConfirmDialog
        open
        title={t('list.archiveListTitle', {name: list.name})}
        description={t('list.archiveListDesc')}
        confirmText={t('common.archive')}
        danger
        onCancel={close}
        onConfirm={() => {
          close()
          void report(listApi.archiveList(list.id))
        }}
      />
    )
  } else if (pending?.type === 'archive-cards') {
    const {list} = pending
    dialog = (
      <ConfirmDialog
        open
        title={t('list.archiveAllCardsTitle', {name: list.name})}
        description={t('list.archiveAllCardsDesc')}
        confirmText={t('common.archive')}
        danger
        onCancel={close}
        onConfirm={() => {
          close()
          void report(cardApi.archiveAllCards(list.id))
        }}
      />
    )
  }

  const face = {
    labels: new Map(ownLabels.map((label) => [label.id, label])),
    levels: new Map(ownLevels.map((level) => [level.id, level])),
    timeZone: settings.timezone,
    now,
  }
  const isToday = view.todayOnly ? (card: Card) => isTodayCard(card, now, settings.timezone) : null

  return (
    <div ref={panelViewRef} className={openCardId === null ? 'panel-view' : 'panel-view drawer-open'}>
      <PanelToolbar
        view={view}
        onChange={updateView}
        labels={ownLabels}
        levels={ownLevels}
        lists={active}
      />
      {error && <p className="form-error panel-error">{error}</p>}
      {!loaded && !error && <p className="hint panel-loading">{t('common.loading')}</p>}
      {loaded && (
        <DndContext
          sensors={sensors}
          collisionDetection={collision}
          onDragStart={onDragStart}
          onDragMove={onDragMove}
          onDragOver={onDragMove}
          onDragEnd={onDragEnd}
          onDragCancel={() => { clearHints(); disarmFlip(listsRef.current) }}
          autoScroll={LIST_AUTO_SCROLL}
        >
          <div ref={listsRef} className={dragActive ? 'lists dragging' : 'lists'}>
            {active.map((list) => (
              <ListColumn
                key={list.id}
                list={list}
                cards={sortedCards(panelCards, list, ownLevels).filter((card) => matchesFilter(card, view.filter))}
                count={manualOrder(panelCards, list.id).length}
                face={face}
                progress={progress}
                attachments={attachmentInfo}
                isToday={isToday}
                moving={isListMoving(list)}
                isCardMoving={isCardMoving}
                selectedCardId={openCardId}
                listHint={listHint?.overId === list.id ? listHint : null}
                cardHint={cardHint?.listId === list.id ? cardHint : null}
                composer={composer?.listId === list.id ? composer.end : null}
                actions={columnActions(list)}
                readOnly={readOnly}
              />
            ))}
            {!readOnly && (
              <button type="button" className="add-list" onClick={() => setPending({type: 'create'})}>
                {t('list.createShort')}
              </button>
            )}
          </div>
          <DragOverlay dropAnimation={null}>
            {draggingCard && (
              <div className="card drag-overlay">
                <p className="card-title">{draggingCard.title}</p>
              </div>
            )}
          </DragOverlay>
        </DndContext>
      )}
      {loaded && <CardDrawer panel={panel} cardId={openCardId} openKey={location.key} focus={focus} onClose={closeCard} onNotice={showToast} />}
      {dialog}
      {toastElement}
    </div>
  )
}
