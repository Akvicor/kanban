import {useLayoutEffect, useSyncExternalStore} from 'react'
import {applyPalette, resolvePalette, SYSTEM_DARK_QUERY, type Palette, type PalettePreference} from './palette'

function subscribeSystemDark(onChange: () => void): () => void {
  const query = window.matchMedia(SYSTEM_DARK_QUERY)
  query.addEventListener('change', onChange)
  return () => query.removeEventListener('change', onChange)
}

function getSystemDark(): boolean {
  return window.matchMedia(SYSTEM_DARK_QUERY).matches
}

/**
 * 按配色偏好把配色应用到页面，并返回实际配色。
 * 偏好为跟随系统时，系统外观变化后立即切换，不需要刷新。
 */
export function usePalette(preference: PalettePreference): Palette {
  const systemDark = useSyncExternalStore(subscribeSystemDark, getSystemDark, () => false)
  const palette = resolvePalette(preference, systemDark)
  useLayoutEffect(() => {
    applyPalette(palette)
  }, [palette])
  return palette
}
