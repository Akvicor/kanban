import {afterEach, describe, expect, it, vi} from 'vitest'
import {registerServiceWorker} from './registerServiceWorker'

/** 提供 Service Worker 接口和安全上下文，返回注册函数的替身。 */
function stubBrowser(secure: boolean) {
  const register = vi.fn(async () => ({}) as ServiceWorkerRegistration)
  vi.stubGlobal('isSecureContext', secure)
  Object.defineProperty(navigator, 'serviceWorker', {value: {register}, configurable: true})
  return register
}

describe('registerServiceWorker', () => {
  afterEach(() => {
    vi.unstubAllEnvs()
    vi.unstubAllGlobals()
    Reflect.deleteProperty(navigator, 'serviceWorker')
  })

  it('生产构建且在安全上下文中时，页面加载后注册 /sw.js', () => {
    vi.stubEnv('PROD', true)
    const register = stubBrowser(true)
    registerServiceWorker()
    window.dispatchEvent(new Event('load'))
    expect(register).toHaveBeenCalledWith('/sw.js')
  })

  it('开发环境或非安全上下文中不注册', () => {
    stubBrowser(true)
    const addListener = vi.spyOn(window, 'addEventListener')
    vi.stubEnv('PROD', false)
    registerServiceWorker()

    vi.stubEnv('PROD', true)
    vi.stubGlobal('isSecureContext', false)
    registerServiceWorker()
    expect(addListener).not.toHaveBeenCalledWith('load', expect.anything())
    addListener.mockRestore()
  })
})
