import {fireEvent, render, screen} from '@testing-library/react'
import {createMemoryRouter, RouterProvider} from 'react-router-dom'
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'
import {ResultCode} from '../../api/client'
import {reloadToHome} from '../../session/accounts/reload'
import {ACCOUNTS_STORAGE_KEY, currentToken, readAccounts, type AccountsState} from '../../session/accounts/storage'
import {SessionProvider} from '../../session/SessionProvider'
import {FakeWebSocket} from '../../test/fakeWebSocket'
import {LoginPage} from './LoginPage'

vi.mock('../../session/accounts/reload', () => ({reloadToHome: vi.fn()}))

const settings = {
  timezone: 'UTC',
  palette: 'clean', locale: '', pan_modifier: 'ctrl',
  main_board_id: null,
  open_main_board_on_home: false,
  remind_template: '',
  due_template: '',
  shortcuts: {},
  editor_mode: 'wysiwyg' as const,
}

function meOf(id: number, username: string) {
  return {account: {id, username, role: 'user'}, settings, device_id: 7, shortcut_defaults: {}, shortcut_actions: []}
}

function renderLogin(state?: unknown) {
  const router = createMemoryRouter(
    [
      {path: '/login', element: <LoginPage />},
      {path: '/', element: <p>主页内容</p>},
      {path: '/board/:id', element: <p>看板内容</p>},
    ],
    {initialEntries: [{pathname: '/login', state}]},
  )
  render(
    <SessionProvider>
      <RouterProvider router={router} />
    </SessionProvider>,
  )
}

function fillAndSubmit(username = ' alice ') {
  fireEvent.change(screen.getByLabelText('用户名'), {target: {value: username}})
  fireEvent.change(screen.getByLabelText('密码'), {target: {value: 'password-123'}})
  fireEvent.click(screen.getByRole('button', {name: '登录'}))
}

/** 按路径返回响应：/api/auth/login 返回 login，/api/user/me 返回 me，其余成功。 */
function mockServer(responses: Record<string, unknown>) {
  const fetchMock = vi.fn(async (url: string) => new Response(JSON.stringify(responses[url] ?? {code: ResultCode.Succeeded})))
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('LoginPage', () => {
  beforeEach(() => {
    FakeWebSocket.reset()
    vi.stubGlobal('WebSocket', FakeWebSocket)
    vi.mocked(reloadToHome).mockClear()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    localStorage.clear()
  })

  it('登录成功后保存为当前账号、进入主页并建立同步连接', async () => {
    const fetchMock = mockServer({'/api/auth/login': {code: ResultCode.Succeeded, data: {token: 'new-token', me: meOf(1, 'alice')}}})
    renderLogin()
    fillAndSubmit()

    expect(await screen.findByText('主页内容')).toBeInTheDocument()
    expect(currentToken()).toBe('new-token')
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toBe('/api/auth/login')
    expect(JSON.parse(init.body as string)).toMatchObject({username: 'alice', password: 'password-123'})

    const socket = FakeWebSocket.latest()
    expect(socket.url).toMatch(/\/api\/sync\/ws$/)
    socket.open()
    expect(socket.sent[0]).toEqual({token: 'new-token', last_revision: 0})
  })

  it('登录失败时显示服务端信息', async () => {
    mockServer({'/api/auth/login': {code: ResultCode.Unauthorized}})
    renderLogin()
    fillAndSubmit()

    expect(await screen.findByRole('alert')).toHaveTextContent('请重新登录')
    expect(readAccounts().accounts).toEqual([])
  })

  it('当前账号失效时提示重新登录并预填用户名，重新登录后原位置更新令牌', async () => {
    const saved: AccountsState = {
      accounts: [
        {userId: 2, username: 'bob', nickname: '', token: 'bob-token'},
        {userId: 1, username: 'alice', nickname: '甲', token: null},
      ],
      current: 1,
    }
    localStorage.setItem(ACCOUNTS_STORAGE_KEY, JSON.stringify(saved))
    mockServer({'/api/auth/login': {code: ResultCode.Succeeded, data: {token: 'renewed', me: meOf(1, 'alice')}}})
    renderLogin({from: '/board/5'})

    expect(screen.getByRole('status')).toHaveTextContent('账号「甲」的登录已失效')
    expect(screen.getByLabelText('用户名')).toHaveValue('alice')
    expect(screen.getByRole('button', {name: '切换到其他账号'})).toBeInTheDocument()

    fillAndSubmit('alice')
    expect(await screen.findByText('看板内容')).toBeInTheDocument()
    expect(readAccounts().accounts.map((item) => [item.userId, item.token])).toEqual([
      [2, 'bob-token'],
      [1, 'renewed'],
    ])
  })

  it('重新登录时换成了其他账号，回到主页而不是原地址', async () => {
    localStorage.setItem(ACCOUNTS_STORAGE_KEY, JSON.stringify({accounts: [{userId: 1, username: 'alice', nickname: '', token: null}], current: 1}))
    mockServer({'/api/auth/login': {code: ResultCode.Succeeded, data: {token: 'carol-token', me: meOf(3, 'carol')}}})
    renderLogin({from: '/board/5'})

    fillAndSubmit('carol')
    expect(await screen.findByText('主页内容')).toBeInTheDocument()
    expect(readAccounts()).toMatchObject({current: 3, accounts: [{userId: 1}, {userId: 3}]})
  })

  it('已登录时添加账号：加在列表最后，切换过去并刷新页面', async () => {
    localStorage.setItem(ACCOUNTS_STORAGE_KEY, JSON.stringify({accounts: [{userId: 1, username: 'alice', nickname: '', token: 'alice-token'}], current: 1}))
    mockServer({
      '/api/user/me': {code: ResultCode.Succeeded, data: meOf(1, 'alice')},
      '/api/auth/login': {code: ResultCode.Succeeded, data: {token: 'bob-token', me: meOf(2, 'bob')}},
    })
    renderLogin({addAccount: true, from: '/settings'})

    expect(await screen.findByText('登录新账号')).toBeInTheDocument()
    fillAndSubmit('bob')
    await vi.waitFor(() => expect(reloadToHome).toHaveBeenCalledOnce())
    expect(readAccounts()).toMatchObject({current: 2, accounts: [{userId: 1}, {userId: 2, token: 'bob-token'}]})
  })
})
