import {fireEvent, render, screen, waitFor, within} from '@testing-library/react'
import {MemoryRouter} from 'react-router-dom'
import {afterEach, describe, expect, it, vi} from 'vitest'
import {ResultCode} from '../../api/client'
import type {Attachment, Card, List, Me, NotifyChannel, NotifyDelivery, Panel, Task} from '../../api/types'
import {SessionContext, type SessionValue} from '../../session/context'
import {EntityType, SyncStore} from '../../sync/store'
import {CardDetail} from './CardDetail'

// 真实的 Markdown 编辑器依赖浏览器排版，测试中用文本框代替，只验证保存和模式切换的逻辑。
vi.mock('../../markdown/MarkdownEditor', () => ({
  default: ({initialValue, onChange, onModeChange}: {initialValue: string; onChange: (v: string) => void; onModeChange: (m: string) => void}) => (
    <>
      <textarea aria-label="描述" defaultValue={initialValue} onChange={(e) => onChange(e.target.value)} />
      <button type="button" onClick={() => onModeChange('markup')}>
        切换到源码
      </button>
    </>
  ),
}))

const settings = {timezone: 'UTC', palette: 'clean' as const, locale: '', pan_modifier: 'ctrl', main_board_id: null, open_main_board_on_home: false, remind_template: '', due_template: '', shortcuts: {}, editor_mode: 'wysiwyg' as const}
const me: Me = {account: {id: 1, username: 'alice', role: 'user'}, settings, device_id: 1, shortcut_defaults: {}, shortcut_actions: []}
const panel: Panel = {id: 10, board_id: 1, name: '默认', position: 1, archived_at: null}
const emptyOp = {add_labels: [], remove_labels: [], start: '' as const, complete: '' as const}

function list(id: number, fields: Partial<List> = {}): List {
  return {
    id,
    panel_id: 10,
    name: `列表 ${id}`,
    color: '',
    position: id,
    show_age: true,
    sort_mode: 'manual',
    sort_dir: 'asc',
    head_add: 'head',
    tail_add: 'tail',
    remind_off: false,
    due_off: false,
    rules: {create: emptyOp, enter: emptyOp, exit: emptyOp},
    archived_at: null,
    ...fields,
  }
}

const baseCard: Card = {
  id: 7,
  panel_id: 10,
  list_id: 5,
  position: 100,
  title: '整理预算',
  description: '**重点**',
  label_ids: [],
  priority_level_id: null,
  remind_at: null,
  due_at: null,
  remind_notify: false,
  due_notify: false,
  created_at: '2026-01-01T00:00:00Z',
  started_at: null,
  completed_at: null,
  timer_seconds: 0,
  timer_started_at: null,
  archived_at: null,
  cover_attachment_id: null,
  notify_channel_ids: [],
}

function task(id: number, fields: Partial<Task> = {}): Task {
  return {id, card_id: 7, parent_id: null, title: `任务 ${id}`, done: false, position: id * 100, ...fields}
}

/** 用面板内容（同步序号 3）初始化本地数据并渲染卡片详情。 */
function renderDetail(
  options: {card?: Partial<Card>; lists?: List[]; tasks?: Task[]; attachments?: Attachment[]; channels?: NotifyChannel[]; deliveries?: NotifyDelivery[]} = {},
) {
  const sync = new SyncStore(settings)
  sync.applySnapshot(
    {settings, devices: [], folders: [], boards: [], panels: [panel], labels: [], priority_levels: [], notify_channels: options.channels ?? []},
    1,
  )
  const card = {...baseCard, ...options.card}
  sync.applyPanelContent({
    panel_id: 10,
    revision: 3,
    lists: options.lists ?? [list(5)],
    cards: card.archived_at === null ? [card] : [],
    tasks: options.tasks ?? [],
    card_actions: [{id: 1, card_id: 7, type: 'create', data: {list_id: 5}, created_at: '2026-01-01T00:00:00Z'}],
    card_links: [],
    attachments: options.attachments ?? [],
    notify_deliveries: options.deliveries ?? [],
  })
  if (card.archived_at !== null) {
    sync.applyBundle({revision: 3, cards: [card], tasks: options.tasks ?? [], card_actions: [], card_links: [], attachments: [], notify_deliveries: []})
  }
  const value: SessionValue = {status: 'authenticated', me, sync, login: vi.fn(), logout: vi.fn()}
  const onNotice = vi.fn()
  const onClose = vi.fn()
  render(
    <MemoryRouter>
      <SessionContext.Provider value={value}>
        <CardDetail panel={panel} cardId={7} focus={null} onClose={onClose} onNotice={onNotice} />
      </SessionContext.Provider>
    </MemoryRouter>,
  )
  return {sync, onNotice, onClose}
}

