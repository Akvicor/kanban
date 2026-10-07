import {describe, expect, it} from 'vitest'
import {deviceName} from './deviceName'

describe('deviceName', () => {
  it('按浏览器和系统生成设备名', () => {
    expect(
      deviceName('Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0 Safari/537.36'),
    ).toBe('Chrome · macOS')
    expect(
      deviceName('Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1'),
    ).toBe('Safari · iOS')
    expect(deviceName('Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/140.0 Safari/537.36 Edg/140.0')).toBe('Edge · Windows')
    expect(deviceName('curl/8.0')).toBe('未知设备')
  })

  it('桌面客户端按 KanbanApp 标记识别', () => {
    expect(
      deviceName('Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) kanban-app/1.2.0 Chrome/146.0 Electron/44.5.1 Safari/537.36 KanbanApp/1.2.0'),
    ).toBe('桌面客户端 · Windows')
    expect(deviceName('Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/146.0 Electron/44.5.1 Safari/537.36 KanbanApp/0.0.0-g9ad87dd')).toBe('桌面客户端 · Linux')
  })
})
