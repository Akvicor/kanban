import {fireEvent, render, screen, waitFor, within} from '@testing-library/react'
import {MemoryRouter} from 'react-router-dom'
import {afterEach, describe, expect, it, vi} from 'vitest'
import {ResultCode} from '../../api/client'
import type {Me, NotifyChannel} from '../../api/types'
import {SidebarContext} from '../../layout/sidebar'
import {SessionContext, type SessionValue} from '../../session/context'
import {SyncStore} from '../../sync/store'
import {ChannelsPage} from './ChannelsPage'

const settings = {timezone: 'UTC', palette: 'clean' as const, locale: '', pan_modifier: 'ctrl', main_board_id: null, open_main_board_on_home: false, remind_template: '', due_template: '', shortcuts: {}, editor_mode: 'wysiwyg' as const}
const me: Me = {account: {id: 1, username: 'alice', role: 'user'}, settings, device_id: 1, shortcut_defaults: {}, shortcut_actions: []}
const channel: NotifyChannel = {id: 5, name: '手机', api: 'https://msg.example.com', format: 'markdown', token_set: true, sign_set: true, created_at: '2026-01-01T00:00:00Z'}

function renderPage(responses: object[]) {
  const fetchMock = vi.fn(async () => new Response(JSON.stringify(responses.shift())))
  vi.stubGlobal('fetch', fetchMock)
  const sync = new SyncStore(settings)
  sync.applySnapshot({settings, devices: [], folders: [], boards: [], panels: [], labels: [], priority_levels: [], notify_channels: [channel]}, 1)
  const value: SessionValue = {status: 'authenticated', me, sync, login: vi.fn(), logout: vi.fn()}
  render(
    <MemoryRouter>
      <SessionContext.Provider value={value}>
        <SidebarContext.Provider value={{toggle: vi.fn(), close: vi.fn()}}>
          <ChannelsPage />
        </SidebarContext.Provider>
      </SessionContext.Provider>
    </MemoryRouter>,
  )
  return fetchMock
}

function requestBody(fetchMock: ReturnType<typeof vi.fn>, index: number): [string, unknown] {
  const [path, init] = fetchMock.mock.calls[index] as unknown as [string, RequestInit]
  return [path, JSON.parse(init.body as string)]
}

describe('ChannelsPage', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('编辑渠道时 Token 和 Sign 留空表示保留原值', async () => {
    const fetchMock = renderPage([{code: ResultCode.Succeeded, data: {...channel, name: '电脑'}, revision: 2}])

    fireEvent.click(screen.getByRole('button', {name: '编辑'}))
    const dialog = await screen.findByRole('dialog')
    fireEvent.change(within(dialog).getByLabelText('名字'), {target: {value: '电脑'}})
    fireEvent.click(within(dialog).getByRole('button', {name: '保存'}))

    await waitFor(() => expect(screen.getByText('电脑')).toBeInTheDocument())
    expect(requestBody(fetchMock, 0)).toEqual([
      '/api/notify_channel/update',
      {id: 5, name: '电脑', api: 'https://msg.example.com', token: '', sign: '', format: 'markdown'},
    ])
  })

  it('测试发送失败时在渠道下显示错误信息', async () => {
    renderPage([{code: ResultCode.Failed}])

    fireEvent.click(screen.getByRole('button', {name: '测试发送'}))

    expect(await screen.findByText('服务器错误')).toBeInTheDocument()
  })

  it('删除渠道需要确认，确认后从列表中移除', async () => {
    const fetchMock = renderPage([{code: ResultCode.Succeeded, revision: 2}])

    fireEvent.click(screen.getByRole('button', {name: '删除'}))
    fireEvent.click(within(await screen.findByRole('dialog')).getByRole('button', {name: '删除'}))

    await waitFor(() => expect(screen.queryByText('手机')).toBeNull())
    expect(requestBody(fetchMock, 0)).toEqual(['/api/notify_channel/delete', {id: 5}])
  })
})
