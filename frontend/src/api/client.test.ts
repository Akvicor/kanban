import {afterEach, describe, expect, it, vi} from 'vitest'
import {ACCOUNTS_STORAGE_KEY, readAccounts, type AccountsState} from '../session/accounts/storage'
import {ApiError, errorMessage, http, onUnauthorized, pendingRequestCount, ResultCode} from './client'

/** 让 fetch 返回指定的响应体，并记录请求。 */
function mockFetch(body: unknown) {
  const fetchMock = vi.fn(async () => new Response(JSON.stringify(body)))
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

function authorization(fetchMock: ReturnType<typeof mockFetch>, index = 0): string | undefined {
  const [, init] = fetchMock.mock.calls[index] as unknown as [string, RequestInit]
  return (init.headers as Record<string, string>).Authorization
}

/** 本设备保存两个账号，alice 为当前账号。 */
function seedAccounts() {
  const state: AccountsState = {
    accounts: [
      {userId: 1, username: 'alice', nickname: '', token: 'alice-token'},
      {userId: 2, username: 'bob', nickname: '', token: 'bob-token'},
    ],
    current: 1,
  }
  localStorage.setItem(ACCOUNTS_STORAGE_KEY, JSON.stringify(state))
}

describe('http', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    localStorage.clear()
    onUnauthorized(null)
  })

  it('成功时返回 data，并带上当前账号的令牌', async () => {
    seedAccounts()
    const fetchMock = mockFetch({code: ResultCode.Succeeded, data: {id: 1}})
    await expect(http.post('/user/settings/update', {palette: 'dark'})).resolves.toEqual({id: 1})

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toBe('/api/user/settings/update')
    expect(authorization(fetchMock)).toBe('Bearer alice-token')
    expect(init.body).toBe('{"palette":"dark"}')
  })

  it('可以显式使用指定账号的令牌', async () => {
    seedAccounts()
    const fetchMock = mockFetch({code: ResultCode.Succeeded})
    await http.post('/auth/logout', {}, {token: 'bob-token'})
    expect(authorization(fetchMock)).toBe('Bearer bob-token')
  })

  it('业务失败时抛出带服务端信息的错误', async () => {
    mockFetch({code: 1105})
    const error = await http.post('/admin/user/create').catch((err: unknown) => err)
    expect(error).toBeInstanceOf(ApiError)
    expect((error as ApiError).code).toBe(1105)
    expect(errorMessage(error)).toBe('用户名已被使用')
  })

  it.each([1002, 1003, 1004])('当前账号的令牌失效（%i）时标记为已失效并通知', async (code) => {
    seedAccounts()
    const handler = vi.fn()
    onUnauthorized(handler)
    mockFetch({code})

    await expect(http.get('/user/me')).rejects.toBeInstanceOf(ApiError)
    expect(handler).toHaveBeenCalledOnce()
    const {accounts, current} = readAccounts()
    expect(current).toBe(1)
    expect(accounts.map((account) => account.token)).toEqual([null, 'bob-token'])
  })

  it('其他账号的令牌失效时只标记该账号，不通知', async () => {
    seedAccounts()
    const handler = vi.fn()
    onUnauthorized(handler)
    mockFetch({code: 1002})

    await expect(http.post('/auth/logout', {}, {token: 'bob-token'})).rejects.toBeInstanceOf(ApiError)
    expect(handler).not.toHaveBeenCalled()
    expect(readAccounts().accounts.map((account) => account.token)).toEqual(['alice-token', null])
  })

  it('统计尚未结束的请求', async () => {
    let respond: (response: Response) => void = () => {}
    vi.stubGlobal('fetch', vi.fn(() => new Promise<Response>((resolve) => (respond = resolve))))
    const request = http.get('/user/me')
    expect(pendingRequestCount()).toBe(1)
    respond(new Response(JSON.stringify({code: ResultCode.Succeeded})))
    await request
    expect(pendingRequestCount()).toBe(0)
  })

  it('网络错误时给出可展示的信息', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => {
      throw new TypeError('network')
    }))
    await expect(http.get('/user/me')).rejects.toThrow('无法连接服务器')
    expect(pendingRequestCount()).toBe(0)
  })
})
