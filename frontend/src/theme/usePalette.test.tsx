import {act, renderHook} from '@testing-library/react'
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'
import {usePalette} from './usePalette'

/** 可以手动切换的系统外观，替代 jsdom 中缺失的 matchMedia。 */
function mockSystemAppearance(initialDark: boolean) {
  let dark = initialDark
  const listeners = new Set<() => void>()
  vi.stubGlobal('matchMedia', (query: string) => ({
    media: query,
    get matches() {
      return dark
    },
    addEventListener: (_type: string, listener: () => void) => listeners.add(listener),
    removeEventListener: (_type: string, listener: () => void) => listeners.delete(listener),
  }))
  return {
    setDark(value: boolean) {
      dark = value
      listeners.forEach((listener) => listener())
    },
  }
}

describe('usePalette', () => {
  let appearance: ReturnType<typeof mockSystemAppearance>

  beforeEach(() => {
    appearance = mockSystemAppearance(false)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('跟随系统时随系统外观立即切换', () => {
    const {result} = renderHook(() => usePalette('system'))
    expect(result.current).toBe('clean')
    expect(document.documentElement.dataset.palette).toBe('clean')

    act(() => appearance.setDark(true))
    expect(result.current).toBe('dark')
    expect(document.documentElement.dataset.palette).toBe('dark')

    act(() => appearance.setDark(false))
    expect(result.current).toBe('clean')
  })

  it('指定配色时忽略系统外观变化', () => {
    const {result} = renderHook(() => usePalette('paper'))
    act(() => appearance.setDark(true))
    expect(result.current).toBe('paper')
    expect(document.documentElement.dataset.palette).toBe('paper')
  })

  it('偏好改变时重新应用配色', () => {
    const {result, rerender} = renderHook(({preference}) => usePalette(preference), {
      initialProps: {preference: 'clean' as 'clean' | 'dark'},
    })
    rerender({preference: 'dark'})
    expect(result.current).toBe('dark')
    expect(document.documentElement.dataset.palette).toBe('dark')
  })
})
