import {act, render, screen} from '@testing-library/react'
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'
import {ResultCode, getToken, setToken} from '../api/client'
import type {Me} from '../api/types'
import {useSession} from './context'
import {SessionProvider} from './SessionProvider'

const settings = {timezone: 'UTC', palette: 'clean' as const, locale: '', pan_modifier: 'ctrl', main_board_id: null, open_main_board_on_home: false, remind_template: '', due_template: '', shortcuts: {}, editor_mode: 'wysiwyg' as const}
const me: Me = {account: {id: 1, username: 'alice', role: 'user'}, settings, device_id: 1, shortcut_defaults: {}, shortcut_actions: []}

function Status() {
  return <p>{useSession().status}</p>
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
    setToken('device-token')
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
    render(
      <SessionProvider>
        <Status />
      </SessionProvider>,
    )

    await act(() => vi.advanceTimersByTimeAsync(0))
    expect(screen.getByText('unreachable')).toBeInTheDocument()
    expect(getToken()).toBe('device-token')

    // 第一次重试等 1 秒，仍是服务端错误；第二次重试等 2 秒后成功。
    await act(() => vi.advanceTimersByTimeAsync(1000))
    expect(screen.getByText('unreachable')).toBeInTheDocument()
    await act(() => vi.advanceTimersByTimeAsync(2000))
    expect(screen.getByText('authenticated')).toBeInTheDocument()
    expect(getToken()).toBe('device-token')
  })

  it('服务端返回未登录时清除令牌并进入未登录', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({code: ResultCode.Unauthorized, msg: '请先登录'}))))
    render(
      <SessionProvider>
        <Status />
      </SessionProvider>,
    )

    await act(() => vi.advanceTimersByTimeAsync(0))
    expect(screen.getByText('anonymous')).toBeInTheDocument()
    expect(getToken()).toBeNull()
  })
})
