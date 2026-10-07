import {fireEvent, render, screen, waitFor, within} from '@testing-library/react'
import {afterEach, describe, expect, it, vi} from 'vitest'
import {ResultCode} from '../../api/client'
import type {Me} from '../../api/types'
import {SessionContext, type SessionValue} from '../../session/context'
import {SyncStore} from '../../sync/store'
import {ShortcutSection} from './ShortcutSection'

const shortcuts = {open_card: ['E', 'Enter'], edit_labels: ['L'], edit_title: ['T']}

const me: Me = {
  account: {id: 1, username: 'alice', role: 'user'},
  settings: {
    timezone: 'UTC',
    palette: 'clean', locale: '', pan_modifier: 'ctrl',
    main_board_id: null,
    open_main_board_on_home: false,
    remind_template: '',
    due_template: '',
    shortcuts,
    editor_mode: 'wysiwyg',
  },
  device_id: 7,
  shortcut_defaults: shortcuts,
  shortcut_actions: ['open_card', 'edit_labels', 'edit_title'],
}

function renderSection() {
  const sync = new SyncStore(me.settings)
  const value: SessionValue = {status: 'authenticated', me, sync, login: vi.fn(), logout: vi.fn()}
  render(
    <SessionContext.Provider value={value}>
      <ShortcutSection />
    </SessionContext.Provider>,
  )
  return sync
}

function startCapture(label: string) {
  const row = screen.getByRole('row', {name: new RegExp(label)})
  fireEvent.click(within(row).getByRole('button', {name: '添加'}))
}

describe('ShortcutSection', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('按键已被其他操作使用时提示冲突，不保存', () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    renderSection()

    startCapture('修改卡片标题')
    fireEvent.keyDown(window, {code: 'KeyL'})
    expect(screen.getByText(/已绑定到「编辑卡片标签」/)).toBeInTheDocument()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('添加不冲突的按键时保存该操作的全部按键，并立即显示', async () => {
    const saved = {...me.settings, shortcuts: {...shortcuts, edit_title: ['T', 'R']}}
    const fetchMock = vi.fn(async () => new Response(JSON.stringify({code: ResultCode.Succeeded, data: saved, revision: 3})))
    vi.stubGlobal('fetch', fetchMock)
    const sync = renderSection()

    startCapture('修改卡片标题')
    fireEvent.keyDown(window, {code: 'KeyR'})

    await waitFor(() => expect(sync.getState().settings).toEqual(saved))
    const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(JSON.parse(init.body as string)).toEqual({shortcuts: {edit_title: ['T', 'R']}})
    expect(screen.getByText('R')).toBeInTheDocument()
  })
})
