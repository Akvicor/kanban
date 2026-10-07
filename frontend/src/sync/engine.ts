import type {SyncEvent, SyncMessage} from '../api/types'
import {backoffDelay} from '../utils/backoff'
import type {SyncStore} from './store'

/** 服务端关闭连接时使用的关闭码，与后端 global/hub 一致。 */
export const CloseCode = {
  Revoked: 4001,
} as const

/** 发现序号缺口后，等待缺失变更到达的时间。不同写入的推送可能先后交错，短暂等待即可补上。 */
export const GAP_WAIT_MS = 2000

export interface SyncEngineOptions {
  url: string
  token: string
  store: SyncStore
  /** 设备令牌已吊销时调用。之后不再重连。 */
  onRevoked: () => void
}

/** 同步连接地址，与页面同源。 */
export function syncUrl(location: Location): string {
  const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${location.host}/api/sync/ws`
}

/**
 * 同步连接：连接后发送令牌和已应用的序号，按序号把快照和变更应用到 SyncStore。
 *
 * - 变更必须按序号逐条应用；遇到缺口时先暂存后面的变更，等待 GAP_WAIT_MS，仍未补上就重连补齐。
 * - 连接断开后带着已应用的序号重连；页面回到前台或网络恢复时立即重连。
 * - 已应用的序号只保存在内存中，刷新页面后从 0 开始，由服务端发送快照。
 */
export class SyncEngine {
  private readonly options: SyncEngineOptions
  private socket: WebSocket | null = null
  private applied = 0
  private readonly pending = new Map<number, SyncEvent>()
  private gapTimer: ReturnType<typeof setTimeout> | null = null
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private attempts = 0
  private stopped = false

  constructor(options: SyncEngineOptions) {
    this.options = options
  }

  start(): void {
    window.addEventListener('online', this.reconnectNow)
    document.addEventListener('visibilitychange', this.onVisibilityChange)
    this.connect()
  }

  stop(): void {
    this.stopped = true
    window.removeEventListener('online', this.reconnectNow)
    document.removeEventListener('visibilitychange', this.onVisibilityChange)
    this.clearTimers()
    this.socket?.close()
    this.socket = null
  }

  private connect(): void {
    if (this.stopped || this.socket) {
      return
    }
    const socket = new WebSocket(this.options.url)
    this.socket = socket
    socket.onopen = () => {
      socket.send(JSON.stringify({token: this.options.token, last_revision: this.applied}))
    }
    socket.onmessage = (message: MessageEvent<string>) => {
      if (this.socket === socket) {
        this.handle(JSON.parse(message.data) as SyncMessage)
      }
    }
    socket.onclose = (event: CloseEvent) => {
      if (this.socket !== socket) {
        return
      }
      this.socket = null
      this.clearGapTimer()
      if (event.code === CloseCode.Revoked) {
        this.stop()
        this.options.onRevoked()
        return
      }
      this.scheduleReconnect()
    }
  }

  private handle(message: SyncMessage): void {
    switch (message.type) {
      case 'snapshot':
        this.options.store.applySnapshot(message.snapshot, message.revision)
        this.applied = message.revision
        this.drain()
        return
      case 'events':
        for (const event of message.events) {
          if (event.revision > this.applied) {
            this.pending.set(event.revision, event)
          }
        }
        this.drain()
        return
      case 'caught_up':
        this.attempts = 0
        if (this.applied < message.revision) {
          // 服务端已发送到 message.revision，本地仍有缺口，说明消息丢失，重连补齐。
          this.restart()
        }
        return
    }
  }

  /** 按序号应用暂存的变更，直到遇到缺口。 */
  private drain(): void {
    for (const revision of this.pending.keys()) {
      if (revision <= this.applied) {
        this.pending.delete(revision)
      }
    }
    let next = this.pending.get(this.applied + 1)
    while (next) {
      this.options.store.applyEvent(next)
      this.pending.delete(next.revision)
      this.applied = next.revision
      next = this.pending.get(this.applied + 1)
    }
    if (this.pending.size === 0) {
      this.clearGapTimer()
    } else if (!this.gapTimer) {
      this.gapTimer = setTimeout(() => {
        this.gapTimer = null
        if (this.pending.size > 0) {
          this.restart()
        }
      }, GAP_WAIT_MS)
    }
  }

  /** 关闭当前连接并立即重连，从已应用的序号补齐。 */
  private restart(): void {
    const socket = this.socket
    this.socket = null
    this.pending.clear()
    this.clearGapTimer()
    socket?.close()
    this.connect()
  }

  private scheduleReconnect(): void {
    if (this.stopped || this.reconnectTimer) {
      return
    }
    const delay = backoffDelay(this.attempts)
    this.attempts += 1
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.connect()
    }, delay)
  }

  private reconnectNow = (): void => {
    if (this.stopped || this.socket) {
      return
    }
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    this.connect()
  }

  private onVisibilityChange = (): void => {
    if (document.visibilityState === 'visible') {
      this.reconnectNow()
    }
  }

  private clearGapTimer(): void {
    if (this.gapTimer) {
      clearTimeout(this.gapTimer)
      this.gapTimer = null
    }
  }

  private clearTimers(): void {
    this.clearGapTimer()
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
  }
}
