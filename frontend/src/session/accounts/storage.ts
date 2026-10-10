/**
 * 本设备保存的已登录账号：账号列表和当前账号，存在 localStorage 的同一个键中，所有标签页共用。
 *
 * - 列表顺序由用户拖动调整，新登录的账号加在最后，最多 MAX_ACCOUNTS 个（已失效的也计入）。
 * - 每个账号保存自己的设备令牌；令牌为 null 表示登录已失效，账号仍留在原位置，重新登录后更新令牌。
 * - 请求层按当前账号取令牌（见 api/client.ts）。
 *
 * 本模块只读写本地数据，不发请求；切换、退出等需要请求服务端的流程见 actions.ts。
 */

/** 保存在本设备的一个已登录账号。 */
export interface StoredAccount {
  userId: number
  username: string
  nickname: string
  /** 设备令牌，null 表示登录已失效。 */
  token: string | null
}

/** 账号列表和当前账号的用户 ID（没有当前账号时为 null）。 */
export interface AccountsState {
  accounts: StoredAccount[]
  current: number | null
}

/** 本设备保存账号列表的键。 */
export const ACCOUNTS_STORAGE_KEY = 'kanban.accounts'

/** 最多保存的账号数。 */
export const MAX_ACCOUNTS = 10

const EMPTY: AccountsState = {accounts: [], current: null}

type Listener = () => void
const listeners = new Set<Listener>()

/** 最近一次读取或写入的内容和解析结果。内容不变时返回同一个对象，供 useSyncExternalStore 使用。 */
let cached: {raw: string | null; state: AccountsState} = {raw: null, state: EMPTY}

function isAccount(value: unknown): value is StoredAccount {
  if (typeof value !== 'object' || value === null) return false
  const account = value as Record<string, unknown>
  return (
    typeof account.userId === 'number' &&
    typeof account.username === 'string' &&
    typeof account.nickname === 'string' &&
    (account.token === null || typeof account.token === 'string')
  )
}

/** 解析保存的内容，格式不对的部分丢弃；当前账号不在列表中时视为没有当前账号。 */
function parseAccounts(raw: string | null): AccountsState {
  if (!raw) return EMPTY
  try {
    const value = JSON.parse(raw) as {accounts?: unknown; current?: unknown}
    const accounts = Array.isArray(value.accounts) ? value.accounts.filter(isAccount) : []
    const current = typeof value.current === 'number' && accounts.some((account) => account.userId === value.current) ? value.current : null
    return {accounts, current}
  } catch {
    return EMPTY
  }
}

/** 当前保存的账号列表。存储不可用时返回本次打开页面期间在内存中的列表。 */
export function readAccounts(): AccountsState {
  let raw: string | null
  try {
    raw = localStorage.getItem(ACCOUNTS_STORAGE_KEY)
  } catch {
    return cached.state
  }
  if (raw !== cached.raw) {
    cached = {raw, state: parseAccounts(raw)}
  }
  return cached.state
}

function emit(): void {
  listeners.forEach((listener) => listener())
}

/** 写入新的账号列表并通知订阅者。存储不可用时只保留在内存中。 */
function save(next: AccountsState): void {
  const raw = JSON.stringify(next)
  cached = {raw, state: next}
  try {
    localStorage.setItem(ACCOUNTS_STORAGE_KEY, raw)
  } catch {
    // 存储不可用（例如隐私模式配额为 0）时，本次打开页面期间仍可使用。
  }
  emit()
}

/** 订阅本标签页对账号列表的修改；其他标签页的修改见 watchOtherTabs。 */
export function subscribeAccounts(listener: Listener): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

/** 当前账号，没有时为 null。 */
export function currentAccount(state: AccountsState = readAccounts()): StoredAccount | null {
  return state.accounts.find((account) => account.userId === state.current) ?? null
}

/** 当前账号的令牌；没有当前账号或登录已失效时为 null。 */
export function currentToken(): string | null {
  return currentAccount()?.token ?? null
}

/** 账号的身份信息，来自登录响应或当前用户。 */
export interface AccountProfile {
  id: number
  username: string
  nickname?: string
}

/**
 * 保存一次登录：账号已在列表中（含已失效的）时更新原位置的令牌和名称，顺序不变；否则加在列表最后。
 * 返回被替换的旧令牌（没有或已失效时为 null），调用方负责吊销。不修改当前账号。
 */
