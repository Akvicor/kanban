import {act, render, screen} from '@testing-library/react'
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'
import {ResultCode} from '../api/client'
import type {Me} from '../api/types'
import {reloadToHome} from './accounts/reload'
import {ACCOUNTS_STORAGE_KEY, currentToken, readAccounts, type AccountsState} from './accounts/storage'
import {useSession} from './context'
import {SessionProvider} from './SessionProvider'

vi.mock('./accounts/reload', () => ({reloadToHome: vi.fn()}))

const settings = {timezone: 'UTC', palette: 'clean' as const, locale: '', pan_modifier: 'ctrl', main_board_id: null, open_main_board_on_home: false, remind_template: '', due_template: '', shortcuts: {}, editor_mode: 'wysiwyg' as const}
const me: Me = {account: {id: 1, username: 'alice', role: 'user'}, settings, device_id: 1, shortcut_defaults: {}, shortcut_actions: []}

const saved: AccountsState = {
  accounts: [
    {userId: 1, username: 'alice', nickname: '', token: 'device-token'},
    {userId: 2, username: 'bob', nickname: '', token: 'bob-token'},
  ],
  current: 1,
}

function Status() {
  return <p>{useSession().status}</p>
}

function renderSession() {
  render(
    <SessionProvider>
      <Status />
    </SessionProvider>,
  )
}

/** 同步连接不在这里验证，用不会连接的替身代替。 */
class IdleSocket {
  addEventListener() {}
  close() {}
  send() {}
}

describe('SessionProvider', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.stubGlobal('WebSocket', IdleSocket)
    vi.mocked(reloadToHome).mockClear()
    localStorage.setItem(ACCOUNTS_STORAGE_KEY, JSON.stringify(saved))
  })
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
    localStorage.clear()
  })

  it('连不上服务端时保留令牌并自动重试，连上后进入已登录', async () => {
    const fetchMock = vi
      .fn()
      .mockRejectedValueOnce(new TypeError('Failed to fetch'))
      .mockResolvedValueOnce(new Response('bad gateway', {status: 502}))
      .mockImplementation(async (path: string) => new Response(JSON.stringify(path === '/api/user/me' ? {code: ResultCode.Succeeded, data: me} : {code: ResultCode.Succeeded})))
    vi.stubGlobal('fetch', fetchMock)
    renderSession()

    await act(() => vi.advanceTimersByTimeAsync(0))
    expect(screen.getByText('unreachable')).toBeInTheDocument()
    expect(currentToken()).toBe('device-token')

    // 第一次重试等 1 秒，仍是服务端错误；第二次重试等 2 秒后成功。
    await act(() => vi.advanceTimersByTimeAsync(1000))
    expect(screen.getByText('unreachable')).toBeInTheDocument()
    await act(() => vi.advanceTimersByTimeAsync(2000))
    expect(screen.getByText('authenticated')).toBeInTheDocument()
    expect(currentToken()).toBe('device-token')
  })

  it('令牌已失效时把当前账号标记为已失效并进入未登录，不再重试', async () => {
    const fetchMock = vi.fn(async () => new Response(JSON.stringify({code: 1002})))
    vi.stubGlobal('fetch', fetchMock)
    renderSession()

    await act(() => vi.advanceTimersByTimeAsync(0))
    expect(screen.getByText('anonymous')).toBeInTheDocument()
    await act(() => vi.advanceTimersByTimeAsync(60_000))
    expect(screen.getByText('anonymous')).toBeInTheDocument()
    expect(fetchMock).toHaveBeenCalledOnce()
    expect(readAccounts()).toMatchObject({current: 1, accounts: [{userId: 1, token: null}, {userId: 2, token: 'bob-token'}]})
  })

  it('其他标签页换了当前账号时刷新页面，只调整顺序时不刷新', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({code: ResultCode.Succeeded, data: me}))))
    renderSession()
    await act(() => vi.advanceTimersByTimeAsync(0))

    const fromOtherTab = (state: AccountsState) => {
      localStorage.setItem(ACCOUNTS_STORAGE_KEY, JSON.stringify(state))
      window.dispatchEvent(new StorageEvent('storage', {key: ACCOUNTS_STORAGE_KEY}))
    }
    fromOtherTab({accounts: [saved.accounts[1], saved.accounts[0]], current: 1})
    expect(reloadToHome).not.toHaveBeenCalled()
    fromOtherTab({accounts: saved.accounts, current: 2})
    expect(reloadToHome).toHaveBeenCalledOnce()
  })
})
