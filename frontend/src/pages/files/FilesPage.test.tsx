import {fireEvent, render, screen, waitFor, within} from '@testing-library/react'
import {MemoryRouter} from 'react-router-dom'
import {afterEach, describe, expect, it, vi} from 'vitest'
import {ResultCode} from '../../api/client'
import type {Me, UserFile} from '../../api/types'
import {SidebarContext} from '../../layout/sidebar'
import {SessionContext, type SessionValue} from '../../session/context'
import {SyncStore} from '../../sync/store'
import {FilesPage} from './FilesPage'

const settings = {timezone: 'UTC', palette: 'clean' as const, locale: '', pan_modifier: 'ctrl', main_board_id: null, open_main_board_on_home: false, remind_template: '', due_template: '', shortcuts: {}, editor_mode: 'wysiwyg' as const}
const me: Me = {account: {id: 1, username: 'alice', role: 'user'}, settings, device_id: 1, shortcut_defaults: {}, shortcut_actions: []}

function file(id: number, name: string, references: number): UserFile {
  return {
    id, name, size: 1024, mime_type: 'text/plain', image: false, width: 0, height: 0, reference_count: references,
    zero_at: references === 0 ? '2026-02-01T00:00:00Z' : null, first_referenced_at: '2026-01-01T00:00:00Z',
  }
}

describe('FilesPage', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('只有不再被引用的文件可以删除，删除后从列表中移除', async () => {
    const responses: object[] = [
      {code: ResultCode.Succeeded, data: {revision: 3, files: [file(1, '在用.txt', 2), file(2, '不用了.txt', 0)]}},
      {code: ResultCode.Succeeded, revision: 4},
    ]
    const fetchMock = vi.fn(async () => new Response(JSON.stringify(responses.shift())))
    vi.stubGlobal('fetch', fetchMock)
    const sync = new SyncStore(settings)
    const value: SessionValue = {status: 'authenticated', me, sync, login: vi.fn(), logout: vi.fn()}
    render(
      <MemoryRouter>
        <SessionContext.Provider value={value}>
          <SidebarContext.Provider value={{toggle: vi.fn(), close: vi.fn()}}>
            <FilesPage />
          </SidebarContext.Provider>
        </SessionContext.Provider>
      </MemoryRouter>,
    )

    const unused = (await screen.findByRole('button', {name: '不用了.txt'})).closest('.file-row') as HTMLElement
    const inUse = screen.getByRole('button', {name: '在用.txt'}).closest('.file-row') as HTMLElement
    expect(within(inUse).getByRole('button', {name: '删除'})).toBeDisabled()
    // 未被引用的文件排在前面。
    expect(document.querySelector('.file-row:not(.file-head) .file-name')?.textContent).toBe('不用了.txt')

    fireEvent.click(within(unused).getByRole('button', {name: '删除'}))
    fireEvent.click(within(await screen.findByRole('dialog')).getByRole('button', {name: '删除'}))
    await waitFor(() => expect(screen.queryByRole('button', {name: '不用了.txt'})).toBeNull())
    const [path, init] = fetchMock.mock.calls[1] as unknown as [string, RequestInit]
    expect(path).toBe('/api/user_file/delete')
    expect(JSON.parse(init.body as string)).toEqual({id: 2})
  })
})
