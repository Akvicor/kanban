import {describe, expect, it} from 'vitest'
import type {Device, List, Settings} from '../api/types'
import {EntityType, SyncStore} from './store'

const settings: Settings = {
  timezone: 'UTC',
  palette: 'clean', locale: '', pan_modifier: 'ctrl',
  main_board_id: null,
  open_main_board_on_home: false,
  remind_template: '',
  due_template: '',
  shortcuts: {},
  editor_mode: 'wysiwyg' as const,
}

function list(id: number, panelId: number, name: string): List {
  const op = {add_labels: [], remove_labels: [], start: '' as const, complete: '' as const}
  return {
    id,
    panel_id: panelId,
    name,
    color: '',
    position: id,
    show_age: true,
    sort_mode: 'manual',
    sort_dir: 'asc',
    head_add: 'head',
    tail_add: 'tail',
    remind_off: false,
    due_off: false,
    rules: {create: op, enter: op, exit: op},
    archived_at: null,
  }
}

function device(id: number, lastActive: string): Device {
  return {id, name: `设备 ${id}`, created_at: lastActive, last_active_at: lastActive}
}

describe('SyncStore', () => {
  it('快照整体替换数据，设备按最后活跃时间倒序', () => {
    const store = new SyncStore(settings)
    store.applySnapshot({settings: {...settings, palette: 'dark'}, devices: [device(1, '2026-01-01'), device(2, '2026-02-01')], folders: [], boards: [], panels: [], labels: [], priority_levels: [], notify_channels: []}, 5)
    expect(store.getState().settings.palette).toBe('dark')
    expect(store.getState().devices.map((d) => d.id)).toEqual([2, 1])
  })

  it('只用序号更大的数据覆盖本地数据', () => {
    const store = new SyncStore(settings)
    store.applySnapshot({settings, devices: [], folders: [], boards: [], panels: [], labels: [], priority_levels: [], notify_channels: []}, 3)

    // HTTP 响应先到（序号 5），之后推送的序号 4 和 5 都不能覆盖它。
    store.applyLocal(EntityType.Settings, 0, {...settings, palette: 'paper'}, 5)
    store.applyEvent({revision: 4, op: 'upsert', type: 'settings', id: 1, data: {...settings, palette: 'dark'}})
    store.applyEvent({revision: 5, op: 'upsert', type: 'settings', id: 1, data: {...settings, palette: 'paper'}})
    expect(store.getState().settings.palette).toBe('paper')

    store.applyEvent({revision: 6, op: 'upsert', type: 'settings', id: 1, data: {...settings, palette: 'system'}})
    expect(store.getState().settings.palette).toBe('system')
  })

  it('设备新增、更新和删除', () => {
    const store = new SyncStore(settings)
    store.applySnapshot({settings, devices: [device(1, '2026-01-01')], folders: [], boards: [], panels: [], labels: [], priority_levels: [], notify_channels: []}, 1)
    store.applyEvent({revision: 2, op: 'upsert', type: 'device', id: 2, data: device(2, '2026-03-01')})
    expect(store.getState().devices.map((d) => d.id)).toEqual([2, 1])

    store.applyLocal(EntityType.Device, 1, null, 3)
    // 删除之后，更早序号的推送不会让设备重新出现。
    store.applyEvent({revision: 2, op: 'upsert', type: 'device', id: 1, data: device(1, '2026-04-01')})
    expect(store.getState().devices.map((d) => d.id)).toEqual([2])
  })

  it('按面板加载的内容不覆盖推送带来的较新数据，并移除已不属于该面板的列表', () => {
    const store = new SyncStore(settings)
    store.applySnapshot({settings, devices: [], folders: [], boards: [], panels: [], labels: [], priority_levels: [], notify_channels: []}, 5)
    // 加载请求在途时：列表 1 在序号 8 被改名，列表 3 在序号 6 被推送（之后在序号 7 被移走或删除前的旧数据）。
    store.applyEvent({revision: 8, op: 'upsert', type: 'list', id: 1, data: list(1, 10, '新名称')})
    store.applyEvent({revision: 6, op: 'upsert', type: 'list', id: 3, data: list(3, 10, '旧列表')})
    store.applyPanelContent({panel_id: 10, revision: 7, lists: [list(1, 10, '旧名称'), list(2, 10, '列表 2')], cards: [], tasks: [], card_actions: [], card_links: [], attachments: [], notify_deliveries: []})

    const lists = store.getState().lists.sort((a, b) => a.id - b.id)
    expect(lists.map((l) => `${l.id}:${l.name}`)).toEqual(['1:新名称', '2:列表 2'])
    expect(store.getState().loadedPanels.has(10)).toBe(true)

    // 快照清空按面板加载的数据，面板需要重新加载。
    store.applySnapshot({settings, devices: [], folders: [], boards: [], panels: [], labels: [], priority_levels: [], notify_channels: []}, 9)
    expect(store.getState().lists).toEqual([])
    expect(store.getState().loadedPanels.has(10)).toBe(false)
  })

  it('没有产生变更的写入和未知类型的变更不改动数据', () => {
    const store = new SyncStore(settings)
    const before = store.getState()
    store.applyLocal(EntityType.Settings, 0, {...settings, palette: 'dark'}, 0)
    store.applyEvent({revision: 1, op: 'upsert', type: 'unknown_entity', id: 1, data: {}})
    expect(store.getState()).toBe(before)
  })

  it('加载用户文件列表时移除已删除的文件，保留更新的推送，收到快照后需要重新加载', () => {
    const store = new SyncStore(settings)
    const file = (id: number, name: string) => ({
      id, name, size: 1, mime_type: 'text/plain', image: false, width: 0, height: 0,
      reference_count: 0, zero_at: null, first_referenced_at: '2026-01-01',
    })
    store.applyEvent({revision: 3, op: 'upsert', type: EntityType.UserFile, id: 1, data: file(1, '旧文件')})
    store.applyEvent({revision: 9, op: 'upsert', type: EntityType.UserFile, id: 2, data: file(2, '推送的新名称')})
    // 列表读取于序号 5：文件 1 已不在其中（删除），文件 2 的推送更新，不被列表覆盖。
    store.applyUserFiles({revision: 5, files: [file(2, '列表中的名称'), file(3, '新文件')]})
    expect(store.getState().userFilesLoaded).toBe(true)
    expect(store.getState().userFiles.map((f) => [f.id, f.name])).toEqual([
      [2, '推送的新名称'],
      [3, '新文件'],
    ])
    store.applySnapshot({settings, devices: [], folders: [], boards: [], panels: [], labels: [], priority_levels: [], notify_channels: []}, 10)
    expect(store.getState().userFilesLoaded).toBe(false)
  })
})
