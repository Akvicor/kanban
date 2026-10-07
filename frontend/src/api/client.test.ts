import {afterEach, describe, expect, it, vi} from 'vitest'
import {ApiError, errorMessage, getToken, http, onUnauthorized, ResultCode, setToken, TOKEN_STORAGE_KEY} from './client'

/** 让 fetch 返回指定的响应体，并记录请求。 */
function mockFetch(body: unknown) {
  const fetchMock = vi.fn(async () => new Response(JSON.stringify(body)))
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('http', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    localStorage.clear()
    onUnauthorized(null)
  })

  it('成功时返回 data，并带上登录令牌', async () => {
    setToken('abc')
    const fetchMock = mockFetch({code: ResultCode.Succeeded, data: {id: 1}})
    await expect(http.post('/user/settings/update', {palette: 'dark'})).resolves.toEqual({id: 1})

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toBe('/api/user/settings/update')
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer abc')
    expect(init.body).toBe('{"palette":"dark"}')
  })

  it('业务失败时抛出带服务端信息的错误', async () => {
    mockFetch({code: 1105})
    const error = await http.post('/admin/user/create').catch((err: unknown) => err)
    expect(error).toBeInstanceOf(ApiError)
    expect((error as ApiError).code).toBe(1105)
    expect(errorMessage(error)).toBe('用户名已被使用')
  })

  it('登录失效时清除令牌并通知', async () => {
    setToken('expired')
    const handler = vi.fn()
    onUnauthorized(handler)
    mockFetch({code: ResultCode.Unauthorized, msg: '登录已失效'})

    await expect(http.get('/user/me')).rejects.toBeInstanceOf(ApiError)
    expect(handler).toHaveBeenCalledOnce()
    expect(getToken()).toBeNull()
    expect(localStorage.getItem(TOKEN_STORAGE_KEY)).toBeNull()
  })

  it('网络错误时给出可展示的信息', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => {
      throw new TypeError('network')
    }))
    await expect(http.get('/user/me')).rejects.toThrow('无法连接服务器')
  })
})