export function saveLogin(profile: AccountProfile, token: string): {replacedToken: string | null} {
  const state = readAccounts()
  const entry: StoredAccount = {userId: profile.id, username: profile.username, nickname: profile.nickname ?? '', token}
  const existing = state.accounts.find((account) => account.userId === profile.id)
  const accounts = existing ? state.accounts.map((account) => (account.userId === profile.id ? entry : account)) : [...state.accounts, entry]
  save({...state, accounts})
  return {replacedToken: existing?.token && existing.token !== token ? existing.token : null}
}

/** 登录这个账号会不会超过数量上限：账号不在列表中且列表已满。 */
export function exceedsLimit(userId: number | null): boolean {
  const {accounts} = readAccounts()
  return accounts.length >= MAX_ACCOUNTS && !accounts.some((account) => account.userId === userId)
}

/** 设置当前账号，null 表示没有当前账号。 */
export function setCurrentAccount(userId: number | null): void {
  save({...readAccounts(), current: userId})
}

/** 用取回的当前用户或同步数据中的账号更新该账号保存的名称，名称没有变化时不写入。 */
export function updateAccountProfile(profile: AccountProfile): void {
  const state = readAccounts()
  const nickname = profile.nickname ?? ''
  const account = state.accounts.find((item) => item.userId === profile.id)
  if (!account || (account.username === profile.username && account.nickname === nickname)) return
  save({...state, accounts: state.accounts.map((item) => (item === account ? {...item, username: profile.username, nickname} : item))})
}

/** 令牌失效：把持有该令牌的账号标记为已失效，账号保留在原位置。 */
export function expireToken(token: string): void {
  const state = readAccounts()
  if (!state.accounts.some((account) => account.token === token)) return
  save({...state, accounts: state.accounts.map((account) => (account.token === token ? {...account, token: null} : account))})
}

/** 从列表中移除账号；移除的是当前账号时，当前账号变为空，由调用方决定切到哪个账号。 */
export function removeAccount(userId: number): void {
  const state = readAccounts()
  save({accounts: state.accounts.filter((account) => account.userId !== userId), current: state.current === userId ? null : state.current})
}

/** 清空全部账号。 */
export function clearAccounts(): void {
  save(EMPTY)
}

/** 把 userId 移到 targetId 之前或之后，用于拖动排序。 */
export function moveAccount(userId: number, targetId: number, position: 'before' | 'after'): void {
  const state = readAccounts()
  const moving = state.accounts.find((account) => account.userId === userId)
  if (!moving || userId === targetId) return
  const rest = state.accounts.filter((account) => account.userId !== userId)
  const targetIndex = rest.findIndex((account) => account.userId === targetId)
  if (targetIndex < 0) return
  rest.splice(position === 'before' ? targetIndex : targetIndex + 1, 0, moving)
  save({...state, accounts: rest})
}

/** 当前登录身份是否不同：令牌不同，或令牌都为空（已失效或没有账号）但用户不同。 */
export function sessionChanged(before: AccountsState, after: AccountsState): boolean {
  const a = currentAccount(before)
  const b = currentAccount(after)
  const tokenA = a?.token ?? null
  const tokenB = b?.token ?? null
  if (tokenA !== tokenB) return true
  return tokenA === null && (a?.userId ?? null) !== (b?.userId ?? null)
}

/**
 * 监听其他标签页对账号列表的修改：当前登录身份变化时调用 onSessionChange（由调用方刷新页面），
 * 否则只更新本标签页的列表。返回取消监听的函数。
 */
export function watchOtherTabs(onSessionChange: () => void): () => void {
  // 本标签页最后确认的列表。不能直接用 readAccounts 的缓存作为修改前的状态：
  // 其他标签页写入后、storage 事件到达前，本标签页的读取可能已经取到新内容。
  let known = readAccounts()
  const unsubscribe = subscribeAccounts(() => {
    known = readAccounts()
  })
  const onStorage = (event: StorageEvent) => {
    // key 为 null 表示其他标签页清空了整个 localStorage。
    if (event.key !== ACCOUNTS_STORAGE_KEY && event.key !== null) return
    const before = known
    known = readAccounts()
    if (sessionChanged(before, known)) {
      onSessionChange()
      return
    }
    emit()
  }
  window.addEventListener('storage', onStorage)
  return () => {
    window.removeEventListener('storage', onStorage)
    unsubscribe()
  }
}
