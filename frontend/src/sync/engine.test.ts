import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'
import type {Settings, SyncEvent} from '../api/types'
import {FakeWebSocket} from '../test/fakeWebSocket'
import {CloseCode, GAP_WAIT_MS, SyncEngine} from './engine'
import {SyncStore} from './store'

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

function paletteEvent(revision: number, palette: Settings['palette']): SyncEvent {
  return {revision, op: 'upsert', type: 'settings', id: 1, data: {...settings, palette}}
}

describe('SyncEngine', () => {
  let store: SyncStore
  let onRevoked: ReturnType<typeof vi.fn<() => void>>
  let engine: SyncEngine

  beforeEach(() => {
    vi.useFakeTimers()
    vi.stubGlobal('WebSocket', FakeWebSocket)
    FakeWebSocket.reset()
    store = new SyncStore(settings)
    onRevoked = vi.fn<() => void>()
    engine = new SyncEngine({url: 'ws://test/api/sync/ws', token: 'token', store, onRevoked})
    engine.start()
  })

  afterEach(() => {
    engine.stop()
    vi.unstubAllGlobals()
    vi.useRealTimers()
  })

  /** 打开连接并收到序号为 revision 的快照。 */
  function connectWithSnapshot(revision: number) {
    const socket = FakeWebSocket.latest()
    socket.open()
    socket.receive({type: 'snapshot', revision, snapshot: {settings, devices: [], folders: [], boards: [], panels: [], labels: [], priority_levels: [], notify_channels: []}})
    socket.receive({type: 'caught_up', revision})
    return socket
  }

  it('连接后发送令牌和已应用的序号，按序号应用变更', () => {
    const socket = connectWithSnapshot(3)
    expect(socket.sent[0]).toEqual({token: 'token', last_revision: 0})

    socket.receive({type: 'events', events: [paletteEvent(4, 'dark')]})
    expect(store.getState().settings.palette).toBe('dark')
  })

  it('乱序到达的变更先暂存，补上缺口后一起应用', () => {
    const socket = connectWithSnapshot(3)
    socket.receive({type: 'events', events: [paletteEvent(5, 'paper')]})
    expect(store.getState().settings.palette).toBe('clean')

    socket.receive({type: 'events', events: [paletteEvent(4, 'dark')]})
    expect(store.getState().settings.palette).toBe('paper')
    vi.advanceTimersByTime(GAP_WAIT_MS)
    expect(FakeWebSocket.instances).toHaveLength(1)
  })

  it('缺口一直补不上时重连，从已应用的序号补齐', () => {
    const socket = connectWithSnapshot(3)
    socket.receive({type: 'events', events: [paletteEvent(4, 'dark'), paletteEvent(6, 'paper')]})
    vi.advanceTimersByTime(GAP_WAIT_MS)

    expect(socket.closed).toBe(true)
    const next = FakeWebSocket.latest()
    expect(next).not.toBe(socket)
    next.open()
    expect(next.sent[0]).toEqual({token: 'token', last_revision: 4})
  })

  it('连接断开后带着已应用的序号重连', () => {
    const socket = connectWithSnapshot(7)
    socket.serverClose(1006)
    expect(FakeWebSocket.instances).toHaveLength(1)

    vi.advanceTimersByTime(1000)
    const next = FakeWebSocket.latest()
    next.open()
    expect(next.sent[0]).toEqual({token: 'token', last_revision: 7})
  })

  it('令牌被吊销时通知退出登录，不再重连', () => {
    const socket = connectWithSnapshot(1)
    socket.serverClose(CloseCode.Revoked)
    expect(onRevoked).toHaveBeenCalledOnce()

    vi.advanceTimersByTime(60000)
    expect(FakeWebSocket.instances).toHaveLength(1)
  })
})
