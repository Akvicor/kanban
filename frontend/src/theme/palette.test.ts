import {afterEach, describe, expect, it} from 'vitest'
import {
  applyPalette,
  cachePreference,
  DEFAULT_PREFERENCE,
  PREFERENCE_STORAGE_KEY,
  readCachedPreference,
  resolvePalette,
} from './palette'

describe('resolvePalette', () => {
  it('跟随系统时亮色用清爽、暗色用暗夜', () => {
    expect(resolvePalette('system', false)).toBe('clean')
    expect(resolvePalette('system', true)).toBe('dark')
  })

  it('指定配色时不随系统外观变化', () => {
    for (const palette of ['clean', 'dark', 'paper'] as const) {
      expect(resolvePalette(palette, false)).toBe(palette)
      expect(resolvePalette(palette, true)).toBe(palette)
    }
  })
})

describe('配色偏好缓存', () => {
  afterEach(() => localStorage.clear())

  it('没有缓存或缓存无效时使用默认偏好', () => {
    expect(readCachedPreference()).toBe(DEFAULT_PREFERENCE)
    localStorage.setItem(PREFERENCE_STORAGE_KEY, 'unknown')
    expect(readCachedPreference()).toBe(DEFAULT_PREFERENCE)
  })

  it('读回已缓存的偏好', () => {
    cachePreference('system')
    expect(readCachedPreference()).toBe('system')
  })
})

describe('applyPalette', () => {
  it('在根元素上标记当前配色', () => {
    applyPalette('paper')
    expect(document.documentElement.dataset.palette).toBe('paper')
  })
})
