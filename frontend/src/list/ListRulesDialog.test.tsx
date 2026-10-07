import {fireEvent, render, screen, waitFor, within} from '@testing-library/react'
import {afterEach, describe, expect, it, vi} from 'vitest'
import {ResultCode} from '../api/client'
import type {List, Me} from '../api/types'
import {SessionContext, type SessionValue} from '../session/context'
import {SyncStore} from '../sync/store'
import {ListRulesDialog} from './ListRulesDialog'

const emptyOp = {add_labels: [], remove_labels: [], start: '' as const, complete: '' as const}

const list: List = {
  id: 5,
  panel_id: 10,
  name: '完成',
  color: '',
  position: 1,
  show_age: true,
  sort_mode: 'manual',
  sort_dir: 'asc',
  head_add: 'head',
  tail_add: 'tail',
  remind_off: false,
  due_off: false,
  rules: {create: emptyOp, enter: emptyOp, exit: emptyOp},
  archived_at: null,
}

const settings = {timezone: 'UTC', palette: 'clean' as const, locale: '', pan_modifier: 'ctrl', main_board_id: null, open_main_board_on_home: false, remind_template: '', due_template: '', shortcuts: {}, editor_mode: 'wysiwyg' as const}
const me: Me = {account: {id: 1, username: 'alice', role: 'user'}, settings, device_id: 1, shortcut_defaults: {}, shortcut_actions: []}

function renderDialog() {
  const sync = new SyncStore(settings)
  sync.applySnapshot(
    {
      settings,
      devices: [],
      folders: [],
      boards: [],
      panels: [],
      labels: [
        {id: 1, panel_id: 10, name: 'running', color: '#3E63DD', position: 1},
        {id: 2, panel_id: 10, name: 'finished', color: '#30A46C', position: 2},
        {id: 3, panel_id: 99, name: '其他面板', color: '#000000', position: 1},
      ],
      priority_levels: [],
      notify_channels: [],
    },
    1,
  )
  const value: SessionValue = {status: 'authenticated', me, sync, login: vi.fn(), logout: vi.fn()}
  const onClose = vi.fn()
  render(
    <SessionContext.Provider value={value}>
      <ListRulesDialog list={list} onClose={onClose} />
    </SessionContext.Provider>,
  )
  return {sync, onClose}
}

describe('ListRulesDialog', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('只列出本面板的标签，保存勾选的标签和时间动作', async () => {
    const fetchMock = vi.fn(async () => new Response(JSON.stringify({code: ResultCode.Succeeded, data: list, revision: 2})))
    vi.stubGlobal('fetch', fetchMock)
    const {onClose} = renderDialog()
    expect(screen.queryByText('其他面板')).toBeNull()

    const enter = screen.getByRole('heading', {name: '移入'}).closest('section')!
    const [addField, removeField] = within(enter).getAllByText(/标签$/).map((el) => el.closest('.field')!)
    fireEvent.click(within(addField as HTMLElement).getByRole('button', {name: 'finished'}))
    fireEvent.click(within(removeField as HTMLElement).getByRole('button', {name: 'running'}))
    const setTime = within(enter).getByRole('button', {name: '完成时间'})
    fireEvent.keyDown(setTime, {key: 'Enter'})
    fireEvent.click(await screen.findByRole('menuitemradio', {name: '设置（为空时写入）'}))
    const exit = screen.getByRole('heading', {name: '移出'}).closest('section')!
    const clearTime = within(exit).getByRole('button', {name: '完成时间'})
    fireEvent.keyDown(clearTime, {key: 'Enter'})
    fireEvent.click(await screen.findByRole('menuitemradio', {name: '清空'}))
    fireEvent.click(screen.getByRole('button', {name: '保存'}))

    await waitFor(() => expect(onClose).toHaveBeenCalled())
    const [, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    expect(JSON.parse(init.body as string)).toEqual({
      id: 5,
      rules: {
        create: emptyOp,
        enter: {add_labels: [2], remove_labels: [1], start: '', complete: 'set'},
        exit: {...emptyOp, complete: 'clear'},
      },
    })
  })
})
