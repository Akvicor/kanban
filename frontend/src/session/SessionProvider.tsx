import {useCallback, useEffect, useMemo, useState, useSyncExternalStore, type ReactNode} from 'react'
import * as authApi from '../api/auth'
import {ApiError, ResultCode, clearToken, getToken, onUnauthorized, setToken} from '../api/client'
import type {Me} from '../api/types'
import {fetchMe} from '../api/user'
import {applyLocalePreference} from '../i18n'
import {SyncEngine, syncUrl} from '../sync/engine'
import {SyncStore} from '../sync/store'
import {cachePreference, readCachedPreference, type PalettePreference} from '../theme/palette'
import {backoffDelay} from '../utils/backoff'
import {usePalette} from '../theme/usePalette'
import {SessionContext, type SessionStatus, type SessionValue} from './context'
import {deviceName} from './deviceName'

/** 已登录时的会话：当前用户和其同步数据。 */
interface Signed {
  me: Me
  sync: SyncStore
}

const noSubscribe = () => () => {}

/**
 * 登录状态、当前用户和同步数据。
 * 打开页面时用本设备保存的令牌取回当前用户，连不上服务端时保留令牌并自动重试；
 * 登录后建立同步连接，令牌被吊销或任何请求返回未登录时回到未登录状态。
 * 配色以同步数据中的个人设置为准，并缓存到本设备，供下次打开页面时的首屏使用。
 */
export function SessionProvider({children}: {children: ReactNode}) {
  const [status, setStatus] = useState<SessionStatus>(() => (getToken() ? 'loading' : 'anonymous'))
  const [signed, setSigned] = useState<Signed | null>(null)

  const signOut = useCallback(() => {
    clearToken()
    setSigned(null)
    setStatus('anonymous')
  }, [])

  const signIn = useCallback((me: Me) => {
    setSigned({me, sync: new SyncStore(me.settings)})
    setStatus('authenticated')
  }, [])

  useEffect(() => {
    onUnauthorized(signOut)
    return () => onUnauthorized(null)
  }, [signOut])

  // 用本设备的令牌确认登录状态。服务端返回未登录时，请求层已清除令牌并调用 signOut；
  // 连不上服务端或服务端出错时保留令牌，按 backoffDelay 自动重试，页面回到前台或网络恢复时立即重试。
  const confirming = status === 'loading' || status === 'unreachable'
  useEffect(() => {
    if (!confirming) {
      return
    }
    let cancelled = false
    let inFlight = false
    let attempts = 0
    let timer: ReturnType<typeof setTimeout> | undefined
    const attempt = () => {
      if (cancelled || inFlight) return
      clearTimeout(timer)
      inFlight = true
      fetchMe()
        .then((me) => !cancelled && signIn(me))
        .catch((err: unknown) => {
          if (cancelled || (err instanceof ApiError && err.code === ResultCode.Unauthorized)) return
          setStatus('unreachable')
          timer = setTimeout(attempt, backoffDelay(attempts))
          attempts += 1
        })
        .finally(() => {
          inFlight = false
        })
    }
    const onVisible = () => document.visibilityState === 'visible' && attempt()
    window.addEventListener('online', attempt)
    document.addEventListener('visibilitychange', onVisible)
    attempt()
    return () => {
      cancelled = true
      clearTimeout(timer)
      window.removeEventListener('online', attempt)
      document.removeEventListener('visibilitychange', onVisible)
    }
  }, [confirming, signIn])

  // 登录后建立同步连接，退出登录或换成另一个会话时关闭。
  useEffect(() => {
    const token = getToken()
    if (!signed || !token) {
      return
    }
    const engine = new SyncEngine({url: syncUrl(window.location), token, store: signed.sync, onRevoked: signOut})
    engine.start()
    // 写入文件 Cookie。失败只影响图片等文件的加载，Cookie 通常已在上次打开时写好，因此不打断会话。
    authApi.writeFileCookie().catch(() => {})
    return () => engine.stop()
  }, [signed, signOut])

  const store = signed?.sync
  const syncedPalette = useSyncExternalStore(store?.subscribe ?? noSubscribe, () => store?.getState().settings.palette)
  const preference: PalettePreference = syncedPalette ?? readCachedPreference()
  usePalette(preference)
  useEffect(() => {
    if (syncedPalette) {
      cachePreference(syncedPalette)
    }
  }, [syncedPalette])

  // 语言随同步设置应用；未登录或设置未加载时保持本设备缓存的语言（登录页等首屏使用）。
  const syncedLocale = useSyncExternalStore(store?.subscribe ?? noSubscribe, () => store?.getState().settings.locale)
  useEffect(() => {
    if (syncedLocale !== undefined) {
      applyLocalePreference(syncedLocale)
    }
  }, [syncedLocale])

  const login = useCallback(
    async (username: string, password: string) => {
      const result = await authApi.login(username, password, deviceName(navigator.userAgent))
      setToken(result.token)
      signIn(result.me)
    },
    [signIn],
  )

  const logout = useCallback(async () => {
    try {
      await authApi.logout()
    } finally {
      signOut()
    }
  }, [signOut])

  const value = useMemo<SessionValue>(
    () => ({status, me: signed?.me ?? null, sync: signed?.sync ?? null, login, logout}),
    [status, signed, login, logout],
  )
  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>
}
