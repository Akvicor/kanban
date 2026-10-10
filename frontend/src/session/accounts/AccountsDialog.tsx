import {DndContext, pointerWithin, useDraggable, useDroppable, type DragEndEvent, type DragMoveEvent} from '@dnd-kit/core'
import {createContext, useCallback, useContext, useState} from 'react'
import {useLocation, useNavigate} from 'react-router-dom'
import {ConfirmDialog} from '../../components/ConfirmDialog'
import {Dialog} from '../../components/Dialog'
import {useDragSensors} from '../../hooks/useDragSensors'
import {useT} from '../../i18n'
import {displayName} from '../account'
import {canLeavePage, logoutAccount, logoutAll, switchAccount} from './actions'
import {MAX_ACCOUNTS, moveAccount, type StoredAccount} from './storage'
import {useAccounts} from './useAccounts'
import './AccountsDialog.css'

interface AccountsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /**
   * 只用于切换：不列出当前账号，不能排序，也没有登录新账号和退出当前、全部账号。
   * 当前账号失效、停留在登录页时使用。
   */
  switchOnly?: boolean
}

/** 拖动时的放置位置：放在 overId 之前或之后。 */
interface DropHint {
  overId: number
  position: 'before' | 'after'
}

const HintContext = createContext<DropHint | null>(null)

/** 等待二次确认的退出操作。 */
type PendingSignOut = {kind: 'account'; account: StoredAccount} | {kind: 'current'} | {kind: 'all'}

/** 开始拖动时指针的纵坐标，鼠标和触摸都适用。 */
function startY(event: Event): number {
  if ('touches' in event) {
    const touch = (event as TouchEvent).touches[0] ?? (event as TouchEvent).changedTouches[0]
    return touch?.clientY ?? 0
  }
  return (event as PointerEvent).clientY
}

/**
 * 账号弹窗：本设备已登录的账号，拖动左侧把手调整顺序；每个账号可以切换或退出。
 * 还可以登录新账号、退出当前账号、退出全部账号。所有退出都需要二次确认；
 * 切换和会刷新页面的退出在有上传或请求进行中时不执行，提示稍后再试。
 */
