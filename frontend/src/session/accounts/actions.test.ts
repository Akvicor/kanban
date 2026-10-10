import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'
import {ResultCode} from '../../api/client'
import type {Me} from '../../api/types'
import {uploadQueue} from '../../upload/queue'
import {addAccount, canLeavePage, logoutAccount, logoutAll, switchAccount} from './actions'
import {reloadToHome} from './reload'
import {ACCOUNTS_STORAGE_KEY, readAccounts, type AccountsState, type StoredAccount} from './storage'

vi.mock('./reload', () => ({reloadToHome: vi.fn()}))

function account(userId: number, token: string | null = `token-${userId}`): StoredAccount {
  return {userId, username: `user${userId}`, nickname: '', token}
}

function seed(state: AccountsState) {
  localStorage.setItem(ACCOUNTS_STORAGE_KEY, JSON.stringify(state))
}

/** 记录每个请求的路径和令牌；codes 按路径和令牌指定失败的结果码。 */
function mockServer(codes: Record<string, number> = {}) {
  const calls: string[] = []
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url: string, init: RequestInit) => {
      const token = ((init.headers as Record<string, string>).Authorization ?? '').replace('Bearer ', '')
      const call = `${url} ${token}`
      calls.push(call)
      return new Response(JSON.stringify({code: codes[call] ?? ResultCode.Succeeded}))
    }),
  )
  return calls
}

const settings = {timezone: 'UTC', palette: 'clean' as const, locale: '', pan_modifier: 'ctrl', main_board_id: null, open_main_board_on_home: false, remind_template: '', due_template: '', shortcuts: {}, editor_mode: 'wysiwyg' as const}

function loginResult(userId: number, token: string): {token: string; me: Me} {
  return {token, me: {account: {id: userId, username: `user${userId}`, role: 'user'}, settings, device_id: 1, shortcut_defaults: {}, shortcut_actions: []}}
}

describe('账号操作', () => {
  beforeEach(() => vi.mocked(reloadToHome).mockClear())
  afterEach(() => {
    vi.unstubAllGlobals()
    localStorage.clear()
  })

  it('切换时注销原账号的文件令牌，为目标账号签发后切换并刷新', async () => {
    seed({accounts: [account(1), account(2)], current: 1})
    const calls = mockServer()
    await switchAccount(2)
    expect(calls).toEqual(['/api/auth/file_cookie/revoke token-1', '/api/auth/file_cookie token-2'])
    expect(readAccounts().current).toBe(2)
    expect(reloadToHome).toHaveBeenCalledOnce()
  })

  it('切换到已失效的账号时不签发文件令牌，刷新后进入重新登录', async () => {
    seed({accounts: [account(1), account(2, null)], current: 1})
    const calls = mockServer()
    await switchAccount(2)
    expect(calls).toEqual(['/api/auth/file_cookie/revoke token-1'])
    expect(readAccounts().current).toBe(2)
    expect(reloadToHome).toHaveBeenCalledOnce()
  })

  it('已登录时添加账号：加在最后并切换过去', async () => {
    seed({accounts: [account(1)], current: 1})
    const calls = mockServer()
    await addAccount(loginResult(2, 'token-2'))
    expect(calls).toEqual(['/api/auth/file_cookie/revoke token-1', '/api/auth/file_cookie token-2'])
    expect(readAccounts()).toMatchObject({current: 2, accounts: [{userId: 1}, {userId: 2, token: 'token-2'}]})
    expect(reloadToHome).toHaveBeenCalledOnce()
  })

  it('重复登录已在列表中的账号时原位置更新令牌，并吊销旧令牌', async () => {
    seed({accounts: [account(1), account(2)], current: 1})
    const calls = mockServer()
    await addAccount(loginResult(2, 'renewed-2'))
    expect(calls).toEqual(['/api/auth/logout token-2', '/api/auth/file_cookie/revoke token-1', '/api/auth/file_cookie renewed-2'])
    expect(readAccounts().accounts.map((item) => [item.userId, item.token])).toEqual([
      [1, 'token-1'],
      [2, 'renewed-2'],
    ])
  })

  it('退出其他账号时不刷新，并为当前账号重新签发文件 Cookie', async () => {
    seed({accounts: [account(1), account(2)], current: 1})
    const calls = mockServer()
    await logoutAccount(2)
    expect(calls).toEqual(['/api/auth/logout token-2', '/api/auth/file_cookie token-1'])
    expect(readAccounts()).toEqual({accounts: [account(1)], current: 1})
    expect(reloadToHome).not.toHaveBeenCalled()
  })

  it('退出已失效的账号时只从列表移除', async () => {
    seed({accounts: [account(1), account(2, null)], current: 1})
    const calls = mockServer()
    await logoutAccount(2)
    expect(calls).toEqual([])
    expect(readAccounts().accounts).toEqual([account(1)])
  })

  it('退出当前账号后切换到列表中的第一个账号；吊销失败时本地照常移除', async () => {
    seed({accounts: [account(1), account(2), account(3)], current: 2})
    const calls = mockServer({'/api/auth/logout token-2': ResultCode.Failed})
    await logoutAccount(2)
    expect(calls).toEqual(['/api/auth/logout token-2', '/api/auth/file_cookie token-1'])
    expect(readAccounts()).toMatchObject({current: 1, accounts: [{userId: 1}, {userId: 3}]})
    expect(reloadToHome).toHaveBeenCalledOnce()
  })

  it('退出最后一个账号后没有当前账号', async () => {
    seed({accounts: [account(1)], current: 1})
    mockServer()
    await logoutAccount(1)
    expect(readAccounts()).toEqual({accounts: [], current: null})
    expect(reloadToHome).toHaveBeenCalledOnce()
  })

  it('退出全部账号时逐个吊销有效令牌并清空列表', async () => {
    seed({accounts: [account(1), account(2, null), account(3)], current: 3})
    const calls = mockServer()
    await logoutAll()
    expect(calls).toEqual(['/api/auth/logout token-1', '/api/auth/logout token-3'])
    expect(readAccounts()).toEqual({accounts: [], current: null})
    expect(reloadToHome).toHaveBeenCalledOnce()
  })

  it('有上传进行中或请求未结束时不能离开页面', async () => {
    expect(canLeavePage()).toBe(true)
    const busy = vi.spyOn(uploadQueue, 'isBusy').mockReturnValue(true)
    expect(canLeavePage()).toBe(false)
    busy.mockRestore()

    seed({accounts: [account(1), account(2)], current: 1})
    let respond: (response: Response) => void = () => {}
    vi.stubGlobal('fetch', vi.fn(() => new Promise<Response>((resolve) => (respond = resolve))))
    const pending = logoutAccount(2)
    expect(canLeavePage()).toBe(false)
    respond(new Response(JSON.stringify({code: ResultCode.Succeeded})))
    vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({code: ResultCode.Succeeded}))))
    await pending
    expect(canLeavePage()).toBe(true)
  })
})
