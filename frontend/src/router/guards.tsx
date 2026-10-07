import type {ReactNode} from 'react'
import {Navigate, useLocation} from 'react-router-dom'
import {useSession} from '../session/context'
import {useT} from '../i18n'

/** 未登录时跳到登录页，登录后回到原来的地址；连不上服务端时显示正在重试。 */
export function RequireAuth({children}: {children: ReactNode}) {
  const t = useT()
  const {status} = useSession()
  const location = useLocation()
  if (status === 'loading') {
    return <div className="page-loading">{t('common.loading')}</div>
  }
  if (status === 'unreachable') {
    return <div className="page-loading">{t('router.reconnecting')}</div>
  }
  if (status === 'anonymous') {
    return <Navigate to="/login" replace state={{from: location.pathname}} />
  }
  return children
}

/** 非管理员访问管理页面时回到主页。 */
export function RequireAdmin({children}: {children: ReactNode}) {
  const {me} = useSession()
  if (me?.account.role !== 'admin') {
    return <Navigate to="/" replace />
  }
  return children
}
