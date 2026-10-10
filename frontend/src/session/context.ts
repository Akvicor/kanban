import {createContext, useContext, useSyncExternalStore} from 'react'
import type {Me} from '../api/types'
import type {SyncStore} from '../sync/store'
import {t} from '../i18n'

/**
 * 当前登录状态：正在确认、连不上服务端（保留令牌并自动重试）、未登录、已登录。
 * 只有服务端明确返回会话失效时才进入未登录；当前账号失效时它仍留在账号列表中，标记为已失效。
 */
export type SessionStatus = 'loading' | 'unreachable' | 'anonymous' | 'authenticated'

export interface SessionValue {
  status: SessionStatus
  /** 已登录时的当前用户，其余状态为 null。其中的个人设置只是登录时的值，之后以 sync 中的为准。 */
  me: Me | null
  /** 已登录时当前用户的同步数据，其余状态为 null。 */
  sync: SyncStore | null
  /** 登录。已登录时是添加另一个账号并切换过去（刷新页面）；切换、退出见 accounts/actions.ts。 */
  login: (username: string, password: string) => Promise<void>
}

export const SessionContext = createContext<SessionValue | null>(null)

export function useSession(): SessionValue {
  const value = useContext(SessionContext)
  if (!value) {
    throw new Error(t('error.useSession'))
  }
  return value
}

/** 返回已登录的当前用户。只能在需要登录的页面中使用。账号以同步数据为准，其他设备改名后会更新。 */
export function useMe(): Me {
  const {me, sync} = useSession()
  const account = useSyncExternalStore(
    sync?.subscribe ?? (() => () => {}),
    () => sync?.getState().account ?? null,
  )
  if (!me) {
    throw new Error(t('session.notSignedIn'))
  }
  return account ? {...me, account} : me
}