/** 按顺序返回给定的接口响应，并记录请求。 */
function mockFetch(...responses: object[]) {
  const fetchMock = vi.fn(async () => new Response(JSON.stringify(responses.shift() ?? {code: ResultCode.Succeeded})))
  vi.stubGlobal('fetch', fetchMock)
  return {
    body: (index: number) => JSON.parse((fetchMock.mock.calls[index] as unknown as [string, RequestInit])[1].body as string),
    path: (index: number) => (fetchMock.mock.calls[index] as unknown as [string])[0],
    calls: () => fetchMock.mock.calls.length,
  }
}

describe('CardDetail', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('选择通知渠道，并在渠道下显示提醒和截止的发送状态', async () => {
    const channels: NotifyChannel[] = [
      {id: 1, name: '手机', api: 'https://a.example.com', format: 'markdown', token_set: true, sign_set: true, created_at: '2026-01-01T00:00:00Z'},
      {id: 2, name: '电脑', api: 'https://b.example.com', format: 'text', token_set: true, sign_set: true, created_at: '2026-01-01T00:00:00Z'},
    ]
    const remindAt = '2026-01-02T08:00:00Z'
    const api = mockFetch({code: ResultCode.Succeeded, data: {...baseCard, remind_at: remindAt, remind_notify: true, notify_channel_ids: [1, 2]}, revision: 4})
    renderDetail({
      card: {remind_at: remindAt, remind_notify: true, due_at: '2999-01-01T00:00:00Z', due_notify: true, notify_channel_ids: [1]},
      channels,
      deliveries: [
        {id: 3, card_id: 7, channel_id: 1, kind: 'remind', target_at: remindAt, sent_at: '2026-01-02T08:00:10Z', attempts: 1, next_attempt_at: remindAt, last_error: ''},
      ],
    })

    expect(screen.getByText('提醒：已发送 2026-01-02 08:00')).toBeInTheDocument()
    expect(screen.getByText('截止：未到时间')).toBeInTheDocument()

    fireEvent.keyDown(screen.getByRole('button', {name: '选择通知渠道'}), {key: 'Enter'})
    fireEvent.click(await screen.findByRole('menuitemcheckbox', {name: '电脑'}))

    await waitFor(() => expect(screen.getByText('电脑')).toBeInTheDocument())
    expect(api.path(0)).toBe('/api/card/notify_channel')
    expect(api.body(0)).toEqual({id: 7, channel_id: 2, on: true})
  })

  it('修改标题时带上开始编辑时的同步序号，保存后立即显示新标题', async () => {
    const api = mockFetch({code: ResultCode.Succeeded, data: {...baseCard, title: '整理 10 月预算'}, revision: 4})
    renderDetail()

    fireEvent.click(screen.getByRole('heading', {name: '整理预算'}))
    const input = screen.getByLabelText('卡片标题')
    fireEvent.change(input, {target: {value: '整理 10 月预算'}})
    fireEvent.keyDown(input, {key: 'Enter'})

    await screen.findByRole('heading', {name: '整理 10 月预算'})
    expect(api.path(0)).toBe('/api/card/title')
    expect(api.body(0)).toEqual({id: 7, text: '整理 10 月预算', base_revision: 3})
  })

  it('标题冲突时保留草稿并显示其他设备的标题，选择覆盖时以最新序号重新提交', async () => {
    const api = mockFetch(
      {code: ResultCode.Conflict, msg: '标题已在其他设备上修改', data: {...baseCard, title: '设备二的标题'}, revision: 9},
      {code: ResultCode.Succeeded, data: {...baseCard, title: '我的草稿'}, revision: 10},
    )
    const {onNotice, sync} = renderDetail()
    fireEvent.click(screen.getByRole('heading', {name: '整理预算'}))
    const input = screen.getByLabelText('卡片标题')
    fireEvent.change(input, {target: {value: '我的草稿'}})
    fireEvent.blur(input)

    expect(await screen.findByRole('alert')).toHaveTextContent('标题已在其他设备上改为「设备二的标题」')
    expect(screen.getByLabelText('卡片标题')).toHaveValue('我的草稿')
    expect(sync.versionOf(EntityType.Card, 7)).toBe(9)
    // 冲突期间失去焦点不会再次提交。
    fireEvent.blur(screen.getByLabelText('卡片标题'))
    expect(api.calls()).toBe(1)

    fireEvent.click(screen.getByRole('button', {name: '用我的标题覆盖'}))
    await screen.findByRole('heading', {name: '我的草稿'})
    expect(api.body(1)).toEqual({id: 7, text: '我的草稿', base_revision: 9})
    expect(onNotice).not.toHaveBeenCalled()
  })

  it('描述冲突时保留草稿，可以查看最新内容、放弃或仍然保存', async () => {
    const api = mockFetch(
      {code: ResultCode.Conflict, msg: '描述已在其他设备上修改', data: {...baseCard, description: '设备二的描述'}, revision: 9},
      {code: ResultCode.Succeeded, data: {...baseCard, description: '我的描述'}, revision: 10},
    )
    renderDetail()
    fireEvent.click(screen.getByRole('button', {name: '编辑'}))
    fireEvent.change(await screen.findByLabelText('描述'), {target: {value: '我的描述'}})
    fireEvent.click(screen.getByRole('button', {name: '保存'}))

    expect(await screen.findByRole('alert')).toHaveTextContent('描述已在其他设备上修改')
    expect(screen.getByLabelText('描述')).toHaveValue('我的描述')
    fireEvent.click(screen.getByRole('button', {name: '查看最新内容'}))
    expect(await screen.findByText('设备二的描述')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', {name: '仍然保存'}))
    await waitFor(() => expect(screen.queryByLabelText('描述')).toBeNull())
    expect(await screen.findByText('我的描述')).toBeInTheDocument()
    expect(api.body(1)).toEqual({id: 7, text: '我的描述', base_revision: 9})
  })

  it('描述按 Markdown 渲染，包括扩展语法', async () => {
    mockFetch()
    renderDetail({card: {description: '**重点** ==高亮== ++下划线++\n\n- [x] 已完成的检查项'}})
    expect((await screen.findByText('重点')).tagName).toBe('STRONG')
    expect(screen.getByText('高亮').tagName).toBe('MARK')
    expect(screen.getByText('下划线').tagName).toBe('INS')
    const description = screen.getByRole('heading', {name: /描述/}).closest('section')!
    expect(within(description as HTMLElement).getByRole('checkbox')).toBeChecked()
  })

  it('编辑描述保存失败时保留草稿，切换编辑模式时保存到个人设置', async () => {
    const api = mockFetch(
      {code: ResultCode.Succeeded, data: {...settings, editor_mode: 'markup'}, revision: 4},
      {code: ResultCode.Failed, msg: '无法连接服务器'},
    )
    const {onNotice, sync} = renderDetail()

    fireEvent.click(screen.getByRole('button', {name: '编辑'}))
    fireEvent.click(await screen.findByRole('button', {name: '切换到源码'}))
    await waitFor(() => expect(sync.getState().settings.editor_mode).toBe('markup'))
    expect(api.path(0)).toBe('/api/user/settings/update')
    expect(api.body(0)).toEqual({editor_mode: 'markup'})

    fireEvent.change(screen.getByLabelText('描述'), {target: {value: '新的描述'}})
    fireEvent.click(screen.getByRole('button', {name: '保存'}))
    await waitFor(() => expect(onNotice).toHaveBeenCalledWith('服务器错误'))
    expect(api.body(1)).toEqual({id: 7, text: '新的描述', base_revision: 3})
    expect(screen.getByLabelText('描述')).toHaveValue('新的描述')
  })

  it('在新建任务的输入框粘贴多行时一次创建多个任务', async () => {
    const created = [task(1, {title: '买菜'}), task(2, {title: '做饭'})]
    const api = mockFetch({code: ResultCode.Succeeded, data: created, revision: 4})
    renderDetail()

    const input = screen.getByLabelText('添加任务，粘贴多行可一次添加多个')
    fireEvent.paste(input, {clipboardData: {getData: () => '买菜\n\n  做饭  \n'}})

    await screen.findByText('做饭')
    expect(api.body(0)).toEqual({card_id: 7, parent_id: null, titles: ['买菜', '做饭']})
    expect(screen.getByText('0/2')).toBeInTheDocument()
  })

  it('勾选父任务后它和下级一起显示为完成，删除任务后从本地移除', async () => {
    const api = mockFetch(
      {code: ResultCode.Succeeded, data: [task(1, {done: true}), task(2, {parent_id: 1, done: true})], revision: 4},
      {code: ResultCode.Succeeded, revision: 5},
    )
    const {sync} = renderDetail({tasks: [task(1), task(2, {parent_id: 1})]})

    fireEvent.click(screen.getAllByRole('checkbox', {name: '完成'})[0])
    await screen.findByText('2/2')
    expect(api.body(0)).toEqual({id: 1, done: true})

    // 任务的操作按钮在悬停或聚焦时才显示。
    fireEvent.click(screen.getAllByRole('button', {name: '删除任务', hidden: true})[1])
    await waitFor(() => expect(sync.getState().tasks.map((t) => t.id)).toEqual([1]))
    expect(api.path(1)).toBe('/api/task/delete')
    expect(sync.versionOf(EntityType.Task, 2)).toBe(5)
  })

  it('卡片在卡片归档中时只能查看', () => {
    const api = mockFetch()
    renderDetail({card: {archived_at: '2026-02-01T00:00:00Z', list_id: null, position: null}, tasks: [task(1)]})

    expect(screen.getByText(/卡片在卡片归档中，只能查看/)).toBeInTheDocument()
    expect(screen.queryByRole('button', {name: '编辑'})).toBeNull()
    expect(screen.queryByRole('button', {name: '归档卡片'})).toBeNull()
    expect(screen.queryByLabelText('添加任务，粘贴多行可一次添加多个')).toBeNull()
    expect(screen.getByRole('checkbox', {name: '完成'})).toBeDisabled()
    fireEvent.click(screen.getByRole('heading', {name: '整理预算'}))
    expect(screen.queryByLabelText('卡片标题')).toBeNull()
    expect(api.calls()).toBe(0)
  })

  it('列表在列表归档中时其中的卡片只能查看，操作记录显示列表名', () => {
    renderDetail({lists: [list(5, {name: '待办', archived_at: '2026-02-01T00:00:00Z'})]})
    expect(screen.getByText(/列表在列表归档中，只能查看/)).toBeInTheDocument()
    const log = screen.getByRole('heading', {name: '操作记录'}).closest('section')!
    expect(within(log as HTMLElement).getByText('创建于「待办」')).toBeInTheDocument()
  })
})

