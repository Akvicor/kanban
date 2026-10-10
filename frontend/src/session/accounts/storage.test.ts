import {afterEach, describe, expect, it, vi} from 'vitest'
import {
  ACCOUNTS_STORAGE_KEY,
  exceedsLimit,
  MAX_ACCOUNTS,
  moveAccount,
  readAccounts,
  saveLogin,
  sessionChanged,
  watchOtherTabs,
  type AccountsState,
  type StoredAccount,
} from './storage'

function account(userId: number, token: string | null = `token-${userId}`): StoredAccount {
  return {userId, username: `user${userId}`, nickname: '', token}
}

function seed(state: AccountsState) {
  localStorage.setItem(ACCOUNTS_STORAGE_KEY, JSON.stringify(state))
}

/** 模拟其他标签页写入账号列表：直接改 localStorage 并派发 storage 事件。 */
function writeFromOtherTab(state: AccountsState) {
  const raw = JSON.stringify(state)
  localStorage.setItem(ACCOUNTS_STORAGE_KEY, raw)
  window.dispatchEvent(new StorageEvent('storage', {key: ACCOUNTS_STORAGE_KEY, newValue: raw}))
}

describe('账号列表', () => {
  afterEach(() => localStorage.clear())

  it('新账号加在最后；已在列表中的账号原位置更新令牌，并返回被替换的旧令牌', () => {
    seed({accounts: [account(1), account(2, null)], current: 1})

    expect(saveLogin({id: 3, username: 'user3'}, 'new-3')).toEqual({replacedToken: null})
    expect(saveLogin({id: 2, username: 'user2'}, 'new-2')).toEqual({replacedToken: null})
    expect(saveLogin({id: 1, username: 'user1', nickname: '甲'}, 'new-1')).toEqual({replacedToken: 'token-1'})

    const {accounts, current} = readAccounts()
    expect(accounts.map((item) => [item.userId, item.token])).toEqual([
      [1, 'new-1'],
      [2, 'new-2'],
      [3, 'new-3'],
    ])
    expect(accounts[0].nickname).toBe('甲')
    expect(current).toBe(1)
  })

  it('列表已满时只能登录已在列表中的账号', () => {
    seed({accounts: Array.from({length: MAX_ACCOUNTS}, (_, index) => account(index + 1, index === 0 ? null : `token-${index + 1}`)), current: 2})
    expect(exceedsLimit(1)).toBe(false)
    expect(exceedsLimit(99)).toBe(true)
  })

  it('拖动把账号放到另一个账号之前或之后', () => {
    seed({accounts: [account(1), account(2), account(3)], current: 1})
    moveAccount(3, 1, 'before')
    expect(readAccounts().accounts.map((item) => item.userId)).toEqual([3, 1, 2])
    moveAccount(3, 2, 'after')
    expect(readAccounts().accounts.map((item) => item.userId)).toEqual([1, 2, 3])
  })

  it('当前令牌不同，或令牌都为空但用户不同时，登录身份变化', () => {
    const base: AccountsState = {accounts: [account(1), account(2, null), account(3, null)], current: 1}
    expect(sessionChanged(base, {...base, accounts: [account(2, null), account(1), account(3, null)]})).toBe(false)
    expect(sessionChanged(base, {...base, current: 2})).toBe(true)
    expect(sessionChanged(base, {...base, accounts: [account(1, 'renewed'), account(2, null), account(3, null)]})).toBe(true)
    expect(sessionChanged({...base, current: 2}, {...base, current: 3})).toBe(true)
    expect(sessionChanged(base, {accounts: [], current: null})).toBe(true)
  })

  it('其他标签页换了当前账号时通知刷新，只改顺序时更新列表', () => {
    seed({accounts: [account(1), account(2)], current: 1})
    const onSessionChange = vi.fn()
    const stop = watchOtherTabs(onSessionChange)

    writeFromOtherTab({accounts: [account(2), account(1)], current: 1})
    expect(onSessionChange).not.toHaveBeenCalled()
    expect(readAccounts().accounts.map((item) => item.userId)).toEqual([2, 1])

    // 事件到达前本标签页已读到新内容，仍能识别出当前账号变化。
    localStorage.setItem(ACCOUNTS_STORAGE_KEY, JSON.stringify({accounts: [account(2), account(1)], current: 2}))
    readAccounts()
    window.dispatchEvent(new StorageEvent('storage', {key: ACCOUNTS_STORAGE_KEY}))
    expect(onSessionChange).toHaveBeenCalledOnce()
    stop()
  })
})
