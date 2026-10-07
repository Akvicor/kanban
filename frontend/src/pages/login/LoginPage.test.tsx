import {fireEvent, render, screen} from '@testing-library/react'
import {createMemoryRouter, RouterProvider} from 'react-router-dom'
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'
import {getToken, ResultCode} from '../../api/client'
import {SessionProvider} from '../../session/SessionProvider'
import {FakeWebSocket} from '../../test/fakeWebSocket'
import {LoginPage} from './LoginPage'

const me = {
  account: {id: 1, username: 'alice', role: 'user'},
  settings: {
    timezone: 'UTC',
    palette: 'clean', locale: '', pan_modifier: 'ctrl',
    main_board_id: null,
    open_main_board_on_home: false,
    remind_template: '',
    due_template: '',
    shortcuts: {},
  editor_mode: 'wysiwyg' as const,
  },
  device_id: 7,
  shortcut_defaults: {},
  shortcut_actions: [],
}

function renderLogin() {
  const router = createMemoryRouter(
    [
      {path: '/login', element: <LoginPage />},
      {path: '/', element: <p>主页内容</p>},
    ],
    {initialEntries: ['/login']},
  )
  render(
    <SessionProvider>
      <RouterProvider router={router} />
    </SessionProvider>,
  )
}

function fillAndSubmit() {
  fireEvent.change(screen.getByLabelText('用户名'), {target: {value: ' alice '}})
  fireEvent.change(screen.getByLabelText('密码'), {target: {value: 'password-123'}})
  fireEvent.click(screen.getByRole('button', {name: '登录'}))
}

describe('LoginPage', () => {
  beforeEach(() => {
    FakeWebSocket.reset()
    vi.stubGlobal('WebSocket', FakeWebSocket)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    localStorage.clear()
  })

  it('登录成功后保存令牌、进入主页并建立同步连接', async () => {
    const fetchMock = vi.fn(async () => new Response(JSON.stringify({code: ResultCode.Succeeded, data: {token: 'new-token', me}})))
    vi.stubGlobal('fetch', fetchMock)
    renderLogin()
    fillAndSubmit()

    expect(await screen.findByText('主页内容')).toBeInTheDocument()
    expect(getToken()).toBe('new-token')
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toBe('/api/auth/login')
    expect(JSON.parse(init.body as string)).toMatchObject({username: 'alice', password: 'password-123'})

    const socket = FakeWebSocket.latest()
    expect(socket.url).toMatch(/\/api\/sync\/ws$/)
    socket.open()
    expect(socket.sent[0]).toEqual({token: 'new-token', last_revision: 0})
  })

  it('登录失败时显示服务端信息', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({code: ResultCode.Unauthorized}))))
    renderLogin()
    fillAndSubmit()

    expect(await screen.findByRole('alert')).toHaveTextContent('请重新登录')
    expect(getToken()).toBeNull()
  })
})
