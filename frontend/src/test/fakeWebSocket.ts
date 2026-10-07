/** 测试用的 WebSocket 替身：记录发送的消息，由测试主动触发打开、收到消息和关闭。 */
export class FakeWebSocket {
  static instances: FakeWebSocket[] = []

  readonly url: string
  readonly sent: unknown[] = []
  closed = false
  onopen: (() => void) | null = null
  onmessage: ((event: {data: string}) => void) | null = null
  onclose: ((event: {code: number}) => void) | null = null

  constructor(url: string) {
    this.url = url
    FakeWebSocket.instances.push(this)
  }

  static latest(): FakeWebSocket {
    return FakeWebSocket.instances[FakeWebSocket.instances.length - 1]
  }

  static reset(): void {
    FakeWebSocket.instances = []
  }

  send(data: string): void {
    this.sent.push(JSON.parse(data))
  }

  close(): void {
    this.closed = true
  }

  open(): void {
    this.onopen?.()
  }

  receive(message: unknown): void {
    this.onmessage?.({data: JSON.stringify(message)})
  }

  /** 模拟服务端关闭连接。 */
  serverClose(code: number): void {
    this.closed = true
    this.onclose?.({code})
  }
}
