import {useDraggable, useDroppable} from '@dnd-kit/core'
import {useCallback, useRef, useState, type CSSProperties} from 'react'
import type {Card, Label, PriorityLevel} from '../api/types'
import {Menu, MenuItem, MenuSeparator} from '../components/Menu'
import {dueStatus, formatAge, formatDuration, timerSeconds} from './model'
import './CardItem.css'
import {useT} from '../i18n'

/** 两次点击在这个间隔内算作双击，打开卡片详情。 */
const DOUBLE_CLICK_MS = 400

export interface CardFaceContext {
  labels: Map<number, Label>
  levels: Map<number, PriorityLevel>
  timeZone: string
  now: number
  showAge: boolean
  progress: {done: number; total: number}
  /** 附件数和封面缩略图地址（没有封面时为 null）。 */
  attachments: {count: number; coverUrl: string | null}
}

interface CardItemProps {
  card: Card
  context: CardFaceContext
  /** 不满足「只看今日」时变灰，仍可点击和拖动。 */
  dimmed: boolean
  selected: boolean
  /** 已松手、移动尚未生效：保持拖动时的半透明，直到卡片出现在新位置或请求结束。 */
  moving: boolean
  /** 拖动中的落点提示：放在这张卡片的上方或下方。 */
  dropHint: 'before' | 'after' | null
  onOpen: () => void
  onHover: (hovered: boolean) => void
  /** 放入剪贴板，供列菜单粘贴到列首。不传时不显示卡片菜单，例如只读查看。 */
  onCopy?: () => void
  onCut?: () => void
  /** 从菜单打开卡片详情，不随双击切换关闭。 */
  onEdit?: () => void
}

/**
 * 卡片正面：封面、标签、标题，以及优先级、到期状态、任务进度、定时器、附件数和卡龄。
 * 连点两下打开卡片详情；详情已打开时再连点两下这张卡片则关闭。单次点击不打开。菜单中的「编辑」也会打开。
 * 触屏上浏览器不保证合成双击事件，因此这里按两次点击的间隔自行判断。
 * 鼠标按下后移动 5 像素开始拖动，触屏上长按开始拖动。
 */
export function CardItem({card, context, dimmed, selected, moving, dropHint, onOpen, onHover, onCopy, onCut, onEdit}: CardItemProps) {
  const t = useT()
  const dragId = `card:${card.id}`
  const editable = onCopy !== undefined
  const {attributes, listeners, setNodeRef: setDragRef, isDragging} = useDraggable({id: dragId, disabled: !editable})
  const [menuOpen, setMenuOpen] = useState(false)
  /** 上一次点击的时刻，两次点击在阈值内算作双击。 */
  const lastClickRef = useRef(0)
  const {setNodeRef: setDropRef} = useDroppable({id: dragId})
  const setRef = useCallback(
    (element: HTMLElement | null) => {
      setDragRef(element)
      setDropRef(element)
    },
    [setDragRef, setDropRef],
  )
  const status = dueStatus(card, context.now, context.timeZone)
  const level = card.priority_level_id === null ? null : context.levels.get(card.priority_level_id)
  const labels = card.label_ids.map((id) => context.labels.get(id)).filter((label): label is Label => label !== undefined)
  const running = card.timer_started_at !== null
  const seconds = timerSeconds(card, context.now)

  let className = 'card'
  if (dimmed) className += ' dimmed'
  if (selected) className += ' selected'
  if (isDragging || moving) className += ' dragging'
  if (dropHint) className += ` drop-${dropHint}`

  return (
    <article
      ref={setRef}
      data-flip-key={`c:${card.id}`}
      {...(editable ? {...attributes, ...listeners} : {})}
      className={className}
      onClick={(event) => {
        if ((event.target as HTMLElement).closest('button')) return
        const now = Date.now()
        if (now - lastClickRef.current < DOUBLE_CLICK_MS) {
          lastClickRef.current = 0
          onOpen()
          return
        }
        lastClickRef.current = now
      }}
      onMouseEnter={() => onHover(true)}
      onMouseLeave={() => onHover(false)}
      aria-label={card.title}
    >
      {context.attachments.coverUrl && <img className="cover" src={context.attachments.coverUrl} alt="" loading="lazy" draggable={false} />}
      {labels.length > 0 && (
        <div className="labels">
          {labels.map((label) => (
            <span key={label.id} className="label" style={{'--c': label.color} as CSSProperties}>
              {label.name || '\u00a0'}
            </span>
          ))}
        </div>
      )}
      {editable && (
        <Menu
          open={menuOpen}
          onOpenChange={setMenuOpen}
          trigger={
            <button
              type="button"
              className="card-more"
              aria-label={t('common.menuOf', {name: card.title})}
              onPointerDown={(event) => event.stopPropagation()}
              onClick={(event) => {
                event.stopPropagation()
              }}
            >
              ⋯
            </button>
          }
        >
          <MenuItem onSelect={() => onEdit?.()}>{t('common.edit')}</MenuItem>
          <MenuSeparator />
          <MenuItem
            onSelect={() => {
              onCopy()
            }}
          >
            {t('common.copy')}
          </MenuItem>
          <MenuItem
            onSelect={() => {
              onCut?.()
            }}
          >
            {t('common.cut')}
          </MenuItem>
        </Menu>
      )}
      <p className="card-title">{card.title}</p>
      <div className="meta">
        {level && (
          <span className="prio" style={{'--c': level.color} as CSSProperties}>
            {level.name}
          </span>
        )}
        {status && <span className={`due ${status}`}>{t(status === 'overdue' ? 'card.overdue' : status === 'today' ? 'card.today' : 'card.soon')}</span>}
        {context.progress.total > 0 && (
          <span className="meta-item" title={t('card.taskProgress')}>
            ☑ {context.progress.done}/{context.progress.total}
          </span>
        )}
        {(running || seconds > 0) && (
          <span className={running ? 'meta-item timer running' : 'meta-item timer'} title={t('card.timer')}>
            ⏱ {formatDuration(seconds)}
          </span>
        )}
        {context.attachments.count > 0 && (
          <span className="meta-item" title={t('card.attachmentCount')}>
            📎 {context.attachments.count}
          </span>
        )}
        {card.description && (
          <span className="meta-item" title={t('card.hasDescription')}>
            ≡
          </span>
        )}
        {context.showAge && (
          <span className="meta-item age" title={t('card.age')}>
            {formatAge(card.created_at, context.now)}
          </span>
        )}
      </div>
    </article>
  )
}
