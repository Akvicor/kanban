import {useCallback, useEffect, useMemo, useState, useSyncExternalStore, type ReactNode} from 'react'
import * as authApi from '../api/auth'
import {ApiError, ResultCode, isSessionError, onUnauthorized} from '../api/client'
import type {Me} from '../api/types'
import {fetchMe} from '../api/user'
import {applyLocalePreference, t} from '../i18n'
import {SyncEngine, syncUrl} from '../sync/engine'
import {SyncStore} from '../sync/store'
import {cachePreference, readCachedPreference, type PalettePreference} from '../theme/palette'
import {backoffDelay} from '../utils/backoff'
import {usePalette} from '../theme/usePalette'
import {addAccount, rememberLogin} from './accounts/actions'
import {reloadToHome} from './accounts/reload'
import {currentToken, exceedsLimit, expireToken, MAX_ACCOUNTS, setCurrentAccount, updateAccountProfile, watchOtherTabs} from './accounts/storage'
import {SessionContext, type SessionStatus, type SessionValue} from './context'
import {deviceName} from './deviceName'

/** 已登录时的会话：当前用户和其同步数据。 */
interface Signed {
  me: Me
  sync: SyncStore
}

const noSubscribe = () => () => {}

/**
 * 登录状态、当前用户和同步数据。当前账号来自本设备保存的账号列表（见 accounts/storage.ts）。
 * 打开页面时用当前账号的令牌取回当前用户，连不上服务端时保留令牌并自动重试；
 * 登录后建立同步连接。令牌失效（任何请求返回会话失效码，或同步连接被吊销）时，当前账号标记为已失效并回到未登录，
 * 登录页据此提示重新登录。其他标签页换了当前账号时刷新本页。
 * 配色以同步数据中的个人设置为准，并缓存到本设备，供下次打开页面时的首屏使用。
 */
export function SessionProvider({children}: {children: ReactNode}) {
  const [status, setStatus] = useState<SessionStatus>(() => (currentToken() ? 'loading' : 'anonymous'))
  const [signed, setSigned] = useState<Signed | null>(null)

  const signOut = useCallback(() => {
    setSigned(null)
    setStatus('anonymous')
  }, [])

  const signIn = useCallback((me: Me) => {
    updateAccountProfile(me.account)
    setSigned({me, sync: new SyncStore(me.settings)})
    setStatus('authenticated')
  }, [])

  // 请求层已把失效的账号标记为已失效，这里只回到未登录。
  useEffect(() => {
    onUnauthorized(signOut)
    return () => onUnauthorized(null)
  }, [signOut])

  useEffect(() => watchOtherTabs(reloadToHome), [])

  // 用当前账号的令牌确认登录状态。令牌失效时请求层已标记账号并调用 signOut，这里不再重试；
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
          if (cancelled || isSessionError(err)) return
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

  // 登录后建立同步连接，令牌被吊销、退出登录或换成另一个会话时关闭。
  useEffect(() => {
    const token = currentToken()
    if (!signed || !token) {
      return
    }
    const onRevoked = () => {
      expireToken(token)
      signOut()
    }
    const engine = new SyncEngine({url: syncUrl(window.location), token, store: signed.sync, onRevoked})
    engine.start()
    // 签发文件 Cookie。失败只影响图片等文件的加载，Cookie 通常已在切换或上次打开时写好，因此不打断会话。
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

  // 其他设备修改用户名或昵称后，同步更新本设备账号列表中保存的名称。
  const syncedAccount = useSyncExternalStore(store?.subscribe ?? noSubscribe, () => store?.getState().account)
  useEffect(() => {
    if (syncedAccount) {
      updateAccountProfile(syncedAccount)
    }
  }, [syncedAccount])

  // 登录：已登录时是添加另一个账号，切换过去并刷新页面；未登录时（包括重新登录已失效的账号）直接进入已登录。
  // 新账号会超过数量上限时吊销刚签发的令牌并报错。
  const login = useCallback(
    async (username: string, password: string) => {
      const result = await authApi.login(username, password, deviceName(navigator.userAgent))
      if (exceedsLimit(result.me.account.id)) {
        await authApi.logout(result.token).catch(() => {})
        // 登出请求会清除浏览器中的文件 Cookie，已登录时为当前账号重新签发。
        if (status === 'authenticated') await authApi.writeFileCookie().catch(() => {})
        throw new ApiError(ResultCode.Failed, t('account.limitReached', {max: MAX_ACCOUNTS}))
      }
      if (status === 'authenticated') {
        await addAccount(result)
        return
      }
      await rememberLogin(result)
      setCurrentAccount(result.me.account.id)
      signIn(result.me)
    },
    [status, signIn],
  )

  const value = useMemo<SessionValue>(
    () => ({status, me: signed?.me ?? null, sync: signed?.sync ?? null, login}),
    [status, signed, login],
  )
  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>
}