export function AccountsDialog({open, onOpenChange, switchOnly = false}: AccountsDialogProps) {
  const t = useT()
  const navigate = useNavigate()
  const location = useLocation()
  const {accounts, current} = useAccounts()
  const sensors = useDragSensors()
  const [hint, setHint] = useState<DropHint | null>(null)
  const [pending, setPending] = useState<PendingSignOut | null>(null)
  const [working, setWorking] = useState(false)
  const [error, setError] = useState('')

  const rows = switchOnly ? accounts.filter((account) => account.userId !== current) : accounts
  const sortable = !switchOnly && rows.length > 1
  const full = accounts.length >= MAX_ACCOUNTS

  /** 执行操作；leavesPage 表示操作会刷新页面，需要先确认没有进行中的上传和请求。 */
  async function run(leavesPage: boolean, action: () => Promise<void>) {
    if (leavesPage && !canLeavePage()) {
      setError(t('account.busy'))
      return
    }
    setError('')
    setWorking(true)
    try {
      await action()
    } finally {
      setWorking(false)
    }
  }

  function confirmSignOut() {
    const target = pending
    setPending(null)
    if (!target) return
    if (target.kind === 'all') {
      void run(true, logoutAll)
    } else if (target.kind === 'current' && current !== null) {
      void run(true, () => logoutAccount(current))
    } else if (target.kind === 'account') {
      void run(target.account.userId === current, () => logoutAccount(target.account.userId))
    }
  }

  function onDragMove(event: DragMoveEvent) {
    if (!event.over) {
      setHint(null)
      return
    }
    const rect = event.over.rect
    const ratio = (startY(event.activatorEvent) + event.delta.y - rect.top) / rect.height
    setHint({overId: Number(event.over.id), position: ratio < 0.5 ? 'before' : 'after'})
  }

  function onDragEnd(event: DragEndEvent) {
    const target = hint
    setHint(null)
    if (target && event.over) {
      moveAccount(Number(event.active.id), target.overId, target.position)
    }
  }

  function addAccount() {
    onOpenChange(false)
    navigate('/login', {state: {addAccount: true, from: location.pathname}})
  }

  const pendingName = pending?.kind === 'account' ? displayName(pending.account) : ''
  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange} title={switchOnly ? t('account.switchTitle') : t('account.title')} description={sortable ? t('account.description') : undefined}>
        <DndContext sensors={sensors} collisionDetection={pointerWithin} onDragMove={onDragMove} onDragOver={onDragMove} onDragEnd={onDragEnd} onDragCancel={() => setHint(null)}>
          <HintContext.Provider value={hint}>
            <ul className="account-list">
              {rows.map((account) => (
                <AccountRow
                  key={account.userId}
                  account={account}
                  isCurrent={account.userId === current}
                  sortable={sortable}
                  disabled={working}
                  onSwitch={() => void run(true, () => switchAccount(account.userId))}
                  onSignOut={() => setPending({kind: 'account', account})}
                />
              ))}
            </ul>
          </HintContext.Provider>
        </DndContext>
        {error && (
          <p className="form-error" role="alert">
            {error}
          </p>
        )}
        {!switchOnly && (
          <div className="account-actions">
            <button type="button" className="btn" disabled={working || full} onClick={addAccount}>
              {t('account.addAccount')}
            </button>
            <button type="button" className="btn danger" disabled={working} onClick={() => setPending({kind: 'current'})}>
              {t('account.signOutCurrent')}
            </button>
            {accounts.length > 1 && (
              <button type="button" className="btn danger" disabled={working} onClick={() => setPending({kind: 'all'})}>
                {t('account.signOutAll')}
              </button>
            )}
          </div>
        )}
        {!switchOnly && full && <p className="hint">{t('account.limitReached', {max: MAX_ACCOUNTS})}</p>}
      </Dialog>
      <ConfirmDialog
        open={pending !== null}
        title={
          pending?.kind === 'all' ? t('account.signOutAllTitle') : pending?.kind === 'current' ? t('account.signOutCurrent') : t('account.signOutTitle', {name: pendingName})
        }
        description={
          pending?.kind === 'all'
            ? t('account.signOutAllDesc', {count: accounts.length})
            : pending?.kind === 'current' || (pending?.kind === 'account' && pending.account.userId === current)
              ? t('account.signOutCurrentDesc')
              : t('account.signOutDesc')
        }
        confirmText={t('account.signOut')}
        danger
        onConfirm={confirmSignOut}
        onCancel={() => setPending(null)}
      />
    </>
  )
}

interface AccountRowProps {
  account: StoredAccount
  isCurrent: boolean
  sortable: boolean
  disabled: boolean
  onSwitch: () => void
  onSignOut: () => void
}

/** 账号列表中的一行：左侧拖动把手，中间名称和状态，右侧切换（或重新登录）和退出。 */
function AccountRow({account, isCurrent, sortable, disabled, onSwitch, onSignOut}: AccountRowProps) {
  const t = useT()
  const hint = useContext(HintContext)
  const {listeners, setNodeRef: setDragRef, setActivatorNodeRef, isDragging} = useDraggable({id: account.userId, disabled: !sortable})
  const {setNodeRef: setDropRef} = useDroppable({id: account.userId, disabled: !sortable})
  const setRef = useCallback(
    (element: HTMLElement | null) => {
      setDragRef(element)
      setDropRef(element)
    },
    [setDragRef, setDropRef],
  )
  const expired = account.token === null

  let className = 'account-row'
  if (isDragging) className += ' dragging'
  if (hint?.overId === account.userId) className += ` drop-${hint.position}`

  return (
    <li ref={setRef} className={className}>
      {sortable && (
        <span ref={setActivatorNodeRef} className="account-handle" aria-label={t('account.dragHandle')} title={t('account.dragHandle')} {...listeners}>
          ⋮⋮
        </span>
      )}
      <div className="account-info">
        <strong>
          {displayName(account)}
          {isCurrent && <span className="tag">{t('account.current')}</span>}
          {expired && <span className="tag danger">{t('account.expired')}</span>}
        </strong>
        <span className="hint">{account.username}</span>
      </div>
      {!isCurrent && (
        <button type="button" className="btn sm" disabled={disabled} onClick={onSwitch}>
          {expired ? t('account.relogin') : t('account.switch')}
        </button>
      )}
      <button type="button" className="btn sm danger" disabled={disabled} onClick={onSignOut}>
        {t('account.signOut')}
      </button>
    </li>
  )
}