function fileAttachment(id: number, name: string, fields: Partial<Attachment> = {}): Attachment {
  return {
    id, card_id: 7, type: 'file', name, url: '', favicon: '', created_at: '2026-01-01T00:00:00Z',
    file: {user_file_id: id, size: 2048, mime_type: 'text/plain', image: false, width: 0, height: 0},
    ...fields,
  }
}

describe('CardDetail 附件', () => {
  afterEach(() => vi.unstubAllGlobals())

  const section = () => screen.getByRole('heading', {name: /^附件/}).closest('section') as HTMLElement

  it('附件按添加时间从新到旧显示；卡片只读时不能上传或添加链接', () => {
    mockFetch()
    renderDetail({
      card: {archived_at: '2026-02-01T00:00:00Z', list_id: null, position: null},
      attachments: [fileAttachment(1, '较早.txt'), fileAttachment(2, '较新.txt')],
    })
    const names = within(section()).getAllByRole('button', {name: /\.txt$/}).filter((el) => el.className === 'attachment-name')
    expect(names.map((el) => el.textContent)).toEqual(['较新.txt', '较早.txt'])
    expect(within(section()).getAllByText(/^2 KB · /)).toHaveLength(2)
    expect(within(section()).queryByRole('button', {name: '上传文件'})).toBeNull()
    expect(within(section()).queryByRole('button', {name: '添加链接'})).toBeNull()
  })

  it('选择文件后先查询 sha256，系统已有时不上传内容，直接新建附件', async () => {
    const created = fileAttachment(5, '报告.txt')
    const api = mockFetch(
      {code: ResultCode.Succeeded, data: {exists: true, session_id: 0, received: 0, done: false}},
      {code: ResultCode.Succeeded, data: created, revision: 4},
    )
    renderDetail()
    fireEvent.change(within(section()).getByLabelText('选择要上传的文件'), {target: {files: [new File(['Hello'], '报告.txt')]}})

    // 上传进度中也会显示文件名，因此等到附件列表中出现这个附件。
    await waitFor(() => expect(section().querySelector('.attachment-name')?.textContent).toBe('报告.txt'))
    expect(api.path(0)).toBe('/api/upload/prepare')
    expect(api.path(1)).toBe('/api/attachment/create_file')
    expect(api.body(1)).toEqual({card_id: 7, sha256: '185f8db32271fe25f561a6fc938b2e264306ec304eda518007d1764826381969', name: '报告.txt'})
  })

  it('在卡片详情中粘贴网址时新建链接附件', async () => {
    const link: Attachment = {...fileAttachment(6, 'https://example.com/a'), type: 'link', file: null, url: 'https://example.com/a'}
    const api = mockFetch({code: ResultCode.Succeeded, data: link, revision: 4})
    renderDetail()
    const event = new Event('paste', {bubbles: true, cancelable: true})
    Object.assign(event, {clipboardData: {files: [], getData: () => 'https://example.com/a'}})
    window.dispatchEvent(event)

    expect(await within(section()).findByText('https://example.com/a')).toBeInTheDocument()
    expect(api.path(0)).toBe('/api/attachment/create_link')
    expect(api.body(0)).toEqual({card_id: 7, url: 'https://example.com/a', name: ''})
  })
})
