import '@testing-library/jest-dom/vitest'
import {cleanup} from '@testing-library/react'
import {afterEach} from 'vitest'

// 测试断言按简体中文界面文案编写，固定测试语言，避免 jsdom 浏览器语言把界面解析成英文。
Object.defineProperty(navigator, 'languages', {value: ['zh-CN'], configurable: true})

// jsdom 没有 matchMedia。默认按系统亮色处理；需要切换系统外观的测试自行替换。
if (!window.matchMedia) {
  window.matchMedia = (query: string) =>
    ({
      media: query,
      matches: false,
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
    }) as MediaQueryList
}

// jsdom 没有 ResizeObserver。测试中不触发尺寸变化，提供空实现即可。
if (!window.ResizeObserver) {
  window.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
}

afterEach(() => {
  cleanup()
})
